package handler

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/meteormin/wuwa-tracker/config"
	report "github.com/meteormin/wuwa-tracker/internal/reporter"
	"github.com/meteormin/wuwa-tracker/internal/scanner"
	"github.com/meteormin/wuwa-tracker/internal/service"
	"github.com/meteormin/wuwa-tracker/internal/types"
	"github.com/meteormin/wuwa-tracker/locales"
)

// Handler 는 HTTP 요청을 처리하고 service 계층에 위임하는 핸들러 구조체입니다.
type Handler struct {
	svc *service.Service
}

// NewHandler 는 새로운 Handler 구조체 인스턴스를 생성하고 의존성을 주입합니다.
func NewHandler(svc *service.Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

// Track 은 사용자가 제출한 Kurogame 가챠 로그 URL을 기반으로 데이터를 페치하고,
// BadgerDB에 기존 기록과 병합 저장한 뒤 최신 통계 데이터를 반환합니다.
func (h *Handler) Track(c fiber.Ctx) error {
	var req types.TrackRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newInvalidRequestBodyErr(err))
	}

	statsResponse, err := h.svc.TrackURL(req.URL)
	if err != nil {
		return h.handleTrackErr(c, err)
	}
	return c.JSON(statsResponse)
}

// Scan 은 로컬 게임 로그 파일에서 가챠 기록 URL을 추출합니다.
func (h *Handler) Scan(c fiber.Ctx) error {
	var req types.ScanRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newInvalidRequestBodyErr(err))
	}

	url, err := h.svc.ScanURL(req.Path)
	if err != nil {
		return h.handleScanErr(c, err)
	}
	return c.JSON(types.ScanResponse{
		Success: true,
		URL:     url,
	})
}

// Upload 는 클라이언트로부터 직접 JSON 형태의 가챠 데이터 세트를 입력받아 DB에 병합 저장하고,
// 즉시 분석 통계를 산출하여 반환합니다. 외부 API 요청 없이 오프라인 분석 및 테스트가 가능합니다.
func (h *Handler) Upload(c fiber.Ctx) error {
	var req types.UploadRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newInvalidRequestBodyErr(err))
	}

	statsResponse, err := h.svc.Upload(req.FetchResult)
	if err != nil {
		return h.handleUploadErr(c, err)
	}
	return c.JSON(statsResponse)
}

func (h *Handler) handleScanErr(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrMissingScanPath):
		return c.Status(fiber.StatusBadRequest).JSON(errMissingScanPath)
	case errors.Is(err, scanner.ErrURLNotFound):
		return c.Status(fiber.StatusNotFound).JSON(errScanURLNotFound)
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(errScanFailed)
	}
}

func (h *Handler) handleTrackErr(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrMissingURL):
		return c.Status(fiber.StatusBadRequest).JSON(errMissingURL)
	case errors.Is(err, service.ErrMissingPlayerID):
		return c.Status(fiber.StatusBadRequest).JSON(errMissingPlayerIDInURL)
	case errors.Is(err, service.ErrInvalidURL):
		return c.Status(fiber.StatusBadRequest).JSON(errInvalidURLFormat)
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseSaveFailed)
	}
}

func (h *Handler) handleUploadErr(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrMissingPlayerID):
		return c.Status(fiber.StatusBadRequest).JSON(errPlayerIDRequired)
	case errors.Is(err, service.ErrEmptyUploadData):
		return c.Status(fiber.StatusBadRequest).JSON(errEmptyUploadData)
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseSaveFailed)
	}
}

// GetStats 는 DB에 저장된 특정 플레이어의 가챠 데이터를 조회하여 통계 데이터를 산출합니다.
func (h *Handler) GetStats(c fiber.Ctx) error {
	playerID := c.Params("playerId")
	if playerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errMissingPlayerID)
	}

	statsResponse, err := h.svc.GetStats(playerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseQueryFailed)
	}
	return c.JSON(statsResponse)
}

// ListPlayers 는 DB에 기록이 저장된 모든 고유 플레이어 ID 리스트를 반환합니다.
func (h *Handler) ListPlayers(c fiber.Ctx) error {
	players, err := h.svc.ListPlayers()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseListPlayersFailed)
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
		"luckScoreThresholds": h.svc.LuckScoreThresholds(),
	})
}

// GetI18n 은 프론트엔드와 HTML 리포트에서 공유하는 UI 번역 리소스를 반환합니다.
func (h *Handler) GetI18n(c fiber.Ctx) error {
	lang := c.Query("lang", config.DefaultLanguage)
	resolvedLang, translations, err := locales.LoadUITranslationsWithFallback(lang)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to load translations",
		})
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"lang":         resolvedLang,
		"translations": translations,
	})
}

// ExportReport 는 특정 플레이어의 가챠 데이터를 지정된 포맷(html, json, csv)으로 익스포트하여 다운로드하도록 합니다.
func (h *Handler) ExportReport(c fiber.Ctx) error {
	playerID := c.Params("playerId")
	if playerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errMissingPlayerID)
	}

	formatParam := strings.ToLower(c.Query("format", config.DefaultReportFormat))
	lang := c.Query("lang", config.DefaultLanguage)
	var format report.Format
	switch formatParam {
	case "json":
		format = report.FormatJSON
	case "csv":
		format = report.FormatCSV
	case "html":
		format = report.FormatHTML
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Unsupported format: " + formatParam,
		})
	}

	// 메모리 버퍼에 리포트 내용 쓰기
	var buf bytes.Buffer
	if err := h.svc.ExportReport(&buf, playerID, format, lang); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to generate report: " + err.Error(),
		})
	}

	// 컨텐트 타입 지정
	switch format {
	case report.FormatJSON:
		c.Type("json")
	case report.FormatCSV:
		c.Set("Content-Type", "text/csv; charset=utf-8")
	case report.FormatHTML:
		c.Type("html")
	}

	// 파일 다운로드 응답 전송
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"report_%s.%s\"", playerID, formatParam))
	return c.Send(buf.Bytes())
}
