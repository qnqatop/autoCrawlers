package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"qnqa-auto-crawlers/pkg/app"
	"qnqa-auto-crawlers/pkg/db"
	"qnqa-auto-crawlers/pkg/logger"
	"qnqa-auto-crawlers/pkg/rabbitmq"

	"github.com/BurntSushi/toml"
	"github.com/go-pg/pg/v10"
)

func main() {
	lg := logger.NewLogger(true)

	var cfg app.Config
	if _, err := toml.DecodeFile("./cfg/local.cfg", &cfg); err != nil {
		lg.Errorf("decoding toml: %v", err)
		os.Exit(1)
	}

	// Инициализация подключения к базе данных
	dbc, err := initDatabaseConnection(lg, cfg.Database, false)
	if err != nil {
		lg.Errorf("connect main db: %v", err)
		os.Exit(1)
	}
	defer dbc.Close()

	// Инициализация подключения к RabbitMQ
	var rmq *rabbitmq.Client
	if cfg.RabbitMQ.URL != "" {

		rmq, err = rabbitmq.NewClient(cfg.RabbitMQ.URL)
		if err != nil {
			lg.Errorf("connect to RabbitMQ: %v", err)
			panic(err)
		}
		defer rmq.Close()
	}

	// Инициализация приложения
	a := app.New(dbc, rmq, cfg, lg)

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем обработку сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем приложение в отдельной горутине
	go func() {
		if err := a.Run(ctx); err != nil {
			lg.Errorf("Application error: %v", err)
			cancel()
		}
	}()

	// Ожидаем сигнал завершения
	<-sigChan
	lg.Printf("Received shutdown signal")
	cancel()
}

func initDatabaseConnection(lg logger.Logger, connOps *pg.Options, sqlVerbose bool) (*pg.DB, error) {
	dbc := pg.Connect(connOps)
	if sqlVerbose {
		queryLogger := logger.NewSimpleLogger(log.New(os.Stderr, "Q", log.LstdFlags))
		dbc.AddQueryHook(&db.QueryLogger{SimpleLogger: queryLogger})
	}
	var v string
	if _, err := dbc.QueryOne(pg.Scan(&v), "select version()"); err != nil {
		return nil, err
	}
	lg.Printf("%s", v)
	return dbc, nil
}
