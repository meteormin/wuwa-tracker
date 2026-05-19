package handler

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
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

// Track 은 사용자가 제출한 Kurogame 가챠 로그 URL을 기반으로 데이터를 페치하고,
// BadgerDB에 덮어쓰기 저장한 뒤 최신 통계 데이터를 반환합니다.
func (h *Handler) Track(c fiber.Ctx) error {
	var req types.TrackRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Success:  false,
			Error:    "invalid request body: " + err.Error(),
			ErrorKey: "err.invalid_request_body",
		})
	}

	targetURL := strings.TrimSpace(req.URL)
	targetURL = strings.ReplaceAll(targetURL, "\\", "")
	if targetURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Success:  false,
			Error:    "missing url parameter",
			ErrorKey: "err.missing_url",
		})
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Success:  false,
			Error:    "invalid url format",
			ErrorKey: "err.invalid_url_format",
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
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Success:  false,
			Error:    "missing player_id in url",
			ErrorKey: "err.missing_player_id_in_url",
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
			return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
				Success:  false,
				Error:    "failed to save records to database",
				ErrorKey: "err.database_save_failed",
			})
		}
	}

	// 동기화된 DB 기반으로 최신 통계 계산 후 반환
	return h.returnPlayerStats(c, playerID)
}

// Upload 는 클라이언트로부터 직접 JSON 형태의 가챠 데이터 세트를 입력받아 DB에 덮어쓰기 저장하고,
// 즉시 분석 통계를 산출하여 반환합니다. 외부 API 요청 없이 오프라인 분석 및 테스트가 가능합니다.
func (h *Handler) Upload(c fiber.Ctx) error {
	var req types.UploadRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Success:  false,
			Error:    "invalid request body: " + err.Error(),
			ErrorKey: "err.invalid_request_body",
		})
	}

	playerID := strings.TrimSpace(req.PlayerID)
	if playerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Success:  false,
			Error:    "playerId is required",
			ErrorKey: "err.player_id_required",
		})
	}

	if len(req.Data) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Success:  false,
			Error:    "data map cannot be empty",
			ErrorKey: "err.empty_upload_data",
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
			return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
				Success:  false,
				Error:    "failed to save records to database",
				ErrorKey: "err.database_save_failed",
			})
		}
	}

	// 저장된 데이터를 바탕으로 즉시 가챠 분석 리포트 리턴
	return h.returnPlayerStats(c, playerID)
}

// GetStats 는 DB에 저장된 특정 플레이어의 가챠 데이터를 조회하여 통계 데이터를 산출합니다.
func (h *Handler) GetStats(c fiber.Ctx) error {
	playerID := c.Params("playerId")
	if playerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(types.ErrorResponse{
			Success:  false,
			Error:    "missing playerId parameter",
			ErrorKey: "err.missing_player_id",
		})
	}

	return h.returnPlayerStats(c, playerID)
}

// ListPlayers 는 DB에 기록이 저장된 모든 고유 플레이어 ID 리스트를 반환합니다.
func (h *Handler) ListPlayers(c fiber.Ctx) error {
	players, err := h.db.ListPlayers()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
			Success:  false,
			Error:    "failed to retrieve player list",
			ErrorKey: "err.database_list_players_failed",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"players": players,
	})
}

// GetConfig 는 서버의 설정을 프론트엔드로 전달합니다. (운 점수 임계값 등 포함)
func (h *Handler) GetConfig(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success":             true,
		"luckScoreThresholds": h.cfg.LuckScoreThresholds,
	})
}

// returnPlayerStats 는 BadgerDB에서 플레이어 가챠 데이터를 가져와 통계(Stats)를 계산하고 JSON 응답을 전송하는 헬퍼 함수입니다.
func (h *Handler) returnPlayerStats(c fiber.Ctx, playerID string) error {
	// 통계 계산 엔진 초기화
	calc := tracker.NewStatsCalculator(h.cfg.StandardFiveStarResources)
	statsList := make([]types.Stats, 0, len(h.cfg.GachaTypes.Items))

	for _, gachaType := range h.cfg.GachaTypes.Items {
		records, err := h.db.GetGachaRecords(playerID, gachaType.Key)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ErrorResponse{
				Success:  false,
				Error:    "failed to query player records",
				ErrorKey: "err.database_query_failed",
			})
		}
		// 기록이 비어 있어도 기본 구조체를 바르게 렌더링하기 위해 무조건 추가
		statsList = append(statsList, calc.CalculateStats(records, gachaType))
	}

	return c.JSON(types.StatsResponse{
		Success:  true,
		PlayerID: playerID,
		Stats:    statsList,
	})
}
