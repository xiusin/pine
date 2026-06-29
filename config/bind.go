package config

import (
	"encoding/json"
	"fmt"
)

// Bind 将 Repository 中以 prefix 为前缀的配置绑定到 dst 结构体.
// 类似 Spring 的 @ConfigurationProperties: prefix="app" 时读取 app.* 配置填充 dst.
//
// 实现使用 encoding/json 中转 (marshal map -> unmarshal struct),
// 字段匹配遵循 json tag (如 `json:"name"`) 与大小写不敏感规则.
//
// 用法:
//
//	type AppConfig struct {
//	    Name string `json:"name"`
//	    Port int    `json:"port"`
//	}
//	var app AppConfig
//	if err := config.Bind(repo, "app", &app); err != nil { ... }
//
// dst 必须为非 nil 指针. prefix 为空时绑定整个配置树.
// 注意: time.Duration 字段需以 int(纳秒) 或可被 time.Duration 解析的字符串提供,
// 复杂类型建议直接使用对应的 GetXxx 方法读取.
func Bind(repo Repository, prefix string, dst any) error {
	if repo == nil {
		return fmt.Errorf("config: nil repository")
	}
	if dst == nil {
		return fmt.Errorf("config: nil destination")
	}

	data := repo.All()
	if prefix != "" {
		sub, ok := lookup(data, prefix)
		if !ok {
			// 前缀不存在, 视为无配置可绑定, 不修改 dst
			return nil
		}
		m, ok := sub.(map[string]any)
		if !ok {
			return fmt.Errorf("config: prefix %q is not a map", prefix)
		}
		data = m
	}

	// 规范化后用 JSON 中转绑定, 兼容 yaml 产物的 map[interface{}]interface{} 等.
	b, err := json.Marshal(normalizeValue(data))
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("config: unmarshal: %w", err)
	}
	return nil
}
