package event

// DispatchAsync 异步分发事件.
// 每个监听器在独立 goroutine 中执行, 监听器 panic 不会影响其他监听器.
// 可通过 Flush 等待所有异步监听器执行完成.
func (d *dispatcher) DispatchAsync(event Event) {
	for _, l := range d.collectListeners(event.Name) {
		d.wg.Add(1)
		go func(listener Listener, ev Event) {
			defer d.wg.Done()
			safeCall(listener, ev)
		}(l, event)
	}
}

// Flush 等待所有已派发的异步监听器执行完成.
func (d *dispatcher) Flush() error {
	d.wg.Wait()
	return nil
}
