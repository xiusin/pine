package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// LoadFromYAML 从 YAML 文件加载配置为 map[string]any.
// yaml.v2 解析嵌套 map 默认产物为 map[interface{}]interface{}, 此处统一规范化为 map[string]any.
func LoadFromYAML(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read yaml %q: %w", path, err)
	}
	var out map[string]any
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse yaml %q: %w", path, err)
	}
	return normalizeMap(out), nil
}

// LoadFromJSON 从 JSON 文件加载配置.
func LoadFromJSON(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read json %q: %w", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse json %q: %w", path, err)
	}
	return normalizeMap(out), nil
}

// LoadFromEnv 从环境变量加载配置.
// 仅加载以 prefix 开头的环境变量, 去除前缀后将剩余部分按下划线拆分为嵌套 map, 并小写键名.
//
// 转换示例:
//   - prefix="" 且 APP_NAME=hello          -> {"app": {"name": "hello"}}
//   - prefix="APP_" 且 APP_DB_HOST=localhost -> {"db": {"host": "localhost"}}
//   - prefix="APP_" 且 APP_NAME=hello        -> {"name": "hello"}
func LoadFromEnv(prefix string) map[string]any {
	out := map[string]any{}
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, prefix) {
			continue
		}
		body := strings.TrimPrefix(kv, prefix)
		idx := strings.IndexByte(body, '=')
		if idx < 0 {
			continue
		}
		key := body[:idx]
		val := body[idx+1:]
		if key == "" {
			continue
		}
		// 下划线转嵌套: DB_HOST -> db.host
		parts := strings.Split(key, "_")
		for i, p := range parts {
			parts[i] = strings.ToLower(p)
		}
		assign(out, strings.Join(parts, "."), val)
	}
	return out
}

// LoadFromDotEnv 从 .env 文件加载配置.
// 解析 KEY=VALUE 行 (忽略空行与 # 注释), 支持可选的 export 前缀与两端引号包裹.
// 返回的 map 键为原始 KEY (大小写保留), 值为去引号后的字符串.
func LoadFromDotEnv(path string) (map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dotenv %q: %w", path, err)
	}
	defer f.Close()

	out := map[string]any{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 去除可选 export 前缀
		line = strings.TrimPrefix(line, "export ")
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// 去除两端匹配的引号
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if key == "" {
			continue
		}
		out[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan dotenv %q: %w", path, err)
	}
	return out, nil
}

// Merge 合并多个配置源, 后者覆盖前者 (对嵌套 map 递归合并, 非 map 值直接覆盖).
func Merge(sources ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, src := range sources {
		mergeInto(out, src)
	}
	return out
}

// mergeInto 将 src 递归合并到 dst.
func mergeInto(dst, src map[string]any) {
	for k, v := range src {
		if sv, ok := v.(map[string]any); ok {
			if dv, ok := dst[k].(map[string]any); ok {
				mergeInto(dv, sv)
				continue
			}
			dst[k] = deepCopyMap(sv)
			continue
		}
		dst[k] = v
	}
}

// ---- map 规范化 ----

// normalizeMap 将 map[string]any 中的嵌套 map[interface{}]interface{} (yaml.v2 产物)
// 递归转换为 map[string]any, 便于后续 dot 路径访问与 JSON 中转绑定.
func normalizeMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = normalizeValue(v)
	}
	return out
}

func normalizeValue(v any) any {
	switch val := v.(type) {
	case map[interface{}]any:
		return normalizeInterfaceMap(val)
	case map[string]any:
		return normalizeMap(val)
	case []any:
		for i, e := range val {
			val[i] = normalizeValue(e)
		}
		return val
	default:
		return v
	}
}

func normalizeInterfaceMap(in map[interface{}]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[fmt.Sprintf("%v", k)] = normalizeValue(v)
	}
	return out
}
