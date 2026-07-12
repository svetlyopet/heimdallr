package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

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

	handler.Use(middleware.RequestLimits(middleware.RequestLimitsConfig{
		MaxRequestBodyBytes:   cfg.Server.MaxRequestBodyBytes,
		MaxDecodedOutputBytes: cfg.Server.MaxDecodedOutputBytes,
		MaxPaginationLimit:    cfg.Server.MaxPaginationLimit,
	}))

	api := handler.Group("/api")
	application.RegisterPublicRoutes(api)
	api.Use(middleware.Authentication(application.TokenService(), middleware.AuthenticationConfig{
		SessionCookieName: cfg.Auth.SessionCookieName,
		CSRFCookieName:    cfg.Auth.CSRFCookieName,
		OptionalPaths:     []string{"/api/v1/auth/logout"},
	}))
	application.RegisterProtectedAuthRoutes(api)
	application.RegisterRoutes(api)

	if err = web.RegisterRoutes(handler); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	httpServer := newHTTPServer(addr, handler, cfg.Server)

	return &Server{
		host:       cfg.Server.Host,
		addr:       addr,
		logger:     appLogger,
		handler:    handler,
		httpServer: httpServer,
	}, nil
}

type Server struct {
	host       string
	addr       string
	logger     *logger.Logger
	handler    *gin.Engine
	httpServer *http.Server
}

func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) HTTPHandler() http.Handler {
	return s.handler
}

func (s *Server) HTTPServer() *http.Server {
	return s.httpServer
}

func newHTTPServer(addr string, handler http.Handler, cfg config.ServerConfig) *http.Server {
	if cfg.ReadHeaderTimeout <= 0 {
		cfg.ReadHeaderTimeout = 5 * time.Second
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 15 * time.Second
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 30 * time.Second
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = 60 * time.Second
	}
	if cfg.MaxHeaderBytes <= 0 {
		cfg.MaxHeaderBytes = 1 << 20
	}

	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}
}
