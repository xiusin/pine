package event

import "sync"

// Event 事件对象.
// Name 用于区分事件类型, Payload 携带事件数据.
type Event struct {
	Name    string
	Payload any
}

// Listener 事件监听器.
// 返回的 error 当前不参与流程控制, 仅为后续扩展保留.
type Listener func(event Event) error

// Dispatcher 事件分发器接口.
// 参考 Laravel Event 与 Spring ApplicationEventPublisher.
type Dispatcher interface {
	// Listen 注册监听器到指定事件名.
	Listen(name string, listener Listener)
	// ListenAny 注册通配符监听器, 监听所有事件.
	ListenAny(listener Listener)
	// Dispatch 同步分发事件, 所有监听器在当前 goroutine 依次执行.
	Dispatch(event Event)
	// DispatchAsync 异步分发事件, 每个监听器在独立 goroutine 中执行.
	DispatchAsync(event Event)
	// HasListeners 判断指定事件名是否存在监听器 (不包含通配符监听器).
	HasListeners(name string) bool
	// Flush 等待所有已派发的异步监听器执行完成.
	Flush() error
}

// dispatcher 默认事件分发器实现.
type dispatcher struct {
	mu           sync.RWMutex
	listeners    map[string][]Listener // 事件名 -> 监听器列表
	anyListeners []Listener            // 通配符监听器
	wg           sync.WaitGroup        // 用于异步等待
}

// NewDispatcher 创建新的默认事件分发器.
func NewDispatcher() Dispatcher {
	return &dispatcher{
		listeners: make(map[string][]Listener),
	}
}

// Listen 注册监听器到指定事件名.
func (d *dispatcher) Listen(name string, listener Listener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners[name] = append(d.listeners[name], listener)
}

// ListenAny 注册通配符监听器, 监听所有事件.
func (d *dispatcher) ListenAny(listener Listener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.anyListeners = append(d.anyListeners, listener)
}

// collectListeners 收集指定事件的全部监听器 (具体事件监听器在前, 通配符监听器在后).
// 返回独立的新切片, 调用方可在锁外安全遍历.
func (d *dispatcher) collectListeners(name string) []Listener {
	d.mu.RLock()
	defer d.mu.RUnlock()
	specific := d.listeners[name]
	all := make([]Listener, 0, len(specific)+len(d.anyListeners))
	all = append(all, specific...)
	all = append(all, d.anyListeners...)
	return all
}

// safeCall 安全调用监听器, recover 保护避免 panic 影响其他监听器.
func safeCall(listener Listener, event Event) {
	defer func() { _ = recover() }()
	_ = listener(event)
}

// Dispatch 同步分发事件.
// 监听器按注册顺序执行; 单个监听器 panic 不会影响其他监听器.
func (d *dispatcher) Dispatch(event Event) {
	for _, l := range d.collectListeners(event.Name) {
		safeCall(l, event)
	}
}

// HasListeners 判断指定事件名是否存在监听器 (不包含通配符监听器).
func (d *dispatcher) HasListeners(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.listeners[name]) > 0
}

// 包级默认 dispatcher
var defaultDispatcher = NewDispatcher()

// Listen 在默认 dispatcher 上注册监听器到指定事件名.
func Listen(name string, listener Listener) {
	defaultDispatcher.Listen(name, listener)
}

// ListenAny 在默认 dispatcher 上注册通配符监听器, 监听所有事件.
func ListenAny(listener Listener) {
	defaultDispatcher.ListenAny(listener)
}

// Dispatch 在默认 dispatcher 上同步分发事件.
func Dispatch(event Event) {
	defaultDispatcher.Dispatch(event)
}

// DispatchAsync 在默认 dispatcher 上异步分发事件.
func DispatchAsync(event Event) {
	defaultDispatcher.DispatchAsync(event)
}

// HasListeners 判断默认 dispatcher 上指定事件名是否存在监听器.
func HasListeners(name string) bool {
	return defaultDispatcher.HasListeners(name)
}

// Flush 等待默认 dispatcher 上所有异步监听器执行完成.
func Flush() error {
	return defaultDispatcher.Flush()
}
