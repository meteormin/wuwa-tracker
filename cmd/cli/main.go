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

	"github.com/meteormin/wuwa-tracker/cmd/cli/report"
	"github.com/meteormin/wuwa-tracker/cmd/cli/scan"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
	rep "github.com/meteormin/wuwa-tracker/internal/reporter"
	"github.com/meteormin/wuwa-tracker/internal/scanner"
	"github.com/meteormin/wuwa-tracker/internal/service"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

var buildTag = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	var err error

	switch cmd {
	case "version":
		fmt.Println(buildTag)
	case "scan":
		err = scan.Run(os.Args[2:])
	case "report":
		err = report.Run(os.Args[2:])
	case "run":
		err = runAll(os.Args[2:])
	default:
		fmt.Printf("unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// printUsage 는 CLI 도구의 사용법을 콘솔에 출력합니다.
func printUsage() {
	fmt.Println("Usage: wuwa-tracker <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  version Print build tag")
	fmt.Println("  scan    Scan log files to extract the Wuthering Waves gacha record URL")
	fmt.Println("  report  Fetch gacha records and generate a report (use -url or -f)")
	fmt.Println("  run     Run the entire flow (scan for URL, fetch data, and generate report)")
	fmt.Println()
	fmt.Println("Use 'wuwa-tracker <command> -h' for more information about a command.")
}

// runAll 은 전체 가챠 데이터 추출 및 리포트 생성을 실행하는 run 서브커맨드 로직입니다.
func runAll(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	urlFlag := fs.String("url", "", "Wuthering Waves gacha record URL")
	pathFlag := fs.String("path", "", "Wuthering Waves Game root path to scan for logs")
	formatFlag := fs.String("format", "html", "Report format (json, csv, html)")
	outFlag := fs.String("o", "report", "Output file path (without extension)")
	langFlag := fs.String("lang", "ko", "Report UI language (ko, en)")
	dbPathFlag := fs.String("dbpath", "data/wuwa_badger", "BadgerDB storage directory")
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
		foundURL, err := scanner.FindURLInDirectory(*pathFlag)
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

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var client *tracker.Client
	if *verboseFlag {
		client = tracker.NewClient(&http.Client{
			Transport: &tracker.LoggingTransport{
				Captured: http.DefaultTransport,
			},
			Timeout: 5 * time.Second,
		})
	} else {
		client = tracker.NewClient(&http.Client{
			Timeout: 5 * time.Second,
		})
	}

	localeData := tracker.LoadGachaLocaleWithFallback(client, cfg.GachaLocaleEndpoint, lang)
	cfg.GachaTypes.MapFromLocaleData(localeData)

	badgerDB, err := db.NewBadgerDB(*dbPathFlag)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = badgerDB.Close()
	}()

	calc := tracker.NewStatsCalculator(cfg.StandardFiveStarResources)
	svc, err := service.New(service.Deps{
		DB:     badgerDB,
		Config: cfg,
		Client: client,
		Calc:   calc,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

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

	exporter, err := rep.NewExporter(cfg, format, *langFlag)
	if err != nil {
		return fmt.Errorf("failed to load exporter: %w", err)
	}

	finalOut := *outFlag
	if !strings.HasSuffix(finalOut, "."+*formatFlag) {
		finalOut = finalOut + "." + *formatFlag
	}

	reportData := types.ReportData{
		PlayerID: fetchResult.Payload.PlayerID,
		Stats:    statsResponse.Stats,
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
