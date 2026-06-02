package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gofiber/fiber/v3/log"
	"github.com/meteormin/wuwa-tracker/config"
	"github.com/meteormin/wuwa-tracker/internal/db"
	"github.com/meteormin/wuwa-tracker/internal/service"
	"github.com/meteormin/wuwa-tracker/internal/tracker"
)

const RuntimeServiceName = "wuwa-tracker-runtime"

type ServiceProvider interface {
	Service() *service.Service
}

type RuntimeService struct {
	cfg *config.Config

	mu     sync.RWMutex
	db     *db.BadgerDB
	svc    *service.Service
	cancel context.CancelFunc
	closed bool
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
	client := tracker.NewClient(httpClient, r.cfg.TrackingURL)
	localeData := tracker.LoadGachaLocaleWithFallback(client, r.cfg.Language)
	r.cfg.GachaTypes.MapFromLocaleData(localeData)

	badgerDB, err := db.NewBadgerDB(r.cfg.DBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	calc := tracker.NewStatsCalculator(r.cfg.StandardFiveStarResources, r.cfg.CostPolicy)
	svc, err := service.New(service.Deps{
		DB:     badgerDB,
		Config: r.cfg,
		Client: client,
		Calc:   calc,
	})
	if err != nil {
		_ = badgerDB.Close()
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	if r.cfg.DBGCEnabled {
		gcCtx, cancel := context.WithCancel(ctx)
		if err := badgerDB.StartValueLogGC(gcCtx, db.ValueLogGCOptions{
			Interval:     r.cfg.DBGCInterval,
			DiscardRatio: r.cfg.DBGCDiscardRatio,
		}, func(err error) {
			log.Errorf("Failed to run BadgerDB value log GC: %v\n", err)
		}); err != nil {
			cancel()
			_ = badgerDB.Close()
			return fmt.Errorf("failed to start database gc: %w", err)
		}
		r.cancel = cancel
		log.Infof("BadgerDB value log GC enabled. interval=%s discardRatio=%.2f\n", r.cfg.DBGCInterval, r.cfg.DBGCDiscardRatio)
	}

	r.db = badgerDB
	r.svc = svc
	r.closed = false
	log.Infof("Successfully started BadgerDB engine under directory: %s\n", r.cfg.DBPath)
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
	if r.db == nil {
		return nil
	}
	if err := r.db.Close(); err != nil {
		return err
	}
	r.db = nil
	r.svc = nil
	log.Info("Database connection closed")
	return nil
}

func (r *RuntimeService) Service() *service.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.svc
}
