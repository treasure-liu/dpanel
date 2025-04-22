package notice

import (
	"context"
	"encoding/json"
	"fmt"
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
	filterArgs := filters.NewArgs()
	filterArgs.Add("type", "container")

	// 使用events.ListOptions类型
	eventCh, errCh := docker.Sdk.Client.Events(ctx, events.ListOptions{
		Filters: filterArgs,
	})

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			if err != nil && err != io.EOF {
				slog.Error("监控容器事件出错", "error", err)
			}
			return
		case event := <-eventCh:
			// 处理容器事件
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
				// 获取镜像信息
				imageName := event.Actor.Attributes["image"]
				if imageName == "" {
					// 尝试通过容器ID查询获取镜像信息
					inspectData, err := docker.Sdk.Client.ContainerInspect(context.Background(), containerId)
					if err == nil && inspectData.Config != nil {
						imageName = inspectData.Config.Image
					}
				}
				
				if imageName != "" {
					alertMsg = fmt.Sprintf("容器异常退出: %s (ID: %s)，镜像: %s，退出码: %s", containerName, containerId[:12], imageName, exitCode)
				} else {
					alertMsg = fmt.Sprintf("容器异常退出: %s (ID: %s)，退出码: %s", containerName, containerId[:12], exitCode)
				}
				shouldAlert = true
			}
		}
	case "stop", "kill":
		if config.AlertOnStop {
			// 获取镜像信息
			imageName := event.Actor.Attributes["image"]
			if imageName == "" {
				// 尝试通过容器ID查询获取镜像信息
				inspectData, err := docker.Sdk.Client.ContainerInspect(context.Background(), containerId)
				if err == nil && inspectData.Config != nil {
					imageName = inspectData.Config.Image
				}
			}
			
			if imageName != "" {
				alertMsg = fmt.Sprintf("容器已停止: %s (ID: %s)，镜像: %s", containerName, containerId[:12], imageName)
			} else {
				alertMsg = fmt.Sprintf("容器已停止: %s (ID: %s)", containerName, containerId[:12])
			}
			shouldAlert = true
		}
	case "oom":
		if config.AlertOnOOM {
			// 获取镜像信息
			imageName := event.Actor.Attributes["image"]
			if imageName == "" {
				// 尝试通过容器ID查询获取镜像信息
				inspectData, err := docker.Sdk.Client.ContainerInspect(context.Background(), containerId)
				if err == nil && inspectData.Config != nil {
					imageName = inspectData.Config.Image
				}
			}
			
			if imageName != "" {
				alertMsg = fmt.Sprintf("容器内存溢出(OOM): %s (ID: %s)，镜像: %s", containerName, containerId[:12], imageName)
			} else {
				alertMsg = fmt.Sprintf("容器内存溢出(OOM): %s (ID: %s)", containerName, containerId[:12])
			}
			shouldAlert = true
		}
	}

	if shouldAlert && alertMsg != "" {
		slog.Warn("检测到容器异常", 
			"type", event.Action,
			"container", containerName,
			"id", containerId[:12],
			"alertMsg", alertMsg)
			
		// 发送系统通知
		_ = Message{}.Error("容器告警", alertMsg)
		slog.Info("已发送系统通知", "container", containerName, "type", "系统通知")
		
		// 发送钉钉通知
		err := SendDingtalkMessage("容器告警", alertMsg, TypeError)
		if err != nil {
			analyzeAndLogError("钉钉", err, containerName, string(event.Action))
		} else if GetDingtalkConfig().IsEnabled {
			slog.Info("已发送告警通知", "container", containerName, "type", "钉钉")
		}
		
		// 发送邮件通知
		err = SendEmailMessage("容器告警", alertMsg, TypeError)
		if err != nil {
			analyzeAndLogError("邮件", err, containerName, string(event.Action))
		} else if GetEmailConfig().IsEnabled {
			slog.Info("已发送告警通知", "container", containerName, "type", "邮件")
		}
		
		// 发送飞书通知
		err = SendFeishuMessage("容器告警", alertMsg, TypeError)
		if err != nil {
			analyzeAndLogError("飞书", err, containerName, string(event.Action))
		} else if GetFeishuConfig().IsEnabled {
			slog.Info("已发送告警通知", "container", containerName, "type", "飞书")
		}
		
		// 发送企业微信通知
		err = SendWechatWorkMessage("容器告警", alertMsg, TypeError)
		if err != nil {
			analyzeAndLogError("企业微信", err, containerName, string(event.Action))
		} else if GetWechatWorkConfig().IsEnabled {
			slog.Info("已发送告警通知", "container", containerName, "type", "企业微信")
		}
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
			// 获取镜像信息
			imageName := inspectData.Config.Image
			
			alertMsg := fmt.Sprintf("容器健康检查失败: %s (ID: %s)", containerName, containerId[:12])
			if imageName != "" {
				alertMsg = fmt.Sprintf("容器健康检查失败: %s (ID: %s)，镜像: %s", containerName, containerId[:12], imageName)
			}
			
			// 添加最近的健康检查日志
			if len(inspectData.State.Health.Log) > 0 {
				lastLog := inspectData.State.Health.Log[len(inspectData.State.Health.Log)-1]
				alertMsg += fmt.Sprintf("\n检查输出: %s", lastLog.Output)
			}
			
			// 发送系统通知
			_ = Message{}.Error("容器健康告警", alertMsg)
			slog.Info("已发送系统通知", "container", containerName, "type", "系统通知")
			
			// 发送钉钉通知
			err := SendDingtalkMessage("容器健康告警", alertMsg, TypeError)
			if err != nil {
				analyzeAndLogError("钉钉", err, containerName, "健康检查失败")
			} else if GetDingtalkConfig().IsEnabled {
				slog.Info("已发送告警通知", "container", containerName, "type", "钉钉")
			}
			
			// 发送邮件通知
			err = SendEmailMessage("容器健康告警", alertMsg, TypeError)
			if err != nil {
				analyzeAndLogError("邮件", err, containerName, "健康检查失败")
			} else if GetEmailConfig().IsEnabled {
				slog.Info("已发送告警通知", "container", containerName, "type", "邮件")
			}
			
			// 发送飞书通知
			err = SendFeishuMessage("容器健康告警", alertMsg, TypeError)
			if err != nil {
				analyzeAndLogError("飞书", err, containerName, "健康检查失败")
			} else if GetFeishuConfig().IsEnabled {
				slog.Info("已发送告警通知", "container", containerName, "type", "飞书")
			}
			
			// 发送企业微信通知
			err = SendWechatWorkMessage("容器健康告警", alertMsg, TypeError)
			if err != nil {
				analyzeAndLogError("企业微信", err, containerName, "健康检查失败")
			} else if GetWechatWorkConfig().IsEnabled {
				slog.Info("已发送告警通知", "container", containerName, "type", "企业微信")
			}
		}
	}
}

// analyzeAndLogError 分析通知发送错误并记录详细信息
func analyzeAndLogError(noticeType string, err error, containerName string, eventType string) {
	errMsg := err.Error()
	
	// 创建基本的错误日志属性
	errAttrs := []any{
		"error", errMsg,
		"container", containerName,
		"event", eventType,
	}
	
	// 根据不同通知类型和错误模式分析可能的原因
	var reason string
	switch noticeType {
	case "钉钉":
		if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "timeout") {
			reason = "网络连接问题，请检查网络连接或代理设置"
		} else if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "unauthorized") {
			reason = "Webhook地址无效或已过期，请更新钉钉机器人配置"
		} else if strings.Contains(errMsg, "sign") || strings.Contains(errMsg, "signature") {
			reason = "签名验证失败，请检查Secret配置是否正确"
		} else if strings.Contains(errMsg, "429") || strings.Contains(errMsg, "too many requests") {
			reason = "发送频率超过钉钉限制，请降低发送频率"
		}
		
	case "邮件":
		if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "timeout") {
			reason = "无法连接到SMTP服务器，请检查服务器地址和端口"
		} else if strings.Contains(errMsg, "authentication failed") || strings.Contains(errMsg, "auth") {
			reason = "SMTP认证失败，请检查用户名和密码"
		} else if strings.Contains(errMsg, "tls") || strings.Contains(errMsg, "SSL") {
			reason = "SSL/TLS连接问题，请检查EMAIL_USE_SSL设置是否与服务器匹配"
		} else if strings.Contains(errMsg, "recipient") || strings.Contains(errMsg, "sender") {
			reason = "发件人或收件人地址无效，请检查EMAIL_FROM和EMAIL_TO配置"
		}
		
	case "飞书":
		if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "timeout") {
			reason = "网络连接问题，请检查网络连接或代理设置"
		} else if strings.Contains(errMsg, "sign") || strings.Contains(errMsg, "token") {
			reason = "Webhook地址无效或签名错误，请检查配置"
		} else if strings.Contains(errMsg, "429") || strings.Contains(errMsg, "limit") {
			reason = "发送频率超过飞书限制，请降低发送频率"
		}
		
	case "企业微信":
		if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "timeout") {
			reason = "网络连接问题，请检查网络连接或代理设置"
		} else if strings.Contains(errMsg, "40014") || strings.Contains(errMsg, "invalid") {
			reason = "Webhook地址无效，请检查key参数"
		} else if strings.Contains(errMsg, "45009") || strings.Contains(errMsg, "limit") {
			reason = "企业微信API调用频率限制，请降低发送频率"
		}
	}
	
	// 如果找到可能的原因，添加到日志属性中
	if reason != "" {
		errAttrs = append(errAttrs, "可能原因", reason)
	}
	
	// 记录详细的错误信息
	slog.Error(noticeType+"通知发送失败", errAttrs...)
	
	// 记录调试提示
	slog.Debug("通知发送问题排查建议", 
		"type", noticeType,
		"tip", "请检查"+noticeType+"配置，并确认相关服务运行正常",
		"docker_cmd", "docker logs dpanel | grep \""+noticeType+"通知发送失败\"")
} 