package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/meteormin/wuwa-tracker/cmd/cli/backup"
	dbcmd "github.com/meteormin/wuwa-tracker/cmd/cli/db"
	"github.com/meteormin/wuwa-tracker/cmd/cli/merge"
	"github.com/meteormin/wuwa-tracker/cmd/cli/report"
	"github.com/meteormin/wuwa-tracker/cmd/cli/scan"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
	rep "github.com/meteormin/wuwa-tracker/internal/reporter"
	"github.com/meteormin/wuwa-tracker/internal/scanner"
	"github.com/meteormin/wuwa-tracker/internal/service"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
)

var buildTag = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]
	cfg := config.Default()

	var err error

	switch cmd {
	case "version":
		fmt.Println(buildTag)
	case "scan":
		err = scan.Runner(cfg)(args)
	case "backup":
		cfg.DBPath = extractStringFlag(args, "dbpath", cfg.DBPath)
		err = backup.Runner(cfg)(args)
	case "merge":
		cfg.DBPath = extractStringFlag(args, "dbpath", cfg.DBPath)
		err = merge.Runner(cfg)(args)
	case "db":
		cfg.DBPath = extractStringFlag(args, "dbpath", cfg.DBPath)
		err = dbcmd.Runner(cfg)(args)
	case "report":
		err = runWithRuntime(cfg, args, report.Runner)
	case "run":
		err = runWithRuntime(cfg, args, runAllRunner)
	default:
		fmt.Printf("unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

type cliRuntime struct {
	svc *service.Service
	db  *db.BadgerDB
}

func runWithRuntime(cfg *config.Config, args []string, commandFactory func(*service.Service) func([]string) error) error {
	runtime, err := newCLIRuntime(cfg, args)
	if err != nil {
		return err
	}
	defer func() {
		_ = runtime.Close()
	}()

	return commandFactory(runtime.svc)(args)
}

func newCLIRuntime(cfg *config.Config, args []string) (*cliRuntime, error) {
	cfg.DBPath = extractStringFlag(args, "dbpath", cfg.DBPath)
	client := newTrackerClient(cfg.ResourcesURL, cfg.TrackingURL, cfg.HTTPTimeout, extractBoolFlag(args, "v"))
	badgerDB, err := db.NewBadgerDB(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	calc := tracker.NewStatsCalculator(cfg.StandardFiveStarResources, cfg.CostPolicy)
	svc, err := service.New(service.Deps{
		DB:     badgerDB,
		Config: cfg,
		Client: client,
		Calc:   calc,
	})
	if err != nil {
		_ = badgerDB.Close()
		return nil, fmt.Errorf("failed to initialize service: %w", err)
	}

	return &cliRuntime{
		svc: svc,
		db:  badgerDB,
	}, nil
}

func (r *cliRuntime) Close() error {
	return r.db.Close()
}

func newTrackerClient(resourcesURL, trackingURL string, timeout time.Duration, verbose bool) *tracker.Client {
	if verbose {
		return tracker.NewClient(tracker.Config{
			Client: &http.Client{
				Transport: &tracker.LoggingTransport{
					Captured: http.DefaultTransport,
				},
				Timeout: timeout,
			},
			ResourceURL: resourcesURL,
			TrackingURL: trackingURL,
		})
	}
	return tracker.NewClient(tracker.Config{
		Client: &http.Client{
			Timeout: timeout,
		},
		ResourceURL: resourcesURL,
		TrackingURL: trackingURL,
	})
}

func extractStringFlag(args []string, name, fallback string) string {
	for i, arg := range args {
		if arg == "-"+name || arg == "--"+name {
			if i+1 < len(args) {
				return args[i+1]
			}
			return fallback
		}

		for _, prefix := range []string{"-" + name + "=", "--" + name + "="} {
			if strings.HasPrefix(arg, prefix) {
				return strings.TrimPrefix(arg, prefix)
			}
		}
	}
	return fallback
}

func extractBoolFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == "-"+name || arg == "--"+name {
			return true
		}
		for _, prefix := range []string{"-" + name + "=", "--" + name + "="} {
			if strings.HasPrefix(arg, prefix) {
				return strings.TrimPrefix(arg, prefix) == "true"
			}
		}
	}
	return false
}

// printUsage 는 CLI 도구의 사용법을 콘솔에 출력합니다.
func printUsage() {
	fmt.Println("Usage: wuwa-tracker <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  version Print build tag")
	fmt.Println("  scan    Scan log files to extract the Wuthering Waves gacha record URL")
	fmt.Println("  backup  Create a BadgerDB backup file")
	fmt.Println("  merge   Merge records from a BadgerDB backup file")
	fmt.Println("  db      Inspect and maintain the BadgerDB storage")
	fmt.Println("  report  Fetch gacha records and generate a report (use -url or -f)")
	fmt.Println("  run     Run the entire flow (scan for URL, fetch data, and generate report)")
	fmt.Println()
	fmt.Println("Use 'wuwa-tracker <command> -h' for more information about a command.")
}

func runAllRunner(svc *service.Service) func(args []string) error {
	return func(args []string) error {
		return runAll(svc, args)
	}
}

// runAll 은 전체 가챠 데이터 추출 및 리포트 생성을 실행하는 run 서브커맨드 로직입니다.
func runAll(svc *service.Service, args []string) error {
	defaults := svc.Config()
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	urlFlag := fs.String("url", "", "Wuthering Waves gacha record URL")
	pathFlag := fs.String("path", "", "Wuthering Waves Game root path to scan for logs")
	formatFlag := fs.String("format", defaults.ReportFormat, "Report format (json, csv, html)")
	outFlag := fs.String("o", defaults.ReportOutput, "Output file path (without extension)")
	langFlag := fs.String("lang", defaults.Language, "Report UI language (ko, en)")
	fs.String("dbpath", defaults.DBPath, "BadgerDB storage directory")
	verboseFlag := fs.Bool("v", false, "Enable verbose logging")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// 터미널에서 복사/붙여넣기 시 자동으로 추가되는 백슬래시(\) 이스케이프 문자 제거
	targetURL := strings.ReplaceAll(*urlFlag, "\\", "")
	if targetURL == "" {
		if *pathFlag == "" {
			return fmt.Errorf("no URL or path provided. Please provide either -url or -path")
		}

		fmt.Printf("No URL provided. Attempting to scan from path: %s\n", *pathFlag)
		foundURL, err := svc.Scan(*pathFlag)
		if err != nil {
			return fmt.Errorf("failed to auto-scan URL. Please provide it manually via the -url parameter. (Error: %v)", err)
		}
		targetURL = foundURL
		fmt.Println("Successfully scanned URL.")

		if *verboseFlag {
			fmt.Printf("URL: %s\n", targetURL)
		}
	}

	fmt.Println("Fetching gacha data. Please wait...")

	// URL에서 lang 파라미터 추출하여 지역화에 사용
	var lang string
	if u, err := url.Parse(targetURL); err == nil {
		if q := u.Query(); q.Get("lang") != "" {
			lang = q.Get("lang")
		} else if u.Fragment != "" { // 해시 플래그먼트에 파라미터가 있는 경우 처리
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

	svc.PrepareLocale(lang)

	fetchResult, err := svc.FetchAndSave(targetURL)
	if err != nil {
		return fmt.Errorf("failed to fetch records: %w", err)
	}
	if len(fetchResult.Records) > 0 && *verboseFlag {
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
		return fmt.Errorf("failed to load stats from database: %w", err)
	}

	var format rep.Format
	switch strings.ToLower(*formatFlag) {
	case "json":
		format = rep.FormatJSON
	case "csv":
		format = rep.FormatCSV
	case "html":
		format = rep.FormatHTML
	default:
		return fmt.Errorf("unsupported format: %s", *formatFlag)
	}

	finalOut := *outFlag
	if !strings.HasSuffix(finalOut, "."+*formatFlag) {
		finalOut = finalOut + "." + *formatFlag
	}

	f, err := os.Create(finalOut)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	if len(statsResponse.Stats) == 0 {
		return fmt.Errorf("no valid records found. Report was not generated")
	}

	if err := svc.ExportReport(f, fetchResult.Payload.PlayerID, format, *langFlag); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	fmt.Printf("Report successfully generated! File: %s\n", finalOut)
	return nil
}
