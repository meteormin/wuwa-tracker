package tracker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

const (
	apiGachaRecordQueryEndpoint = "https://gmserver-api.aki-game2.net/gacha/record/query"
	apiGachaLocaleEndpoint      = "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/locales/%s.json"
)

var ErrMissingRequiredParams = errors.New("missing required parameters in url")

type Client struct {
	core            *http.Client
	enabledDebugLog bool
}

// NewClient 는 Client 를 생성합니다.
func NewClient(client *http.Client) *Client {
	return &Client{
		core: client,
	}
}

func (c *Client) EnableDebugLog() {
	c.enabledDebugLog = true
}

func (c *Client) DisableDebugLog() {
	c.enabledDebugLog = false
}

// FetchRecords 는 지정된 배너(gachaType)의 가챠 기록을 가져옵니다.
func (c *Client) FetchRecords(urlStr string, gachaType int) ([]types.Record, error) {
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

	p := types.Payload{
		PlayerID:     q.Get("player_id"),
		ServerID:     q.Get("svr_id"),
		LanguageCode: q.Get("lang"),
		RecordID:     q.Get("record_id"),
		CardPoolID:   q.Get("gacha_id"),
		CardPoolType: gachaType,
	}

	if p.PlayerID == "" || p.ServerID == "" || p.RecordID == "" {
		return nil, ErrMissingRequiredParams
	}

	body, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, apiGachaRecordQueryEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.core.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var gResp types.GachaResponse
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

// FetchGachaLocale 는 원격에서 로컬라이제이션 데이터를 가져와 gachaNamesMap 을 업데이트합니다.
func (c *Client) FetchGachaLocale(lang string) (types.LocaleData, error) {
	if lang == "" {
		lang = "ko"
	}
	urlStr := fmt.Sprintf(apiGachaLocaleEndpoint, lang)
	resp, err := c.core.Get(urlStr)
	if err != nil {
		return types.LocaleData{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return types.LocaleData{}, fmt.Errorf("failed to fetch locale: %d", resp.StatusCode)
	}

	var data types.LocaleData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return types.LocaleData{}, err
	}

	return data, nil
}

func (c *Client) FetchAllRecords(urlStr string, gachaTypes []types.GachaType) map[string][]types.Record {
	// FetchAllRecords 호출 시점의 시간 값을 기준으로 fetchTimestamp를 설정합니다.
	fetchTimestamp := time.Now().Format("20060102150405")
	result := make(map[string][]types.Record)
	for _, gachaType := range gachaTypes {
		records, err := c.FetchRecords(urlStr, gachaType.ID)
		if err != nil {
			log.Printf("Failed to fetch records for gacha type %d: %v\n", gachaType.ID, err)
			continue
		}
		result[gachaType.Key] = records
	}

	if len(result) > 0 && c.enabledDebugLog {
		if err := os.MkdirAll("logs", 0o755); err != nil {
			log.Printf("Warning: failed to create logs directory: %v\n", err)
		} else {
			filePath := fmt.Sprintf("logs/%s.json", fetchTimestamp)
			b, err := json.MarshalIndent(result, "", "    ")
			if err != nil {
				log.Printf("Warning: failed to marshal records: %v\n", err)
			}
			if err := os.WriteFile(filePath, b, 0o644); err != nil {
				log.Printf("Warning: failed to save records JSON to %s: %v\n", filePath, err)
			}
		}
	}

	return result
}

type LoggingTransport struct {
	Captured http.RoundTripper
}

func (l *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Printf("[REQ] %s %s\n", req.Method, req.URL.String())

	startTime := time.Now()

	resp, err := l.Captured.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	duration := time.Since(startTime)

	log.Printf("[RES] %s %s - %s\n", req.Method, req.URL.String(), duration)

	return resp, nil
}
