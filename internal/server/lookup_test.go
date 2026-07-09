package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestLookupServiceGetByIdReturnsServer(t *testing.T) {
	db := newServerTestDB(t)
	repo := NewRepository(db)
	lookup := lookupService{repository: repo}

	created, err := repo.Create(context.Background(), Server{
		ID:       uuid.New(),
		Hostname: "lookup-host.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	got, err := lookup.GetById(context.Background(), created.ID.String())
	require.NoError(t, err)
	require.Equal(t, created.ID, got.Id)
	require.Equal(t, "lookup-host.example.com", got.Hostname)
}

func TestLookupServiceGetByIdReturnsNotFound(t *testing.T) {
	db := newServerTestDB(t)
	lookup := lookupService{repository: NewRepository(db)}

	_, err := lookup.GetById(context.Background(), uuid.New().String())
	require.ErrorIs(t, err, ErrServerNotFound)
}
