package db

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

const backupLoadMaxPendingWrites = 256

type BadgerDB struct {
	core *badger.DB
}

type MergeFromBackupResult struct {
	Players int
	Banners int
	Records int
}

// NewBadgerDB 는 지정된 디렉터리에 BadgerDB 데이터베이스를 기동하고 커넥션을 반환합니다.
// 디버그/압축 통계 로그 스팸을 방지하기 위해 로거는 nil로 초기화합니다.
func NewBadgerDB(path string) (*BadgerDB, error) {
	opts := badger.DefaultOptions(path).WithLogger(nil)
	badgerDB, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerDB{
		core: badgerDB,
	}, nil
}

// Close 는 BadgerDB 데이터베이스 커넥션을 닫습니다.
func (b *BadgerDB) Close() error {
	return b.core.Close()
}

// Backup 은 현재 BadgerDB 데이터를 단일 백업 스트림으로 내보냅니다.
func (b *BadgerDB) Backup(w io.Writer) (uint64, error) {
	return b.core.Backup(w, 0)
}

// MergeFromBackup 은 Badger 백업 스트림을 임시 DB에 복원한 뒤 현재 DB에 가챠 기록을 병합합니다.
func (b *BadgerDB) MergeFromBackup(r io.Reader) (MergeFromBackupResult, error) {
	tmpDir, err := os.MkdirTemp("", "wuwa-tracker-merge-*")
	if err != nil {
		return MergeFromBackupResult{}, err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	imported, err := NewBadgerDB(tmpDir)
	if err != nil {
		return MergeFromBackupResult{}, err
	}
	defer func() {
		_ = imported.Close()
	}()

	if err := imported.core.Load(r, backupLoadMaxPendingWrites); err != nil {
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

// SaveGachaRecords 는 특정 플레이어의 특정 배너 가챠 리스트를 기존 데이터와 병합하여 BadgerDB에 저장합니다.
// 명조 API가 최근 6개월 데이터만 제공하므로, 기존 데이터를 보존하면서 새로운 데이터를 증분 병합합니다.
func (b *BadgerDB) SaveGachaRecords(playerID, cardPoolType string, records []types.Record) error {
	if records == nil {
		records = []types.Record{}
	}

	// 기존 가챠 기록 로드
	existing, err := b.GetGachaRecords(playerID, cardPoolType)
	if err != nil {
		return err
	}

	// 기존 기록과 신규 기록 병합
	merged := MergeRecords(existing, records)

	recordsJSON, err := json.Marshal(merged)
	if err != nil {
		return err
	}

	key := []byte("gacha:" + playerID + ":" + cardPoolType)
	return b.core.Update(func(txn *badger.Txn) error {
		return txn.Set(key, recordsJSON)
	})
}

// GetGachaRecords 는 특정 플레이어의 특정 배너 가챠 기록을 복원하여 반환합니다.
func (b *BadgerDB) GetGachaRecords(playerID, cardPoolType string) ([]types.Record, error) {
	var records []types.Record
	key := []byte("gacha:" + playerID + ":" + cardPoolType)

	err := b.core.View(func(txn *badger.Txn) error {
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
func (b *BadgerDB) ListPlayers() ([]string, error) {
	playersMap := make(map[string]bool)

	err := b.core.View(func(txn *badger.Txn) error {
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

func (b *BadgerDB) listGachaRecordSets() ([]gachaRecordSet, error) {
	var recordSets []gachaRecordSet

	err := b.core.View(func(txn *badger.Txn) error {
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
