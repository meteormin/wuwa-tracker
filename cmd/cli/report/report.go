package report

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
	reporter "github.com/meteormin/wuwa-tracker/internal/reporter"
	"github.com/meteormin/wuwa-tracker/internal/scanner"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

// Run 은 report 서브커맨드를 실행합니다.
// 제공된 가챠 URL을 바탕으로 가챠 통계 데이터를 조회하고 리포트를 생성합니다.
func Run(args []string) error {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	urlFlag := fs.String("url", "", "Wuthering Waves gacha record URL")
	formatFlag := fs.String("format", "html", "Report format (json, csv, html)")
	outFlag := fs.String("o", "report", "Output file path (without extension)")
	verboseFlag := fs.Bool("v", false, "Enable verbose logging")

	if err := fs.Parse(args); err != nil {
		return err
	}

	targetURL := strings.ReplaceAll(*urlFlag, "\\", "")
	if targetURL == "" {
		return fmt.Errorf("url parameter is required. Use -url")
	}

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

	calc := tracker.NewStatsCalculator(cfg.StandardFiveStarResources)

	selectList := tracker.LoadGachaLocaleWithFallback(client, cfg.GachaLocaleEndpoint, lang)
	cfg.GachaTypes.MapFromSelectList(selectList)

	statsList := make([]types.Stats, 0, len(cfg.GachaTypes.Items))

	fetchResult, err := client.FetchAllRecords(targetURL, cfg.GachaTypes.Items)
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

	for _, gachaType := range cfg.GachaTypes.Items {
		records, ok := fetchResult.Records[gachaType.Key]
		if !ok {
			return fmt.Errorf("failed to fetch data: record for %s not found", gachaType.Key)
		}
		statsList = append(statsList, calc.CalculateStats(records, gachaType))
	}

	var format reporter.Format
	switch strings.ToLower(*formatFlag) {
	case "json":
		format = reporter.FormatJSON
	case "csv":
		format = reporter.FormatCSV
	case "html":
		format = reporter.FormatHTML
	default:
		return fmt.Errorf("unsupported format: %s", *formatFlag)
	}

	exporter, err := reporter.NewExporter(cfg, format)
	if err != nil {
		return fmt.Errorf("failed to load exporter: %w", err)
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
