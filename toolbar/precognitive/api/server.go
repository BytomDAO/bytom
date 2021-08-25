package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"

	"github.com/bytom/bytom/toolbar/precognitive/config"
	serverCommon "github.com/bytom/bytom/toolbar/server"
)

type Server struct {
	cfg    *config.Config
	db     *gorm.DB
	engine *gin.Engine
}

func NewApiServer(cfg *config.Config, db *gorm.DB) *Server {
	server := &Server{
		cfg: cfg,
		db:  db,
	}
	if cfg.API.IsReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}
	server.setupRouter()
	return server
}

func (s *Server) setupRouter() {
	r := gin.Default()
	r.Use(serverCommon.Middleware(s))

	v1 := r.Group("/api/v1")
	v1.POST("/list-nodes", serverCommon.HandlerMiddleware(s.ListNodes))

	s.engine = r
}

func (s *Server) Run() {
	s.engine.Run(fmt.Sprintf(":%d", s.cfg.API.ListeningPort))
}
