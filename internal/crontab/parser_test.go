package crontab

import (
	"strings"
	"testing"
)

func TestParseLine(t *testing.T) {
	job, err := ParseLine("*/5 * * * * /usr/bin/backup.sh --full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Schedule != "*/5 * * * *" {
		t.Errorf("schedule = %q, want %q", job.Schedule, "*/5 * * * *")
	}
	if job.Command != "/usr/bin/backup.sh --full" {
		t.Errorf("command = %q", job.Command)
	}
	if job.Name != "backup.sh" {
		t.Errorf("name = %q, want backup.sh", job.Name)
	}
}

func TestParseLineRejectsNonSchedule(t *testing.T) {
	if _, err := ParseLine("not-a-schedule command"); err == nil {
		t.Error("expected error for non-schedule line")
	}
}

func TestParseLineRejectsEnvVars(t *testing.T) {
	if _, err := ParseLine("PATH=/usr/bin"); err == nil {
		t.Error("expected error for env var line")
	}
}

func TestParseReaderSkipsCommentsAndBlank(t *testing.T) {
	input := "# comment\n\n*/5 * * * * /bin/true\n# another\n0 2 * * * /bin/false\n"
	jobs, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(jobs))
	}
	if jobs[0].Command != "/bin/true" || jobs[1].Command != "/bin/false" {
		t.Errorf("unexpected jobs: %+v", jobs)
	}
}

func TestTruncateKeepsUTF8Intact(t *testing.T) {
	s := "héllo wörld — long output with emoji 🎉🎉🎉 and more text"
	out := truncate(s, 10)
	if !strings.HasSuffix(out, "...[truncated]") {
		t.Errorf("expected truncation marker, got %q", out)
	}
	// Truncated portion must be valid UTF-8 (no split rune)
	cut := strings.TrimSuffix(out, "...[truncated]")
	if !utf8Valid(cut) {
		t.Errorf("truncated prefix is not valid UTF-8: %q", cut)
	}
}

func TestTruncateShortStringUntouched(t *testing.T) {
	s := "short"
	if out := truncate(s, 100); out != s {
		t.Errorf("got %q, want %q", out, s)
	}
}

func utf8Valid(s string) bool {
	for _, r := range s {
		if r == 0xFFFD {
			return false
		}
	}
	return true
}
