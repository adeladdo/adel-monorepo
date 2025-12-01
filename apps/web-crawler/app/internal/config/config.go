package config

import (
	"context"
	"log"
	"net/url"
	"time"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	StartURL        *url.URL      `env:"START_URL,required"`
	MaxDepth        int           `env:"MAX_DEPTH,default=3"`
	Workers         int           `env:"WORKERS,default=5"`
	HTTPTimeout     time.Duration `env:"HTTP_TIMEOUT,default=10s"`
	TaskQueueBuffer int           `env:"TASK_QUEUE_BUFFER,default=1000"`
}

func Load(ctx context.Context) (*Config, error) {
	if err := godotenv.Overload(); err != nil {
		log.Println("No .env file found, continuing...")
	}

	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
