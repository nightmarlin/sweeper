package sweeper

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type GameMutator func(ctx context.Context, g *Game) error

type Store interface {
	SaveGame(ctx context.Context, game *Game) error
	GetGame(ctx context.Context, gameID uuid.UUID) (*Game, error)
	MutateGame(
		ctx context.Context,
		gameID uuid.UUID,
		mut GameMutator,
	) (*Game, error)
}

type Service struct {
	store     Store
	idGen     IDGenerator
	numberGen NumberGenerator
}

func NewService(store Store, idGen IDGenerator, numberGen NumberGenerator) Service {
	return Service{
		store:     store,
		idGen:     idGen,
		numberGen: numberGen,
	}
}

func (s Service) StartGame(ctx context.Context, board Board) (*Game, error) {
	g, err := NewGame(ctx, s.idGen, s.numberGen, board)
	if err != nil {
		return nil, fmt.Errorf("creating game: %w", err)
	}
	if err := s.store.SaveGame(ctx, g); err != nil {
		return nil, fmt.Errorf("saving game: %w", err)
	}
	return g, nil
}

func (s Service) GetGame(ctx context.Context, gameID uuid.UUID) (*Game, error) {
	return s.store.GetGame(ctx, gameID)
}

func (s Service) EndGame(ctx context.Context, gameID uuid.UUID) (*Game, error) {
	return s.store.MutateGame(
		ctx,
		gameID,
		func(ctx context.Context, g *Game) error { return g.End() },
	)
}

func (s Service) MakeMove(
	ctx context.Context,
	gameID uuid.UUID,
	ref CellRef,
	toState CellState,
) (*Game, error) {
	return s.store.MutateGame(
		ctx,
		gameID,
		func(ctx context.Context, g *Game) error {
			return g.UpdateCell(ref, toState)
		},
	)
}
