package notice

import (
	"context"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/donknap/dpanel/common/service/docker"
)

// EventMonitor 容器事件监控器，处理Docker API版本兼容性
type EventMonitor struct {
	ctx        context.Context
	filterArgs filters.Args
	eventCh    chan interface{}
	errCh      chan error
	cancelFunc context.CancelFunc
}

// NewEventMonitor 创建新的事件监控器
func NewEventMonitor(ctx context.Context) *EventMonitor {
	childCtx, cancel := context.WithCancel(ctx)
	return &EventMonitor{
		ctx:        childCtx,
		filterArgs: filters.NewArgs(),
		eventCh:    make(chan interface{}),
		errCh:      make(chan error),
		cancelFunc: cancel,
	}
}

// AddTypeFilter 添加类型过滤器
func (m *EventMonitor) AddTypeFilter(typeValue string) *EventMonitor {
	m.filterArgs.Add("type", typeValue)
	return m
}

// Start 开始监控
func (m *EventMonitor) Start() (chan interface{}, chan error) {
	// 使用events.ListOptions类型
	events, errs := docker.Sdk.Client.Events(m.ctx, events.ListOptions{
		Filters: m.filterArgs,
	})

	// 转发事件和错误
	go func() {
		defer close(m.eventCh)
		defer close(m.errCh)
		for {
			select {
			case <-m.ctx.Done():
				return
			case err := <-errs:
				m.errCh <- err
				return
			case event := <-events:
				m.eventCh <- event
			}
		}
	}()

	return m.eventCh, m.errCh
}

// Stop 停止监控
func (m *EventMonitor) Stop() {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
}

// MonitorEvents 监控Docker事件的兼容函数
func MonitorEvents(ctx context.Context, filterType string) (<-chan events.Message, <-chan error) {
	filter := filters.NewArgs()
	filter.Add("type", filterType)
	
	// 使用events.ListOptions类型
	return docker.Sdk.Client.Events(ctx, events.ListOptions{
		Filters: filter,
	})
} 