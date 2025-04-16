package notice

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/donknap/dpanel/common/service/docker"
)

// EventMonitor 容器事件监控器，处理Docker API版本兼容性
type EventMonitor struct {
	ctx        context.Context
	filter     filters.Args
	eventChan  chan interface{}
	errorChan  chan error
	cancelFunc context.CancelFunc
}

// NewEventMonitor 创建新的事件监控器
func NewEventMonitor(ctx context.Context) *EventMonitor {
	childCtx, cancel := context.WithCancel(ctx)
	return &EventMonitor{
		ctx:        childCtx,
		filter:     filters.NewArgs(),
		eventChan:  make(chan interface{}),
		errorChan:  make(chan error),
		cancelFunc: cancel,
	}
}

// AddTypeFilter 添加类型过滤器
func (m *EventMonitor) AddTypeFilter(typeValue string) *EventMonitor {
	m.filter.Add("type", typeValue)
	return m
}

// Start 开始监控
func (m *EventMonitor) Start() (chan interface{}, chan error) {
	// 使用docker sdk的Events方法，处理版本兼容性问题
	events, errs := docker.Sdk.Client.Events(m.ctx, types.EventsOptions{
		Filters: m.filter,
	})

	// 转发事件和错误
	go func() {
		defer close(m.eventChan)
		defer close(m.errorChan)
		for {
			select {
			case <-m.ctx.Done():
				return
			case err := <-errs:
				m.errorChan <- err
				return
			case event := <-events:
				m.eventChan <- event
			}
		}
	}()

	return m.eventChan, m.errorChan
}

// Stop 停止监控
func (m *EventMonitor) Stop() {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
}

// MonitorEvents 监控Docker事件的兼容函数，避免直接使用可能变化的Docker API类型
func MonitorEvents(ctx context.Context, filterType string) (<-chan events.Message, <-chan error) {
	filter := filters.NewArgs()
	filter.Add("type", filterType)
	
	// 调用Docker SDK的事件监控，注意返回类型
	return docker.Sdk.Client.Events(ctx, types.EventsOptions{
		Filters: filter,
	})
} 