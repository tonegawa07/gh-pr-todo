package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config はアプリケーション設定
type Config struct {
	Repos []string `yaml:"repos"`
}

// DefaultPath は設定ファイルのデフォルトパスを返す
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "gh-pr-todo", "config.yml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "gh-pr-todo", "config.yml")
}

// Load は設定ファイルを読み込む。ファイルが存在しなければ空の Config を返す。
func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}
	if path == "" {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
