package notice

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/donknap/dpanel/common/service/docker"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// ContainerAlertConfig 容器告警配置
type ContainerAlertConfig struct {
	Enabled            bool     // 是否启用容器告警
	MonitorContainers  []string // 监控的容器名称或ID（为空表示监控所有容器）
	ExcludeContainers  []string // 排除的容器名称或ID
	AlertOnStop        bool     // 容器停止时是否告警
	AlertOnDie         bool     // 容器异常退出时是否告警
	AlertOnOOM         bool     // 容器OOM时是否告警
	AlertOnHealthCheck bool     // 容器健康检查失败时是否告警
	CheckInterval      int      // 检查间隔（秒）
}

var (
	containerAlertConfig ContainerAlertConfig
	alertConfigMutex     sync.Mutex
	containerAlertCtx    context.Context
	cancelAlertMonitor   context.CancelFunc
)

// LoadContainerAlertConfig 从环境变量加载容器告警配置
func LoadContainerAlertConfig() ContainerAlertConfig {
	alertConfigMutex.Lock()
	defer alertConfigMutex.Unlock()

	enabled := strings.ToLower(os.Getenv("CONTAINER_ALERT_ENABLED")) == "true"
	
	var monitorContainers []string
	if monitorStr := os.Getenv("CONTAINER_ALERT_MONITOR"); monitorStr != "" {
		monitorContainers = strings.Split(monitorStr, ",")
	}
	
	var excludeContainers []string
	if excludeStr := os.Getenv("CONTAINER_ALERT_EXCLUDE"); excludeStr != "" {
		excludeContainers = strings.Split(excludeStr, ",")
	}
	
	alertOnStop := strings.ToLower(os.Getenv("CONTAINER_ALERT_ON_STOP")) == "true"
	alertOnDie := strings.ToLower(os.Getenv("CONTAINER_ALERT_ON_DIE")) != "false" // 默认开启
	alertOnOOM := strings.ToLower(os.Getenv("CONTAINER_ALERT_ON_OOM")) != "false" // 默认开启
	alertOnHealthCheck := strings.ToLower(os.Getenv("CONTAINER_ALERT_ON_HEALTH")) != "false" // 默认开启
	
	checkInterval := 60 // 默认60秒
	if intervalStr := os.Getenv("CONTAINER_ALERT_INTERVAL"); intervalStr != "" {
		if interval, err := json.Number(intervalStr).Int64(); err == nil && interval > 0 {
			checkInterval = int(interval)
		}
	}
	
	containerAlertConfig = ContainerAlertConfig{
		Enabled:            enabled,
		MonitorContainers:  monitorContainers,
		ExcludeContainers:  excludeContainers,
		AlertOnStop:        alertOnStop,
		AlertOnDie:         alertOnDie,
		AlertOnOOM:         alertOnOOM,
		AlertOnHealthCheck: alertOnHealthCheck,
		CheckInterval:      checkInterval,
	}
	
	return containerAlertConfig
}

// InitContainerAlertMonitor 初始化容器监控
func InitContainerAlertMonitor() {
	config := LoadContainerAlertConfig()
	if !config.Enabled {
		slog.Info("容器告警功能未启用")
		return
	}

	if containerAlertCtx != nil && cancelAlertMonitor != nil {
		cancelAlertMonitor() // 取消旧的监控
	}

	containerAlertCtx, cancelAlertMonitor = context.WithCancel(context.Background())

	// 启动容器事件监听
	go monitorContainerEvents(containerAlertCtx)

	// 启动容器健康状态检查
	if config.AlertOnHealthCheck {
		go monitorContainerHealth(containerAlertCtx, config.CheckInterval)
	}

	slog.Info("容器告警功能已启用", 
		"monitorContainers", config.MonitorContainers, 
		"excludeContainers", config.ExcludeContainers,
		"alertOnStop", config.AlertOnStop,
		"alertOnDie", config.AlertOnDie,
		"alertOnOOM", config.AlertOnOOM,
		"alertOnHealthCheck", config.AlertOnHealthCheck,
		"checkInterval", config.CheckInterval)
}

// 监听容器事件
func monitorContainerEvents(ctx context.Context) {
	// 创建过滤器，只监听容器事件
	filter := filters.NewArgs()
	filter.Add("type", "container")
	
	// 获取事件流
	eventChan, errChan := docker.Sdk.Client.Events(ctx, types.EventsOptions{
		Filters: filter,
	})

	// 处理事件
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errChan:
			if err != nil && err != io.EOF {
				slog.Error("容器事件监听错误", "error", err)
			}
			return
		case event := <-eventChan:
			handleContainerEvent(event)
		}
	}
}

// 处理容器事件
func handleContainerEvent(event events.Message) {
	config := containerAlertConfig

	// 检查是否需要监控此容器
	containerName := event.Actor.Attributes["name"]
	containerId := event.ID

	// 如果配置了特定容器，检查当前容器是否在监控列表中
	if len(config.MonitorContainers) > 0 {
		inMonitorList := false
		for _, name := range config.MonitorContainers {
			if name == containerName || strings.HasPrefix(containerId, name) {
				inMonitorList = true
				break
			}
		}
		if !inMonitorList {
			return
		}
	}

	// 检查容器是否在排除列表中
	for _, name := range config.ExcludeContainers {
		if name == containerName || strings.HasPrefix(containerId, name) {
			return
		}
	}

	var alertMsg string
	var shouldAlert bool

	switch event.Action {
	case "die":
		if config.AlertOnDie {
			exitCode := event.Actor.Attributes["exitCode"]
			if exitCode != "0" {
				alertMsg = fmt.Sprintf("容器异常退出: %s (ID: %s)，退出码: %s", containerName, containerId[:12], exitCode)
				shouldAlert = true
			}
		}
	case "stop", "kill":
		if config.AlertOnStop {
			alertMsg = fmt.Sprintf("容器已停止: %s (ID: %s)", containerName, containerId[:12])
			shouldAlert = true
		}
	case "oom":
		if config.AlertOnOOM {
			alertMsg = fmt.Sprintf("容器内存溢出(OOM): %s (ID: %s)", containerName, containerId[:12])
			shouldAlert = true
		}
	}

	if shouldAlert && alertMsg != "" {
		// 发送系统通知
		_ = Message{}.Error("容器告警", alertMsg)
		
		// 发送钉钉通知
		_ = SendDingtalkMessage("容器告警", alertMsg, TypeError)
	}
}

// 定期检查容器健康状态
func monitorContainerHealth(ctx context.Context, intervalSeconds int) {
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkContainersHealth()
		}
	}
}

// 检查所有容器的健康状态
func checkContainersHealth() {
	config := containerAlertConfig
	if !config.AlertOnHealthCheck {
		return
	}

	// 获取所有运行中的容器
	containers, err := docker.Sdk.Client.ContainerList(context.Background(), container.ListOptions{All: false})
	if err != nil {
		slog.Error("获取容器列表失败", "error", err)
		return
	}

	for _, c := range containers {
		containerName := c.Names[0]
		if len(containerName) > 0 && containerName[0] == '/' {
			containerName = containerName[1:]
		}
		containerId := c.ID

		// 检查是否需要监控此容器
		if len(config.MonitorContainers) > 0 {
			inMonitorList := false
			for _, name := range config.MonitorContainers {
				if name == containerName || strings.HasPrefix(containerId, name) {
					inMonitorList = true
					break
				}
			}
			if !inMonitorList {
				continue
			}
		}

		// 检查容器是否在排除列表中
		isExcluded := false
		for _, name := range config.ExcludeContainers {
			if name == containerName || strings.HasPrefix(containerId, name) {
				isExcluded = true
				break
			}
		}
		if isExcluded {
			continue
		}

		// 获取容器详细信息
		inspectData, err := docker.Sdk.Client.ContainerInspect(context.Background(), c.ID)
		if err != nil {
			continue
		}

		// 检查健康状态
		if inspectData.State.Health != nil && inspectData.State.Health.Status == "unhealthy" {
			alertMsg := fmt.Sprintf("容器健康检查失败: %s (ID: %s)", containerName, containerId[:12])
			
			// 添加最近的健康检查日志
			if len(inspectData.State.Health.Log) > 0 {
				lastLog := inspectData.State.Health.Log[len(inspectData.State.Health.Log)-1]
				alertMsg += fmt.Sprintf("\n检查输出: %s", lastLog.Output)
			}
			
			// 发送系统通知
			_ = Message{}.Error("容器健康告警", alertMsg)
			
			// 发送钉钉通知
			_ = SendDingtalkMessage("容器健康告警", alertMsg, TypeError)
		}
	}
} 