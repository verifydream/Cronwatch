package monitor

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"cronwatch/internal/alert"
	"cronwatch/internal/config"
	"cronwatch/internal/crontab"
	"cronwatch/internal/storage"

	"github.com/rs/zerolog"
)

type Monitor struct {
	cfg      *config.Config
	store    *storage.Store
	log      zerolog.Logger
	telegram *alert.TelegramSender
	webhook  *alert.WebhookSender
	alerted  map[string]time.Time // last alert time per job
	mu       sync.Mutex
}

func New(cfg *config.Config, store *storage.Store, log zerolog.Logger) *Monitor {
	return &Monitor{
		cfg:      cfg,
		store:    store,
		log:      log,
		telegram: alert.NewTelegramSender(cfg.TelegramBotToken, cfg.TelegramChatID, log),
		webhook:  alert.NewWebhookSender(cfg.WebhookURL, log),
		alerted:  make(map[string]time.Time),
	}
}

func (m *Monitor) Run(ctx context.Context) {
	interval := time.Duration(m.cfg.PollIntervalSec) * time.Second
	m.log.Info().Int("interval_sec", m.cfg.PollIntervalSec).Msg("Cronwatch daemon started")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log.Info().Msg("Cronwatch daemon stopped")
			return
		case <-ticker.C:
			m.checkAndAlert(ctx)
		}
	}
}

func (m *Monitor) checkAndAlert(ctx context.Context) {
	jobs, err := crontab.ParseFile("/etc/crontab")
	if err != nil {
		m.log.Warn().Err(err).Msg("Failed to read crontab")
		return
	}

	m.log.Debug().Int("jobs", len(jobs)).Msg("Parsed crontab")

	for _, job := range jobs {
		m.processJob(ctx, job)
	}
}

func (m *Monitor) processJob(ctx context.Context, job crontab.Job) {
	result := crontab.Run(ctx, job)

	// Store result
	run := &storage.Run{
		JobName:   job.Name,
		Command:   job.Command,
		StartedAt: result.Started,
		Duration:  result.Duration,
		ExitCode:  result.ExitCode,
		Stdout:    result.Stdout,
		Stderr:    result.Stderr,
	}
	runID, err := m.store.InsertRun(run)
	if err != nil {
		m.log.Error().Err(err).Str("job", job.Name).Msg("Failed to store run")
		return
	}

	m.log.Info().
		Str("job", job.Name).
		Int("exit_code", result.ExitCode).
		Dur("duration", result.Duration).
		Int64("run_id", runID).
		Msg("Job completed")

	// Check for failure
	if result.ExitCode != 0 {
		m.sendAlert(job, result, "failed", runID)
		return
	}

	// Check for anomaly
	if m.isAnomaly(job.Name, result.Duration) {
		m.sendAlert(job, result, "anomaly", runID)
	}
}

func (m *Monitor) isAnomaly(jobName string, duration time.Duration) bool {
	avg, stddev, count, err := m.store.GetRecentStats(jobName, 50)
	if err != nil || count < m.cfg.BaselineMinSamples || stddev == 0 {
		return false
	}

	zscore := (float64(duration.Milliseconds()) - avg) / stddev
	return math.Abs(zscore) > m.cfg.ZScoreThreshold
}

func (m *Monitor) sendAlert(job crontab.Job, result crontab.Result, reason string, runID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Dedup: don't re-alert within cooldown
	if lastAlert, ok := m.alerted[job.Name]; ok {
		if time.Since(lastAlert) < time.Duration(m.cfg.AlertCooldownSec)*time.Second {
			return
		}
	}

	msg := formatAlert(job, result, reason, runID)
	m.telegram.Send(msg)
	m.webhook.Send(msg)
	m.alerted[job.Name] = time.Now()
}

func formatAlert(job crontab.Job, result crontab.Result, reason string, runID int64) string {
	emoji := "🔴"
	if reason == "anomaly" {
		emoji = "⚠️"
	}

	stderr := result.Stderr
	if len(stderr) > 200 {
		stderr = stderr[len(stderr)-200:]
	}

	return fmt.Sprintf(
		"%s *Cronwatch: %s*\n\n"+
			"Job: %s\n"+
			"Reason: %s\n"+
			"Exit code: %d\n"+
			"Duration: %s\n"+
			"Run ID: %d\n"+
			"Stderr (last 200 chars):\n```\n%s\n```",
		emoji, job.Name, job.Name, reason, result.ExitCode, result.Duration, runID, stderr,
	)
}
