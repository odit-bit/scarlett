package store

import (
	"log/slog"
	"os"
	"time"
)

var (
	defaultStoreDir     = "/tmp/scarlett"
	defaultApplyTimeout = 2 * time.Second

	defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	_             = defaultLogger
)

type Options func(*storeOptions)

var defaultOption = storeOptions{
	dir:          defaultStoreDir,
	logger:       defaultLogger,
	loglevel:     int(slog.LevelInfo),
	applyTimeout: 2 * time.Second,
	isBootstrap:  false,
	isPurge:      false,
}

type storeOptions struct {
	dir string

	logger       *slog.Logger
	loglevel     int
	applyTimeout time.Duration

	isBootstrap bool
	isPurge     bool
}

func WithLogger(logger *slog.Logger) Options {
	return func(c *storeOptions) {
		c.logger = logger.With("scope", "store-raft")
	}
}

func WithSlog(level slog.Level) Options {
	return func(so *storeOptions) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
		so.logger = logger.With("scope", "store-raft")
		so.loglevel = int(level)
	}
}

func WithPurge(v bool) Options {
	return func(c *storeOptions) {
		c.isPurge = v
	}
}

func WithBootstraping(v bool) Options {
	return func(c *storeOptions) {
		c.isBootstrap = v
	}
}

func WithTimeout(dur time.Duration) Options {
	return func(c *storeOptions) {
		if dur > 0 {
			c.applyTimeout = dur
		}
		c.applyTimeout = defaultApplyTimeout
	}
}

func CustomDir(path string) Options {
	return func(so *storeOptions) {
		so.dir = path
	}
}

// type StoreOptions func(*Store)
