package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/meteormin/wuwa-tracker/config"
	report "github.com/meteormin/wuwa-tracker/internal/reporter"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

func main() {
	in := flag.String("in", "", "Relative path of the single JSON log file (e.g. logs/20260518143618.json)")
	out := flag.String("out", "report", "Relative path to output report (without extension)")
	formatFlag := flag.String("format", "html", "Report format (json, csv, html)")

	flag.Parse()

	if *in == "" {
		log.Fatalf("Error: -in parameter (relative path of single JSON log file) is required")
	}

	// config 로드
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 로케일 대신 Key 필드 값을 그대로 Name으로 사용 (외부 API 요청 없음)
	for i := range cfg.GachaTypes.Items {
		cfg.GachaTypes.Items[i].Name = cfg.GachaTypes.Items[i].Key
	}

	calc := tracker.NewStatsCalculator(cfg.StandardFiveStarResources)
	statsList := make([]types.Stats, 0, len(cfg.GachaTypes.Items))

	// 단일 JSON 파일을 읽어옵니다.
	b, err := os.ReadFile(*in)
	if err != nil {
		log.Fatalf("Error: failed to read log file %s: %v", *in, err)
	}

	var playerID string
	var recordsMap map[string][]types.Record

	// Try to unmarshal as FetchResult first (new format containing payload and player ID)
	var fetchResult types.FetchResult
	if err := json.Unmarshal(b, &fetchResult); err == nil && len(fetchResult.Records) > 0 {
		playerID = fetchResult.Payload.PlayerID
		recordsMap = fetchResult.Records
	} else {
		// Fallback: try to unmarshal as map[string][]types.Record (old format)
		if err := json.Unmarshal(b, &recordsMap); err != nil {
			log.Fatalf("Error: failed to parse JSON from %s: %v", *in, err)
		}
		// Try to extract player ID from file name if possible
		baseName := filepath.Base(*in)
		baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
		if parts := strings.Split(baseName, "-"); len(parts) > 1 {
			playerID = parts[0]
		} else {
			playerID = baseName
		}
	}

	for _, gachaType := range cfg.GachaTypes.Items {
		records, ok := recordsMap[gachaType.Key]
		if !ok {
			log.Printf("Warning: gacha type key %q not found in log file. Skipping...", gachaType.Key)
			continue
		}
		statsList = append(statsList, calc.CalculateStats(records, gachaType))
	}

	if len(statsList) == 0 {
		log.Fatalf("Error: no valid logs found in %s. Report was not generated.", *in)
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
		log.Fatalf("Failed to get exporter: %v", err)
	}

	finalOut := *out
	if !strings.HasSuffix(finalOut, "."+*formatFlag) {
		finalOut = finalOut + "." + *formatFlag
	}

	// 출력 디렉토리가 존재하지 않는다면 자동 생성합니다.
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
		log.Fatalf("Failed to create report file: %v", err)
	}
	defer func() {
		_ = f.Close()
	}()

	if err := exporter.Export(f, reportData); err != nil {
		log.Fatalf("Failed to export report: %v", err)
	}

	fmt.Printf("Report successfully generated! File: %s\n", finalOut)
}
