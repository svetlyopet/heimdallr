package server

import (
	"context"
	"errors"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type lookupService struct {
	repository Repository
}

func (l lookupService) GetById(ctx context.Context, serverID string) (GetResponse, error) {
	server, err := l.repository.FindById(ctx, serverID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrServerNotFound
		}

		return GetResponse{}, ErrGetServer
	}

	return mapEntityToResponse(server), nil
}

func provideLookupService(i do.Injector) (LookupService, error) {
	return lookupService{repository: do.MustInvoke[Repository](i)}, nil
}
