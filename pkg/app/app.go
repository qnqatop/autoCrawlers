package app

import (
	"qnqa-auto-crawlers/pkg/crawlers/mobilede"
	"qnqa-auto-crawlers/pkg/db"
	"qnqa-auto-crawlers/pkg/logger"

	"github.com/go-pg/pg/v10"
	"github.com/go-redis/redis/v8"
)

type Config struct {
	Database *pg.Options
	Redis    *redis.Options
}

type App struct {
	db  *db.DB
	rcl *redis.Client
	logger.Logger

	mobileDeRepo *db.MobileDeRepo

	MDcrawler *mobilede.Crawler
}

func New(dbc *pg.DB, rcl *redis.Client, lg logger.Logger) *App {
	a := App{
		db:     db.New(dbc, lg),
		Logger: lg,
		rcl:    rcl,
	}

	a.mobileDeRepo = db.NewMobileDERepo(a.db)
	a.MDcrawler = mobilede.NewCrawler(lg, a.mobileDeRepo, a.rcl)

	return &a
}
