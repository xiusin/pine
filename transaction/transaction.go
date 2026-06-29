// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package transaction

import (
	"context"
	"database/sql"
)

// TransactionManager 事务管理器接口.
// 参考 Spring PlatformTransactionManager, 提供事务的开始、提交、回滚能力.
//
// 实现方可基于 *sql.DB、ORM 或其他支持事务的资源封装本接口,
// 上层业务代码面向接口编程, 不与具体资源耦合.
type TransactionManager interface {
	// Begin 开始事务, 返回事务句柄.
	Begin() (Transaction, error)
	// Commit 提交事务.
	Commit(tx Transaction) error
	// Rollback 回滚事务.
	Rollback(tx Transaction) error
}

// Transaction 事务句柄.
// 由 TransactionManager.Begin 返回, 持有底层事务资源 (如 *sql.Tx).
// 调用方在事务逻辑结束后需显式调用 Commit 或 Rollback.
type Transaction interface {
	// Commit 提交事务.
	Commit() error
	// Rollback 回滚事务.
	Rollback() error
}

// TxFunc 事务内执行函数.
// 接收当前事务句柄, 返回 error 控制提交或回滚.
type TxFunc func(tx Transaction) error

// Execute 编程式事务执行.
// 参考 Spring TransactionTemplate.execute:
//   - fn 返回 nil 则 Commit;
//   - fn 返回 error 则 Rollback 并返回该 error;
//   - fn panic 则 Rollback 后重新抛出 panic.
func Execute(tm TransactionManager, fn TxFunc) error {
	tx, err := tm.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// DBTransactionManager 基于 *sql.DB 的事务管理器实现.
// 适配标准库 database/sql, 通过 sql.DB.BeginTx 创建底层事务.
type DBTransactionManager struct {
	db *sql.DB
}

// NewDBTransactionManager 创建基于 *sql.DB 的事务管理器.
func NewDBTransactionManager(db *sql.DB) *DBTransactionManager {
	return &DBTransactionManager{db: db}
}

// Begin 开始事务, 返回包装了 *sql.Tx 的事务句柄.
func (m *DBTransactionManager) Begin() (Transaction, error) {
	tx, err := m.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return &dbTransaction{tx: tx}, nil
}

// Commit 提交事务.
func (m *DBTransactionManager) Commit(tx Transaction) error { return tx.Commit() }

// Rollback 回滚事务.
func (m *DBTransactionManager) Rollback(tx Transaction) error { return tx.Rollback() }

// dbTransaction 基于 *sql.Tx 的事务句柄实现.
type dbTransaction struct {
	tx *sql.Tx
}

// Commit 提交底层 *sql.Tx.
func (t *dbTransaction) Commit() error { return t.tx.Commit() }

// Rollback 回滚底层 *sql.Tx.
func (t *dbTransaction) Rollback() error { return t.tx.Rollback() }
