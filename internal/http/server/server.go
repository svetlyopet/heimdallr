package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/app"
	"github.com/svetlyopet/heimdallr/internal/constants"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"gorm.io/gorm"
)

type Server struct {
	host    string
	addr    string
	db      *gorm.DB
	logger  *logger.Logger
	handler *gin.Engine
}

func NewServer(host string, port string, db *gorm.DB, appLogger *logger.Logger) (*Server, error) {
	handler, err := NewHandler(host, appLogger)
	if err != nil {
		return nil, err
	}

	// Serve frontend static files
	handler.Use(static.Serve("/", static.LocalFile(constants.WebPublicPath, false)))

	// SPA fallback for browser routes, while keeping API routes separate.
	handler.NoRoute(func(ctx *gin.Context) {
		if strings.HasPrefix(ctx.Request.URL.Path, "/api") {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": http.StatusText(http.StatusNotFound),
			})
			return
		}

		ctx.File(constants.WebPublicPath + "/index.html")
	})

	application, err := app.New(db, appLogger)
	if err != nil {
		return nil, err
	}

	api := handler.Group("/api")
	application.RegisterRoutes(api)

	addr := fmt.Sprintf(":%s", port)

	return &Server{
		host:    host,
		addr:    addr,
		db:      db,
		logger:  appLogger,
		handler: handler,
	}, nil
}

func (s *Server) Run() error {
	return s.handler.Run(s.addr)
}
