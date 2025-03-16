package app

import (
	"qnqa-auto-crawlers/pkg/crawlers/mobilede"
	"qnqa-auto-crawlers/pkg/db"
	"qnqa-auto-crawlers/pkg/logger"

	"github.com/go-pg/pg/v10"
)

type Config struct {
	Database *pg.Options
}

type App struct {
	db *db.DB
	logger.Logger

	mobileDeRepo *db.MobileDeRepo

	MDcrawler *mobilede.Crawler
}

func New(dbc *pg.DB, lg logger.Logger) *App {
	a := App{
		db:     db.New(dbc, lg),
		Logger: lg,
	}

	a.mobileDeRepo = db.NewMobileDERepo(a.db)
	a.MDcrawler = mobilede.NewCrawler(lg, a.mobileDeRepo)

	return &a
}
