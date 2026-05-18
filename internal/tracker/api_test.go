package tracker

import (
	"net/http"
	"testing"
)

func TestFetchRecords_MissingParams(t *testing.T) {
	// 필수 파라미터가 누락된 URL을 파싱할 때 에러가 발생하는지 검증합니다.
	invalidURL := "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record?lang=ko"

	c := NewClient(http.DefaultClient)
	_, err := c.FetchRecords(invalidURL, 1)
	if err == nil {
		t.Error("Expected error for missing parameters, but got nil")
	}

	expectedErrMsg := "missing required parameters in url"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error %q, got %q", expectedErrMsg, err.Error())
	}
}
