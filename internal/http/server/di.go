package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/svetlyopet/heimdallr/internal/app"
	"github.com/svetlyopet/heimdallr/internal/config"
	"github.com/svetlyopet/heimdallr/internal/http/middleware"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/web"
)

var Package = do.Package(
	app.Package,
	do.Lazy(provideServer),
)

func provideServer(i do.Injector) (*Server, error) {
	cfg := do.MustInvoke[*config.AppConfig](i)
	appLogger := do.MustInvoke[*logger.Logger](i)
	application := do.MustInvoke[*app.App](i)

	handler, err := NewHandler(cfg.Server.Host, appLogger)
	if err != nil {
		return nil, err
	}

	if err = application.Bootstrap(context.Background()); err != nil {
		return nil, err
	}

	api := handler.Group("/api")
	application.RegisterPublicRoutes(api)
	api.Use(middleware.Authentication(application.TokenService()))
	application.RegisterProtectedAuthRoutes(api)
	application.RegisterRoutes(api)

	if err = web.RegisterRoutes(handler); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf(":%s", cfg.Server.Port)

	return &Server{
		host:    cfg.Server.Host,
		addr:    addr,
		logger:  appLogger,
		handler: handler,
	}, nil
}

type Server struct {
	host    string
	addr    string
	logger  *logger.Logger
	handler *gin.Engine
}

func (s *Server) Run() error {
	return s.handler.Run(s.addr)
}

func (s *Server) HTTPHandler() http.Handler {
	return s.handler
}
