package token

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"gorm.io/gorm"
)

type authTokenRepositoryAdapter struct {
	repository Repository
}

func (a authTokenRepositoryAdapter) DeleteByCreatedBy(ctx context.Context, userID string) error {
	return a.repository.DeleteByCreatedBy(ctx, userID)
}

func (a authTokenRepositoryAdapter) DeleteSessionTokensByCreatedBy(ctx context.Context, userID string) error {
	return a.repository.DeleteSessionTokensByCreatedBy(ctx, userID)
}

func (a authTokenRepositoryAdapter) DeleteSessionTokensByCreatedByExceptCredential(ctx context.Context, userID string, excludePlainToken string) error {
	return a.repository.DeleteSessionTokensByCreatedByExceptHash(ctx, userID, hashToken(excludePlainToken))
}

func (a authTokenRepositoryAdapter) WithTx(tx *gorm.DB) auth.TokenRepository {
	return authTokenRepositoryAdapter{repository: a.repository.WithTx(tx)}
}

func provideAuthTokenRepository(i do.Injector) (auth.TokenRepository, error) {
	return authTokenRepositoryAdapter{repository: do.MustInvoke[Repository](i)}, nil
}

// NewAuthTokenRepository adapts token.Repository for auth service use.
func NewAuthTokenRepository(repository Repository) auth.TokenRepository {
	return authTokenRepositoryAdapter{repository: repository}
}
