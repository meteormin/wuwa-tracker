package models

import (
	"encoding/json"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

// InitDB 는 지정된 디렉터리에 BadgerDB 데이터베이스를 기동하고 커넥션을 반환합니다.
// 디버그/압축 통계 로그 스팸을 방지하기 위해 로거는 nil로 초기화합니다.
func InitDB(dbDir string) (*badger.DB, error) {
	opts := badger.DefaultOptions(dbDir).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// SaveGachaRecords 는 특정 플레이어의 특정 배너 가챠 리스트를 JSON 직렬화하여 BadgerDB에 저장합니다.
// 매번 전체 기록을 Rewrite 하므로 중복 및 동기화 에러 걱정이 없습니다.
func SaveGachaRecords(db *badger.DB, playerID, cardPoolType string, records []types.Record) error {
	if records == nil {
		records = []types.Record{}
	}

	recordsJSON, err := json.Marshal(records)
	if err != nil {
		return err
	}

	key := []byte("gacha:" + playerID + ":" + cardPoolType)
	return db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, recordsJSON)
	})
}

// GetGachaRecords 는 특정 플레이어의 특정 배너 가챠 기록을 복원하여 반환합니다.
func GetGachaRecords(db *badger.DB, playerID, cardPoolType string) ([]types.Record, error) {
	var records []types.Record
	key := []byte("gacha:" + playerID + ":" + cardPoolType)

	err := db.View(func(txn *badger.Txn) error {
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
func ListPlayers(db *badger.DB) ([]string, error) {
	playersMap := make(map[string]bool)

	err := db.View(func(txn *badger.Txn) error {
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
