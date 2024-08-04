package sweeperv1

import (
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/nightmarlin/sweeper"
)

func InternalGameToGame(g *sweeper.Game) *Game {
	cells := make([]*Cell, 0, len(g.Cells))
	for ref, cell := range g.Cells {
		c := &Cell{Row: int32(ref.Row), Column: int32(ref.Column), State: nil}

		switch cell.State {
		case sweeper.CellDefault:
			c.State = &Cell_Unrevealed{Unrevealed: &emptypb.Empty{}}
		case sweeper.CellFlagged:
			c.State = &Cell_Flagged{Flagged: &emptypb.Empty{}}
		case sweeper.CellQuestioned:
			c.State = &Cell_Questioned{Questioned: &emptypb.Empty{}}
		case sweeper.CellRevealed:
			rc := &RevealedCell{}
			if cell.ContainsMine {
				rc.Value = &RevealedCell_Mine{Mine: &emptypb.Empty{}}
			} else {
				rc.Value = &RevealedCell_Clear{
					Clear: &ClearRevealedCell{NeighbouringMines: int32(cell.NeighbouringMines)},
				}
			}

			c.State = &Cell_Revealed{Revealed: rc}
		}

		cells = append(cells, c)
	}

	res := &Game{
		Id:    g.ID.String(),
		State: internalGameStateToGameState(g.State),
		Board: &Board{
			Height: int32(g.Board.Height),
			Width:  int32(g.Board.Width),
			Mines:  int32(g.Board.Mines),
		},
		Cells: cells,
	}

	return res
}

func internalGameStateToGameState(s sweeper.GameState) GameState {
	switch s {
	case sweeper.GameOngoing:
		return GameState_ONGOING
	case sweeper.GameWon:
		return GameState_WON
	case sweeper.GameLost:
		return GameState_LOST
	case sweeper.GameResigned:
		return GameState_RESIGNED
	default:
		return GameState_GAME_STATE_UNKNOWN
	}
}

func CellMoveActionToInternalCellState(m CellMoveAction) sweeper.CellState {
	switch m {
	case CellMoveAction_FLAG:
		return sweeper.CellFlagged
	case CellMoveAction_QUESTION:
		return sweeper.CellQuestioned
	case CellMoveAction_REVEAL:
		return sweeper.CellRevealed
	default:
		return sweeper.CellDefault
	}
}
