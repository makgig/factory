package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/makgig/factory/internal/models"
	"github.com/makgig/factory/internal/pgerrors"
)

const (
	pgRetryDelayShort  = 1 * time.Second
	pgRetryDelayMedium = 3 * time.Second
	pgRetryDelayLong   = 5 * time.Second
)

var pgRetrySchedule = []time.Duration{
	pgRetryDelayShort,
	pgRetryDelayMedium,
	pgRetryDelayLong,
}

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) Storage {
	return &PostgresStorage{pool: pool}
}

// ---- MetricStorage ----

func (p *PostgresStorage) UpdateGauge(name string, value float64) {
	const q = `
		INSERT INTO gauges (name, value) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;
	`

	delays := pgRetrySchedule
	runOnce := func() error {
		_, err := p.pool.Exec(context.Background(), q, name, value)
		return err
	}

	for attempt := 0; ; attempt++ {
		if err := runOnce(); err != nil {
			// ретраим только транспортные PG-ошибки (Class 08)
			if pgerrors.IsRetriable(err) && attempt < len(delays) {
				time.Sleep(delays[attempt])
				continue
			}
			log.Printf("pg: UpdateGauge(%s) error: %v", name, err)
		}
		return
	}
}

func (p *PostgresStorage) UpdateCounter(name string, value int64) {
	const q = `
		INSERT INTO counters (name, delta) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta;
	`

	delays := pgRetrySchedule
	runOnce := func() error {
		_, err := p.pool.Exec(context.Background(), q, name, value)
		return err
	}

	for attempt := 0; ; attempt++ {
		if err := runOnce(); err != nil {
			// ретраим только транспортные PG-ошибки (Class 08)
			if pgerrors.IsRetriable(err) && attempt < len(delays) {
				time.Sleep(delays[attempt])
				continue
			}
			log.Printf("pg: UpdateCounter(%s) error: %v", name, err)
		}
		return
	}
}

func (p *PostgresStorage) GetGauge(name string) (float64, bool) {
	const q = `SELECT value FROM gauges WHERE name = $1;`
	var v float64
	if err := p.pool.QueryRow(context.Background(), q, name).Scan(&v); err != nil {
		return 0, false
	}
	return v, true
}

func (p *PostgresStorage) GetCounter(name string) (int64, bool) {
	const q = `SELECT delta FROM counters WHERE name = $1;`
	var d int64
	if err := p.pool.QueryRow(context.Background(), q, name).Scan(&d); err != nil {
		return 0, false
	}
	return d, true
}

func (p *PostgresStorage) GetAllGauges() map[string]float64 {
	const q = `SELECT name, value FROM gauges;`
	rows, err := p.pool.Query(context.Background(), q)
	if err != nil {
		log.Printf("pg: GetAllGauges query error: %v", err)
		return map[string]float64{}
	}
	defer rows.Close()

	out := make(map[string]float64)
	for rows.Next() {
		var name string
		var v float64
		if err := rows.Scan(&name, &v); err == nil {
			out[name] = v
		}
	}
	return out
}

func (p *PostgresStorage) GetAllCounters() map[string]int64 {
	const q = `SELECT name, delta FROM counters;`
	rows, err := p.pool.Query(context.Background(), q)
	if err != nil {
		log.Printf("pg: GetAllCounters query error: %v", err)
		return map[string]int64{}
	}
	defer rows.Close()

	out := make(map[string]int64)
	for rows.Next() {
		var name string
		var d int64
		if err := rows.Scan(&name, &d); err == nil {
			out[name] = d
		}
	}
	return out
}

func (p *PostgresStorage) UpdateBatch(items []models.Metrics) error {
	if len(items) == 0 {
		return nil
	}

	const upGauge = `
		INSERT INTO gauges (name, value) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;`
	const upCounter = `
		INSERT INTO counters (name, delta) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta;`

	ctx := context.Background()
	delays := pgRetrySchedule

	runOnce := func() error {
		tx, err := p.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("pg begin: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		for _, m := range items {
			switch m.MType {
			case "gauge":
				if m.Value == nil {
					continue
				}
				if _, err := tx.Exec(ctx, upGauge, m.ID, *m.Value); err != nil {
					return fmt.Errorf("update gauge %q: %w", m.ID, err)
				}
			case "counter":
				if m.Delta == nil {
					continue
				}
				if _, err := tx.Exec(ctx, upCounter, m.ID, *m.Delta); err != nil {
					return fmt.Errorf("update counter %q: %w", m.ID, err)
				}
			default:
				// игнор неизвестных типов
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("pg commit: %w", err)
		}
		return nil
	}

	var lastErr error
	for attempt := 0; ; attempt++ {
		if err := runOnce(); err != nil {
			lastErr = err
			if pgerrors.IsRetriable(err) && attempt < len(delays) {
				time.Sleep(delays[attempt])
				continue
			}
			return lastErr
		}
		return nil
	}
}

// ---- FilePersist / BackgroundSaver (для БД не актуальны) ----

func (p *PostgresStorage) SaveToFile() error                  { return nil }
func (p *PostgresStorage) LoadFromFile() error                { return nil }
func (p *PostgresStorage) SetStoreConfig(time.Duration, bool) {}
func (p *PostgresStorage) StartSavingLoop()                   {}
func (p *PostgresStorage) StopSavingLoop()                    {}
