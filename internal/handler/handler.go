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
	"github.com/meteormin/wuwa-tracker/internal/server"
	"github.com/meteormin/wuwa-tracker/internal/service"
	"github.com/meteormin/wuwa-tracker/internal/types"
	"github.com/meteormin/wuwa-tracker/locales"
)

var errRuntimeServiceUnavailable = errors.New("runtime service unavailable")

// RegisterRoutes 는 API 라우트를 등록합니다.
func RegisterRoutes(api fiber.Router) {
	api.Post("/scan", Scan)
	api.Post("/track", Track)
	api.Get("/stats/:playerId", GetStats)
	api.Get("/players", ListPlayers)
	api.Post("/upload", Upload)
	api.Get("/config", GetConfig)
	api.Get("/i18n", GetI18n)
	api.Get("/export/:playerId", ExportReport)
}

// Track 은 사용자가 제출한 Kurogame 가챠 로그 URL을 기반으로 데이터를 페치하고,
// repository에 기존 기록과 병합 저장한 뒤 최신 통계 데이터를 반환합니다.
func Track(c fiber.Ctx) error {
	svc, err := serviceFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseQueryFailed)
	}

	var req types.TrackRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newInvalidRequestBodyErr(err))
	}

	statsResponse, err := svc.TrackURL(req.URL)
	if err != nil {
		return handleTrackErr(c, err)
	}
	return c.JSON(statsResponse)
}

// Scan 은 로컬 게임 로그 파일에서 가챠 기록 URL을 추출합니다.
func Scan(c fiber.Ctx) error {
	svc, err := serviceFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errScanFailed)
	}

	var req types.ScanRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newInvalidRequestBodyErr(err))
	}

	path := strings.TrimSpace(req.Path)
	if path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errMissingScanPath)
	}

	url, err := svc.Scan(path)
	if err != nil {
		return handleScanErr(c, err)
	}
	return c.JSON(types.ScanResponse{
		Success: true,
		URL:     url,
	})
}

// Upload 는 클라이언트로부터 직접 JSON 형태의 가챠 데이터 세트를 입력받아 DB에 병합 저장하고,
// 즉시 분석 통계를 산출하여 반환합니다. 외부 API 요청 없이 오프라인 분석 및 테스트가 가능합니다.
func Upload(c fiber.Ctx) error {
	svc, err := serviceFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseSaveFailed)
	}

	var req types.UploadRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newInvalidRequestBodyErr(err))
	}

	statsResponse, err := svc.Upload(req.FetchResult)
	if err != nil {
		return handleUploadErr(c, err)
	}
	return c.JSON(statsResponse)
}

func handleScanErr(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, scanner.ErrScanPathNotFound):
		return c.Status(fiber.StatusNotFound).JSON(errScanPathNotFound)
	case errors.Is(err, scanner.ErrScanPathAccessDenied):
		return c.Status(fiber.StatusForbidden).JSON(errScanPathAccessDenied)
	case errors.Is(err, scanner.ErrLogFileNotFound):
		return c.Status(fiber.StatusNotFound).JSON(errScanLogFileNotFound)
	case errors.Is(err, scanner.ErrURLNotFound):
		return c.Status(fiber.StatusNotFound).JSON(errScanURLNotFound)
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(errScanFailed)
	}
}

func handleTrackErr(c fiber.Ctx, err error) error {
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

func handleUploadErr(c fiber.Ctx, err error) error {
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
func GetStats(c fiber.Ctx) error {
	svc, err := serviceFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseQueryFailed)
	}

	playerID := c.Params("playerId")
	if playerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errMissingPlayerID)
	}

	statsResponse, err := svc.GetStats(playerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseQueryFailed)
	}
	return c.JSON(statsResponse)
}

// ListPlayers 는 DB에 기록이 저장된 모든 고유 플레이어 ID 리스트를 반환합니다.
func ListPlayers(c fiber.Ctx) error {
	svc, err := serviceFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseListPlayersFailed)
	}

	players, err := svc.ListPlayers()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseListPlayersFailed)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"players": players,
	})
}

// GetConfig 는 서버의 설정을 프론트엔드로 전달합니다. (운 점수 임계값 등 포함)
func GetConfig(c fiber.Ctx) error {
	svc, err := serviceFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errDatabaseQueryFailed)
	}

	return c.JSON(fiber.Map{
		"success":             true,
		"luckScoreThresholds": svc.LuckScoreThresholds(),
	})
}

// GetI18n 은 프론트엔드와 HTML 리포트에서 공유하는 UI 번역 리소스를 반환합니다.
func GetI18n(c fiber.Ctx) error {
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
func ExportReport(c fiber.Ctx) error {
	svc, err := serviceFromCtx(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errReportGenerationFailed)
	}

	playerID := c.Params("playerId")
	if playerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errMissingPlayerID)
	}

	formatParam := c.Query("format", config.DefaultReportFormat)
	lang := c.Query("lang", config.DefaultLanguage)
	format, err := report.ParseFormat(formatParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(newUnsupportedReportFormatErr(formatParam))
	}

	// 메모리 버퍼에 리포트 내용 쓰기
	var buf bytes.Buffer
	if err := svc.ExportReport(&buf, playerID, format, lang); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(errReportGenerationFailed)
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
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"report_%s.%s\"", playerID, format))
	return c.Send(buf.Bytes())
}

func serviceFromCtx(c fiber.Ctx) (*service.Service, error) {
	runtime, ok := fiber.GetService[fiber.Service](c.App().State(), server.RuntimeServiceName)
	if !ok {
		return nil, errRuntimeServiceUnavailable
	}
	provider, ok := runtime.(server.ServiceProvider)
	if !ok {
		return nil, errRuntimeServiceUnavailable
	}
	svc := provider.Service()
	if svc == nil {
		return nil, errRuntimeServiceUnavailable
	}
	return svc, nil
}
