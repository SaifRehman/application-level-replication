//go:build integration

package identity

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func newPgxStoreForTest(t *testing.T) *PgxStore {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	_, err = pool.Exec(context.Background(), Schema)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(), "TRUNCATE digital_identity")
	require.NoError(t, err)

	return NewPgxStore(pool)
}

func TestPgxStoreCRUD(t *testing.T) {
	ctx := context.Background()
	s := newPgxStoreForTest(t)

	in := Identity{FullName: "Ada", Email: "ada@example.com"}
	require.NoError(t, s.Create(ctx, &in))
	require.NotEqual(t, uuid.Nil, in.ID)

	got, err := s.Get(ctx, in.ID)
	require.NoError(t, err)
	require.Equal(t, "Ada", got.FullName)

	all, err := s.List(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)

	in.FullName = "Ada Byron"
	require.NoError(t, s.Update(ctx, &in))

	got, err = s.Get(ctx, in.ID)
	require.NoError(t, err)
	require.Equal(t, "Ada Byron", got.FullName)

	require.NoError(t, s.Delete(ctx, in.ID))
	_, err = s.Get(ctx, in.ID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgxStoreGetMissing(t *testing.T) {
	s := newPgxStoreForTest(t)
	_, err := s.Get(context.Background(), uuid.New())
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgxStoreUpdateMissing(t *testing.T) {
	s := newPgxStoreForTest(t)
	err := s.Update(context.Background(), &Identity{ID: uuid.New(), FullName: "x"})
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgxStoreDeleteMissing(t *testing.T) {
	s := newPgxStoreForTest(t)
	err := s.Delete(context.Background(), uuid.New())
	require.ErrorIs(t, err, ErrNotFound)
}
