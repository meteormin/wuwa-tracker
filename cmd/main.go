package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/meteormin/wuwa-tracker/cmd/backup"
	"github.com/meteormin/wuwa-tracker/cmd/cli"
	dbcmd "github.com/meteormin/wuwa-tracker/cmd/db"
	"github.com/meteormin/wuwa-tracker/cmd/merge"
	"github.com/meteormin/wuwa-tracker/cmd/report"
	"github.com/meteormin/wuwa-tracker/cmd/scan"
	"github.com/meteormin/wuwa-tracker/cmd/serve"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
	rep "github.com/meteormin/wuwa-tracker/internal/reporter"
	"github.com/meteormin/wuwa-tracker/internal/scanner"
	"github.com/meteormin/wuwa-tracker/internal/service"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
)

var buildTag = "dev"

func main() {
	cfg := config.Default()
	args := os.Args[1:]

	var err error
	if cli.HelpRequested(args) {
		printUsage()
		return
	}
	if len(args) > 0 && args[0] == "help" {
		err = printCommandUsage(cfg, buildTag, args[1:])
	} else if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		err = serve.Runner(cfg, buildTag)(args)
	} else {
		cmd := args[0]
		cmdArgs := args[1:]

		switch cmd {
		case "version":
			if cli.HelpRequested(cmdArgs) {
				printVersionUsage()
				break
			}
			fmt.Println(buildTag)
		case "serve":
			err = serve.Runner(cfg, buildTag)(cmdArgs)
		case "scan":
			err = scan.Runner(cfg)(cmdArgs)
		case "backup":
			err = backup.Runner(cfg)(cmdArgs)
		case "merge":
			err = merge.Runner(cfg)(cmdArgs)
		case "db":
			err = dbcmd.Runner(cfg)(cmdArgs)
		case "report":
			if cli.HelpRequested(cmdArgs) {
				report.PrintUsage(cfg)
			} else {
				err = runWithRuntime(cfg, cmdArgs, report.Runner)
			}
		case "run":
			if cli.HelpRequested(cmdArgs) {
				printRunUsage(cfg)
			} else {
				err = runWithRuntime(cfg, cmdArgs, runAllRunner)
			}
		default:
			fmt.Printf("unknown command: %s\n\n", cmd)
			printUsage()
			os.Exit(1)
		}
	}

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func printCommandUsage(cfg *config.Config, buildTag string, args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "help":
		printUsage()
	case "version":
		printVersionUsage()
	case "serve":
		return serve.Runner(cfg, buildTag)([]string{"help"})
	case "scan":
		return scan.Runner(cfg)([]string{"help"})
	case "backup":
		return backup.Runner(cfg)([]string{"help"})
	case "merge":
		return merge.Runner(cfg)([]string{"help"})
	case "db":
		return dbcmd.Runner(cfg)(append([]string{"help"}, args[1:]...))
	case "report":
		report.PrintUsage(cfg)
	case "run":
		printRunUsage(cfg)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
	return nil
}

type cliRuntime struct {
	svc        *service.Service
	repository *db.BadgerRepository
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
	core, err := db.OpenBadger(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger core: %w", err)
	}
	repository, err := db.NewBadgerRepository(core)
	if err != nil {
		_ = core.Close()
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	calc := tracker.NewStatsCalculator(cfg.StandardFiveStarResources, cfg.CostPolicy)
	svc, err := service.New(service.Deps{
		Repository: repository,
		Config:     cfg,
		Client:     client,
		Calc:       calc,
	})
	if err != nil {
		_ = repository.Close()
		return nil, fmt.Errorf("failed to initialize service: %w", err)
	}

	return &cliRuntime{
		svc:        svc,
		repository: repository,
	}, nil
}

func (r *cliRuntime) Close() error {
	return r.repository.Close()
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
	fmt.Println("Usage: wuwa-tracker [command] [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help    Show usage for the CLI or a specific command")
	fmt.Println("  serve   Run the HTTP server (default)")
	fmt.Println("  version Print build tag")
	fmt.Println("  scan    Scan log files to extract the Wuthering Waves gacha record URL")
	fmt.Println("  backup  Create a Badger repository backup file")
	fmt.Println("  merge   Merge records from a Badger repository backup file")
	fmt.Println("  db      Inspect and maintain the Badger repository storage")
	fmt.Println("  report  Fetch gacha records and generate a report (use -url or -f)")
	fmt.Println("  run     Run the entire flow (scan for URL, fetch data, and generate report)")
	fmt.Println()
	fmt.Println("Use 'wuwa-tracker help <command>' or 'wuwa-tracker <command> -h' for more information about a command.")
}

func printVersionUsage() {
	fmt.Println("Usage: wuwa-tracker version")
	fmt.Println()
	fmt.Println("Print build tag.")
}

func runAllRunner(svc *service.Service) func(args []string) error {
	return func(args []string) error {
		return runAll(svc, args)
	}
}

// runAll 은 전체 가챠 데이터 추출 및 리포트 생성을 실행하는 run 서브커맨드 로직입니다.
func runAll(svc *service.Service, args []string) error {
	defaults := svc.Config()
	fs, urlFlag, pathFlag, formatFlag, outFlag, langFlag, verboseFlag := newRunFlagSet(defaults)
	if handled, err := cli.Parse(fs, args); handled || err != nil {
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
		return fmt.Errorf("failed to load stats from repository: %w", err)
	}

	format, err := rep.ParseFormat(*formatFlag)
	if err != nil {
		return err
	}

	finalOut := *outFlag
	if !strings.HasSuffix(finalOut, "."+string(format)) {
		finalOut = finalOut + "." + string(format)
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

func printRunUsage(defaults *config.Config) {
	fs, _, _, _, _, _, _ := newRunFlagSet(defaults)
	fs.Usage()
}

func newRunFlagSet(defaults *config.Config) (*flag.FlagSet, *string, *string, *string, *string, *string, *bool) {
	fs := cli.NewFlagSet("run", "wuwa-tracker run (-url <gacha-url> | -path <game-root-or-log-path>) [arguments]")
	urlFlag := fs.String("url", "", "Wuthering Waves gacha record URL")
	pathFlag := fs.String("path", "", "Wuthering Waves Game root path to scan for logs")
	formatFlag := fs.String("format", defaults.ReportFormat, "Report format (json, csv, html)")
	outFlag := fs.String("o", defaults.ReportOutput, "Output file path (without extension)")
	langFlag := fs.String("lang", defaults.Language, "Report UI language (ko, en)")
	fs.String("dbpath", defaults.DBPath, "Badger repository storage directory")
	verboseFlag := fs.Bool("v", false, "Enable verbose logging")
	return fs, urlFlag, pathFlag, formatFlag, outFlag, langFlag, verboseFlag
}
