package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LoadByProfile 根据 profile 加载配置, 返回基于合并结果构建的 Repository.
//
// 加载顺序 (后者覆盖前者):
//  1. baseDir/config.yaml 作为基础配置 (若存在)
//  2. baseDir/config/{profile}.yaml 作为 profile 覆盖 (profile 为空则跳过)
//  3. 以 APP_ 为前缀的环境变量最终覆盖
//
// 类似 Spring Boot 的 application.yml + application-{profile}.yml + 环境变量机制.
// 基础文件与 profile 文件均不存在时, 仍返回非 nil 仓库 (仅含环境变量配置).
func LoadByProfile(baseDir, profile string) (Repository, error) {
	merged := map[string]any{}

	// 1. 基础配置 config.yaml (不存在时静默跳过)
	basePath := filepath.Join(baseDir, "config.yaml")
	if data, err := LoadFromYAML(basePath); err == nil {
		merged = Merge(merged, data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load base config: %w", err)
	}

	// 2. profile 覆盖 config/{profile}.yaml (不存在时静默跳过)
	if profile != "" {
		profPath := filepath.Join(baseDir, "config", profile+".yaml")
		if data, err := LoadFromYAML(profPath); err == nil {
			merged = Merge(merged, data)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("load profile %q: %w", profile, err)
		}
	}

	// 3. 环境变量最终覆盖 (APP_ 前缀)
	merged = Merge(merged, LoadFromEnv("APP_"))

	return NewWithData(merged), nil
}
