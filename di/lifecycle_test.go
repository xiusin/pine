// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package di

import (
	"errors"
	"sync"
	"testing"
)

// fullLifecycleBean 实现全部生命周期接口，用于验证回调顺序与触发。
type fullLifecycleBean struct {
	mu        sync.Mutex
	name      string
	booted    bool
	shutdown  bool
	callOrder []string
}

func (b *fullLifecycleBean) SetBeanName(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.name = name
	b.callOrder = append(b.callOrder, "SetBeanName")
}

func (b *fullLifecycleBean) Boot() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.booted = true
	b.callOrder = append(b.callOrder, "Boot")
	return nil
}

func (b *fullLifecycleBean) Shutdown() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.shutdown = true
	b.callOrder = append(b.callOrder, "Shutdown")
	return nil
}

func (b *fullLifecycleBean) calls() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, len(b.callOrder))
	copy(out, b.callOrder)
	return out
}

// recordingProcessor 记录 BeforeInit/AfterInit 调用的 bean 名。
type recordingProcessor struct {
	mu          sync.Mutex
	beforeCalls []string
	afterCalls  []string
}

func (p *recordingProcessor) BeforeInit(name string, bean any) (any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.beforeCalls = append(p.beforeCalls, name)
	return bean, nil
}

func (p *recordingProcessor) AfterInit(name string, bean any) (any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.afterCalls = append(p.afterCalls, name)
	return bean, nil
}

// replaceProcessor 在 BeforeInit 阶段替换 bean。
type replaceProcessor struct {
	replacement any
}

func (p *replaceProcessor) BeforeInit(name string, bean any) (any, error) {
	return p.replacement, nil
}

func (p *replaceProcessor) AfterInit(name string, bean any) (any, error) {
	return bean, nil
}

// errBootBean 的 Boot 返回错误。
type errBootBean struct{ err error }

func (b *errBootBean) Boot() error { return b.err }

// errShutdownBean 的 Shutdown 返回错误。
type errShutdownBean struct{ err error }

func (b *errShutdownBean) Shutdown() error { return b.err }

// TestLifecycle_BootAndShutdown 验证单例创建后调用 Boot、注册 Shutdownable，
// 且容器 Shutdown 时触发 bean 的 Shutdown。
func TestLifecycle_BootAndShutdown(t *testing.T) {
	b := NewBuilder()
	bean := &fullLifecycleBean{}
	b.Singleton("lifecycle.bean", func(_ AbstractBuilder) (any, error) {
		return bean, nil
	})

	got, err := b.Get("lifecycle.bean")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != bean {
		t.Fatal("expected same bean instance")
	}
	if !bean.booted {
		t.Error("Boot() should be called after singleton creation")
	}
	if bean.name != "lifecycle.bean" {
		t.Errorf("SetBeanName not called or wrong name: %q", bean.name)
	}

	if err := b.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	if !bean.shutdown {
		t.Error("bean.Shutdown() should be called on container Shutdown")
	}
}

// TestLifecycle_CallOrder 验证回调执行顺序：
//
//	BeforeInit -> SetBeanName -> Boot -> AfterInit -> (Shutdown 在容器关闭时)
func TestLifecycle_CallOrder(t *testing.T) {
	b := NewBuilder()
	bean := &fullLifecycleBean{}
	proc := &recordingProcessor{}
	b.RegisterPostProcessor(proc)
	b.Singleton("ordered.bean", func(_ AbstractBuilder) (any, error) {
		return bean, nil
	})

	if _, err := b.Get("ordered.bean"); err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	calls := bean.calls()
	// BeforeInit / AfterInit 不记录在 bean.callOrder 中，bean 侧仅见 SetBeanName -> Boot
	want := []string{"SetBeanName", "Boot"}
	if len(calls) < len(want) {
		t.Fatalf("expected at least %d calls, got %d: %v", len(want), len(calls), calls)
	}
	for i, w := range want {
		if calls[i] != w {
			t.Errorf("call %d: want %s, got %s (full: %v)", i, w, calls[i], calls)
		}
	}

	proc.mu.Lock()
	defer proc.mu.Unlock()
	if len(proc.beforeCalls) != 1 || proc.beforeCalls[0] != "ordered.bean" {
		t.Errorf("BeforeInit not called correctly: %v", proc.beforeCalls)
	}
	if len(proc.afterCalls) != 1 || proc.afterCalls[0] != "ordered.bean" {
		t.Errorf("AfterInit not called correctly: %v", proc.afterCalls)
	}
}

// TestLifecycle_PostProcessorReplacesBean 验证 BeforeInit 返回值可替换 bean，
// 后续 SetBeanName / Boot 作用于替换后的实例。
func TestLifecycle_PostProcessorReplacesBean(t *testing.T) {
	b := NewBuilder()
	replaced := &fullLifecycleBean{}
	b.Singleton("replaced.bean", func(_ AbstractBuilder) (any, error) {
		return &fullLifecycleBean{}, nil // 原始 bean 会被替换
	})
	b.RegisterPostProcessor(&replaceProcessor{replacement: replaced})

	got, err := b.Get("replaced.bean")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != replaced {
		t.Fatal("post-processor should replace bean with replacement")
	}
	if !replaced.booted {
		t.Error("Boot() should be called on the replacement bean")
	}
	if replaced.name != "replaced.bean" {
		t.Errorf("SetBeanName should be called on replacement: %q", replaced.name)
	}
}

// TestLifecycle_BootTriggersLazySingletons 验证 Boot() 触发懒加载所有单例。
func TestLifecycle_BootTriggersLazySingletons(t *testing.T) {
	b := NewBuilder()
	bean := &fullLifecycleBean{}
	created := false
	b.Singleton("lazy.bean", func(_ AbstractBuilder) (any, error) {
		created = true
		return bean, nil
	})

	if created {
		t.Fatal("singleton should not be created before Boot")
	}
	if err := b.Boot(); err != nil {
		t.Fatalf("Boot failed: %v", err)
	}
	if !created {
		t.Error("Boot should trigger singleton creation")
	}
	if !bean.booted {
		t.Error("Boot should trigger bean Boot()")
	}

	// bootOnce: 再次 Boot 不应重复创建
	if err := b.Boot(); err != nil {
		t.Fatalf("second Boot failed: %v", err)
	}
}

// TestLifecycle_BootError 验证 Boot 回调返回错误时向上传递。
func TestLifecycle_BootError(t *testing.T) {
	b := NewBuilder()
	bootErr := errors.New("boot failed")
	b.Singleton("err.boot", func(_ AbstractBuilder) (any, error) {
		return &errBootBean{err: bootErr}, nil
	})

	_, err := b.Get("err.boot")
	if !errors.Is(err, bootErr) {
		t.Errorf("expected boot error %v, got %v", bootErr, err)
	}
}

// TestLifecycle_ShutdownError 验证 Shutdown 返回错误时向上传递，且仍调用所有 bean。
func TestLifecycle_ShutdownError(t *testing.T) {
	b := NewBuilder()
	shutdownErr := errors.New("shutdown failed")
	okBean := &fullLifecycleBean{}
	b.Singleton("ok.bean", func(_ AbstractBuilder) (any, error) {
		return okBean, nil
	})
	b.Singleton("err.shutdown", func(_ AbstractBuilder) (any, error) {
		return &errShutdownBean{err: shutdownErr}, nil
	})

	if _, err := b.Get("ok.bean"); err != nil {
		t.Fatalf("Get ok.bean failed: %v", err)
	}
	if _, err := b.Get("err.shutdown"); err != nil {
		t.Fatalf("Get err.shutdown failed: %v", err)
	}

	err := b.Shutdown()
	if !errors.Is(err, shutdownErr) {
		t.Errorf("expected shutdown error %v, got %v", shutdownErr, err)
	}
	if !okBean.shutdown {
		t.Error("all beans should still be shut down even when one returns error")
	}
}

// TestLifecycle_NonSingletonNoLifecycle 验证非单例 bean 不触发生命周期回调。
func TestLifecycle_NonSingletonNoLifecycle(t *testing.T) {
	b := NewBuilder()
	bean := &fullLifecycleBean{}
	b.Bind("proto.bean", func(_ AbstractBuilder) (any, error) {
		return bean, nil
	})

	got, err := b.Get("proto.bean")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != bean {
		t.Fatal("expected same bean")
	}
	if bean.booted {
		t.Error("non-singleton should not trigger Boot()")
	}
	if bean.name != "" {
		t.Error("non-singleton should not trigger SetBeanName()")
	}

	if err := b.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	if bean.shutdown {
		t.Error("non-singleton should not be registered for Shutdown")
	}
}

// TestLifecycle_ShutdownIdempotent 验证多次 Shutdown 仅触发一次 bean.Shutdown。
func TestLifecycle_ShutdownIdempotent(t *testing.T) {
	b := NewBuilder()
	bean := &fullLifecycleBean{}
	b.Singleton("idempotent.bean", func(_ AbstractBuilder) (any, error) {
		return bean, nil
	})
	if _, err := b.Get("idempotent.bean"); err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if err := b.Shutdown(); err != nil {
		t.Fatalf("first Shutdown failed: %v", err)
	}
	if !bean.shutdown {
		t.Fatal("first Shutdown should call bean.Shutdown()")
	}
	// 第二次 Shutdown 不应再次调用
	if err := b.Shutdown(); err != nil {
		t.Fatalf("second Shutdown failed: %v", err)
	}
	calls := bean.calls()
	shutdownCount := 0
	for _, c := range calls {
		if c == "Shutdown" {
			shutdownCount++
		}
	}
	if shutdownCount != 1 {
		t.Errorf("expected exactly 1 Shutdown call, got %d", shutdownCount)
	}
}

// TestLifecycle_BootSkipsParamsDefinition 验证 Boot 跳过参数型 Definition（factory 为 nil）。
func TestLifecycle_BootSkipsParamsDefinition(t *testing.T) {
	b := NewBuilder()
	// 参数型 Definition，factory 为 nil，Boot 不应尝试解析它
	b.SetWithParams("params.bean", func(_ AbstractBuilder, params ...any) (any, error) {
		return params[0], nil
	})
	// 正常单例
	bean := &fullLifecycleBean{}
	b.Singleton("normal.bean", func(_ AbstractBuilder) (any, error) {
		return bean, nil
	})

	if err := b.Boot(); err != nil {
		t.Fatalf("Boot should skip params definitions, got error: %v", err)
	}
	if !bean.booted {
		t.Error("Boot should still trigger normal singleton Boot()")
	}
}

// TestLifecycle_BeanNameAwareOnly 验证仅实现 BeanNameAware 的 bean。
func TestLifecycle_BeanNameAwareOnly(t *testing.T) {
	b := NewBuilder()
	bean := &nameOnlyBean{}
	b.Singleton("name.only", func(_ AbstractBuilder) (any, error) {
		return bean, nil
	})

	if _, err := b.Get("name.only"); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if bean.name != "name.only" {
		t.Errorf("expected name %q, got %q", "name.only", bean.name)
	}
}

type nameOnlyBean struct {
	name string
}

func (b *nameOnlyBean) SetBeanName(name string) { b.name = name }

// TestLifecycle_PostProcessorBeforeInitError 验证 BeforeInit 返回错误时中断并向上传递。
func TestLifecycle_PostProcessorBeforeInitError(t *testing.T) {
	b := NewBuilder()
	b.Singleton("pp.err", func(_ AbstractBuilder) (any, error) {
		return &fullLifecycleBean{}, nil
	})
	ppErr := errors.New("before init failed")
	b.RegisterPostProcessor(&errProcessor{beforeErr: ppErr})

	_, err := b.Get("pp.err")
	if !errors.Is(err, ppErr) {
		t.Errorf("expected before-init error %v, got %v", ppErr, err)
	}
}

// TestLifecycle_PostProcessorAfterInitError 验证 AfterInit 返回错误时中断并向上传递。
func TestLifecycle_PostProcessorAfterInitError(t *testing.T) {
	b := NewBuilder()
	b.Singleton("pp.after.err", func(_ AbstractBuilder) (any, error) {
		return &fullLifecycleBean{}, nil
	})
	afterErr := errors.New("after init failed")
	b.RegisterPostProcessor(&errProcessor{afterErr: afterErr})

	_, err := b.Get("pp.after.err")
	if !errors.Is(err, afterErr) {
		t.Errorf("expected after-init error %v, got %v", afterErr, err)
	}
}

type errProcessor struct {
	beforeErr error
	afterErr  error
}

func (p *errProcessor) BeforeInit(name string, bean any) (any, error) {
	if p.beforeErr != nil {
		return bean, p.beforeErr
	}
	return bean, nil
}

func (p *errProcessor) AfterInit(name string, bean any) (any, error) {
	if p.afterErr != nil {
		return bean, p.afterErr
	}
	return bean, nil
}

// TestLifecycle_SingletonCachedAfterBoot 验证 Boot 后单例被缓存，再次 Get 不重复触发 Boot。
func TestLifecycle_SingletonCachedAfterBoot(t *testing.T) {
	b := NewBuilder()
	bootCount := 0
	b.Singleton("cached.bean", func(_ AbstractBuilder) (any, error) {
		return &countingBootBean{count: &bootCount}, nil
	})

	if _, err := b.Get("cached.bean"); err != nil {
		t.Fatalf("first Get failed: %v", err)
	}
	if bootCount != 1 {
		t.Fatalf("expected 1 boot, got %d", bootCount)
	}
	// 再次 Get 应返回缓存实例，不重复 Boot
	if _, err := b.Get("cached.bean"); err != nil {
		t.Fatalf("second Get failed: %v", err)
	}
	if bootCount != 1 {
		t.Errorf("expected cached singleton (1 boot), got %d boots", bootCount)
	}
}

type countingBootBean struct {
	count *int
}

func (b *countingBootBean) Boot() error {
	*b.count++
	return nil
}
