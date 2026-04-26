package identity

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const Schema = `
CREATE TABLE IF NOT EXISTS digital_identity (
  id          UUID PRIMARY KEY,
  full_name   TEXT NOT NULL,
  phone       TEXT NOT NULL DEFAULT '',
  address     TEXT NOT NULL DEFAULT '',
  email       TEXT NOT NULL DEFAULT '',
  passport_no TEXT NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

type PgxStore struct {
	pool *pgxpool.Pool
}

func NewPgxStore(pool *pgxpool.Pool) *PgxStore {
	return &PgxStore{pool: pool}
}

func (s *PgxStore) Create(ctx context.Context, id *Identity) error {
	if id.ID == uuid.Nil {
		id.ID = uuid.New()
	}
	now := time.Now().UTC()
	id.CreatedAt = now
	id.UpdatedAt = now
	_, err := s.pool.Exec(ctx, `
		INSERT INTO digital_identity
		  (id, full_name, phone, address, email, passport_no, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id.ID, id.FullName, id.Phone, id.Address, id.Email, id.PassportNo, id.CreatedAt, id.UpdatedAt,
	)
	return err
}

func (s *PgxStore) Get(ctx context.Context, id uuid.UUID) (*Identity, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, full_name, phone, address, email, passport_no, created_at, updated_at
		FROM digital_identity WHERE id = $1`, id)
	out := &Identity{}
	err := row.Scan(&out.ID, &out.FullName, &out.Phone, &out.Address, &out.Email, &out.PassportNo, &out.CreatedAt, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *PgxStore) List(ctx context.Context) ([]Identity, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, full_name, phone, address, email, passport_no, created_at, updated_at
		FROM digital_identity ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Identity, 0)
	for rows.Next() {
		var i Identity
		if err := rows.Scan(&i.ID, &i.FullName, &i.Phone, &i.Address, &i.Email, &i.PassportNo, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *PgxStore) Update(ctx context.Context, id *Identity) error {
	id.UpdatedAt = time.Now().UTC()
	tag, err := s.pool.Exec(ctx, `
		UPDATE digital_identity
		   SET full_name=$2, phone=$3, address=$4, email=$5, passport_no=$6, updated_at=$7
		 WHERE id=$1`,
		id.ID, id.FullName, id.Phone, id.Address, id.Email, id.PassportNo, id.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PgxStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM digital_identity WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
