package middleware

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/requestlimits"
)

const (
	DefaultMaxRequestBodyBytes int64 = 5 << 20
	DefaultMaxPaginationLimit        = 100
)

type RequestLimitsConfig struct {
	MaxRequestBodyBytes   int64
	MaxDecodedOutputBytes int64
	MaxPaginationLimit    int
}

func RequestLimits(cfg RequestLimitsConfig) gin.HandlerFunc {
	cfg = normalizeRequestLimits(cfg)

	return func(ctx *gin.Context) {
		if err := validatePaginationQuery(ctx.Request, cfg.MaxPaginationLimit); err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, cfg.MaxRequestBodyBytes)
		requestCtx := requestlimits.WithContext(ctx.Request.Context(), requestlimits.Values{
			MaxDecodedOutputBytes: cfg.MaxDecodedOutputBytes,
		})
		ctx.Request = ctx.Request.WithContext(requestCtx)
		ctx.Next()
	}
}

func normalizeRequestLimits(cfg RequestLimitsConfig) RequestLimitsConfig {
	if cfg.MaxRequestBodyBytes <= 0 {
		cfg.MaxRequestBodyBytes = DefaultMaxRequestBodyBytes
	}
	if cfg.MaxDecodedOutputBytes <= 0 {
		cfg.MaxDecodedOutputBytes = requestlimits.DefaultMaxDecodedOutputBytes
	}
	if cfg.MaxPaginationLimit <= 0 {
		cfg.MaxPaginationLimit = DefaultMaxPaginationLimit
	}

	return cfg
}

func validatePaginationQuery(req *http.Request, maxLimit int) error {
	page, pageSet, err := positiveQueryInt(req, "page")
	if err != nil {
		return err
	}
	limit, limitSet, err := positiveQueryInt(req, "limit")
	if err != nil {
		return err
	}

	if !limitSet {
		limit = 10
	}
	if limit > maxLimit {
		return errors.New("limit exceeds maximum")
	}
	if !pageSet {
		page = 1
	}
	if page-1 > math.MaxInt/limit {
		return errors.New("page and limit are too large")
	}

	return nil
}

func positiveQueryInt(req *http.Request, name string) (int, bool, error) {
	value := strings.TrimSpace(req.URL.Query().Get(name))
	if value == "" {
		return 0, false, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, true, errors.New("page and limit must be positive integers")
	}

	return parsed, true, nil
}
