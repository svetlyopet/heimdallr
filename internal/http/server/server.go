package server

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/app"
	"github.com/svetlyopet/heimdallr/internal/http/middleware"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/web"
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

	application, err := app.New(db, appLogger)
	if err != nil {
		return nil, err
	}

	if err = application.Bootstrap(context.Background()); err != nil {
		return nil, err
	}

	api := handler.Group("/api")
	api.Use(middleware.Authentication(application.AuthService(), application.TokenService()))
	application.RegisterRoutes(api)

	if err = web.RegisterRoutes(handler); err != nil {
		return nil, err
	}

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
