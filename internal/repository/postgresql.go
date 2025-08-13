package repository

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/makgig/factory/internal/models"
)

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
	if _, err := p.pool.Exec(context.Background(), q, name, value); err != nil {
		log.Printf("pg: UpdateGauge(%s) error: %v", name, err)
	}
}

func (p *PostgresStorage) UpdateCounter(name string, value int64) {
	const q = `
		INSERT INTO counters (name, delta) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta;
	`
	if _, err := p.pool.Exec(context.Background(), q, name, value); err != nil {
		log.Printf("pg: UpdateCounter(%s) error: %v", name, err)
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
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		// если Commit не был вызван — откатим
		_ = tx.Rollback(ctx)
	}()

	const upGauge = `
		INSERT INTO gauges (name, value) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value;`
	const upCounter = `
		INSERT INTO counters (name, delta) VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta;`

	for _, m := range items {
		switch m.MType {
		case "gauge":
			if m.Value != nil {
				if _, err := tx.Exec(ctx, upGauge, m.ID, *m.Value); err != nil {
					return err
				}
			}
		case "counter":
			if m.Delta != nil {
				if _, err := tx.Exec(ctx, upCounter, m.ID, *m.Delta); err != nil {
					return err
				}
			}
		default:
			// игнорим неизвестные типы, чтобы не ронять всю пачку
			log.Printf("pg: unknown metric type %q for %s — skipped", m.MType, m.ID)
		}
	}
	return tx.Commit(ctx)
}

// ---- FilePersist / BackgroundSaver (для БД не актуальны) ----

func (p *PostgresStorage) SaveToFile() error                  { return nil }
func (p *PostgresStorage) LoadFromFile() error                { return nil }
func (p *PostgresStorage) SetStoreConfig(time.Duration, bool) {}
func (p *PostgresStorage) StartSavingLoop()                   {}
func (p *PostgresStorage) StopSavingLoop()                    {}
