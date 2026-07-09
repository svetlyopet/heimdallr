package server

import (
	"context"
	"errors"

	"github.com/samber/do/v2"
	"github.com/svetlyopet/heimdallr/internal/server/api"
	"gorm.io/gorm"
)

type lookupService struct {
	repository Repository
}

func (l lookupService) GetById(ctx context.Context, serverID string) (api.Server, error) {
	server, err := l.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Server{}, ErrServerNotFound
		}

		return api.Server{}, ErrGetServer
	}

	return mapEntityToResponse(server), nil
}

func provideLookupService(i do.Injector) (LookupService, error) {
	return lookupService{repository: do.MustInvoke[Repository](i)}, nil
}
