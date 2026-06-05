package tracker

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/meteormin/wuwa-tracker/internal/types"
)

func TestFetchRecords_MissingParams(t *testing.T) {
	// 필수 파라미터가 누락된 URL을 파싱할 때 에러가 발생하는지 검증합니다.
	invalidURL := "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record?lang=ko"

	c := NewClient(Config{
		Client:      http.DefaultClient,
		ResourceURL: "https://aki-gm-resources-oversea.aki-game.net",
		TrackingURL: "https://gmserver-api.aki-game2.net",
	})
	_, err := c.ParsePayloadFromURL(invalidURL)
	if err == nil {
		t.Error("Expected error for missing parameters, but got nil")
	}

	expectedErrMsg := "missing required parameters in url"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestParsePayloadFromURL_WithCurrentRecordURL(t *testing.T) {
	targetURL := "https://aki-gm-resources-oversea.aki-game.net/aki/gacha/index.html#/record?svr_id=86d52186155b148b5c138ceb41be9650&player_id=717352365&lang=ko&gacha_id=100064&gacha_type=1&svr_area=global&record_id=c45c9e78822350517b018ffc482907a6&resources_id=2e23eba1044f1f740cddb72f3b982768&platform=Mac"

	c := NewClient(Config{
		Client:      http.DefaultClient,
		ResourceURL: "https://aki-gm-resources-oversea.aki-game.net",
		TrackingURL: "https://gmserver-api.aki-game2.net",
	})
	payload, err := c.ParsePayloadFromURL(targetURL)
	if err != nil {
		t.Fatalf("ParsePayloadFromURL returned error: %v", err)
	}

	if payload.PlayerID != "717352365" {
		t.Fatalf("expected player id 717352365, got %q", payload.PlayerID)
	}
	if payload.ServerID != "86d52186155b148b5c138ceb41be9650" {
		t.Fatalf("expected server id 86d52186155b148b5c138ceb41be9650, got %q", payload.ServerID)
	}
	if payload.RecordID != "c45c9e78822350517b018ffc482907a6" {
		t.Fatalf("expected record id c45c9e78822350517b018ffc482907a6, got %q", payload.RecordID)
	}
	if payload.CardPoolID != "100064" {
		t.Fatalf("expected card pool id 100064, got %q", payload.CardPoolID)
	}
	if payload.LanguageCode != "ko" {
		t.Fatalf("expected language code ko, got %q", payload.LanguageCode)
	}
}

func TestFetchRecordsUsesRecordEndpoint(t *testing.T) {
	client := NewClient(Config{
		Client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost {
					t.Fatalf("expected POST request, got %s", req.Method)
				}
				if req.URL.String() != "https://example.com/gacha/record/query" {
					t.Fatalf("unexpected request URL: %s", req.URL.String())
				}

				var payload types.Payload
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				if payload.CardPoolType != 1 {
					t.Fatalf("expected card pool type 1, got %d", payload.CardPoolType)
				}

				body, err := json.Marshal(types.GachaResponse{Code: 0, Data: []types.Record{}})
				if err != nil {
					t.Fatalf("failed to marshal response: %v", err)
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}),
		},
		ResourceURL: "https://example.com",
		TrackingURL: "https://example.com",
	})

	_, err := client.FetchRecords(types.Payload{CardPoolType: 1})
	if err != nil {
		t.Fatalf("FetchRecords returned error: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
