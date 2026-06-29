// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package transaction

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

// ===== mock TransactionManager (验证 Execute 语义) =====

// fakeTx 记录 Commit / Rollback 调用, 用于验证 Execute 的提交/回滚语义.
type fakeTx struct {
	committed  bool
	rolledBack bool
	commitErr  error
}

func (t *fakeTx) Commit() error {
	if t.commitErr != nil {
		return t.commitErr
	}
	t.committed = true
	return nil
}

func (t *fakeTx) Rollback() error {
	t.rolledBack = true
	return nil
}

// fakeManager 内存版事务管理器, 记录事务创建与提交/回滚计数.
type fakeManager struct {
	beginErr error
	tx       *fakeTx
}

func (m *fakeManager) Begin() (Transaction, error) {
	if m.beginErr != nil {
		return nil, m.beginErr
	}
	if m.tx == nil {
		m.tx = &fakeTx{}
	}
	return m.tx, nil
}

func (m *fakeManager) Commit(tx Transaction) error   { return tx.Commit() }
func (m *fakeManager) Rollback(tx Transaction) error { return tx.Rollback() }

// TestExecuteSuccessCommit 验证 fn 返回 nil 时事务被提交且未回滚.
func TestExecuteSuccessCommit(t *testing.T) {
	m := &fakeManager{}
	var seenTx Transaction
	err := Execute(m, func(tx Transaction) error {
		seenTx = tx
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if seenTx == nil {
		t.Fatal("expected tx passed to fn")
	}
	if !m.tx.committed {
		t.Fatal("expected tx committed")
	}
	if m.tx.rolledBack {
		t.Fatal("expected tx NOT rolled back")
	}
}

// TestExecuteErrorRollback 验证 fn 返回 error 时事务被回滚且 error 被透传.
func TestExecuteErrorRollback(t *testing.T) {
	m := &fakeManager{}
	wantErr := errors.New("boom")
	err := Execute(m, func(tx Transaction) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if m.tx.committed {
		t.Fatal("expected tx NOT committed")
	}
	if !m.tx.rolledBack {
		t.Fatal("expected tx rolled back")
	}
}

// TestExecutePanicRollback 验证 fn panic 时事务被回滚且 panic 被重新抛出.
func TestExecutePanicRollback(t *testing.T) {
	m := &fakeManager{}
	defer func() {
		p := recover()
		if p != "boom" {
			t.Fatalf("expected panic 'boom', got %v", p)
		}
		if !m.tx.rolledBack {
			t.Fatal("expected tx rolled back after panic")
		}
		if m.tx.committed {
			t.Fatal("expected tx NOT committed after panic")
		}
	}()
	_ = Execute(m, func(tx Transaction) error {
		panic("boom")
	})
}

// TestExecuteBeginError 验证 Begin 失败时直接返回错误, 不调用 fn.
func TestExecuteBeginError(t *testing.T) {
	beginErr := errors.New("begin failed")
	m := &fakeManager{beginErr: beginErr}
	called := false
	err := Execute(m, func(tx Transaction) error {
		called = true
		return nil
	})
	if !errors.Is(err, beginErr) {
		t.Fatalf("expected begin error, got %v", err)
	}
	if called {
		t.Fatal("expected fn NOT called when Begin fails")
	}
}

// ===== 最小 in-memory driver (验证 DBTransactionManager 真实 SQL 集成) =====

// fakeDriver 实现一个最小的 database/sql/driver.Driver,
// 用于在不依赖真实数据库的情况下测试 DBTransactionManager.
// 仅支持事务的开始/提交/回滚计数, 不支持实际查询.
type fakeDriver struct {
	mu  sync.Mutex
	tx0 *fakeDriverTx // 最近一次创建的事务, 供测试断言
}

func (d *fakeDriver) Open(name string) (driver.Conn, error) {
	return &fakeConn{d: d}, nil
}

// fakeConn 实现 driver.Conn 与 driver.ConnBeginTx.
type fakeConn struct {
	d *fakeDriver
}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("fakeDriver: Prepare not supported")
}

func (c *fakeConn) Close() error { return nil }

func (c *fakeConn) Begin() (driver.Tx, error) {
	return c.beginTx(context.Background())
}

// BeginTx 满足 driver.ConnBeginTx, sql.DB.BeginTx 会优先调用本方法.
func (c *fakeConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.beginTx(ctx)
}

func (c *fakeConn) beginTx(_ context.Context) (driver.Tx, error) {
	tx := &fakeDriverTx{}
	c.d.mu.Lock()
	c.d.tx0 = tx
	c.d.mu.Unlock()
	return tx, nil
}

// fakeDriverTx 实现 driver.Tx, 记录提交/回滚次数.
type fakeDriverTx struct {
	committed  int32
	rolledBack int32
}

func (t *fakeDriverTx) Commit() error {
	atomic.AddInt32(&t.committed, 1)
	return nil
}

func (t *fakeDriverTx) Rollback() error {
	atomic.AddInt32(&t.rolledBack, 1)
	return nil
}

// fakeDrv 与 fakeDriverOnce 保证 driver 全局只注册一次,
// 每个测试复用同一实例并在用例开始时重置其状态 (tx0 置空).
var (
	fakeDrv        = &fakeDriver{}
	fakeDriverOnce sync.Once
)

// newFakeDB 返回一个基于 fakeDriver 的 *sql.DB, 用于 DBTransactionManager 测试.
// 每次调用会重置共享 driver 的最近事务记录, 保证用例间互不影响.
func newFakeDB(t *testing.T) (*sql.DB, *fakeDriver) {
	t.Helper()
	fakeDriverOnce.Do(func() {
		sql.Register("pine-tx-fake", fakeDrv)
	})
	fakeDrv.mu.Lock()
	fakeDrv.tx0 = nil
	fakeDrv.mu.Unlock()
	db, err := sql.Open("pine-tx-fake", "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db, fakeDrv
}

// TestDBTransactionManagerCommit 验证 DBTransactionManager.Execute 成功时底层事务被提交.
func TestDBTransactionManagerCommit(t *testing.T) {
	db, d := newFakeDB(t)
	defer db.Close()

	tm := NewDBTransactionManager(db)
	err := Execute(tm, func(tx Transaction) error {
		// 事务句柄应可断言为 *dbTransaction 并持有底层 *sql.Tx
		dbTx, ok := tx.(*dbTransaction)
		if !ok {
			t.Fatalf("expected *dbTransaction, got %T", tx)
		}
		if dbTx.tx == nil {
			t.Fatal("expected underlying *sql.Tx not nil")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if d.tx0 == nil || atomic.LoadInt32(&d.tx0.committed) != 1 {
		t.Fatal("expected underlying tx committed once")
	}
	if d.tx0 != nil && atomic.LoadInt32(&d.tx0.rolledBack) != 0 {
		t.Fatal("expected underlying tx NOT rolled back")
	}
}

// TestDBTransactionManagerRollback 验证 DBTransactionManager.Execute 失败时底层事务被回滚.
func TestDBTransactionManagerRollback(t *testing.T) {
	db, d := newFakeDB(t)
	defer db.Close()

	tm := NewDBTransactionManager(db)
	wantErr := errors.New("db boom")
	err := Execute(tm, func(tx Transaction) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
	if d.tx0 == nil || atomic.LoadInt32(&d.tx0.rolledBack) != 1 {
		t.Fatal("expected underlying tx rolled back once")
	}
	if d.tx0 != nil && atomic.LoadInt32(&d.tx0.committed) != 0 {
		t.Fatal("expected underlying tx NOT committed")
	}
}

// TestDBTransactionManagerImplementsInterface 编译期保证 DBTransactionManager
// 实现 TransactionManager 接口.
func TestDBTransactionManagerImplementsInterface(t *testing.T) {
	var _ TransactionManager = (*DBTransactionManager)(nil)
	var _ Transaction = (*dbTransaction)(nil)
}
