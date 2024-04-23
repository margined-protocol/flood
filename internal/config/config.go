package config

import (
	"github.com/BurntSushi/toml"
	"github.com/margined-protocol/flood/internal/types"
)

func LoadConfig(configPath string) (*types.Config, error) {
	var config types.Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
