package config

import (
	"embed"
	"encoding/json"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

//go:embed config.json
var FS embed.FS

type Config struct {
	StandardFiveStarResources types.StandardFiveStarResources
	GachaTypes                types.GachaTypes
	LuckScoreThresholds       []types.LuckScoreThreshold
}

func Load() (*Config, error) {
	var cfg *Config
	raw, err := FS.ReadFile("config.json")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
