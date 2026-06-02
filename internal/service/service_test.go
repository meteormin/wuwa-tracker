package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
	reporter "github.com/meteormin/wuwa-tracker/internal/reporter"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

func TestNewValidatesDependencies(t *testing.T) {
	database := openTestDB(t)
	cfg := testConfig()
	client := tracker.NewClient(http.DefaultClient, cfg.TrackingURL)
	calc := tracker.NewStatsCalculator(cfg.StandardFiveStarResources, cfg.CostPolicy)

	tests := []struct {
		name string
		deps Deps
		err  error
	}{
		{
			name: "missing db",
			deps: Deps{Config: cfg, Client: client, Calc: calc},
			err:  ErrMissingDB,
		},
		{
			name: "missing config",
			deps: Deps{DB: database, Client: client, Calc: calc},
			err:  ErrMissingConfig,
		},
		{
			name: "missing client",
			deps: Deps{DB: database, Config: cfg, Calc: calc},
			err:  ErrMissingClient,
		},
		{
			name: "missing calc",
			deps: Deps{DB: database, Config: cfg, Client: client},
			err:  ErrMissingCalc,
		},
		{
			name: "valid",
			deps: Deps{DB: database, Config: cfg, Client: client, Calc: calc},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := New(tt.deps)
			if tt.err != nil {
				if !errors.Is(err, tt.err) {
					t.Fatalf("expected error %v, got %v", tt.err, err)
				}
				if svc != nil {
					t.Fatalf("expected nil service, got %#v", svc)
				}
				return
			}
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}
			if svc == nil {
				t.Fatal("New returned nil service")
			}
		})
	}
}

func TestServiceUploadSavesRecordsAndReturnsStats(t *testing.T) {
	svc := newTestService(t)
	fetchResult := types.FetchResult{
		Payload: types.Payload{PlayerID: " player-1 "},
		Records: map[string][]types.Record{
			"character": {
				testRecord("character", 5001, 5, "Limited", "2026-05-20 12:00:00"),
				testRecord("character", 3001, 3, "Common", "2026-05-19 12:00:00"),
			},
		},
	}

	statsResponse, err := svc.Upload(fetchResult)
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	if !statsResponse.Success {
		t.Fatal("expected successful stats response")
	}
	if statsResponse.PlayerID != "player-1" {
		t.Fatalf("expected trimmed player id, got %q", statsResponse.PlayerID)
	}
	if len(statsResponse.Stats) != 2 {
		t.Fatalf("expected stats for each configured gacha type, got %d", len(statsResponse.Stats))
	}

	characterStats := statsResponse.Stats[0]
	if characterStats.GachaName != "Character" {
		t.Fatalf("expected character stats first, got %q", characterStats.GachaName)
	}
	if characterStats.TotalPulls != 2 {
		t.Fatalf("expected 2 character pulls, got %d", characterStats.TotalPulls)
	}
	if characterStats.TotalAstrite != 320 {
		t.Fatalf("expected 320 astrite, got %d", characterStats.TotalAstrite)
	}
	if len(characterStats.FiveStars) != 1 {
		t.Fatalf("expected 1 five star, got %d", len(characterStats.FiveStars))
	}
	if statsResponse.Stats[1].TotalPulls != 0 {
		t.Fatalf("expected empty weapon stats, got %d pulls", statsResponse.Stats[1].TotalPulls)
	}

	stored, err := svc.db.GetGachaRecords("player-1", "character")
	if err != nil {
		t.Fatalf("GetGachaRecords returned error: %v", err)
	}
	if !reflect.DeepEqual(stored, fetchResult.Records["character"]) {
		t.Fatalf("stored records mismatch\nexpected: %+v\nactual:   %+v", fetchResult.Records["character"], stored)
	}
}

func TestServiceSaveFetchResultValidation(t *testing.T) {
	svc := newTestService(t)

	tests := []struct {
		name        string
		fetchResult types.FetchResult
		err         error
	}{
		{
			name:        "missing player id",
			fetchResult: types.FetchResult{Payload: types.Payload{PlayerID: " "}, Records: map[string][]types.Record{"character": {}}},
			err:         ErrMissingPlayerID,
		},
		{
			name:        "empty records",
			fetchResult: types.FetchResult{Payload: types.Payload{PlayerID: "player-1"}},
			err:         ErrEmptyUploadData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.SaveFetchResult(tt.fetchResult)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}

func TestServiceGetStatsRequiresPlayerID(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.GetStats(" ")
	if !errors.Is(err, ErrMissingPlayerID) {
		t.Fatalf("expected ErrMissingPlayerID, got %v", err)
	}
}

func TestServiceListPlayers(t *testing.T) {
	svc := newTestService(t)

	for _, playerID := range []string{"player-2", "player-1"} {
		err := svc.SaveFetchResult(types.FetchResult{
			Payload: types.Payload{PlayerID: playerID},
			Records: map[string][]types.Record{
				"character": {testRecord("character", 3001, 3, playerID, "2026-05-20 12:00:00")},
			},
		})
		if err != nil {
			t.Fatalf("SaveFetchResult returned error: %v", err)
		}
	}

	players, err := svc.ListPlayers()
	if err != nil {
		t.Fatalf("ListPlayers returned error: %v", err)
	}
	sort.Strings(players)

	expected := []string{"player-1", "player-2"}
	if !reflect.DeepEqual(players, expected) {
		t.Fatalf("players mismatch\nexpected: %+v\nactual:   %+v", expected, players)
	}
}

func TestServiceTrackURLSanitizesAndFetchesRecords(t *testing.T) {
	cfg := testConfig()
	client := tracker.NewClient(&http.Client{
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

			recordsByType := map[int][]types.Record{
				1: {testRecord("character", 5001, 5, "Limited", "2026-05-20 12:00:00")},
				2: {testRecord("weapon", 4001, 4, "Blade", "2026-05-19 12:00:00")},
			}
			body, err := json.Marshal(types.GachaResponse{
				Code: 0,
				Data: recordsByType[payload.CardPoolType],
			})
			if err != nil {
				t.Fatalf("failed to marshal response: %v", err)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		}),
	}, cfg.TrackingURL)
	svc := newTestServiceWithClient(t, cfg, client)

	statsResponse, err := svc.TrackURL(" https:\\//aki-gm-resources.aki-game.net/gacha?player_id=player-1&svr_id=server-1&record_id=record-1 ")
	if err != nil {
		t.Fatalf("TrackURL returned error: %v", err)
	}

	if statsResponse.PlayerID != "player-1" {
		t.Fatalf("expected player-1, got %q", statsResponse.PlayerID)
	}
	if len(statsResponse.Stats) != 2 {
		t.Fatalf("expected 2 stats entries, got %d", len(statsResponse.Stats))
	}
	if statsResponse.Stats[0].TotalPulls != 1 {
		t.Fatalf("expected 1 character pull, got %d", statsResponse.Stats[0].TotalPulls)
	}
	if statsResponse.Stats[1].TotalPulls != 1 {
		t.Fatalf("expected 1 weapon pull, got %d", statsResponse.Stats[1].TotalPulls)
	}
}

func TestServiceTrackURLValidation(t *testing.T) {
	svc := newTestService(t)

	tests := []struct {
		name      string
		targetURL string
		err       error
	}{
		{name: "missing url", targetURL: "   ", err: ErrMissingURL},
		{name: "missing player id", targetURL: "https://aki-gm-resources.aki-game.net/gacha?svr_id=server-1&record_id=record-1", err: ErrMissingPlayerID},
		{name: "invalid url", targetURL: "://bad-url", err: ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.TrackURL(tt.targetURL)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}

func TestServiceConfigHelpers(t *testing.T) {
	svc := newTestService(t)

	if svc.Config() != svc.cfg {
		t.Fatal("Config did not return service config")
	}
	if !reflect.DeepEqual(svc.LuckScoreThresholds(), svc.cfg.LuckScoreThresholds) {
		t.Fatal("LuckScoreThresholds did not return configured thresholds")
	}

	svc.UseGachaTypeKeysAsNames()
	for _, gachaType := range svc.cfg.GachaTypes.Items {
		if gachaType.Name != gachaType.Key {
			t.Fatalf("expected name %q, got %q", gachaType.Key, gachaType.Name)
		}
	}
}

func TestServiceExportReportJSON(t *testing.T) {
	svc := newTestService(t)
	err := svc.SaveFetchResult(types.FetchResult{
		Payload: types.Payload{PlayerID: "player-1"},
		Records: map[string][]types.Record{
			"character": {testRecord("character", 5001, 5, "Limited", "2026-05-20 12:00:00")},
		},
	})
	if err != nil {
		t.Fatalf("SaveFetchResult returned error: %v", err)
	}

	var out bytes.Buffer
	if err := svc.ExportReport(&out, "player-1", reporter.FormatJSON, "en"); err != nil {
		t.Fatalf("ExportReport returned error: %v", err)
	}

	if !strings.Contains(out.String(), `"playerId": "player-1"`) {
		t.Fatalf("expected exported JSON to include player id, got %s", out.String())
	}
	if !strings.Contains(out.String(), `"totalPulls": 1`) {
		t.Fatalf("expected exported JSON to include stats, got %s", out.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestService(t *testing.T) *Service {
	t.Helper()

	cfg := testConfig()
	return newTestServiceWithClient(t, cfg, tracker.NewClient(http.DefaultClient, cfg.TrackingURL))
}

func newTestServiceWithClient(t *testing.T, cfg *config.Config, client *tracker.Client) *Service {
	t.Helper()

	calc := tracker.NewStatsCalculator(cfg.StandardFiveStarResources, cfg.CostPolicy)
	svc, err := New(Deps{
		DB:     openTestDB(t),
		Config: cfg,
		Client: client,
		Calc:   calc,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	return svc
}

func openTestDB(t *testing.T) *db.BadgerDB {
	t.Helper()

	database, err := db.NewBadgerDB(t.TempDir())
	if err != nil {
		t.Fatalf("NewBadgerDB returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})
	return database
}

func testConfig() *config.Config {
	return &config.Config{
		TrackingURL:               "https://example.com",
		StandardFiveStarResources: []int{9001},
		GachaTypes: types.GachaTypes{
			Items: []types.GachaType{
				{ID: 1, Key: "character", Name: "Character", HasOffBannerDrop: true, BaseRate: 0.8, ExpectedPulls: 55},
				{ID: 2, Key: "weapon", Name: "Weapon", BaseRate: 0.8, ExpectedPulls: 55},
			},
		},
		LuckScoreThresholds: []types.LuckScoreThreshold{
			{MinScore: 100, State: "lucky"},
			{MinScore: 0, State: "normal"},
		},
		CostPolicy: types.CostPolicy{
			AstritePerPull: 160,
		},
	}
}

func testRecord(cardPoolType string, resourceID, qualityLevel int, name, time string) types.Record {
	return types.Record{
		CardPoolType: cardPoolType,
		ResourceID:   resourceID,
		QualityLevel: qualityLevel,
		ResourceType: "Weapon",
		Name:         name,
		Count:        1,
		Time:         time,
	}
}
