// Package config 为 pine 框架提供外部配置加载与管理能力.
//
// 本包补充 pine 在函数式选项 (configurator.go) 之外的配置能力, 支持:
//   - 多源加载: YAML / JSON / 环境变量 / .env 文件
//   - dot 路径访问: config.Get("app.name") 访问 data["app"]["name"]
//   - 类型安全读取: GetString / GetInt / GetBool / GetFloat64 / GetDuration / GetStringSlice
//   - 多环境 Profile: config/dev.yaml 覆盖 config.yaml, 再被环境变量覆盖
//   - 结构体绑定: 类似 Spring @ConfigurationProperties
//
// # 与其他框架的对应关系
//
//	Laravel config("app.name")           -> config.GetString("app.name")
//	Laravel config()->set(k, v)          -> config.Set(k, v)
//	Spring @Value("${app.name}")         -> config.GetString("app.name")
//	Spring @ConfigurationProperties       -> config.Bind(repo, "app", &cfg)
//	Spring profiles (application-{p}.yml) -> config.LoadByProfile(dir, "dev")
//
// # 快速开始
//
//	repo, err := config.LoadByProfile("./config", "dev")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	config.SetDefault(repo)
//	name := config.GetString("app.name", "pine")
//	port := config.GetInt("app.port", 8080)
//
// # 结构体绑定
//
//	type AppConfig struct {
//	    Name string `json:"name"`
//	    Port int    `json:"port"`
//	}
//	var app AppConfig
//	if err := config.Bind(config.Default(), "app", &app); err != nil {
//	    log.Fatal(err)
//	}
//
// # 多源合并
//
//	base, _ := config.LoadFromYAML("config.yaml")
//	env := config.LoadFromEnv("APP_")
//	dotenv, _ := config.LoadFromDotEnv(".env")
//	repo := config.NewWithData(config.Merge(base, dotenv, env))
//
// # LoadByProfile 加载顺序
//
// LoadByProfile 按以下顺序合并配置, 后者覆盖前者:
//  1. baseDir/config.yaml (基础配置)
//  2. baseDir/config/{profile}.yaml (profile 覆盖)
//  3. APP_ 前缀环境变量 (最终覆盖)
//
// 环境变量名转换规则: 前缀去除 + 下划线拆分嵌套 + 键小写.
// 例如 APP_DB_HOST=localhost (prefix="APP_") -> {"db": {"host": "localhost"}}.
package config
