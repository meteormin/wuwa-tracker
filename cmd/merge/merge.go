package merge

import (
	"fmt"
	"os"
	"strings"

	"github.com/meteormin/wuwa-tracker/cmd/cli"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
)

func Runner(cfg *config.Config) func(args []string) error {
	return func(args []string) error {
		return run(cfg, args)
	}
}

// Run 은 merge 서브커맨드를 실행합니다.
// Badger repository 백업 파일을 현재 repository에 가챠 기록 단위로 병합합니다.
func run(cfg *config.Config, args []string) error {
	fs := cli.NewFlagSet("merge", "wuwa-tracker merge -f <backup-file> [arguments]")
	fileFlag := fs.String("f", "", "Path to a Badger repository backup file")
	dbPathFlag := fs.String("dbpath", cfg.DBPath, "Badger repository storage directory")
	if handled, err := cli.Parse(fs, args); handled || err != nil {
		return err
	}

	if strings.TrimSpace(*fileFlag) == "" {
		return fmt.Errorf("backup file path is required. Use -f")
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

	f, err := os.Open(*fileFlag)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	result, err := repository.MergeFromBackup(f)
	if err != nil {
		return fmt.Errorf("failed to merge backup: %w", err)
	}

	fmt.Printf("Backup merged! players=%d banners=%d records=%d\n", result.Players, result.Banners, result.Records)
	return nil
}
