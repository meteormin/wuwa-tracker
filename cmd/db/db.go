package dbcmd

import (
	"flag"
	"fmt"
	"strings"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
)

func Runner(cfg *config.Config) func(args []string) error {
	return func(args []string) error {
		return run(cfg, args)
	}
}

// run 은 db 관리 서브커맨드를 실행합니다.
func run(cfg *config.Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing db command\n\n%s", usage())
	}

	switch args[0] {
	case "stats":
		return runStats(cfg, args[1:])
	case "gc":
		return runGC(cfg, args[1:])
	default:
		return fmt.Errorf("unknown db command: %s\n\n%s", args[0], usage())
	}
}

func usage() string {
	return strings.Join([]string{
		"Usage: wuwa-tracker db <command> [arguments]",
		"",
		"Commands:",
		"  stats  Inspect Badger repository storage size",
		"  gc     Run Badger repository value log garbage collection",
		"",
		"Use 'wuwa-tracker db <command> -h' for more information about a command.",
	}, "\n")
}

func runStats(cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("db stats", flag.ExitOnError)
	dbPathFlag := fs.String("dbpath", cfg.DBPath, "Badger repository storage directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	stats, err := statsFromDBPath(*dbPathFlag)
	if err != nil {
		return fmt.Errorf("failed to inspect repository size: %w", err)
	}

	printStats("DB Stats", stats)
	return nil
}

func runGC(cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("db gc", flag.ExitOnError)
	dbPathFlag := fs.String("dbpath", cfg.DBPath, "Badger repository storage directory")
	discardRatioFlag := fs.Float64("discard-ratio", cfg.DBGCDiscardRatio, "Badger value log discard ratio")
	if err := fs.Parse(args); err != nil {
		return err
	}

	core, err := db.OpenBadger(*dbPathFlag)
	if err != nil {
		return fmt.Errorf("failed to open badger core: %w", err)
	}
	repository, err := db.NewBadgerRepository(core)
	if err != nil {
		_ = core.Close()
		return fmt.Errorf("failed to initialize repository: %w", err)
	}
	defer func() {
		_ = repository.Close()
	}()

	before, err := repository.Stats()
	if err != nil {
		return fmt.Errorf("failed to inspect repository size before gc: %w", err)
	}

	if err := repository.RunValueLogGC(*discardRatioFlag); err != nil {
		return fmt.Errorf("failed to run value log gc: %w", err)
	}

	after, err := repository.Stats()
	if err != nil {
		return fmt.Errorf("failed to inspect repository size after gc: %w", err)
	}

	printStats("Before GC", before)
	fmt.Println()
	printStats("After GC", after)
	return nil
}

func statsFromDBPath(path string) (db.Stats, error) {
	core, err := db.OpenBadger(path)
	if err != nil {
		return db.StatsFromPath(path)
	}
	repository, err := db.NewBadgerRepository(core)
	if err == nil {
		defer func() {
			_ = repository.Close()
		}()
		return repository.Stats()
	}
	_ = core.Close()

	return db.StatsFromPath(path)
}

func printStats(title string, stats db.Stats) {
	fmt.Println(title)
	fmt.Printf("Path: %s\n", stats.Path)
	fmt.Printf("Files: %d\n", stats.FileCount)
	fmt.Printf("Apparent Size: %s (%d bytes)\n", formatBytes(stats.ApparentSizeBytes), stats.ApparentSizeBytes)
	fmt.Printf("Disk Usage: %s (%d bytes)\n", formatBytes(stats.DiskUsageBytes), stats.DiskUsageBytes)
	fmt.Printf("LSM Size: %s (%d bytes)\n", formatBytes(stats.LSMSizeBytes), stats.LSMSizeBytes)
	fmt.Printf("VLog Size: %s (%d bytes)\n", formatBytes(stats.VLogSizeBytes), stats.VLogSizeBytes)
	fmt.Printf("VLog Files: %d\n", stats.VLogCount)
	fmt.Printf("SST Files: %d\n", stats.SSTCount)
	fmt.Printf("MemTable Files: %d\n", stats.MemTableCount)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	value := float64(bytes)
	for _, suffix := range []string{"KB", "MB", "GB", "TB"} {
		value /= unit
		if value < unit {
			return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".") + " " + suffix
		}
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".") + " PB"
}
