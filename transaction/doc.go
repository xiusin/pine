// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

// Package transaction 提供事务抽象与编程式事务支持.
//
// 设计参考 Spring PlatformTransactionManager 与 TransactionTemplate,
// 对标 Spring 中 @Transactional 的底层抽象.
//
// 核心 abstraction:
//   - TransactionManager: 事务管理器接口 (Begin/Commit/Rollback);
//   - Transaction: 事务句柄, 持有底层事务资源;
//   - Execute: 编程式事务, 封装开始-执行-提交/回滚的样板代码.
//
// 框架内置基于标准库 database/sql 的 DBTransactionManager 实现,
// 业务也可自行实现 TransactionManager 接入 ORM 或其他资源.
//
// 用法 (编程式事务):
//
//	import "database/sql"
//	import "github.com/xiusin/pine/transaction"
//
//	tm := transaction.NewDBTransactionManager(db)
//	err := transaction.Execute(tm, func(tx transaction.Transaction) error {
//	    // 在事务内执行 SQL, 失败返回 error 触发回滚
//	    if _, err := tx.(*transaction.DBTransactionManager)... ; err != nil {
//	        return err
//	    }
//	    return nil
//	})
//
// 若需将 *sql.Tx 取出供 ORM 或手写 SQL 使用, 可通过自定义 TransactionManager
// 在事务句柄中暴露底层 *sql.Tx (参考 DBTransactionManager 的实现).
//
// 与 Spring @Transactional 的对应关系:
//
//   - PlatformTransactionManager  -> TransactionManager
//   - TransactionStatus           -> Transaction
//   - TransactionTemplate.execute -> Execute(tm, fn)
//   - DataSourceTransactionManager-> DBTransactionManager
//
// 当前包仅提供编程式事务; 声明式事务 (@Transactional 注解风格) 可在
// 业务层基于本接口结合中间件/AOP 实现, 不在核心包范围内.
package transaction
