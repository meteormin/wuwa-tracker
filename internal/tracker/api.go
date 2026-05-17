package tracker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	apiEndpoint = "https://gmserver-api.aki-game2.net/gacha/record/query"
)

type payload struct {
	PlayerID     string `json:"playerId"`
	ServerID     string `json:"serverId"`
	LanguageCode string `json:"languageCode"`
	RecordID     string `json:"recordId"`
	CardPoolID   string `json:"cardPoolId"`
	CardPoolType int    `json:"cardPoolType"`
}

// FetchRecords 는 지정된 배너(gachaType)의 가챠 기록을 가져옵니다.
func FetchRecords(urlStr string, gachaType int) ([]Record, error) {
	// 터미널에서 복사/붙여넣기 시 자동으로 추가되는 백슬래시(\) 이스케이프 문자 제거
	urlStr = strings.ReplaceAll(urlStr, "\\", "")

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
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

	p := payload{
		PlayerID:     q.Get("player_id"),
		ServerID:     q.Get("svr_id"),
		LanguageCode: q.Get("lang"),
		RecordID:     q.Get("record_id"),
		CardPoolID:   q.Get("gacha_id"),
		CardPoolType: gachaType,
	}

	if p.PlayerID == "" || p.ServerID == "" || p.RecordID == "" {
		return nil, errors.New("missing required parameters in url")
	}

	body, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, apiEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var gResp GachaResponse
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &gResp); err != nil {
		return nil, err
	}

	if gResp.Code != 0 {
		return nil, fmt.Errorf("api error: %s (code: %d)", gResp.Message, gResp.Code)
	}

	return gResp.Data, nil
}

// FetchAll 은 1부터 7까지의 모든 가챠 타입에 대해 데이터를 수집하여 맵으로 반환합니다.
func FetchAll(urlStr string) (map[int][]Record, error) {
	results := make(map[int][]Record)
	gachaTypes := []int{1, 2, 3, 4, 5, 6, 7}

	for _, gt := range gachaTypes {
		records, err := FetchRecords(urlStr, gt)
		if err != nil {
			// 배너 조회가 실패하더라도 전체가 중단되지 않도록 로그(혹은 무시) 처리하고 넘어갈 수 있음
			// 토큰 만료 에러일 경우엔 즉각 반환해야 하므로 에러 메시지 검사
			return nil, err
		}
		if len(records) > 0 {
			results[gt] = records
		}
	}

	return results, nil
}
