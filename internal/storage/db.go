package storage

import (
	"database/sql"
	"math"
	"time"

	_ "modernc.org/sqlite"
)

type Run struct {
	ID        int64
	JobName   string
	Command   string
	StartedAt time.Time
	Duration  time.Duration
	ExitCode  int
	Stdout    string
	Stderr    string
}

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=wal_mode&_pragma=busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_name TEXT NOT NULL,
			command TEXT NOT NULL,
			started_at DATETIME NOT NULL,
			duration_ms INTEGER NOT NULL,
			exit_code INTEGER NOT NULL,
			stdout TEXT DEFAULT '',
			stderr TEXT DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_runs_job ON runs(job_name);
		CREATE INDEX IF NOT EXISTS idx_runs_started ON runs(started_at);
	`)
	return err
}

func (s *Store) InsertRun(r *Run) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO runs (job_name, command, started_at, duration_ms, exit_code, stdout, stderr) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.JobName, r.Command, r.StartedAt, r.Duration.Milliseconds(), r.ExitCode, r.Stdout, r.Stderr,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) QueryRunsByJob(jobName string, limit int) ([]Run, error) {
	rows, err := s.db.Query(
		`SELECT id, job_name, command, started_at, duration_ms, exit_code, stdout, stderr FROM runs WHERE job_name = ? ORDER BY started_at DESC LIMIT ?`,
		jobName, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRuns(rows)
}

func (s *Store) QueryRunsSince(since time.Time) ([]Run, error) {
	rows, err := s.db.Query(
		`SELECT id, job_name, command, started_at, duration_ms, exit_code, stdout, stderr FROM runs WHERE started_at >= ? ORDER BY started_at DESC`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRuns(rows)
}

func (s *Store) GetRun(id int64) (*Run, error) {
	r := &Run{}
	err := s.db.QueryRow(
		`SELECT id, job_name, command, started_at, duration_ms, exit_code, stdout, stderr FROM runs WHERE id = ?`, id,
	).Scan(&r.ID, &r.JobName, &r.Command, &r.StartedAt, &r.Duration, &r.ExitCode, &r.Stdout, &r.Stderr)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (s *Store) GetRecentStats(jobName string, limit int) (avg float64, stddev float64, count int, err error) {
	rows, err := s.db.Query(
		`SELECT duration_ms FROM runs WHERE job_name = ? AND exit_code = 0 ORDER BY started_at DESC LIMIT ?`,
		jobName, limit,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	var durations []float64
	for rows.Next() {
		var d int64
		if err = rows.Scan(&d); err != nil {
			return
		}
		durations = append(durations, float64(d))
	}

	count = len(durations)
	if count == 0 {
		return
	}

	sum := 0.0
	for _, d := range durations {
		sum += d
	}
	avg = sum / float64(count)

	variance := 0.0
	for _, d := range durations {
		diff := d - avg
		variance += diff * diff
	}
	variance /= float64(count)
	stddev = math.Sqrt(variance)

	return
}

func scanRuns(rows *sql.Rows) ([]Run, error) {
	var runs []Run
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.JobName, &r.Command, &r.StartedAt, &r.Duration, &r.ExitCode, &r.Stdout, &r.Stderr); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}
