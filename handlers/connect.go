package handlers

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/nightmarlin/sweeper"
	sweeperv1 "github.com/nightmarlin/sweeper/gen/sweeper/v1"
	"github.com/nightmarlin/sweeper/gen/sweeper/v1/sweeperv1connect"
)

type Connect struct {
	sweeperv1connect.UnimplementedSweeperServiceHandler

	svc sweeper.Service
}

func NewConnect(svc sweeper.Service) Connect {
	return Connect{svc: svc}
}

func (h Connect) StartGame(
	ctx context.Context,
	req *connect.Request[sweeperv1.StartGameRequest],
) (*connect.Response[sweeperv1.StartGameResponse], error) {
	g, err := h.svc.StartGame(
		ctx,
		sweeper.Board{
			Width:  int(req.Msg.Board.Width),
			Height: int(req.Msg.Board.Height),
			Mines:  int(req.Msg.Board.Mines),
		},
	)
	if err != nil {
		return nil, mapErr(err)
	}
	return &connect.Response[sweeperv1.StartGameResponse]{
		Msg: &sweeperv1.StartGameResponse{Game: sweeperv1.InternalGameToGame(g)},
	}, nil
}

func (h Connect) GetGame(
	ctx context.Context,
	req *connect.Request[sweeperv1.GetGameRequest],
) (*connect.Response[sweeperv1.GetGameResponse], error) {
	id, err := parseUUID(req.Msg.GameId)
	if err != nil {
		return nil, err
	}

	g, err := h.svc.GetGame(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}

	return &connect.Response[sweeperv1.GetGameResponse]{
		Msg: &sweeperv1.GetGameResponse{Game: sweeperv1.InternalGameToGame(g)},
	}, nil
}

func (h Connect) MakeMove(
	ctx context.Context,
	req *connect.Request[sweeperv1.MakeMoveRequest],
) (*connect.Response[sweeperv1.MakeMoveResponse], error) {
	id, err := parseUUID(req.Msg.GameId)
	if err != nil {
		return nil, err
	}

	var g *sweeper.Game

	switch m := req.Msg.Move.(type) {
	case *sweeperv1.MakeMoveRequest_End:
		g, err = h.svc.EndGame(ctx, id)

	case *sweeperv1.MakeMoveRequest_Cell:
		g, err = h.svc.MakeMove(
			ctx,
			id,
			sweeper.CellRef{Row: int(m.Cell.Row), Column: int(m.Cell.Column)},
			sweeperv1.CellMoveActionToInternalCellState(m.Cell.Action),
		)

	default:
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			fmt.Errorf("unknown move type: %T", m),
		)
	}
	if err != nil {
		return nil, mapErr(err)
	}

	return &connect.Response[sweeperv1.MakeMoveResponse]{
		Msg: &sweeperv1.MakeMoveResponse{Game: sweeperv1.InternalGameToGame(g)},
	}, nil
}

func parseUUID(id string) (uuid.UUID, error) {
	res, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return res, nil
}

var knownErrs = map[error]connect.Code{
	sweeper.ErrGameNotFound: connect.CodeNotFound,
	sweeper.ErrRevealed:     connect.CodeAlreadyExists,
	sweeper.ErrFlagged:      connect.CodeFailedPrecondition,
	sweeper.ErrOutOfBounds:  connect.CodeOutOfRange,
	sweeper.ErrGameFinished: connect.CodeFailedPrecondition,
}

func mapErr(err error) *connect.Error {
	code := connect.CodeInternal
	for ke, c := range knownErrs {
		if errors.Is(err, ke) {
			code = c
			break
		}
	}

	return connect.NewError(code, err)
}
