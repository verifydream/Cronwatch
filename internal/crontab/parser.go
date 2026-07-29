package crontab

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type Job struct {
	Schedule string // raw cron schedule line
	Command  string // command to execute
	Name     string // derived from command or first arg
	Raw      string // original line
}

func ParseFile(path string) ([]Job, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseReader(f)
}

func ParseReader(r io.Reader) ([]Job, error) {
	var jobs []Job
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		job, err := ParseLine(line)
		if err != nil {
			continue // skip unparseable lines
		}
		jobs = append(jobs, job)
	}
	return jobs, scanner.Err()
}

func ParseLine(line string) (Job, error) {
	fields := strings.Fields(line)
	if len(fields) < 6 {
		return Job{}, fmt.Errorf("not enough fields: %s", line)
	}

	// Check for environment variable definitions (VAR=value)
	if strings.Contains(fields[0], "=") && len(fields) == 1 {
		return Job{}, fmt.Errorf("env var, not a job: %s", line)
	}

	// Standard cron: min hour dom month dow command
	// Check if first 5 fields are cron schedule-like
	for i := 0; i < 5; i++ {
		f := fields[i]
		if !isScheduleField(f) {
			return Job{}, fmt.Errorf("field %d '%s' doesn't look like a schedule", i, f)
		}
	}

	schedule := strings.Join(fields[:5], " ")
	command := strings.Join(fields[5:], " ")

	// Derive name from command
	name := deriveName(command)

	return Job{
		Schedule: schedule,
		Command:  command,
		Name:     name,
		Raw:      line,
	}, nil
}

func isScheduleField(s string) bool {
	// Must contain only digits, *, /, -, comma, or @
	for _, c := range s {
		if !((c >= '0' && c <= '9') || c == '*' || c == '/' || c == '-' || c == ',' || c == '@') {
			return false
		}
	}
	return len(s) > 0
}

func deriveName(command string) string {
	// Take first word of command as name
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "unknown"
	}
	// Strip path, keep basename
	name := fields[0]
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}
