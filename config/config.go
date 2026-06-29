package config

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Repository 配置仓库接口, 提供类型安全的配置读取.
// 类似 Laravel 的 Config Repository 与 Spring 的 Environment.
type Repository interface {
	// Get 按 dot 路径读取配置, 未命中时返回 defaultVal[0] (若提供), 否则 nil.
	Get(key string, defaultVal ...any) any
	// GetString 读取字符串, 非 string 类型用 fmt.Sprintf 转换.
	GetString(key string, defaultVal ...string) string
	// GetInt 读取整型, 支持数值与可解析为整型的字符串.
	GetInt(key string, defaultVal ...int) int
	// GetBool 读取布尔值, 字符串 "1/true/yes/on" 视为 true (大小写不敏感).
	GetBool(key string, defaultVal ...bool) bool
	// GetFloat64 读取浮点数, 支持数值与可解析为浮点的字符串.
	GetFloat64(key string, defaultVal ...float64) float64
	// GetDuration 读取 time.Duration, 支持 Duration/int(纳秒)/可解析字符串.
	GetDuration(key string, defaultVal ...time.Duration) time.Duration
	// GetStringSlice 读取字符串切片, 支持 []string/[]any/逗号分隔字符串.
	GetStringSlice(key string, defaultVal ...[]string) []string
	// Has 判断 key 是否存在.
	Has(key string) bool
	// Set 按 dot 路径设置配置, 自动创建中间 map.
	Set(key string, value any)
	// All 返回所有配置的深拷贝.
	All() map[string]any
}

// repository 基于 map 的内存配置仓库实现, 支持 dot 路径访问嵌套配置.
type repository struct {
	mu   sync.RWMutex
	data map[string]any
}

// New 创建空配置仓库.
func New() Repository {
	return &repository{data: map[string]any{}}
}

// NewWithData 基于给定 data 创建配置仓库, data 会被深拷贝以避免外部修改污染.
func NewWithData(data map[string]any) Repository {
	return &repository{data: deepCopyMap(data)}
}

func (r *repository) Get(key string, defaultVal ...any) any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := lookup(r.data, key)
	if !ok {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return nil
	}
	return v
}

func (r *repository) GetString(key string, defaultVal ...string) string {
	v := r.Get(key)
	if v == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func (r *repository) GetInt(key string, defaultVal ...int) int {
	v := r.Get(key)
	if n, ok := toInt(v); ok {
		return n
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0
}

func (r *repository) GetBool(key string, defaultVal ...bool) bool {
	v := r.Get(key)
	if b, ok := toBool(v); ok {
		return b
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return false
}

func (r *repository) GetFloat64(key string, defaultVal ...float64) float64 {
	v := r.Get(key)
	if f, ok := toFloat64(v); ok {
		return f
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0
}

func (r *repository) GetDuration(key string, defaultVal ...time.Duration) time.Duration {
	v := r.Get(key)
	if v == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	switch d := v.(type) {
	case time.Duration:
		return d
	case int:
		return time.Duration(d)
	case int64:
		return time.Duration(d)
	case float64:
		return time.Duration(d)
	case string:
		if d, err := time.ParseDuration(d); err == nil {
			return d
		}
		if n, err := strconv.Atoi(d); err == nil {
			return time.Duration(n)
		}
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0
}

func (r *repository) GetStringSlice(key string, defaultVal ...[]string) []string {
	v := r.Get(key)
	if v == nil {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, len(s))
		for i, e := range s {
			out[i] = fmt.Sprintf("%v", e)
		}
		return out
	case string:
		if s == "" {
			return []string{}
		}
		parts := strings.Split(s, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		return parts
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return nil
}

func (r *repository) Has(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := lookup(r.data, key)
	return ok
}

func (r *repository) Set(key string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	assign(r.data, key, value)
}

func (r *repository) All() map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return deepCopyMap(r.data)
}

// ---- 类型转换辅助 ----

// toInt 尝试将任意值转为 int, 成功返回 (n, true).
func toInt(v any) (int, bool) {
	if v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	case string:
		if n, err := strconv.Atoi(n); err == nil {
			return n, true
		}
	case bool:
		if n {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// toBool 尝试将任意值转为 bool, 成功返回 (b, true).
func toBool(v any) (bool, bool) {
	if v == nil {
		return false, false
	}
	switch b := v.(type) {
	case bool:
		return b, true
	case int:
		return b != 0, true
	case int8:
		return b != 0, true
	case int16:
		return b != 0, true
	case int32:
		return b != 0, true
	case int64:
		return b != 0, true
	case uint:
		return b != 0, true
	case uint8:
		return b != 0, true
	case uint16:
		return b != 0, true
	case uint32:
		return b != 0, true
	case uint64:
		return b != 0, true
	case float32:
		return b != 0, true
	case float64:
		return b != 0, true
	case string:
		if b, err := strconv.ParseBool(b); err == nil {
			return b, true
		}
		lower := strings.ToLower(b)
		return lower == "yes" || lower == "on", true
	}
	return false, false
}

// toFloat64 尝试将任意值转为 float64, 成功返回 (f, true).
func toFloat64(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case string:
		if n, err := strconv.ParseFloat(n, 64); err == nil {
			return n, true
		}
	}
	return 0, false
}

// ---- dot 路径访问 ----

// lookup 按 dot 路径在 data 中查找, 返回找到的值与是否命中.
// 例: lookup(data, "app.db.host") 访问 data["app"]["db"]["host"].
func lookup(data map[string]any, key string) (any, bool) {
	if key == "" {
		return data, true
	}
	parts := strings.Split(key, ".")
	var cur any = data
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := m[p]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// assign 按 dot 路径在 data 中赋值, 自动创建中间 map.
func assign(data map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	cur := data
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = value
			return
		}
		next, ok := cur[p].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
}

// deepCopyMap 递归拷贝 map[string]any, 避免外部修改污染仓库内部状态.
func deepCopyMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		if m, ok := v.(map[string]any); ok {
			dst[k] = deepCopyMap(m)
		} else {
			dst[k] = v
		}
	}
	return dst
}

// ---- 包级默认实例 ----

// defaultRepo 包级默认仓库, 供便捷函数访问.
var defaultRepo Repository = New()

// SetDefault 设置包级默认仓库. 传入 nil 时忽略.
func SetDefault(r Repository) {
	if r == nil {
		return
	}
	defaultRepo = r
}

// Default 返回包级默认仓库.
func Default() Repository {
	return defaultRepo
}

// 以下为包级便捷函数, 委托到 defaultRepo, 类似 Laravel 的全局 config() 助手.

func Get(key string, defaultVal ...any) any {
	return defaultRepo.Get(key, defaultVal...)
}

func GetString(key string, defaultVal ...string) string {
	return defaultRepo.GetString(key, defaultVal...)
}

func GetInt(key string, defaultVal ...int) int {
	return defaultRepo.GetInt(key, defaultVal...)
}

func GetBool(key string, defaultVal ...bool) bool {
	return defaultRepo.GetBool(key, defaultVal...)
}

func GetFloat64(key string, defaultVal ...float64) float64 {
	return defaultRepo.GetFloat64(key, defaultVal...)
}

func GetDuration(key string, defaultVal ...time.Duration) time.Duration {
	return defaultRepo.GetDuration(key, defaultVal...)
}

func GetStringSlice(key string, defaultVal ...[]string) []string {
	return defaultRepo.GetStringSlice(key, defaultVal...)
}

func Has(key string) bool {
	return defaultRepo.Has(key)
}

func Set(key string, value any) {
	defaultRepo.Set(key, value)
}

func All() map[string]any {
	return defaultRepo.All()
}
