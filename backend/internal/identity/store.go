package identity

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	Create(ctx context.Context, id *Identity) error
	Get(ctx context.Context, id uuid.UUID) (*Identity, error)
	List(ctx context.Context) ([]Identity, error)
	Update(ctx context.Context, id *Identity) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type MemoryStore struct {
	mu   sync.RWMutex
	rows map[uuid.UUID]Identity
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[uuid.UUID]Identity{}}
}

func (s *MemoryStore) Create(_ context.Context, id *Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id.ID == uuid.Nil {
		id.ID = uuid.New()
	}
	now := time.Now().UTC()
	id.CreatedAt = now
	id.UpdatedAt = now
	s.rows[id.ID] = *id
	return nil
}

func (s *MemoryStore) Get(_ context.Context, id uuid.UUID) (*Identity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	row, ok := s.rows[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &row, nil
}

func (s *MemoryStore) List(_ context.Context) ([]Identity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Identity, 0, len(s.rows))
	for _, r := range s.rows {
		out = append(out, r)
	}
	return out, nil
}

func (s *MemoryStore) Update(_ context.Context, id *Identity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.rows[id.ID]
	if !ok {
		return ErrNotFound
	}
	id.CreatedAt = existing.CreatedAt
	id.UpdatedAt = time.Now().UTC()
	s.rows[id.ID] = *id
	return nil
}

func (s *MemoryStore) Delete(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rows[id]; !ok {
		return ErrNotFound
	}
	delete(s.rows, id)
	return nil
}
