// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package sessions

import (
	"sync"
	"time"

	"github.com/xiusin/pine/cache"
	"github.com/xiusin/pine/contracts"
)

const (
	Modified = iota
	Destroyed
	Saved
)

// 闪存数据在 data map 中使用的内部键.
// 采用 new/old 双 map 语义: Flash 写入 new, Save 时 new 转为 old (供下次请求读取),
// 下次 Save 时 old 自然被替换 (过期). 参考 Laravel flash 语义.
const (
	flashNewKey = "__flash_new__"
	flashOldKey = "__flash_old__"
)

type Session struct {
	sync.RWMutex
	id            string
	data          map[string]any
	status        int
	store         contracts.SessionStore
	cookie        *Cookie
	cookieName    string                    // cookie 名称, Destroy/Regenerate 时用于删除/更新 cookie
	cookieOptions contracts.CookieOptions   // 写入 cookie 时使用的选项
	expires       time.Duration             // cookie / session 过期时长
}

func newSession(id string, store contracts.SessionStore, cookie *Cookie, cookieName string, cookieOptions contracts.CookieOptions, expires time.Duration) (*Session, error) {
	entity := map[string]any{}
	sess := &Session{
		id:            id,
		store:         store,
		cookie:        cookie,
		cookieName:    cookieName,
		cookieOptions: cookieOptions,
		expires:       expires,
	}

	if err := store.Get(sess.key(), &entity); err != nil && err != cache.ErrKeyNotFound {
		return nil, err
	}

	sess.data = entity
	sess.status = Modified

	return sess, nil
}

func (sess *Session) GetId() string { return sess.id }

func (sess *Session) Set(key string, val any) {
	sess.Lock()
	defer sess.Unlock()
	sess.data[key] = val
	sess.status = Modified
}

// All 返回 session 全部数据的浅拷贝副本, 避免外部直接修改内部 map.
func (sess *Session) All() map[string]any {
	sess.RLock()
	defer sess.RUnlock()

	cp := make(map[string]any, len(sess.data))
	for k, v := range sess.data {
		cp[k] = v
	}
	return cp
}

func (sess *Session) Get(key string) any {
	sess.RLock()
	defer sess.RUnlock()

	return sess.data[key]
}

// Remove 移除某个key
func (sess *Session) Remove(key string) {
	sess.Lock()
	defer sess.Unlock()
	delete(sess.data, key)
	sess.status = Modified
}

// Save 持久化 session 数据.
// 修复: 加锁避免与 Set/Get/Remove 并发竞争; 新增 Saved 状态, 已保存则跳过;
// 不再清空 data, 以便请求内继续读取; 保存前轮转闪存数据.
func (sess *Session) Save() error {
	sess.Lock()
	defer sess.Unlock()

	if sess.status == Destroyed {
		return nil
	}
	if sess.status == Saved {
		return nil
	}

	sess.rotateFlash()

	err := sess.store.Save(sess.key(), &sess.data)
	if err != nil {
		return err
	}
	sess.status = Saved
	return nil
}

// rotateFlash 轮转闪存数据. 调用方需持有写锁.
// 将本轮 new 转为 old (供下次请求读取), 上一轮 old 被替换即视为过期.
func (sess *Session) rotateFlash() {
	if sess.data == nil {
		return
	}
	newFlash, _ := sess.data[flashNewKey].(map[string]any)
	sess.data[flashOldKey] = newFlash
	if newFlash != nil {
		sess.data[flashNewKey] = map[string]any{}
	} else {
		delete(sess.data, flashNewKey)
	}
}

// Destroy 销毁整个sess信息
func (sess *Session) Destroy() error {
	sess.Lock()
	defer sess.Unlock()

	sess.data = nil
	sess.status = Destroyed

	// 修复: cookie 名应是 cookieName, 而非 session ID 值.
	sess.cookie.Delete(sess.cookieName)
	return sess.store.Delete(sess.key())
}

// Has 检查是否存在Key
func (sess *Session) Has(key string) bool {
	sess.RLock()
	defer sess.RUnlock()
	var exist bool
	if sess.data != nil {
		_, exist = sess.data[key]
	}
	return exist
}

// Regenerate 重新生成 session ID, 用于防止 session fixation 攻击.
// destroy 为 true 时删除旧 ID 对应的存储数据; 当前数据迁移到新 ID.
func (sess *Session) Regenerate(destroy bool) error {
	sess.Lock()
	defer sess.Unlock()

	oldKey := sess.key()
	data := sess.data
	if destroy {
		_ = sess.store.Delete(oldKey)
	}
	sess.id = sessionId()
	sess.data = data
	sess.status = Modified
	sess.cookie.SetWithOptions(sess.cookieName, sess.id, sess.cookieOptions, int(sess.expires.Seconds()))
	return nil
}

// Flush 清空当前 session 的全部数据, 但保留 cookie 与存储 key (区别于 Destroy).
func (sess *Session) Flush() {
	sess.Lock()
	defer sess.Unlock()
	sess.data = map[string]any{}
	sess.status = Modified
}

// Flash 写入闪存数据, 数据在下次请求后自动过期.
func (sess *Session) Flash(key string, value any) {
	sess.Lock()
	defer sess.Unlock()
	flash, _ := sess.data[flashNewKey].(map[string]any)
	if flash == nil {
		flash = map[string]any{}
		sess.data[flashNewKey] = flash
	}
	flash[key] = value
	sess.status = Modified
}

// GetFlash 读取并清除闪存数据. 读取后该闪存项不再保留到下次请求.
func (sess *Session) GetFlash(key string) (any, bool) {
	sess.Lock()
	defer sess.Unlock()
	if flash, ok := sess.data[flashNewKey].(map[string]any); ok {
		if v, exists := flash[key]; exists {
			delete(flash, key)
			sess.status = Modified
			return v, true
		}
	}
	if flash, ok := sess.data[flashOldKey].(map[string]any); ok {
		if v, exists := flash[key]; exists {
			delete(flash, key)
			sess.status = Modified
			return v, true
		}
	}
	return nil, false
}

// makeKey 存储session的key
func (sess *Session) key() string {
	return "sess_" + sess.id
}
