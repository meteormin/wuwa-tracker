package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
	"github.com/meteormin/wuwa-tracker/internal/service"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
)

const RuntimeServiceName = "wuwa-tracker-runtime"

type ServiceProvider interface {
	fiber.Service
	Service() *service.Service
}

type RuntimeService struct {
	cfg *config.Config

	mu         sync.RWMutex
	repository *db.BadgerRepository
	svc        *service.Service
	cancel     context.CancelFunc
	closed     bool
}

func NewRuntimeService(cfg *config.Config) *RuntimeService {
	return &RuntimeService{
		cfg: cfg,
	}
}

func (r *RuntimeService) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.svc != nil {
		return nil
	}

	httpClient := &http.Client{Timeout: r.cfg.HTTPTimeout}
	client := tracker.NewClient(tracker.Config{
		Client:      httpClient,
		ResourceURL: r.cfg.ResourcesURL,
		TrackingURL: r.cfg.TrackingURL,
	})
	localeData := tracker.LoadGachaLocaleWithFallback(client, r.cfg.Language)
	r.cfg.GachaTypes.MapFromLocaleData(localeData)

	core, err := db.OpenBadger(r.cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open badger core: %w", err)
	}
	repository, err := db.NewBadgerRepository(core)
	if err != nil {
		_ = core.Close()
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	calc := tracker.NewStatsCalculator(r.cfg.StandardFiveStarResources, r.cfg.CostPolicy)
	svc, err := service.New(service.Deps{
		Repository: repository,
		Config:     r.cfg,
		Client:     client,
		Calc:       calc,
	})
	if err != nil {
		_ = repository.Close()
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	if r.cfg.DBGCEnabled {
		gcCtx, cancel := context.WithCancel(ctx)
		if err := repository.StartValueLogGC(gcCtx, db.ValueLogGCOptions{
			Interval:     r.cfg.DBGCInterval,
			DiscardRatio: r.cfg.DBGCDiscardRatio,
		}, func(err error) {
			log.Errorf("Failed to run Badger repository value log GC: %v\n", err)
		}); err != nil {
			cancel()
			_ = repository.Close()
			return fmt.Errorf("failed to start repository gc: %w", err)
		}
		r.cancel = cancel
		log.Infof("Badger repository value log GC enabled. interval=%s discardRatio=%.2f\n", r.cfg.DBGCInterval, r.cfg.DBGCDiscardRatio)
	}

	r.repository = repository
	r.svc = svc
	r.closed = false
	log.Infof("Successfully started Badger repository under directory: %s\n", r.cfg.DBPath)
	return nil
}

func (r *RuntimeService) String() string {
	return RuntimeServiceName
}

func (r *RuntimeService) State(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "unavailable", err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.svc == nil {
		return "starting", nil
	}
	if r.closed {
		return "closed", nil
	}
	return "ready", nil
}

func (r *RuntimeService) Terminate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	if r.repository == nil {
		return nil
	}
	if err := r.repository.Close(); err != nil {
		return err
	}
	r.repository = nil
	r.svc = nil
	log.Info("Repository connection closed")
	return nil
}

func (r *RuntimeService) Service() *service.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.svc
}
