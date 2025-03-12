package main

import (
	"context"
	"log/slog"
	"os"
	"qnqa-auto-crawlers/pkg/crawlers/mobilede"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cr := mobilede.NewCrawler(logger)

	if err := cr.BrandParse(ctx); err != nil {
		logger.ErrorContext(ctx, "BrandParse failed", "error", err)
		os.Exit(1)
	}
	logger.Info("END")
}
