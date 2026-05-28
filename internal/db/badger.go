package db

import (
	"encoding/json"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

type BadgerDB struct {
	core *badger.DB
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
