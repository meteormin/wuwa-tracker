package models

import (
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GachaRecord 는 SQLite 데이터베이스에 가챠 단일 뽑기 기록을 저장하기 위한 GORM 모델입니다.
type GachaRecord struct {
	ID           uint      `gorm:"primaryKey;autoIncrement;column:id"`
	PlayerID     string    `gorm:"index;column:player_id;type:text;not null"`
	CardPoolType string    `gorm:"index;column:card_pool_type;type:text;not null"`
	ResourceID   int       `gorm:"column:resource_id;not null"`
	QualityLevel int       `gorm:"column:quality_level;not null"`
	ResourceType string    `gorm:"column:resource_type;type:text;not null"`
	Name         string    `gorm:"column:name;type:text;not null"`
	Count        int       `gorm:"column:count;not null"`
	Time         string    `gorm:"column:time;type:text;not null"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName 은 가챠 기록 테이블의 이름을 지정합니다.
func (GachaRecord) TableName() string {
	return "gacha_records"
}

// InitDB 는 CGO가 필요 없는 Glebarez 순수 Go SQLite 드라이버를 사용하여 데이터베이스를 초기화하고 
// 자동 마이그레이션을 실행합니다.
func InitDB(dbPath string) (*gorm.DB, error) {
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(sqlite.Open(dbPath), config)
	if err != nil {
		return nil, err
	}

	// 테이블 스키마 자동 생성 및 동기화
	err = db.AutoMigrate(&GachaRecord{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
