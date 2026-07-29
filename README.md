# Cronwatch

Lightweight cron job monitor with logging and alerting. Single binary, zero external dependencies.

## Features

- Wraps your existing crontab — no changes needed
- Logs every execution to SQLite (stdout, stderr, exit code, duration)
- Telegram and webhook alerts on failures
- Anomaly detection — alerts when a job takes 10x longer than normal (even if exit 0)
- CLI commands: report, history, retry

## Install

```bash
go build -o cronwatch .
```

## Usage

```bash
# Start daemon
./cronwatch daemon --config config.yaml

# Daily report
./cronwatch report

# Job history
./cronwatch history my-job

# Retry a failed run
./cronwatch retry 42
```

## Config

Copy `config.example.yaml` to `config.yaml` and fill in your settings:

```yaml
telegram_bot_token: "YOUR_BOT_TOKEN"
telegram_chat_id: "YOUR_CHAT_ID"
poll_interval_sec: 60
alert_cooldown_sec: 3600
zscore_threshold: 3.0
db_path: "cronwatch.db"
```

## Requirements

- Go 1.22+
- Read access to `/etc/crontab`
