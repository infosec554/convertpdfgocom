package api

import (
	"github.com/gin-gonic/gin"

	"convertpdfgo/api/handlers"
	"convertpdfgo/config"
	"convertpdfgo/pkg/logger"
	"convertpdfgo/service"
)

type Server struct {
	router *gin.Engine
	cfg    *config.Config
	log    logger.ILogger
}

func New(cfg *config.Config, log logger.ILogger, services service.IServiceManager) *Server {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")

	handler := handlers.New(services, cfg.AdminUserID)

	router.GET("/stats", handler.GetStatsPage)
	router.GET("/api/stats", handler.GetStatsAPI)

	return &Server{
		router: router,
		cfg:    cfg,
		log:    log,
	}
}

func (s *Server) Run() error {
	return s.router.Run(s.cfg.AppPort)
}
