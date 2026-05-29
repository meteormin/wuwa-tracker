package config

import (
	"embed"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

//go:embed config.json
var FS embed.FS

const (
	EnvVarHost        = "WUWA_TRACKER_HOST"
	EnvVarPort        = "WUWA_TRACKER_PORT"
	EnvVarDBPath      = "WUWA_TRACKER_DB_PATH"
	EnvVarCORSOrigins = "WUWA_TRACKER_CORS_ORIGINS"

	DefaultServerHost     = "127.0.0.1"
	DefaultServerPort     = "3000"
	DefaultDBPath         = "data/wuwa_badger"
	DefaultCORSLocalhost  = "http://localhost:5173"
	DefaultCORSLoopback   = "http://127.0.0.1:5173"
	DefaultCORSIPv6       = "http://[::1]:5173"
	DefaultReportFormat   = "html"
	DefaultReportOutput   = "report"
	DefaultLanguage       = "ko"
	DefaultHTTPTimeout    = 5 * time.Second
	DefaultAstritePerPull = 160
)

type Config struct {
	GachaLocaleEndpoint       string                          `json:"gachaLocaleEndpoint"`
	StandardFiveStarResources types.StandardFiveStarResources `json:"standardFiveStarResources"`
	GachaTypes                types.GachaTypes                `json:"gachaTypes"`
	LuckScoreThresholds       []types.LuckScoreThreshold      `json:"luckScoreThresholds"`
	CostPolicy                types.CostPolicy                `json:"costPolicy"`
	ServerHost                string                          `json:"-"`
	ServerPort                string                          `json:"-"`
	DBPath                    string                          `json:"-"`
	CORSOrigins               []string                        `json:"-"`
	ReportFormat              string                          `json:"-"`
	ReportOutput              string                          `json:"-"`
	Language                  string                          `json:"-"`
	HTTPTimeout               time.Duration                   `json:"-"`
}

func Default() *Config {
	cfg := &Config{}
	cfg.applyDefaults()
	cfg.applyEnv()
	return cfg
}

func Load() (*Config, error) {
	cfg := Default()
	raw, err := FS.ReadFile("config.json")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, cfg); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	cfg.applyEnv()
	return cfg, nil
}

func (cfg *Config) applyDefaults() {
	if cfg.ServerHost == "" {
		cfg.ServerHost = DefaultServerHost
	}
	if cfg.ServerPort == "" {
		cfg.ServerPort = DefaultServerPort
	}
	if cfg.DBPath == "" {
		cfg.DBPath = DefaultDBPath
	}
	if len(cfg.CORSOrigins) == 0 {
		cfg.CORSOrigins = defaultCORSOrigins()
	}
	if cfg.ReportFormat == "" {
		cfg.ReportFormat = DefaultReportFormat
	}
	if cfg.ReportOutput == "" {
		cfg.ReportOutput = DefaultReportOutput
	}
	if cfg.Language == "" {
		cfg.Language = DefaultLanguage
	}
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = DefaultHTTPTimeout
	}
	if cfg.CostPolicy.AstritePerPull == 0 {
		cfg.CostPolicy.AstritePerPull = DefaultAstritePerPull
	}
}

func (cfg *Config) applyEnv() {
	if value := os.Getenv(EnvVarHost); value != "" {
		cfg.ServerHost = value
	}
	if value := os.Getenv(EnvVarPort); value != "" {
		cfg.ServerPort = value
	}
	if value := os.Getenv(EnvVarDBPath); value != "" {
		cfg.DBPath = value
	}
	if value := os.Getenv(EnvVarCORSOrigins); value != "" {
		cfg.CORSOrigins = splitCSV(value)
	}
}

func defaultCORSOrigins() []string {
	return []string{
		DefaultCORSLocalhost,
		DefaultCORSLoopback,
		DefaultCORSIPv6,
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
