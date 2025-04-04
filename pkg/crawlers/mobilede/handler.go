package mobilede

import (
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
// @Router /api/mobilede/parse-brands [post]
func (h *Server) Brands(c echo.Context) error {
	if err := h.crawler.BrandParse(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to parse brands",
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
// @Router /api/mobilede/parse-models [post]
func (h *Server) Models(c echo.Context) error {
	// Проверяем наличие брендов
	var count int
	if _, err := h.dbc.QueryOne(&count, "SELECT COUNT(*) FROM brands"); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to check brands",
		})
	}

	if count == 0 {
		return c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Message: "No brands found. Please parse brands first.",
		})
	}

	if err := h.crawler.ModelParse(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Message: "Failed to parse models",
		})
	}

	return c.JSON(http.StatusOK, Response{
		Success: true,
		Message: "Model parsing started successfully",
	})
}

// Response представляет стандартный ответ API
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
