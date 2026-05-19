package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/server/db"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

// Handler 는 HTTP 요청을 처리하고 데이터베이스에 협업하는 핸들러 구조체입니다.
type Handler struct {
	db  *db.BadgerDB
	cfg *config.Config
}

// NewHandler 는 새로운 Handler 구조체 인스턴스를 생성하고 의존성을 주입합니다.
func NewHandler(badgerDB *db.BadgerDB, cfg *config.Config) *Handler {
	return &Handler{
		db:  badgerDB,
		cfg: cfg,
	}
}

// TrackRequest 는 가챠 기록 조회를 위한 URL 입력 요청 데이터 구조체입니다.
type TrackRequest struct {
	URL string `json:"url"`
}

// UploadRequest 는 JSON 로그 데이터를 직접 업로드하기 위한 구조체입니다.
type UploadRequest struct {
	PlayerID string                    `json:"playerId"`
	Data     map[string][]types.Record `json:"data"`
}

// StatsResponse 는 프론트엔드로 반환될 표준 통계 응답 데이터 구조체입니다.
type StatsResponse struct {
	Success  bool          `json:"success"`
	PlayerID string        `json:"playerId"`
	Stats    []types.Stats `json:"stats"`
}

// Track 은 사용자가 제출한 Kurogame 가챠 로그 URL을 기반으로 데이터를 페치하고,
// BadgerDB에 덮어쓰기 저장한 뒤 최신 통계 데이터를 반환합니다.
func (h *Handler) Track(c *fiber.Ctx) error {
	var req TrackRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid request body: " + err.Error(),
		})
	}

	targetURL := strings.TrimSpace(req.URL)
	targetURL = strings.ReplaceAll(targetURL, "\\", "")
	if targetURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "missing url parameter",
		})
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid url format",
		})
	}

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

	// 각 배너 타입별 가챠 기록 동기화
	for _, gachaType := range h.cfg.GachaTypes.Items {
		records, err := client.FetchRecords(targetURL, gachaType.ID)
		if err != nil {
			// 개별 배너 페치 실패 시 에러 로그는 패스하고 진행
			continue
		}

		// BadgerDB에 덮어쓰기 방식으로 저장 (기존 데이터와 자동 정합성 일치)
		err = h.db.SaveGachaRecords(playerID, gachaType.Key, records)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "failed to save records to database",
			})
		}
	}

	// 동기화된 DB 기반으로 최신 통계 계산 후 반환
	return h.returnPlayerStats(c, playerID)
}

// Upload 는 클라이언트로부터 직접 JSON 형태의 가챠 데이터 세트를 입력받아 DB에 덮어쓰기 저장하고,
// 즉시 분석 통계를 산출하여 반환합니다. 외부 API 요청 없이 오프라인 분석 및 테스트가 가능합니다.
func (h *Handler) Upload(c *fiber.Ctx) error {
	var req UploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "invalid request body: " + err.Error(),
		})
	}

	playerID := strings.TrimSpace(req.PlayerID)
	if playerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "playerId is required",
		})
	}

	if len(req.Data) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "data map cannot be empty",
		})
	}

	// 맵 데이터를 BadgerDB에 배너별로 저장
	for _, gachaType := range h.cfg.GachaTypes.Items {
		records, ok := req.Data[gachaType.Key]
		if !ok {
			// 업로드 데이터에 특정 배너가 누락되었을 경우 빈 배열로 처리하여 정합성 유지
			records = []types.Record{}
		}

		err := h.db.SaveGachaRecords(playerID, gachaType.Key, records)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "failed to save records to database",
			})
		}
	}

	// 저장된 데이터를 바탕으로 즉시 가챠 분석 리포트 리턴
	return h.returnPlayerStats(c, playerID)
}

// GetStats 는 DB에 저장된 특정 플레이어의 가챠 데이터를 조회하여 통계 데이터를 산출합니다.
func (h *Handler) GetStats(c *fiber.Ctx) error {
	playerID := c.Params("playerId")
	if playerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "missing playerId parameter",
		})
	}

	return h.returnPlayerStats(c, playerID)
}

// ListPlayers 는 DB에 기록이 저장된 모든 고유 플레이어 ID 리스트를 반환합니다.
func (h *Handler) ListPlayers(c *fiber.Ctx) error {
	players, err := h.db.ListPlayers()
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

// returnPlayerStats 는 BadgerDB에서 플레이어 가챠 데이터를 가져와 통계(Stats)를 계산하고 JSON 응답을 전송하는 헬퍼 함수입니다.
func (h *Handler) returnPlayerStats(c *fiber.Ctx, playerID string) error {
	// 통계 계산 엔진 초기화
	calc := tracker.NewStatsCalculator(h.cfg.StandardFiveStarResources)
	statsList := make([]types.Stats, 0, len(h.cfg.GachaTypes.Items))

	for _, gachaType := range h.cfg.GachaTypes.Items {
		records, err := h.db.GetGachaRecords(playerID, gachaType.Key)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "failed to query player records",
			})
		}
		// 기록이 비어 있어도 기본 구조체를 바르게 렌더링하기 위해 무조건 추가
		statsList = append(statsList, calc.CalculateStats(records, gachaType))
	}

	return c.JSON(StatsResponse{
		Success:  true,
		PlayerID: playerID,
		Stats:    statsList,
	})
}
