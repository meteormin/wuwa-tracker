package db

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"

	"github.com/dgraph-io/badger/v4"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

const backupLoadMaxPendingWrites = 256
const defaultValueLogGCDiscardRatio = 0.5

var ErrMissingBadgerDB = errors.New("missing badger db")

type BadgerRepository struct {
	db          *badger.DB
	mergePolicy MergePolicy
	mu          sync.Mutex
	gcWorker    *valueLogGCWorker
	closed      bool
}

type BadgerRepositoryOption func(*BadgerRepository)

func WithMergePolicy(policy MergePolicy) BadgerRepositoryOption {
	return func(repository *BadgerRepository) {
		if policy != nil {
			repository.mergePolicy = policy
		}
	}
}

type MergeFromBackupResult struct {
	Players int
	Banners int
	Records int
}

type Stats struct {
	Path              string
	ApparentSizeBytes int64
	DiskUsageBytes    int64
	LSMSizeBytes      int64
	VLogSizeBytes     int64
	FileCount         int
	VLogCount         int
	SSTCount          int
	MemTableCount     int
}

type ValueLogGCOptions struct {
	Interval     time.Duration
	DiscardRatio float64
}

type valueLogGCWorker struct {
	stop context.CancelFunc
	done chan struct{}
	once sync.Once
}

// OpenBadger 는 지정된 디렉터리에 Badger 엔진을 기동하고 반환합니다.
func OpenBadger(path string) (*badger.DB, error) {
	opts := badger.DefaultOptions(path).WithLogger(nil)
	return badger.Open(opts)
}

// NewBadgerRepository 는 주입된 Badger 엔진을 가챠 기록 repository로 감쌉니다.
func NewBadgerRepository(db *badger.DB, opts ...BadgerRepositoryOption) (*BadgerRepository, error) {
	if db == nil {
		return nil, ErrMissingBadgerDB
	}
	repository := &BadgerRepository{
		db:          db,
		mergePolicy: SequenceMergePolicy{},
	}
	for _, opt := range opts {
		opt(repository)
	}
	return repository, nil
}

// Close 는 Badger 기반 repository 커넥션을 닫습니다.
func (b *BadgerRepository) Close() error {
	b.StopValueLogGC()

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	if err := b.db.Close(); err != nil {
		return err
	}
	b.closed = true
	return nil
}

// Backup 은 현재 repository 데이터를 단일 백업 스트림으로 내보냅니다.
func (b *BadgerRepository) Backup(w io.Writer) (uint64, error) {
	return b.db.Backup(w, 0)
}

// Stats 는 repository 디렉터리의 논리 크기와 실제 디스크 사용량을 집계합니다.
func (b *BadgerRepository) Stats() (Stats, error) {
	stats, err := StatsFromPath(b.db.Opts().Dir)
	if err != nil {
		return Stats{}, err
	}

	lsmSize, vlogSize := b.db.Size()
	stats.LSMSizeBytes = lsmSize
	stats.VLogSizeBytes = vlogSize
	if badgerSize := lsmSize + vlogSize; badgerSize > 0 {
		stats.ApparentSizeBytes = badgerSize
	}
	return stats, nil
}

// StatsFromPath 는 지정된 경로의 파일 크기 통계를 재귀적으로 집계합니다.
func StatsFromPath(path string) (Stats, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Stats{}, err
	}
	if !info.IsDir() {
		return Stats{}, fmt.Errorf("path is not a directory: %s", path)
	}

	stats := Stats{
		Path: path,
	}
	err = filepath.WalkDir(path, func(currentPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}

		stats.FileCount++
		stats.ApparentSizeBytes += fileInfo.Size()
		stats.DiskUsageBytes += diskUsageBytes(fileInfo)

		switch filepath.Ext(currentPath) {
		case ".vlog":
			stats.VLogCount++
		case ".sst":
			stats.SSTCount++
		case ".mem":
			stats.MemTableCount++
		}
		return nil
	})
	if err != nil {
		return Stats{}, err
	}

	return stats, nil
}

// RunValueLogGC 는 정리 가능한 Badger value log를 가능한 만큼 정리합니다.
func (b *BadgerRepository) RunValueLogGC(discardRatio float64) error {
	normalizedDiscardRatio, err := normalizeValueLogGCDiscardRatio(discardRatio)
	if err != nil {
		return err
	}

	for {
		if err := b.runValueLogGCOnce(normalizedDiscardRatio); err != nil {
			if errors.Is(err, badger.ErrNoRewrite) {
				return nil
			}
			return err
		}
	}
}

// StartValueLogGC 는 서버 환경에서 주기적으로 value log GC를 1회씩 시도합니다.
func (b *BadgerRepository) StartValueLogGC(ctx context.Context, opts ValueLogGCOptions, onError func(error)) error {
	if opts.Interval <= 0 {
		return nil
	}

	discardRatio, err := normalizeValueLogGCDiscardRatio(opts.DiscardRatio)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return fmt.Errorf("repository is closed")
	}
	if b.gcWorker != nil {
		return nil
	}

	workerCtx, stop := context.WithCancel(ctx)
	worker := &valueLogGCWorker{
		stop: stop,
		done: make(chan struct{}),
	}
	b.gcWorker = worker

	go func() {
		defer close(worker.done)

		ticker := time.NewTicker(opts.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				if err := b.runValueLogGCOnce(discardRatio); err != nil && !errors.Is(err, badger.ErrNoRewrite) {
					if onError != nil {
						onError(err)
					}
				}
			}
		}
	}()

	return nil
}

// StopValueLogGC 는 실행 중인 value log GC 루프를 멈춥니다.
func (b *BadgerRepository) StopValueLogGC() {
	b.mu.Lock()
	worker := b.gcWorker
	b.gcWorker = nil
	b.mu.Unlock()

	worker.Stop()
}

// Stop 은 value log GC 루프를 멈추고 진행 중인 GC 1회가 끝날 때까지 대기합니다.
func (w *valueLogGCWorker) Stop() {
	if w == nil {
		return
	}
	w.once.Do(func() {
		w.stop()
		<-w.done
	})
}

func (b *BadgerRepository) runValueLogGCOnce(discardRatio float64) error {
	return b.db.RunValueLogGC(discardRatio)
}

func normalizeValueLogGCDiscardRatio(discardRatio float64) (float64, error) {
	if discardRatio <= 0 {
		return defaultValueLogGCDiscardRatio, nil
	}
	if discardRatio >= 1 {
		return 0, fmt.Errorf("discard ratio must be less than 1: %.2f", discardRatio)
	}
	return discardRatio, nil
}

// MergeFromBackup 은 Badger 백업 스트림을 임시 DB에 복원한 뒤 현재 DB에 가챠 기록을 병합합니다.
func (b *BadgerRepository) MergeFromBackup(r io.Reader) (MergeFromBackupResult, error) {
	tmpDir, err := os.MkdirTemp("", "wuwa-tracker-merge-*")
	if err != nil {
		return MergeFromBackupResult{}, err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	importedCore, err := OpenBadger(tmpDir)
	if err != nil {
		return MergeFromBackupResult{}, err
	}
	imported, err := NewBadgerRepository(importedCore)
	if err != nil {
		_ = importedCore.Close()
		return MergeFromBackupResult{}, err
	}
	defer func() {
		_ = imported.Close()
	}()

	if err := imported.db.Load(r, backupLoadMaxPendingWrites); err != nil {
		return MergeFromBackupResult{}, err
	}

	recordSets, err := imported.listGachaRecordSets()
	if err != nil {
		return MergeFromBackupResult{}, err
	}

	players := make(map[string]bool)
	result := MergeFromBackupResult{}
	for _, recordSet := range recordSets {
		if err := b.SaveGachaRecords(recordSet.playerID, recordSet.cardPoolType, recordSet.records); err != nil {
			return MergeFromBackupResult{}, err
		}
		players[recordSet.playerID] = true
		result.Banners++
		result.Records += len(recordSet.records)
	}
	result.Players = len(players)

	return result, nil
}

// SaveGachaRecords 는 특정 플레이어의 특정 배너 가챠 리스트를 기존 데이터와 병합하여 repository에 저장합니다.
// 명조 API가 최근 6개월 데이터만 제공하므로, 기존 데이터를 보존하면서 새로운 데이터를 증분 병합합니다.
func (b *BadgerRepository) SaveGachaRecords(playerID, cardPoolType string, records []types.Record) error {
	if records == nil {
		records = []types.Record{}
	}

	// 기존 가챠 기록 로드
	existing, err := b.GetGachaRecords(playerID, cardPoolType)
	if err != nil {
		return err
	}

	// 기존 기록과 신규 기록 병합
	merged := b.mergePolicy.Merge(existing, records)

	recordsJSON, err := json.Marshal(merged)
	if err != nil {
		return err
	}

	key := []byte("gacha:" + playerID + ":" + cardPoolType)
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, recordsJSON)
	})
}

// GetGachaRecords 는 특정 플레이어의 특정 배너 가챠 기록을 복원하여 반환합니다.
func (b *BadgerRepository) GetGachaRecords(playerID, cardPoolType string) ([]types.Record, error) {
	var records []types.Record
	key := []byte("gacha:" + playerID + ":" + cardPoolType)

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &records)
		})
	})
	if err != nil {
		return nil, err
	}

	if records == nil {
		records = []types.Record{}
	}

	return records, nil
}

// ListPlayers 는 접두사 "gacha:"를 가진 모든 키를 Key-Only 스캔으로 초고속 순회하여
// 고유 플레이어 ID 목록을 수 마이크로초 내에 파싱 및 리스트업합니다.
func (b *BadgerRepository) ListPlayers() ([]string, error) {
	playersMap := make(map[string]bool)

	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // 값은 읽지 않고 키명만 조회하여 성능 극대화

		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("gacha:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().Key()
			parts := strings.Split(string(key), ":")
			if len(parts) >= 2 && parts[0] == "gacha" {
				playersMap[parts[1]] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	players := make([]string, 0, len(playersMap))
	for p := range playersMap {
		players = append(players, p)
	}

	return players, nil
}

type gachaRecordSet struct {
	playerID     string
	cardPoolType string
	records      []types.Record
}

func (b *BadgerRepository) listGachaRecordSets() ([]gachaRecordSet, error) {
	var recordSets []gachaRecordSet

	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true

		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("gacha:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := string(it.Item().Key())
			parts := strings.SplitN(key, ":", 3)
			if len(parts) != 3 || parts[0] != "gacha" || parts[1] == "" || parts[2] == "" {
				return fmt.Errorf("invalid gacha key: %s", key)
			}

			var records []types.Record
			if err := it.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &records)
			}); err != nil {
				return err
			}
			if records == nil {
				records = []types.Record{}
			}

			recordSets = append(recordSets, gachaRecordSet{
				playerID:     parts[1],
				cardPoolType: parts[2],
				records:      records,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return recordSets, nil
}
