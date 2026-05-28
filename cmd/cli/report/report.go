package report

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
	reporter "github.com/meteormin/wuwa-tracker/internal/reporter"
	"github.com/meteormin/wuwa-tracker/internal/scanner"
	"github.com/meteormin/wuwa-tracker/internal/service"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

// Run 은 report 서브커맨드를 실행합니다.
// -url 제공 시 온라인 모드, -f 제공 시 오프라인 모드로 동작합니다.
func Run(args []string) error {
	defaults := config.Default()
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	urlFlag := fs.String("url", "", "Wuthering Waves gacha record URL")
	fileFlag := fs.String("f", "", "Path to a local JSON log file (offline mode)")
	formatFlag := fs.String("format", defaults.ReportFormat, "Report format (json, csv, html)")
	outFlag := fs.String("o", defaults.ReportOutput, "Output file path (without extension)")
	langFlag := fs.String("lang", defaults.Language, "Report UI language (ko, en)")
	dbPathFlag := fs.String("dbpath", defaults.DBPath, "BadgerDB storage directory")
	verboseFlag := fs.Bool("v", false, "Enable verbose logging")

	if err := fs.Parse(args); err != nil {
		return err
	}

	targetURL := strings.ReplaceAll(*urlFlag, "\\", "")

	if targetURL != "" && *fileFlag != "" {
		return fmt.Errorf("cannot use both -url and -f at the same time")
	}
	if targetURL == "" && *fileFlag == "" {
		return fmt.Errorf("either -url or -f parameter is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg.DBPath = *dbPathFlag

	client := newTrackerClient(*verboseFlag, cfg.HTTPTimeout)

	calc := tracker.NewStatsCalculator(cfg.StandardFiveStarResources)
	badgerDB, err := db.NewBadgerDB(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = badgerDB.Close()
	}()

	svc, err := service.New(service.Deps{
		DB:     badgerDB,
		Config: cfg,
		Client: client,
		Calc:   calc,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	var (
		statsList []types.Stats
		playerID  string
	)

	if *fileFlag != "" {
		// 오프라인 모드: 로컬 JSON 파일에서 리포트 생성
		statsList, playerID, err = runOffline(cfg, svc, *fileFlag)
	} else {
		// 온라인 모드: URL에서 가챠 데이터를 가져와 리포트 생성
		statsList, playerID, err = runOnline(cfg, client, svc, targetURL, *verboseFlag)
	}
	if err != nil {
		return err
	}

	if len(statsList) == 0 {
		return fmt.Errorf("no valid records found. Report was not generated")
	}

	return exportReport(cfg, statsList, playerID, *formatFlag, *outFlag, *langFlag)
}

// runOffline 은 로컬 JSON 파일에서 데이터를 읽어 통계를 계산합니다.
func runOffline(cfg *config.Config, svc *service.Service, filePath string) ([]types.Stats, string, error) {
	fmt.Printf("Reading local file: %s\n", filePath)

	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read log file %s: %w", filePath, err)
	}

	var playerID string
	var recordsMap map[string][]types.Record

	// FetchResult 포맷 시도 (신규 포맷)
	var fetchResult types.FetchResult
	if err := json.Unmarshal(b, &fetchResult); err == nil && len(fetchResult.Records) > 0 {
		playerID = fetchResult.Payload.PlayerID
		recordsMap = fetchResult.Records
	} else {
		// Legacy 포맷: map[string][]types.Record
		if err := json.Unmarshal(b, &recordsMap); err != nil {
			return nil, "", fmt.Errorf("failed to parse JSON from %s: %w", filePath, err)
		}
		playerID = playerIDFromFileName(filePath)
	}
	if playerID == "" {
		playerID = playerIDFromFileName(filePath)
	}
	fetchResult.Payload.PlayerID = playerID
	fetchResult.Records = recordsMap

	// 오프라인에서는 locale API를 호출하지 않으므로 Key를 Name으로 사용
	for i := range cfg.GachaTypes.Items {
		cfg.GachaTypes.Items[i].Name = cfg.GachaTypes.Items[i].Key
	}

	if _, err := svc.Upload(fetchResult); err != nil {
		return nil, "", err
	}
	statsResponse, err := svc.GetStats(playerID)
	if err != nil {
		return nil, "", err
	}

	return statsResponse.Stats, statsResponse.PlayerID, nil
}

func playerIDFromFileName(filePath string) string {
	baseName := filepath.Base(filePath)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if parts := strings.Split(baseName, "-"); len(parts) > 1 {
		return parts[0]
	}
	return baseName
}

// runOnline 은 URL에서 가챠 데이터를 가져와 통계를 계산합니다.
func runOnline(cfg *config.Config, client *tracker.Client, svc *service.Service, targetURL string, verbose bool) ([]types.Stats, string, error) {
	fmt.Println("Fetching gacha data. Please wait...")

	// URL에서 lang 파라미터 추출하여 지역화에 사용
	var lang string
	if u, err := url.Parse(targetURL); err == nil {
		if q := u.Query(); q.Get("lang") != "" {
			lang = q.Get("lang")
		} else if u.Fragment != "" {
			parts := strings.SplitN(u.Fragment, "?", 2)
			if len(parts) == 2 {
				q, _ := url.ParseQuery(parts[1])
				if q.Get("lang") != "" {
					lang = q.Get("lang")
				}
			}
		}
	}

	if lang == "" {
		lang = scanner.GetSystemLocale()
	}

	localeData := tracker.LoadGachaLocaleWithFallback(client, cfg.GachaLocaleEndpoint, lang)
	cfg.GachaTypes.MapFromLocaleData(localeData)

	fetchResult, err := svc.FetchAndSave(targetURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch records: %w", err)
	}

	// verbose 모드 시 로그 저장
	if len(fetchResult.Records) > 0 && verbose {
		timestamp := time.Now().Format("20060102150405")
		if err := os.MkdirAll("logs", 0o755); err != nil {
			log.Printf("Warning: failed to create logs directory: %v\n", err)
		} else {
			filePath := fmt.Sprintf("logs/%s-%s.json", fetchResult.Payload.PlayerID, timestamp)
			b, err := json.MarshalIndent(fetchResult, "", "    ")
			if err == nil {
				if err := os.WriteFile(filePath, b, 0o644); err != nil {
					log.Printf("Warning: failed to save records JSON to %s: %v\n", filePath, err)
				}
			}
		}
	}

	statsResponse, err := svc.GetStats(fetchResult.Payload.PlayerID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load stats from database: %w", err)
	}

	return statsResponse.Stats, statsResponse.PlayerID, nil
}

func newTrackerClient(verbose bool, timeout time.Duration) *tracker.Client {
	if verbose {
		return tracker.NewClient(&http.Client{
			Transport: &tracker.LoggingTransport{
				Captured: http.DefaultTransport,
			},
			Timeout: timeout,
		})
	}
	return tracker.NewClient(&http.Client{
		Timeout: timeout,
	})
}

// exportReport 는 통계 데이터를 지정된 포맷으로 파일에 출력합니다.
func exportReport(cfg *config.Config, statsList []types.Stats, playerID, formatFlag, outFlag, lang string) error {
	var format reporter.Format
	switch strings.ToLower(formatFlag) {
	case "json":
		format = reporter.FormatJSON
	case "csv":
		format = reporter.FormatCSV
	case "html":
		format = reporter.FormatHTML
	default:
		return fmt.Errorf("unsupported format: %s", formatFlag)
	}

	exporter, err := reporter.NewExporter(cfg, format, lang)
	if err != nil {
		return fmt.Errorf("failed to load exporter: %w", err)
	}

	finalOut := outFlag
	if !strings.HasSuffix(finalOut, "."+formatFlag) {
		finalOut = finalOut + "." + formatFlag
	}

	// 출력 디렉토리가 존재하지 않으면 자동 생성
	outDir := filepath.Dir(finalOut)
	if outDir != "." && outDir != "" {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			log.Printf("Warning: failed to create output directory %s: %v\n", outDir, err)
		}
	}

	reportData := types.ReportData{
		PlayerID: playerID,
		Stats:    statsList,
	}

	f, err := os.Create(finalOut)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	if err := exporter.Export(f, reportData); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	fmt.Printf("Report successfully generated! File: %s\n", finalOut)
	return nil
}
