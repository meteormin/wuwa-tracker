package handler

import (
	"github.com/meteormin/wuwa-tracker/internal/types"
)

// 미리 정의된 에러 응답 객체들입니다.
var (
	// errMissingURL 은 URL 파라미터가 누락되었을 때의 에러 응답입니다.
	errMissingURL = types.ErrorResponse{
		Success:  false,
		Error:    "missing url parameter",
		ErrorKey: "err.missing_url",
	}

	// errMissingScanPath 는 로컬 로그 스캔 경로가 누락되었을 때의 에러 응답입니다.
	errMissingScanPath = types.ErrorResponse{
		Success:  false,
		Error:    "missing scan path parameter",
		ErrorKey: "err.missing_scan_path",
	}

	// errScanURLNotFound 는 로컬 로그 파일에서 가챠 URL을 찾지 못했을 때의 에러 응답입니다.
	errScanURLNotFound = types.ErrorResponse{
		Success:  false,
		Error:    "gacha url not found in the log",
		ErrorKey: "err.scan_url_not_found",
	}

	errScanPathNotFound = types.ErrorResponse{
		Success:  false,
		Error:    "scan path not found",
		ErrorKey: "err.scan_path_not_found",
	}

	errScanPathAccessDenied = types.ErrorResponse{
		Success:  false,
		Error:    "scan path access denied",
		ErrorKey: "err.scan_path_access_denied",
	}

	errScanLogFileNotFound = types.ErrorResponse{
		Success:  false,
		Error:    "log file not found",
		ErrorKey: "err.scan_log_file_not_found",
	}

	// errScanFailed 는 로컬 로그 스캔 중 예상하지 못한 오류가 발생했을 때의 에러 응답입니다.
	errScanFailed = types.ErrorResponse{
		Success:  false,
		Error:    "failed to scan log files",
		ErrorKey: "err.scan_failed",
	}

	// errInvalidURLFormat 은 URL 형식이 올바르지 않을 때의 에러 응답입니다.
	errInvalidURLFormat = types.ErrorResponse{
		Success:  false,
		Error:    "invalid url format",
		ErrorKey: "err.invalid_url_format",
	}

	// errMissingPlayerIDInURL 은 URL 쿼리 파라미터에서 player_id를 찾을 수 없을 때의 에러 응답입니다.
	errMissingPlayerIDInURL = types.ErrorResponse{
		Success:  false,
		Error:    "missing player_id in url",
		ErrorKey: "err.missing_player_id_in_url",
	}

	// errDatabaseSaveFailed 는 가챠 기록을 repository에 저장하는 데 실패했을 때의 에러 응답입니다.
	errDatabaseSaveFailed = types.ErrorResponse{
		Success:  false,
		Error:    "failed to save records to database",
		ErrorKey: "err.database_save_failed",
	}

	// errPlayerIDRequired 는 플레이어 ID가 누락되었을 때의 에러 응답입니다.
	errPlayerIDRequired = types.ErrorResponse{
		Success:  false,
		Error:    "playerId is required",
		ErrorKey: "err.player_id_required",
	}

	// errEmptyUploadData 는 업로드할 가챠 기록 맵이 비어 있을 때의 에러 응답입니다.
	errEmptyUploadData = types.ErrorResponse{
		Success:  false,
		Error:    "data map cannot be empty",
		ErrorKey: "err.empty_upload_data",
	}

	// errMissingPlayerID 는 가챠 통계 조회 시 플레이어 ID 파라미터가 누락되었을 때의 에러 응답입니다.
	errMissingPlayerID = types.ErrorResponse{
		Success:  false,
		Error:    "missing playerId parameter",
		ErrorKey: "err.missing_player_id",
	}

	// errDatabaseListPlayersFailed 는 DB에서 플레이어 목록을 조회하는 데 실패했을 때의 에러 응답입니다.
	errDatabaseListPlayersFailed = types.ErrorResponse{
		Success:  false,
		Error:    "failed to retrieve player list",
		ErrorKey: "err.database_list_players_failed",
	}

	// errDatabaseQueryFailed 는 DB에서 플레이어 가챠 데이터를 조회하는 데 실패했을 때의 에러 응답입니다.
	errDatabaseQueryFailed = types.ErrorResponse{
		Success:  false,
		Error:    "failed to query player records",
		ErrorKey: "err.database_query_failed",
	}

	errReportGenerationFailed = types.ErrorResponse{
		Success:  false,
		Error:    "failed to generate report",
		ErrorKey: "err.report_generation_failed",
	}
)

// newInvalidRequestBodyErr 는 올바르지 않은 요청 바디 수신 시 동적 에러 상세를 포함한 에러 응답을 생성합니다.
func newInvalidRequestBodyErr(err error) types.ErrorResponse {
	return types.ErrorResponse{
		Success:  false,
		Error:    "invalid request body: " + err.Error(),
		ErrorKey: "err.invalid_request_body",
	}
}

func newUnsupportedReportFormatErr(format string) types.ErrorResponse {
	return types.ErrorResponse{
		Success:  false,
		Error:    "unsupported report format: " + format,
		ErrorKey: "err.unsupported_report_format",
	}
}
