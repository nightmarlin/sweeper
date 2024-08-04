package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/nightmarlin/sweeper"
)

type Store struct {
	mux sync.RWMutex
	s   map[uuid.UUID]sweeper.Game
}

func NewStore() *Store {
	return &Store{s: make(map[uuid.UUID]sweeper.Game)}
}

func (s *Store) SaveGame(_ context.Context, g *sweeper.Game) error {
	defer s.mux.Unlock()
	s.mux.Lock()

	s.s[g.ID] = *g
	return nil
}

func (s *Store) getGame(gameID uuid.UUID) (*sweeper.Game, error) {
	g, ok := s.s[gameID]
	if !ok {
		return nil, sweeper.ErrGameNotFound
	}
	return &g, nil
}

func (s *Store) GetGame(_ context.Context, gameID uuid.UUID) (*sweeper.Game, error) {
	defer s.mux.RUnlock()
	s.mux.RLock()
	return s.getGame(gameID)
}

func (s *Store) MutateGame(
	ctx context.Context,
	gameID uuid.UUID,
	mut sweeper.GameMutator,
) (*sweeper.Game, error) {
	defer s.mux.Unlock()
	s.mux.Lock()

	g, err := s.getGame(gameID)
	if err != nil {
		return nil, err
	}

	if err := mut(ctx, g); err != nil {
		return nil, fmt.Errorf("error in mutator: %w", err)
	}

	s.s[gameID] = *g
	return g, nil
}
