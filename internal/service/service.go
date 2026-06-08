package service

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/meteormin/wuwa-tracker/config"
	reporter "github.com/meteormin/wuwa-tracker/internal/reporter"
	"github.com/meteormin/wuwa-tracker/internal/scanner"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
	"github.com/meteormin/wuwa-tracker/internal/types"
)

var (
	ErrMissingURL      = errors.New("missing url")
	ErrInvalidURL      = errors.New("invalid url")
	ErrMissingPlayerID = errors.New("missing player id")
	ErrEmptyUploadData = errors.New("empty upload data")
	ErrMissingRepo     = errors.New("missing repository")
	ErrMissingConfig   = errors.New("missing config")
	ErrMissingClient   = errors.New("missing tracker client")
	ErrMissingCalc     = errors.New("missing stats calculator")
)

type GachaRepository interface {
	SaveGachaRecords(playerID, cardPoolType string, records []types.Record) error
	GetGachaRecords(playerID, cardPoolType string) ([]types.Record, error)
	ListPlayers() ([]string, error)
}

type GachaClient interface {
	ParsePayloadFromURL(urlStr string) (types.Payload, error)
	FetchAllRecords(payload types.Payload, gachaTypes []types.GachaType) (*types.FetchResult, error)
	FetchGachaLocale(lang string) (types.LocaleData, error)
}

type StatsCalculator interface {
	Calc(records []types.Record, gachaType types.GachaType) types.Stats
}

type Deps struct {
	Repository GachaRepository
	Config     *config.Config
	Client     GachaClient
	Calc       StatsCalculator
}

type Service struct {
	repo   GachaRepository
	cfg    *config.Config
	client GachaClient
	calc   StatsCalculator
}

func New(deps Deps) (*Service, error) {
	if deps.Repository == nil {
		return nil, ErrMissingRepo
	}
	if deps.Config == nil {
		return nil, ErrMissingConfig
	}
	if deps.Client == nil {
		return nil, ErrMissingClient
	}
	if deps.Calc == nil {
		return nil, ErrMissingCalc
	}

	return &Service{
		repo:   deps.Repository,
		cfg:    deps.Config,
		client: deps.Client,
		calc:   deps.Calc,
	}, nil
}

func (s *Service) LuckScoreThresholds() []types.LuckScoreThreshold {
	return s.cfg.LuckScoreThresholds
}

func (s *Service) Config() *config.Config {
	return s.cfg
}

func (s *Service) PrepareLocale(lang string) {
	localeData := tracker.LoadGachaLocaleWithFallback(s.client, lang)
	s.cfg.GachaTypes.MapFromLocaleData(localeData)
}

func (s *Service) UseGachaTypeKeysAsNames() {
	for i := range s.cfg.GachaTypes.Items {
		s.cfg.GachaTypes.Items[i].Name = s.cfg.GachaTypes.Items[i].Key
	}
}

func (s *Service) Scan(path string) (string, error) {
	logPaths, err := scanner.ExpandLogPaths(path, s.cfg.ScanLogPaths)
	if err != nil {
		return "", err
	}
	return scanner.FindURLInDirectory(logPaths, s.cfg.ResourcesURL)
}

func (s *Service) TrackURL(targetURL string) (types.StatsResponse, error) {
	fetchResult, err := s.FetchAndSave(targetURL)
	if err != nil {
		return types.StatsResponse{}, err
	}
	return s.GetStats(fetchResult.Payload.PlayerID)
}

func (s *Service) FetchAndSave(targetURL string) (*types.FetchResult, error) {
	targetURL = sanitizeURL(targetURL)
	if targetURL == "" {
		return nil, ErrMissingURL
	}

	payload, err := s.client.ParsePayloadFromURL(targetURL)
	if err != nil {
		if errors.Is(err, tracker.ErrMissingRequiredParams) {
			return nil, ErrMissingPlayerID
		}
		return nil, ErrInvalidURL
	}

	fetchResult, err := s.client.FetchAllRecords(payload, s.cfg.GachaTypes.Items)
	if err != nil {
		return nil, err
	}
	fetchResult.Payload = payload
	if len(fetchResult.Records) == 0 {
		return nil, ErrInvalidURL
	}

	if err := s.SaveFetchResult(*fetchResult); err != nil {
		return nil, err
	}
	return fetchResult, nil
}

func (s *Service) Upload(fetchResult types.FetchResult) (types.StatsResponse, error) {
	if err := s.SaveFetchResult(fetchResult); err != nil {
		return types.StatsResponse{}, err
	}
	return s.GetStats(fetchResult.Payload.PlayerID)
}

func (s *Service) SaveFetchResult(fetchResult types.FetchResult) error {
	playerID := strings.TrimSpace(fetchResult.Payload.PlayerID)
	if playerID == "" {
		return ErrMissingPlayerID
	}
	if len(fetchResult.Records) == 0 {
		return ErrEmptyUploadData
	}

	for _, gachaType := range s.cfg.GachaTypes.Items {
		records, ok := fetchResult.Records[gachaType.Key]
		if !ok {
			records = []types.Record{}
		}

		if err := s.repo.SaveGachaRecords(playerID, gachaType.Key, records); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetStats(playerID string) (types.StatsResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return types.StatsResponse{}, ErrMissingPlayerID
	}

	statsList := make([]types.Stats, 0, len(s.cfg.GachaTypes.Items))
	for _, gachaType := range s.cfg.GachaTypes.Items {
		records, err := s.repo.GetGachaRecords(playerID, gachaType.Key)
		if err != nil {
			return types.StatsResponse{}, err
		}
		statsList = append(statsList, s.calc.Calc(records, gachaType))
	}

	return types.StatsResponse{
		Success:  true,
		PlayerID: playerID,
		Stats:    statsList,
	}, nil
}

func (s *Service) ListPlayers() ([]string, error) {
	return s.repo.ListPlayers()
}

func (s *Service) ExportReport(w io.Writer, playerID string, format reporter.Format, lang string) error {
	statsResponse, err := s.GetStats(playerID)
	if err != nil {
		return err
	}

	exporter, err := reporter.NewExporter(s.cfg, format, lang)
	if err != nil {
		return err
	}

	reportData := types.ReportData{
		PlayerID: statsResponse.PlayerID,
		Stats:    statsResponse.Stats,
	}
	if len(reportData.Stats) == 0 {
		return fmt.Errorf("no valid records found")
	}
	return exporter.Export(w, reportData)
}

func sanitizeURL(targetURL string) string {
	targetURL = strings.TrimSpace(targetURL)
	return strings.ReplaceAll(targetURL, "\\", "")
}
