// Package queue provides asynq client and server factories for Zosmed services.
package queue

import (
	"fmt"

	"github.com/hibiken/asynq"
)

// NewClient returns an asynq.Client connected to the given Redis URL.
// The caller is responsible for calling Client.Close when done.
func NewClient(redisURL string) (*asynq.Client, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("queue: parse redis URL: %w", err)
	}
	return asynq.NewClient(opt), nil
}

// NewServer returns an asynq.Server configured for task processing.
// concurrency sets the number of concurrent worker goroutines.
// The caller must call Server.Run or Server.Start to begin processing.
func NewServer(redisURL string, concurrency int) (*asynq.Server, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, fmt.Errorf("queue: parse redis URL: %w", err)
	}
	srv := asynq.NewServer(opt, asynq.Config{
		Concurrency: concurrency,
	})
	return srv, nil
}
