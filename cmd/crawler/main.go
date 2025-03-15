package main

import (
	"context"
	"log/slog"
	"os"

	"qnqa-auto-crawlers/pkg/app"
	"qnqa-auto-crawlers/pkg/db"
	"qnqa-auto-crawlers/pkg/logger"

	"github.com/BurntSushi/toml"
	"github.com/go-pg/pg/v10"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	lg := logger.NewLogger(true)

	var cfg Config
	if _, err := toml.DecodeFile("./cfg/local.cfg", &cfg); err != nil {
		lg.Errorf("decoding toml: %v", err)
		os.Exit(1)
	}

	dbc, err := initDatabaseConnection(lg, cfg.Database, false)
	if err != nil {
		lg.Errorf("connect main db: %v", err)
		os.Exit(1)
	}

	a := app.New(dbc, lg)

	if err = a.MDcrawler.ModelParse(context.Background()); err != nil {
		lg.Errorf("BrandParse failed : %v", err)
		os.Exit(1)
	}
	lg.Printf("END")
}

func initDatabaseConnection(lg logger.Logger, connOps *pg.Options, sqlVerbose bool) (*pg.DB, error) {
	dbc := pg.Connect(connOps)
	if sqlVerbose {
		queryLogger := logger.NewSimpleLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		})))
		dbc.AddQueryHook(&db.QueryLogger{SimpleLogger: queryLogger})
	}
	var v string
	if _, err := dbc.QueryOne(pg.Scan(&v), "select version()"); err != nil {
		return nil, err
	}
	lg.Printf("%s", v)
	return dbc, nil
}

type Config struct {
	Database *pg.Options
}
