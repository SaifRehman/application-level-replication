package identity

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMemoryStoreCRUD(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	id := Identity{FullName: "Ada Lovelace", Email: "ada@example.com"}
	require.NoError(t, s.Create(ctx, &id))
	require.NotEqual(t, uuid.Nil, id.ID)
	require.False(t, id.CreatedAt.IsZero())
	require.False(t, id.UpdatedAt.IsZero())

	got, err := s.Get(ctx, id.ID)
	require.NoError(t, err)
	require.Equal(t, "Ada Lovelace", got.FullName)
	require.Equal(t, "ada@example.com", got.Email)

	all, err := s.List(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)

	id.FullName = "Ada Byron"
	require.NoError(t, s.Update(ctx, &id))

	got, err = s.Get(ctx, id.ID)
	require.NoError(t, err)
	require.Equal(t, "Ada Byron", got.FullName)
	require.True(t, got.UpdatedAt.After(got.CreatedAt) || got.UpdatedAt.Equal(got.CreatedAt))

	require.NoError(t, s.Delete(ctx, id.ID))

	_, err = s.Get(ctx, id.ID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMemoryStoreGetMissing(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Get(context.Background(), uuid.New())
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMemoryStoreUpdateMissing(t *testing.T) {
	s := NewMemoryStore()
	err := s.Update(context.Background(), &Identity{ID: uuid.New(), FullName: "x"})
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMemoryStoreDeleteMissing(t *testing.T) {
	s := NewMemoryStore()
	err := s.Delete(context.Background(), uuid.New())
	require.ErrorIs(t, err, ErrNotFound)
}
