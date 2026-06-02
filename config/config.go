package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

const (
	EnvVarHost        = "WUWA_TRACKER_HOST"
	EnvVarPort        = "WUWA_TRACKER_PORT"
	EnvVarDBPath      = "WUWA_TRACKER_DB_PATH"
	EnvVarCORSOrigins = "WUWA_TRACKER_CORS_ORIGINS"

	DefaultGachaLocaleEndpoint = "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/locales"
	DefaultServerHost          = "127.0.0.1"
	DefaultServerPort          = "3000"
	DefaultDBPath              = "data/wuwa_badger"
	DefaultCORSLocalhost       = "http://localhost:5173"
	DefaultCORSLoopback        = "http://127.0.0.1:5173"
	DefaultCORSIPv6            = "http://[::1]:5173"
	DefaultReportFormat        = "html"
	DefaultReportOutput        = "report"
	DefaultLanguage            = "ko"
	DefaultHTTPTimeout         = 5 * time.Second
	DefaultAstritePerPull      = 160
)

var (
	DefaultScanLogPaths = []string{
		// Windows 기본 로그 경로입니다.
		filepath.Join("Client", "Saved", "Logs", "Client.log"),
		filepath.Join("Client", "Binaries", "Win64", "ThirdParty", "KrPcSdk_Global", "KRSDKRes", "KRSDKWebView", "debug.log"),
		// macOS 앱 컨테이너 로그 경로입니다.
		filepath.Join("Data", "Library", "Logs", "Client", "Client.log"),
		// 사용자가 Logs 하위 경로를 직접 입력한 경우의 fallback 입니다.
		filepath.Join("Client", "Client.log"),
		"Client.log",
	}

	DefaultStandardFiveStarResources = []int{
		1203, // 앙코
		1301, // 카카루
		1405, // 감심
		1104, // 능양
		1503, // 벨리나
	}

	DefaultGachaTypes = []types.GachaType{
		{
			ID:               1,
			Key:              "characterEvent",
			HasOffBannerDrop: true,
			BaseRate:         0.8,
			ExpectedPulls:    80,
		},
		{
			ID:               2,
			Key:              "weaponEvent",
			HasOffBannerDrop: false,
			BaseRate:         0.8,
			ExpectedPulls:    55,
		},
		{
			ID:               3,
			Key:              "characterPermanent",
			HasOffBannerDrop: false,
			BaseRate:         0.8,
			ExpectedPulls:    55,
		},
		{
			ID:               4,
			Key:              "weaponPermanent",
			HasOffBannerDrop: false,
			BaseRate:         0.8,
			ExpectedPulls:    55,
		},
		{
			ID:               5,
			Key:              "beginner",
			HasOffBannerDrop: true,
			BaseRate:         0.8,
			ExpectedPulls:    55,
		},
		{
			ID:               6,
			Key:              "beginnerSelect",
			HasOffBannerDrop: false,
			BaseRate:         0.8,
			ExpectedPulls:    55,
		},
		{
			ID:               8,
			Key:              "characterNovice",
			HasOffBannerDrop: true,
			BaseRate:         0.8,
			ExpectedPulls:    80,
		},
		{
			ID:               9,
			Key:              "weaponNovice",
			HasOffBannerDrop: false,
			BaseRate:         0.8,
			ExpectedPulls:    55,
		},
	}

	DefaultLuckScoreThresholds = []types.LuckScoreThreshold{
		{
			MinScore: 0.0,
			State:    "worst",
		},
		{
			MinScore: 85.0,
			State:    "bad",
		},
		{
			MinScore: 95.0,
			State:    "normal",
		},
		{
			MinScore: 102.0,
			State:    "good",
		},
		{
			MinScore: 115.0,
			State:    "best",
		},
	}
)

type Config struct {
	GachaLocaleEndpoint       string                     `json:"gachaLocaleEndpoint"`
	StandardFiveStarResources []int                      `json:"standardFiveStarResources"`
	GachaTypes                types.GachaTypes           `json:"gachaTypes"`
	LuckScoreThresholds       []types.LuckScoreThreshold `json:"luckScoreThresholds"`
	CostPolicy                types.CostPolicy           `json:"costPolicy"`
	ScanLogPaths              []string                   `json:"scanLogPaths"`
	ServerHost                string                     `json:"serverHost"`
	ServerPort                string                     `json:"serverPort"`
	DBPath                    string                     `json:"dbPath"`
	CORSOrigins               []string                   `json:"corsOrigins"`
	ReportFormat              string                     `json:"reportFormat"`
	ReportOutput              string                     `json:"reportOutput"`
	Language                  string                     `json:"language"`
	HTTPTimeout               time.Duration              `json:"httpTimeout"`
}

func Default() *Config {
	cfg := &Config{
		GachaLocaleEndpoint:       DefaultGachaLocaleEndpoint,
		StandardFiveStarResources: slicesClone(DefaultStandardFiveStarResources),
		GachaTypes: types.GachaTypes{
			Items: slicesClone(DefaultGachaTypes),
		},
		LuckScoreThresholds: slicesClone(DefaultLuckScoreThresholds),
		CostPolicy: types.CostPolicy{
			AstritePerPull: DefaultAstritePerPull,
		},
		ScanLogPaths: slicesClone(DefaultScanLogPaths),
		ServerHost:   DefaultServerHost,
		ServerPort:   DefaultServerPort,
		DBPath:       DefaultDBPath,
		CORSOrigins:  defaultCORSOrigins(),
		ReportFormat: DefaultReportFormat,
		ReportOutput: DefaultReportOutput,
		Language:     DefaultLanguage,
		HTTPTimeout:  DefaultHTTPTimeout,
	}
	cfg.applyEnv()
	return cfg
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
		cfg.CORSOrigins = splitComma(value)
	}
}

func defaultCORSOrigins() []string {
	return []string{
		DefaultCORSLocalhost,
		DefaultCORSLoopback,
		DefaultCORSIPv6,
	}
}

func splitComma(value string) []string {
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

func slicesClone[S ~[]E, E any](s S) S {
	return append(S(nil), s...)
}
