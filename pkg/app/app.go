package app

import (
	"context"
	"fmt"
	"time"

	"qnqa-auto-crawlers/pkg/api"
	"qnqa-auto-crawlers/pkg/crawlers/mobilede"
	"qnqa-auto-crawlers/pkg/db"
	"qnqa-auto-crawlers/pkg/logger"
	"qnqa-auto-crawlers/pkg/rabbitmq"

	"github.com/go-pg/pg/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"golang.org/x/sync/errgroup"
)

// Config представляет конфигурацию приложения
type Config struct {
	Database *pg.Options
	RabbitMQ struct {
		URL string
	}
	API struct {
		Addr string
	}
	HttpConfig HttpConfig
	Mobilede   mobilede.Config
}

type HttpConfig struct {
	Host        string
	Port        int
	RPCPartners map[string]string
	Environment string
	SentryDSN   string
	LogRPCCalls bool
	DebugRPC    struct {
		Enabled      bool
		RPCTargetURL string
		RPCDocURL    string
	}
}

// App представляет основное приложение
type App struct {
	Config   Config
	hc       HttpConfig
	Logger   logger.Logger
	DB       *db.DB
	RabbitMQ *rabbitmq.Client
	mdServer *mobilede.Server
	mdRepo   *db.MobileDeRepo
	echo     *echo.Echo
}

// New создает новое приложение
func New(dbc *pg.DB, rmq *rabbitmq.Client, cfg Config, lg logger.Logger) *App {
	app := &App{
		Logger:   lg,
		DB:       db.New(dbc, lg),
		RabbitMQ: rmq,
		echo:     echo.New(),
		hc:       cfg.HttpConfig,
	}
	app.mdRepo = db.NewMobileDERepo(app.DB)
	app.mdServer = mobilede.New(lg, app.DB, app.mdRepo, rmq, cfg.Mobilede)

	api.Init()
	// Middleware
	app.echo.Use(middleware.Logger())
	app.echo.Use(middleware.Recover())
	app.echo.Use(middleware.CORS())

	app.registerAPIHandler()
	return app
}

// Run запускает все компоненты приложения
// Run is a function that runs application.
func (a *App) Run(appContext context.Context) error {
	runGroup, appContext := errgroup.WithContext(appContext)

	runGroup.Go(a.runHTTPServer(appContext, a.hc.Host, a.hc.Port))

	return runGroup.Wait()
}

// runHTTPServer is a function that starts http listener using labstack/echo.
func (a *App) runHTTPServer(appContext context.Context, host string, port int) func() error {
	return func() error {
		listenAddress := fmt.Sprintf("%s:%d", host, port)
		a.Logger.Printf("starting http listener at http://%s\n", listenAddress)
		a.Logger.Printf("swagger - http://%s/swagger/index.html\n", listenAddress)
		eg, appContext := errgroup.WithContext(appContext)
		eg.Go(func() error {
			defer a.Logger.Printf("http listener stopped")
			<-appContext.Done()

			shutdownCtx, cancel := context.WithTimeout(appContext, time.Minute)
			defer cancel()
			return a.echo.Shutdown(shutdownCtx)
		})
		eg.Go(func() error {
			return a.echo.Start(listenAddress)
		})
		return eg.Wait()
	}
}

// registerAPIHandler adds handler rpc into a.echo instance.
func (a *App) registerAPIHandler() {
	a.echo.GET("/swagger/*", echoSwagger.WrapHandler)

	//a.echo.GET("/api/check-partitions", checkPartitions)

	mbdeGroup := a.echo.Group("/api/mbde")

	// Маршруты Server
	mbdeGroup.GET("/parse-brands", a.mdServer.Brands)
	mbdeGroup.GET("/parse-models", a.mdServer.Models)
	mbdeGroup.GET("/parse-list-search", a.mdServer.ListSearch)
}
