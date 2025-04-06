package mobilede

import (
	"context"
	"net/http"

	"qnqa-auto-crawlers/pkg/db"
	"qnqa-auto-crawlers/pkg/logger"
	"qnqa-auto-crawlers/pkg/rabbitmq"

	"github.com/labstack/echo/v4"
)

type Server struct {
	logger  logger.Logger
	dbc     *db.DB
	crawler *Crawler
}

// New создает новый обработчик API
func New(logger logger.Logger, dbc *db.DB, repo *db.MobileDeRepo, rmq *rabbitmq.Client) *Server {
	return &Server{
		logger:  logger,
		dbc:     dbc,
		crawler: NewCrawler(logger, repo, rmq),
	}
}

// Brands обрабатывает запрос на парсинг брендов
// @Summary Parse brands from Server
// @Description Start parsing brands from Server
// @Tags Server
// @Accept json
// @Produce json
// @Success 200 {object} Response
// @Router /api/mbde/parse-brands [get]
func (h *Server) Brands(c echo.Context) error {
	err := h.crawler.BrandParse(context.Background())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Brand parsing started successfully",
	})
}

// Models обрабатывает запрос на парсинг моделей
// @Summary Parse models from Server
// @Description Start parsing models from Server
// @Tags Server
// @Accept json
// @Produce json
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /api/mbde/parse-models [get]
func (h *Server) Models(c echo.Context) error {
	// Проверяем наличие брендов
	var data struct {
		Count int `pg:"count"`
	}
	if _, err := h.dbc.QueryOne(&data, "SELECT COUNT(*) FROM brands"); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: err.Error(),
		})
	}

	if data.Count == 0 {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "No brands found. Please parse brands first.",
		})
	}

	if err := h.crawler.ModelParse(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Model parsing started successfully",
	})
}

// ListSearch обрабатывает запрос на парсинг страниц авто
// @Summary Parse brands from Server
// @Description Start parsing brands from Server
// @Tags Server
// @Accept json
// @Produce json
// @Success 200 {object} Response
// @Router /api/mbde/parse-list-search [get]
func (h *Server) ListSearch(c echo.Context) error {
	err := h.crawler.ListSearch(context.Background())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "ListSearch parsing started successfully",
	})
}
