package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"meerkat-v0/db"
	"meerkat-v0/utils"
)

func NewMetricsID(instance, service, monType, name string) utils.EntityID {
	return utils.EntityID{
		Kind: "monitor",
		Labels: map[string]string{
			"instance": instance,
			"service":  service,
			"type":     monType,
			"name":     name,
		},
	}
}

func NewMetricsIDFromServiceID(serviceID utils.EntityID, monType, name string) utils.EntityID {
	return NewMonitorID(
		serviceID.Labels["instance"],
		serviceID.Labels["name"],
		monType,
		name,
	)
}

type MetricType string

const (
	MetricGauge     MetricType = "gauge"
	MetricCounter   MetricType = "counter"
	MetricHistogram MetricType = "histogram"
)

type MetricsSample struct {
	ID        utils.EntityID
	Timestamp time.Time
	Type      MetricType
	Name      string
	Value     float64
	Labels    map[string]string
}

type MetricsSink interface {
	Emit(context.Context, MetricsSample) error
}

type MetricsRepo interface {
	InsertSample(context.Context, MetricsSample) error
}

type DBMetricsSink struct {
	metricsRepo MetricsRepo
}

func NewDBMetricsSink(metricsRepo MetricsRepo) *DBMetricsSink {
	return &DBMetricsSink{
		metricsRepo: metricsRepo,
	}
}

func (s *DBMetricsSink) Emit(ctx context.Context, sample MetricsSample) error {
	return s.metricsRepo.InsertSample(ctx, sample)
}

func BuildMetrics(metricsRepo MetricsRepo, serviceID utils.EntityID, rawCfg []byte) (utils.EntityID, *EntityInstance, error) {
	var id utils.EntityID
	var cfg EntityConfig
	err := json.Unmarshal(rawCfg, &cfg)
	if err != nil {
		return id, nil, err
	}

	problems := cfg.Valid(context.TODO())
	if len(problems) > 0 {
		return id, nil, NewValidationError(problems, serviceID.Labels["name"], cfg.Name)
	}

	id = NewMonitorIDFromServiceID(serviceID, cfg.Type, cfg.Name)

	sink := NewDBMetricsSink(metricsRepo)

	// TODO: Replace with modules
	var entity Entity
	switch cfg.Type {
	case "cpu":
		entity = &CPUMetrics{
			sink: sink,
		}
	default:
		return id, nil, fmt.Errorf("unknown metrics type: %s", cfg.Type)
	}

	err = entity.Configure(id, rawCfg)
	if err != nil {
		return id, nil, err
	}

	return id, NewEntityInstance(id, entity, cfg, rawCfg), nil
}

func RunMetrics(logger *utils.Logger, inst *EntityInstance) {
	interval := inst.Cfg.Interval
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := inst.Ent.Run(inst.ctx)
			if errors.Is(err, context.Canceled) {
				logger.Warn("Metrics tick error", "id", inst.ID.Canonical(), "err", err)
				continue
			}
		case <-inst.ctx.Done():
			return
		}
	}
}

type CPUMetrics struct {
	ID   utils.EntityID
	sink MetricsSink
}

func (m *CPUMetrics) Run(ctx context.Context) error {
	contents, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return err
	}
	values := strings.Split(string(contents), " ")
	f64, err := strconv.ParseFloat(values[0], 32)
	if err != nil {
		return err
	}
	return m.sink.Emit(ctx, MetricsSample{
		ID:        m.ID,
		Timestamp: time.Now(),
		Type:      MetricGauge,
		Name:      "cpu_loadavg",
		Value:     f64,
		Labels: map[string]string{
			"span": "1m",
		},
	})
}

func (m *CPUMetrics) Configure(id utils.EntityID, cfg []byte) error {
	m.ID = id
	return nil
}

func (m *CPUMetrics) Eq(newCfg []byte) (bool, error) {
	return true, nil
}

type SqliteMetricsRepo struct {
	readDB     *db.Queries
	writeDB    *db.Queries
	entityRepo EntityRepo
}

func NewSqliteMetricsRepo(readDB *db.Queries, writeDB *db.Queries, entityRepo EntityRepo) *SqliteMetricsRepo {
	return &SqliteMetricsRepo{
		readDB:     readDB,
		writeDB:    writeDB,
		entityRepo: entityRepo,
	}
}

func (r *SqliteMetricsRepo) InsertSample(ctx context.Context, sample MetricsSample) error {
	eId, err := r.readDB.GetEntityID(ctx, sample.ID.Canonical())
	if err != nil {
		return err
	}

	labels, err := json.Marshal(sample.Labels)
	if err != nil {
		return err
	}

	_, err = r.writeDB.InsertMetrics(ctx, db.InsertMetricsParams{
		EntityID: eId,
		Ts:       sample.Timestamp,
		Type:     string(sample.Type),
		Value:    sample.Value,
		Name:     sample.Name,
		Labels:   labels,
	})
	if err != nil {
		return err
	}

	return nil
}
