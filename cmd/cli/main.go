package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/meteormin/wuwa-tracker/internal/report"
	"github.com/meteormin/wuwa-tracker/internal/scanner"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
)

func main() {
	urlFlag := flag.String("url", "", "Wuthering Waves gacha record URL")
	pathFlag := flag.String("path", "", "Wuthering Waves Game root path to scan for logs")
	formatFlag := flag.String("format", "html", "Report format (json, csv, html)")
	outFlag := flag.String("out", "report", "Output file path (without extension)")

	flag.Parse()

	targetURL := *urlFlag

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

	recordsMap, err := tracker.FetchAll(targetURL)
	if err != nil {
		log.Fatalf("Failed to fetch data: %v", err)
	}

	statsMap := make(map[int]tracker.Stats)
	for gachaType, records := range recordsMap {
		statsMap[gachaType] = tracker.CalculateStats(gachaType, records)
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

	exporter, err := report.GetExporter(format)
	if err != nil {
		log.Fatalf("Failed to load exporter: %v", err)
	}

	finalOut := *outFlag
	if !strings.HasSuffix(finalOut, "."+*formatFlag) {
		finalOut = finalOut + "." + *formatFlag
	}

	if err := exporter.Export(statsMap, finalOut); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	fmt.Printf("Report successfully generated! File: %s\n", finalOut)
}
