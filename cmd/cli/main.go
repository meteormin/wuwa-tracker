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

	"github.com/meteormin/wuwa-tracker/config"
	report "github.com/meteormin/wuwa-tracker/internal/reporter"
	"github.com/meteormin/wuwa-tracker/internal/scanner"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

func main() {
	urlFlag := flag.String("url", "", "Wuthering Waves gacha record URL")
	pathFlag := flag.String("path", "", "Wuthering Waves Game root path to scan for logs")
	formatFlag := flag.String("format", "html", "Report format (json, csv, html)")
	outFlag := flag.String("out", "report", "Output file path (without extension)")
	verboseFlag := flag.Bool("v", false, "Enable verbose logging")

	flag.Parse()

	// 터미널에서 복사/붙여넣기 시 자동으로 추가되는 백슬래시(\) 이스케이프 문자 제거
	targetURL := strings.ReplaceAll(*urlFlag, "\\", "")
	if targetURL == "" {
		if *pathFlag == "" {
			log.Fatalf("No URL or path provided. Please provide either -url or -path.")
		}

		fmt.Printf("No URL provided. Attempting to scan from path: %s\n", *pathFlag)
		foundURL, err := scanner.FindURLInDirectory(*pathFlag)
		if err != nil {
			log.Fatalf("Failed to auto-scan URL. Please provide it manually via the -url parameter. (Error: %v)", err)
		}
		targetURL = foundURL
		fmt.Println("Successfully scanned URL.")
	}

	fmt.Println("Fetching gacha data. Please wait...")

	// Extract lang from URL to fetch correct localized banner names
	var lang string
	if u, err := url.Parse(targetURL); err == nil {
		if q := u.Query(); q.Get("lang") != "" {
			lang = q.Get("lang")
		} else if u.Fragment != "" { // Sometimes parameters are in the hash fragment
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
		log.Panicf("Failed to load config: %v", err)
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

	calc := tracker.NewStatsCalculator(cfg.StandardFiveStarResources)

	selectList := tracker.LoadGachaLocaleWithFallback(client, cfg.GachaLocaleEndpoint, lang)
	cfg.GachaTypes.MapFromSelectList(selectList)

	statsList := make([]types.Stats, 0, len(cfg.GachaTypes.Items))

	fetchResult, err := client.FetchAllRecords(targetURL, cfg.GachaTypes.Items)
	if err != nil {
		log.Fatalf("Failed to fetch records: %v", err)
	}
	if len(fetchResult.Records) > 0 && *verboseFlag {
		timestamp := time.Now().Format("20060102150405")
		if err := os.MkdirAll("logs", 0o755); err != nil {
			log.Printf("Warning: failed to create logs directory: %v\n", err)
		} else {
			filePath := fmt.Sprintf("logs/%s-%s.json", fetchResult.Payload.PlayerID, timestamp)
			b, err := json.MarshalIndent(fetchResult, "", "    ")
			if err != nil {
				log.Printf("Warning: failed to marshal records: %v\n", err)
			}
			if err := os.WriteFile(filePath, b, 0o644); err != nil {
				log.Printf("Warning: failed to save records JSON to %s: %v\n", filePath, err)
			}
		}
	}

	for _, gachaType := range cfg.GachaTypes.Items {
		records, ok := fetchResult.Records[gachaType.Key]
		if !ok {
			log.Fatalf("Failed to fetch data: %v", err)
		}
		statsList = append(statsList, calc.CalculateStats(records, gachaType))
	}

	var format report.Format
	switch strings.ToLower(*formatFlag) {
	case "json":
		format = report.FormatJSON
	case "csv":
		format = report.FormatCSV
	case "html":
		format = report.FormatHTML
	default:
		log.Fatalf("Unsupported format: %s", *formatFlag)
	}

	exporter, err := report.NewExporter(cfg, format)
	if err != nil {
		log.Fatalf("Failed to load exporter: %v", err)
	}

	finalOut := *outFlag
	if !strings.HasSuffix(finalOut, "."+*formatFlag) {
		finalOut = finalOut + "." + *formatFlag
	}

	reportData := types.ReportData{
		PlayerID: fetchResult.Payload.PlayerID,
		Stats:    statsList,
	}

	f, err := os.Create(finalOut)
	if err != nil {
		log.Fatalf("Failed to create report file: %v", err)
	}
	defer f.Close()

	if err := exporter.Export(f, reportData); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	fmt.Printf("Report successfully generated! File: %s\n", finalOut)
}
