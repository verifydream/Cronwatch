package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cronwatch/internal/config"
	"cronwatch/internal/crontab"
	"cronwatch/internal/logger"
	"cronwatch/internal/monitor"
	"cronwatch/internal/report"
	"cronwatch/internal/storage"

	"github.com/spf13/cobra"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "cronwatch",
	Short: "Lightweight cron job monitor with logging and alerting",
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the monitoring daemon",
	RunE:  runDaemon,
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show daily run summary",
	RunE:  runReport,
}

var historyCmd = &cobra.Command{
	Use:   "history [job-name]",
	Short: "View run history for a job",
	Args:  cobra.ExactArgs(1),
	RunE:  runHistory,
}

var retryCmd = &cobra.Command{
	Use:   "retry [run-id]",
	Short: "Re-execute a failed job by run ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runRetry,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "Config file path")
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(retryCmd)
}

func runDaemon(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	log := logger.Setup(cfg.LogLevel)
	store, err := storage.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	mon := monitor.New(cfg, store, log)
	mon.Run(ctx)
	return nil
}

func runReport(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	store, err := storage.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	defer store.Close()

	runs, err := store.QueryRunsSince(todayStart())
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	fmt.Print(report.DailySummary(runs))
	return nil
}

func runHistory(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	store, err := storage.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	defer store.Close()

	runs, err := store.QueryRunsByJob(args[0], 20)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	fmt.Print(report.JobHistory(runs))
	return nil
}

func runRetry(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	store, err := storage.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	defer store.Close()

	var runID int64
	fmt.Sscanf(args[0], "%d", &runID)
	run, err := store.GetRun(runID)
	if err != nil {
		return fmt.Errorf("run not found: %w", err)
	}

	fmt.Printf("Re-running: %s\n", run.Command)
	ctx := context.Background()
	job := crontab.Job{Name: run.JobName, Command: run.Command}
	result := crontab.Run(ctx, job)
	fmt.Printf("Exit code: %d, Duration: %s\n", result.ExitCode, result.Duration)

	return nil
}

func todayStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
