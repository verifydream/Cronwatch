package report

import (
	"fmt"
	"strings"
	"time"

	"cronwatch/internal/storage"
)

func DailySummary(runs []storage.Run) string {
	if len(runs) == 0 {
		return "No runs recorded today."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 Cronwatch Daily Report — %s\n\n", time.Now().Format("2006-01-02")))

	// Group by job
	byJob := make(map[string][]storage.Run)
	for _, r := range runs {
		byJob[r.JobName] = append(byJob[r.JobName], r)
	}

	failed := 0
	for _, r := range runs {
		if r.ExitCode != 0 {
			failed++
		}
	}

	sb.WriteString(fmt.Sprintf("Total runs: %d | Failed: %d | Success rate: %.0f%%\n\n",
		len(runs), failed, float64(len(runs)-failed)/float64(len(runs))*100))

	for job, jobRuns := range byJob {
		jobFailed := 0
		var totalDur time.Duration
		for _, r := range jobRuns {
			if r.ExitCode != 0 {
				jobFailed++
			}
			totalDur += r.Duration
		}
		avgDur := totalDur / time.Duration(len(jobRuns))
		status := "✅"
		if jobFailed > 0 {
			status = "❌"
		}
		sb.WriteString(fmt.Sprintf("%s %s: %d runs (avg %s, %d failed)\n",
			status, job, len(jobRuns), avgDur.Round(time.Millisecond), jobFailed))
	}

	return sb.String()
}

func JobHistory(runs []storage.Run) string {
	if len(runs) == 0 {
		return "No runs found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("History for %s (last %d runs):\n\n", runs[0].JobName, len(runs)))

	for _, r := range runs {
		status := "✅"
		if r.ExitCode != 0 {
			status = "❌"
		}
		sb.WriteString(fmt.Sprintf("%s #%d | %s | %s | exit=%d\n",
			status, r.ID, r.StartedAt.Format("Jan 02 15:04"), r.Duration.Round(time.Millisecond), r.ExitCode))
	}

	return sb.String()
}
