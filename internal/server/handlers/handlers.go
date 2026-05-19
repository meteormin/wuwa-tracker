package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/server/models"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
	"github.com/meteormin/wuwa-tracker/internal/types"
	"gorm.io/gorm"
)

// TrackRequest 는 가챠 기록 조회를 위한 URL 입력 요청 데이터 구조체입니다.
type TrackRequest struct {
	URL string `json:"url"`
}

// StatsResponse 는 프론트엔드로 반환될 표준 통계 응답 데이터 구조체입니다.
type StatsResponse struct {
	Success  bool          `json:"success"`
	PlayerID string        `json:"playerId"`
	Stats    []types.Stats `json:"stats"`
}

// TrackHandler 는 사용자가 제출한 Kurogame 가챠 로그 URL을 기반으로 데이터를 페치하고,
// SQLite DB에 갱신 및 저장한 뒤 최신 통계 데이터를 반환합니다.
func TrackHandler(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req TrackRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "invalid request body",
			})
		}

		targetURL := strings.TrimSpace(req.URL)
		targetURL = strings.ReplaceAll(targetURL, "\\", "")
		if targetURL == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "empty url",
			})
		}

		u, err := url.Parse(targetURL)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "failed to parse url",
			})
		}

		// URL fragment 혹은 query parameters 에서 player_id 파싱
		var q url.Values
		if u.Fragment != "" {
			parts := strings.SplitN(u.Fragment, "?", 2)
			if len(parts) == 2 {
				q, _ = url.ParseQuery(parts[1])
			} else {
				q = u.Query()
			}
		} else {
			q = u.Query()
		}

		playerID := q.Get("player_id")
		if playerID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "missing player_id in url",
			})
		}

		// Kurogame API 클라이언트 생성
		httpClient := &http.Client{Timeout: 10 * time.Second}
		client := tracker.NewClient(httpClient)

		// 다국어 배너 이름 매핑을 위한 설정 로드
		cfg, err := config.LoadConfig()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "failed to load config",
			})
		}

		// 기본 한국어("ko") 배너 정보 페치
		localeData, err := client.FetchGachaLocale("ko")
		if err != nil {
			// 언어 페치 실패 시에도 동작하도록 처리
			localeData = types.LocaleData{SelectList: map[string]string{}}
		}
		cfg.GachaTypes.MapFromSelectList(localeData.SelectList)

		// 각 배너 타입별 가챠 기록 동기화
		for _, gachaType := range cfg.GachaTypes.Items {
			records, err := client.FetchRecords(targetURL, gachaType.ID)
			if err != nil {
				// 개별 배너 페치 실패 시 에러 로그는 패스하고 진행
				continue
			}

			// 데이터 정합성 보장을 위한 멱등성 보장 처리 (기존 플레이어 배너 데이터 삭제 후 인서트)
			err = db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Where("player_id = ? AND card_pool_type = ?", playerID, gachaType.Key).Delete(&models.GachaRecord{}).Error; err != nil {
					return err
				}

				if len(records) > 0 {
					dbRecords := make([]models.GachaRecord, 0, len(records))
					for _, r := range records {
						dbRecords = append(dbRecords, models.GachaRecord{
							PlayerID:     playerID,
							CardPoolType: gachaType.Key,
							ResourceID:   r.ResourceID,
							QualityLevel: r.QualityLevel,
							ResourceType: r.ResourceType,
							Name:         r.Name,
							Count:        r.Count,
							Time:         r.Time,
						})
					}
					if err := tx.Create(&dbRecords).Error; err != nil {
						return err
					}
				}
				return nil
			})

			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"error":   "database write transaction failed",
				})
			}
		}

		// 동기화된 DB 기반으로 최신 통계 계산 후 반환
		return returnPlayerStats(c, db, playerID, cfg)
	}
}

// GetStatsHandler 는 DB에 저장된 특정 플레이어의 가챠 데이터를 조회하여 통계 데이터를 산출합니다.
func GetStatsHandler(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		playerID := c.Params("playerId")
		if playerID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "missing playerId parameter",
			})
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "failed to load config",
			})
		}

		// 다국어 배너 매핑
		httpClient := &http.Client{Timeout: 5 * time.Second}
		client := tracker.NewClient(httpClient)
		localeData, _ := client.FetchGachaLocale("ko")
		cfg.GachaTypes.MapFromSelectList(localeData.SelectList)

		return returnPlayerStats(c, db, playerID, cfg)
	}
}

// ListPlayersHandler 는 DB에 기록이 저장된 모든 고유 플레이어 ID 리스트를 반환합니다.
func ListPlayersHandler(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var players []string
		err := db.Model(&models.GachaRecord{}).Distinct().Pluck("player_id", &players).Error
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "failed to retrieve player list",
			})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"players": players,
		})
	}
}

// returnPlayerStats 는 SQLite DB에서 플레이어 가챠 데이터를 가져와 통계(Stats)를 계산하고 JSON 응답을 전송하는 헬퍼 함수입니다.
func returnPlayerStats(c *fiber.Ctx, db *gorm.DB, playerID string, cfg *config.Config) error {
	var dbRecords []models.GachaRecord
	err := db.Where("player_id = ?", playerID).Find(&dbRecords).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to query player records",
		})
	}

	// DB 레코드를 []types.Record 구조로 맵핑 및 그룹화
	recordsMap := make(map[string][]types.Record)
	for _, dr := range dbRecords {
		recordsMap[dr.CardPoolType] = append(recordsMap[dr.CardPoolType], types.Record{
			CardPoolType: dr.CardPoolType,
			ResourceID:   dr.ResourceID,
			QualityLevel: dr.QualityLevel,
			ResourceType: dr.ResourceType,
			Name:         dr.Name,
			Count:        dr.Count,
			Time:         dr.Time,
		})
	}

	// 통계 계산 엔진 초기화
	calc := tracker.NewStatsCalculator(cfg.StandardFiveStarResources)
	statsList := make([]types.Stats, 0, len(cfg.GachaTypes.Items))

	for _, gachaType := range cfg.GachaTypes.Items {
		records := recordsMap[gachaType.Key]
		// 기록이 비어 있어도 기본 구조체를 바르게 렌더링하기 위해 무조건 추가
		statsList = append(statsList, calc.CalculateStats(records, gachaType))
	}

	return c.JSON(StatsResponse{
		Success:  true,
		PlayerID: playerID,
		Stats:    statsList,
	})
}
