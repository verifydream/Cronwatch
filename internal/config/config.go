package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	TelegramBotToken  string  `mapstructure:"telegram_bot_token"`
	TelegramChatID    string  `mapstructure:"telegram_chat_id"`
	WebhookURL        string  `mapstructure:"webhook_url"`
	PollIntervalSec   int     `mapstructure:"poll_interval_sec"`
	AlertCooldownSec  int     `mapstructure:"alert_cooldown_sec"`
	ZScoreThreshold   float64 `mapstructure:"zscore_threshold"`
	BaselineMinSamples int    `mapstructure:"baseline_min_samples"`
	DBPath            string  `mapstructure:"db_path"`
	LogLevel          string  `mapstructure:"log_level"`
}

func DefaultConfig() *Config {
	return &Config{
		PollIntervalSec:    60,
		AlertCooldownSec:   3600,
		ZScoreThreshold:    3.0,
		BaselineMinSamples: 10,
		DBPath:             "cronwatch.db",
		LogLevel:           "info",
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	v := viper.New()
	v.SetConfigFile(path)
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
