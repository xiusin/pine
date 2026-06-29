package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRepositoryGetTypes(t *testing.T) {
	r := NewWithData(map[string]any{
		"str":      "hello",
		"num":      42,
		"flag":     true,
		"pi":       3.14,
		"dur":      "5s",
		"list":     []string{"a", "b"},
		"anylist":  []any{"x", "y"},
		"csv":      "a, b, c",
		"numstr":   "123",
		"boolstr":  "yes",
		"floatstr": "3.5",
	})

	if got := r.GetString("str"); got != "hello" {
		t.Errorf("GetString = %q, want %q", got, "hello")
	}
	if got := r.GetInt("num"); got != 42 {
		t.Errorf("GetInt = %d, want 42", got)
	}
	if got := r.GetInt("numstr"); got != 123 {
		t.Errorf("GetInt(numstr) = %d, want 123", got)
	}
	if got := r.GetBool("flag"); got != true {
		t.Errorf("GetBool = %v, want true", got)
	}
	if got := r.GetBool("boolstr"); got != true {
		t.Errorf("GetBool(boolstr=yes) = %v, want true", got)
	}
	if got := r.GetFloat64("pi"); got != 3.14 {
		t.Errorf("GetFloat64 = %v, want 3.14", got)
	}
	if got := r.GetFloat64("floatstr"); got != 3.5 {
		t.Errorf("GetFloat64(floatstr) = %v, want 3.5", got)
	}
	if got := r.GetDuration("dur"); got != 5*time.Second {
		t.Errorf("GetDuration = %v, want 5s", got)
	}
	if got := r.GetStringSlice("list"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("GetStringSlice(list) = %v", got)
	}
	if got := r.GetStringSlice("anylist"); len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Errorf("GetStringSlice(anylist) = %v", got)
	}
	if got := r.GetStringSlice("csv"); len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("GetStringSlice(csv) = %v", got)
	}
}

func TestRepositoryDefaults(t *testing.T) {
	r := NewWithData(map[string]any{"present": "value"})
	if got := r.GetString("missing", "fallback"); got != "fallback" {
		t.Errorf("GetString default = %q", got)
	}
	if got := r.GetInt("missing", 99); got != 99 {
		t.Errorf("GetInt default = %d", got)
	}
	if got := r.GetBool("missing", true); got != true {
		t.Errorf("GetBool default = %v", got)
	}
	if got := r.GetFloat64("missing", 1.5); got != 1.5 {
		t.Errorf("GetFloat64 default = %v", got)
	}
	if got := r.GetDuration("missing", 2*time.Second); got != 2*time.Second {
		t.Errorf("GetDuration default = %v", got)
	}
	if got := r.GetStringSlice("missing", []string{"d"}); len(got) != 1 || got[0] != "d" {
		t.Errorf("GetStringSlice default = %v", got)
	}
	if got := r.GetString("missing"); got != "" {
		t.Errorf("GetString no default = %q, want empty", got)
	}
	if got := r.GetInt("missing"); got != 0 {
		t.Errorf("GetInt no default = %d, want 0", got)
	}
}

func TestRepositoryDotPath(t *testing.T) {
	r := NewWithData(map[string]any{
		"app": map[string]any{
			"name": "pine",
			"db": map[string]any{
				"host": "localhost",
				"port": 3306,
			},
		},
	})
	if got := r.GetString("app.name"); got != "pine" {
		t.Errorf("app.name = %q", got)
	}
	if got := r.GetString("app.db.host"); got != "localhost" {
		t.Errorf("app.db.host = %q", got)
	}
	if got := r.GetInt("app.db.port"); got != 3306 {
		t.Errorf("app.db.port = %d", got)
	}
	if !r.Has("app.db.host") {
		t.Error("Has(app.db.host) should be true")
	}
	if r.Has("app.db.missing") {
		t.Error("Has(app.db.missing) should be false")
	}
	if r.Has("app.db.host.deep") {
		t.Error("Has(app.db.host.deep) should be false (host is string)")
	}
	if got := r.Get("app.db").(map[string]any)["host"]; got != "localhost" {
		t.Errorf("Get(app.db) nested = %v", got)
	}
}

func TestRepositorySetAndAll(t *testing.T) {
	r := New()
	r.Set("app.name", "pine")
	r.Set("app.db.host", "localhost")
	r.Set("app.db.port", 3306)
	if got := r.GetString("app.name"); got != "pine" {
		t.Errorf("Set/Get app.name = %q", got)
	}
	if got := r.GetString("app.db.host"); got != "localhost" {
		t.Errorf("Set/Get app.db.host = %q", got)
	}
	if got := r.GetInt("app.db.port"); got != 3306 {
		t.Errorf("Set/Get app.db.port = %d", got)
	}

	// All() 返回深拷贝, 外部修改不影响仓库
	all := r.All()
	all["app"] = "tampered"
	if got := r.GetString("app.name"); got != "pine" {
		t.Errorf("All() returned shared map, app.name got = %q", got)
	}
}

func TestRepositoryOverwriteNonMap(t *testing.T) {
	r := New()
	r.Set("a.b", "leaf")
	// 在已有 string 值上设置更深路径, 应覆盖为 map
	r.Set("a.b.c", "deep")
	if got := r.GetString("a.b.c"); got != "deep" {
		t.Errorf("overwrite non-map: a.b.c = %q", got)
	}
}

func TestLoadFromJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := []byte(`{"app":{"name":"pine","port":8080,"debug":true}}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := LoadFromJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	r := NewWithData(data)
	if got := r.GetString("app.name"); got != "pine" {
		t.Errorf("app.name = %q", got)
	}
	if got := r.GetInt("app.port"); got != 8080 {
		t.Errorf("app.port = %d", got)
	}
	if got := r.GetBool("app.debug"); got != true {
		t.Errorf("app.debug = %v", got)
	}
}

func TestLoadFromJSONMissing(t *testing.T) {
	_, err := LoadFromJSON("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for missing json file")
	}
}

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte("app:\n  name: pine\n  port: 8080\n  debug: true\n  db:\n    host: localhost\n    port: 3306\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := LoadFromYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	r := NewWithData(data)
	if got := r.GetString("app.name"); got != "pine" {
		t.Errorf("app.name = %q", got)
	}
	if got := r.GetInt("app.port"); got != 8080 {
		t.Errorf("app.port = %d", got)
	}
	if got := r.GetBool("app.debug"); got != true {
		t.Errorf("app.debug = %v", got)
	}
	// 验证 yaml.v2 嵌套 map[interface{}]interface{} 已被规范化为 map[string]any
	if got := r.GetString("app.db.host"); got != "localhost" {
		t.Errorf("app.db.host = %q (nested map normalization)", got)
	}
	if got := r.GetInt("app.db.port"); got != 3306 {
		t.Errorf("app.db.port = %d", got)
	}
}

func TestLoadFromYAMLMissing(t *testing.T) {
	_, err := LoadFromYAML("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing yaml file")
	}
}

func TestLoadFromEnv(t *testing.T) {
	// 使用唯一前缀避免 CI 环境干扰
	t.Setenv("PINE_TEST_NAME", "pine")
	t.Setenv("PINE_TEST_DB_HOST", "localhost")
	t.Setenv("PINE_TEST_DB_PORT", "3306")
	data := LoadFromEnv("PINE_TEST_")
	r := NewWithData(data)
	if got := r.GetString("name"); got != "pine" {
		t.Errorf("name = %q", got)
	}
	if got := r.GetString("db.host"); got != "localhost" {
		t.Errorf("db.host = %q", got)
	}
	if got := r.GetString("db.port"); got != "3306" {
		t.Errorf("db.port = %q", got)
	}
}

func TestLoadFromEnvEmptyPrefix(t *testing.T) {
	// 空前缀 + APP_NAME -> {"app": {"name": ...}}
	t.Setenv("PINE_EMPTY_APP_NAME", "val")
	data := LoadFromEnv("PINE_EMPTY_")
	if got := lookupString(data, "app.name"); got != "val" {
		t.Errorf("empty prefix transform: app.name = %q", got)
	}
}

func TestLoadFromDotEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := []byte("# comment\nAPP_NAME=pine\nAPP_PORT=8080\nexport APP_DEBUG=true\nAPP_QUOTED=\"quoted value\"\nAPP_SINGLE='single'\n\nAPP_EMPTY=\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := LoadFromDotEnv(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := data["APP_NAME"]; got != "pine" {
		t.Errorf("APP_NAME = %v", got)
	}
	if got := data["APP_PORT"]; got != "8080" {
		t.Errorf("APP_PORT = %v", got)
	}
	if got := data["APP_DEBUG"]; got != "true" {
		t.Errorf("APP_DEBUG = %v", got)
	}
	if got := data["APP_QUOTED"]; got != "quoted value" {
		t.Errorf("APP_QUOTED = %v", got)
	}
	if got := data["APP_SINGLE"]; got != "single" {
		t.Errorf("APP_SINGLE = %v", got)
	}
	if got := data["APP_EMPTY"]; got != "" {
		t.Errorf("APP_EMPTY = %v, want empty", got)
	}
	if _, ok := data["APP_NONEXISTENT"]; ok {
		t.Error("APP_NONEXISTENT should not be present")
	}
}

func TestMerge(t *testing.T) {
	base := map[string]any{
		"app": map[string]any{
			"name": "pine",
			"port": 8080,
		},
		"keep": "base",
	}
	overlay := map[string]any{
		"app": map[string]any{
			"port":  9090,
			"debug": true,
		},
		"add": "overlay",
	}
	merged := Merge(base, overlay)
	r := NewWithData(merged)
	if got := r.GetString("app.name"); got != "pine" {
		t.Errorf("app.name = %q (from base)", got)
	}
	if got := r.GetInt("app.port"); got != 9090 {
		t.Errorf("app.port = %d, want 9090 (overlay)", got)
	}
	if got := r.GetBool("app.debug"); got != true {
		t.Errorf("app.debug = %v (overlay)", got)
	}
	if got := r.GetString("keep"); got != "base" {
		t.Errorf("keep = %q (base preserved)", got)
	}
	if got := r.GetString("add"); got != "overlay" {
		t.Errorf("add = %q (overlay)", got)
	}
	// Merge 不应修改输入
	if v, ok := lookup(base, "app.port"); !ok || v != 8080 {
		t.Errorf("base mutated: app.port = %v (ok=%v)", v, ok)
	}
}

func TestLoadByProfile(t *testing.T) {
	dir := t.TempDir()
	base := []byte("app:\n  name: pine\n  port: 8080\n  env: base\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), base, 0o644); err != nil {
		t.Fatal(err)
	}
	profDir := filepath.Join(dir, "config")
	if err := os.Mkdir(profDir, 0o755); err != nil {
		t.Fatal(err)
	}
	prof := []byte("app:\n  env: dev\n  port: 9090\n")
	if err := os.WriteFile(filepath.Join(profDir, "dev.yaml"), prof, 0o644); err != nil {
		t.Fatal(err)
	}
	// 环境变量覆盖: APP_APP_PORT -> app.port
	t.Setenv("APP_APP_PORT", "7070")
	// 环境变量新增: APP_LOG_LEVEL -> log.level
	t.Setenv("APP_LOG_LEVEL", "debug")

	repo, err := LoadByProfile(dir, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if got := repo.GetString("app.name"); got != "pine" {
		t.Errorf("app.name = %q (from base)", got)
	}
	if got := repo.GetString("app.env"); got != "dev" {
		t.Errorf("app.env = %q (from profile)", got)
	}
	if got := repo.GetInt("app.port"); got != 7070 {
		t.Errorf("app.port = %d, want 7070 (from env override)", got)
	}
	if got := repo.GetString("log.level"); got != "debug" {
		t.Errorf("log.level = %q (from env addition)", got)
	}
}

func TestLoadByProfileNoBase(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APP_ONLY_ENV", "yes")
	repo, err := LoadByProfile(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := repo.GetString("only.env"); got != "yes" {
		t.Errorf("only.env = %q (env only, no base file)", got)
	}
}

func TestBind(t *testing.T) {
	type DBConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	type AppConfig struct {
		Name  string   `json:"name"`
		Port  int      `json:"port"`
		Debug bool     `json:"debug"`
		Tags  []string `json:"tags"`
		DB    DBConfig `json:"db"`
	}
	r := NewWithData(map[string]any{
		"app": map[string]any{
			"name":  "pine",
			"port":  8080,
			"debug": true,
			"tags":  []any{"web", "go"},
			"db": map[string]any{
				"host": "localhost",
				"port": 3306,
			},
		},
	})
	var cfg AppConfig
	if err := Bind(r, "app", &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "pine" {
		t.Errorf("Name = %q", cfg.Name)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d", cfg.Port)
	}
	if cfg.Debug != true {
		t.Errorf("Debug = %v", cfg.Debug)
	}
	if len(cfg.Tags) != 2 || cfg.Tags[0] != "web" || cfg.Tags[1] != "go" {
		t.Errorf("Tags = %v", cfg.Tags)
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("DB.Host = %q", cfg.DB.Host)
	}
	if cfg.DB.Port != 3306 {
		t.Errorf("DB.Port = %d", cfg.DB.Port)
	}
}

func TestBindEmptyPrefix(t *testing.T) {
	type Root struct {
		Name string `json:"name"`
	}
	r := NewWithData(map[string]any{"name": "root"})
	var cfg Root
	if err := Bind(r, "", &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "root" {
		t.Errorf("Name = %q", cfg.Name)
	}
}

func TestBindMissingPrefix(t *testing.T) {
	type Cfg struct {
		Name string `json:"name"`
	}
	r := NewWithData(map[string]any{"other": "x"})
	var cfg Cfg
	if err := Bind(r, "app", &cfg); err != nil {
		t.Fatalf("expected nil error for missing prefix, got %v", err)
	}
	if cfg.Name != "" {
		t.Errorf("Name = %q, want empty (no binding)", cfg.Name)
	}
}

func TestBindNonMapPrefix(t *testing.T) {
	r := NewWithData(map[string]any{"app": "scalar"})
	var cfg struct {
		Name string `json:"name"`
	}
	if err := Bind(r, "app", &cfg); err == nil {
		t.Error("expected error when prefix points to non-map")
	}
}

func TestBindNilArgs(t *testing.T) {
	if err := Bind(nil, "app", &struct{}{}); err == nil {
		t.Error("expected error for nil repository")
	}
	if err := Bind(New(), "app", nil); err == nil {
		t.Error("expected error for nil destination")
	}
}

func TestPackageLevelDefault(t *testing.T) {
	old := Default()
	defer SetDefault(old)
	r := NewWithData(map[string]any{"app": map[string]any{"name": "pine", "port": 8080}})
	SetDefault(r)
	if got := GetString("app.name"); got != "pine" {
		t.Errorf("package GetString = %q", got)
	}
	if got := GetInt("app.port"); got != 8080 {
		t.Errorf("package GetInt = %d", got)
	}
	if !Has("app.name") {
		t.Error("package Has(app.name) should be true")
	}
	Set("app.name", "changed")
	if got := GetString("app.name"); got != "changed" {
		t.Errorf("package Set/Get = %q", got)
	}
	if got := All()["app"].(map[string]any)["name"]; got != "changed" {
		t.Errorf("package All = %v", got)
	}
}

func TestSetDefaultNil(t *testing.T) {
	old := Default()
	SetDefault(nil)
	if Default() != old {
		t.Error("SetDefault(nil) should not change default repo")
	}
}

// lookupString 在 map 中按 dot 路径取 string, 测试辅助.
func lookupString(data map[string]any, key string) string {
	v, ok := lookup(data, key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
