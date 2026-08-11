package crontab

import (
	"bytes"
	"context"
	"os/exec"
	"time"
	"unicode/utf8"
)

type Result struct {
	Job      Job
	Started  time.Time
	Duration time.Duration
	ExitCode int
	Stdout   string
	Stderr   string
}

func Run(ctx context.Context, job Job) Result {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "sh", "-c", job.Command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return Result{
		Job:      job,
		Started:  start,
		Duration: duration,
		ExitCode: exitCode,
		Stdout:   truncate(stdout.String(), 4000),
		Stderr:   truncate(stderr.String(), 4000),
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	// Don't split a UTF-8 rune — drop incomplete trailing rune
	for len(cut) > 0 && !utf8.RuneStart(cut[len(cut)-1]) {
		cut = cut[:len(cut)-1]
	}
	// Now trim the last rune if it's incomplete (RuneStart byte not followed by full rune)
	if len(cut) > 0 {
		r, size := utf8.DecodeLastRuneInString(cut)
		if r == utf8.RuneError && size <= 1 {
			cut = cut[:len(cut)-1]
		}
	}
	return cut + "...[truncated]"
}
