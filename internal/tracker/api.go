package tracker

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

var (
	ErrMissingRequiredParams = errors.New("missing required parameters in url")
	ErrInvalidGachaURL       = errors.New("invalid gacha url or unsupported domain")
)

type Client struct {
	core *http.Client

	resourceURL string
	trackingURL string
}

type Config struct {
	Client      *http.Client
	ResourceURL string
	TrackingURL string
}

// NewClient 는 Client 를 생성합니다.
func NewClient(config Config) *Client {
	return &Client{
		core:        config.Client,
		resourceURL: config.ResourceURL,
		trackingURL: config.TrackingURL,
	}
}

// ParsePayloadFromURL 은 가챠 로그 URL 에서 types.Payload를 추출합니다.
func (c *Client) ParsePayloadFromURL(urlStr string) (types.Payload, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return types.Payload{}, err
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
	}

	if p.PlayerID == "" || p.ServerID == "" || p.RecordID == "" {
		return types.Payload{}, ErrMissingRequiredParams
	}

	return p, nil
}

// FetchRecords 는 지정된 배너(gachaType)의 가챠 기록을 가져옵니다.
func (c *Client) FetchRecords(payload types.Payload) ([]types.Record, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint, err := url.JoinPath(c.trackingURL, "/gacha/record/query")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(body))
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

	endpoint, err := url.JoinPath(c.resourceURL, "/aki/gacha/locales", lang+".json")
	if err != nil {
		return types.LocaleData{}, err
	}

	resp, err := c.core.Get(endpoint)
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

// FetchAllRecords 는 모든 배너의 가챠 기록을 가져오고, URL로부터 파싱된 Payload와 함께 단일 구조체(FetchResult)로 반환합니다.
func (c *Client) FetchAllRecords(payload types.Payload, gachaTypes []types.GachaType) (*types.FetchResult, error) {
	result := make(map[string][]types.Record)
	for _, gachaType := range gachaTypes {
		payload.CardPoolType = gachaType.ID
		records, err := c.FetchRecords(payload)
		if err != nil {
			log.Printf("Failed to fetch records for gacha type %d: %v\n", gachaType.ID, err)
			continue
		}
		result[gachaType.Key] = records
	}

	return &types.FetchResult{
		Payload: payload,
		Records: result,
	}, nil
}

type LoggingTransport struct {
	Captured http.RoundTripper
}

func (l *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	startTime := time.Now()

	resp, err := l.Captured.RoundTrip(req)
	if err != nil {
		log.Printf("[Client] %s %s - Error: %v\n", req.Method, req.URL.String(), err)
		return nil, err
	}

	duration := time.Since(startTime)

	log.Printf("[Client] %s %s - %d - %s\n", req.Method, req.URL.String(), resp.StatusCode, duration)

	return resp, nil
}
