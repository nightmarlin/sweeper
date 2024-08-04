package sweeper

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type GameState int

const (
	GameOngoing  = GameState(iota) // The Game is still in progress.
	GameWon                        // The player won the Game.
	GameLost                       // The player lost the Game.
	GameResigned                   // The player chose to end the Game.
)

type Board struct {
	Width, Height int
	Mines         int
}

// safeCells returns the number of Cells that don't contain a mine.
func (b Board) safeCells() int {
	return (b.Width * b.Height) - b.Mines
}

type CellState int

const (
	CellDefault    = CellState(iota) // No modifiers.
	CellFlagged                      // Flagged as containing a mine and cannot be opened.
	CellQuestioned                   // Marked as potentially containing a mine, but can still be opened.
	CellRevealed                     // Revealed by the player.
)

type CellRef struct {
	Row, Column int
}

type Cell struct {
	ContainsMine      bool
	NeighbouringMines int
	State             CellState
}

type Game struct {
	ID    uuid.UUID
	State GameState
	Board Board
	Cells map[CellRef]Cell
}

// An IDGenerator generates globally unique IDs.
type IDGenerator func() uuid.UUID

// A NumberGenerator generates a number between 0 and N (exclusive).
//
//	ng(1) => 0
//	ng(2) => 0 | 1
//
// > This type is compatible with rand.IntN.
type NumberGenerator func(n int) int

func NewGame(
	ctx context.Context,
	idGen IDGenerator,
	numberGen NumberGenerator,
	board Board,
) (*Game, error) {
	boardSize := board.Height * board.Width
	switch {
	case board.Height <= 0, board.Width <= 0:
		return nil, fmt.Errorf(
			"%w: invalid board dimensions (%dx%d)",
			ErrOutOfBounds, board.Height, board.Width,
		)
	case board.Mines <= 0:
		return nil, fmt.Errorf("%w: board must have at least 1 mine", ErrOutOfBounds)
	case board.Mines >= boardSize:
		return nil, fmt.Errorf("%w: board must have at least 1 free space", ErrOutOfBounds)
	}

	g := Game{
		ID:    idGen(),
		State: GameOngoing,
		Board: board,
		Cells: make(map[CellRef]Cell, board.Height*board.Width),
	}

	for row := range board.Height {
		for col := range board.Width {
			g.Cells[CellRef{Row: row, Column: col}] = Cell{}
		}
	}

	for range board.Mines {
		for {
			// this may take a while due to RNG so allow it to be cancelled.
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			ref := CellRef{Row: numberGen(board.Height), Column: numberGen(board.Width)}

			c := g.Cells[ref]
			if c.ContainsMine {
				// don't add a mine if one already present
				continue
			}
			c.ContainsMine = true
			g.Cells[ref] = c
			break
		}
	}

	for cr, c := range g.Cells {
		c.NeighbouringMines = g.countNeighbouringMines(cr)
		g.Cells[cr] = c
	}

	for {
		ref := CellRef{Row: numberGen(board.Height), Column: numberGen(board.Width)}
		if g.Cells[ref].ContainsMine {
			continue
		}
		g.revealCell(ref)
		break
	}

	return &g, nil
}

// countNeighbouringMines counts the number of mines in the cells surrounding
// the Cell at CellRef.
func (g *Game) countNeighbouringMines(ref CellRef) (n int) {
	for row := -1; row <= 1; row++ {
		for col := -1; col <= 1; col++ {
			if row == 0 && col == 0 {
				continue
			}

			lookup := CellRef{Row: ref.Row + row, Column: ref.Column + col}
			if g.Cells[lookup].ContainsMine {
				n++
			}
		}
	}

	return n
}

func (g *Game) checkBounds(ref CellRef) bool {
	return 0 <= ref.Row && ref.Row < g.Board.Height &&
		0 <= ref.Column && ref.Column < g.Board.Width
}

func (g *Game) finished() bool { return g.State != GameOngoing }

func (g *Game) tryWin() {
	// victory is obtained by revealing all non-mine squares or flagging all mines.
	if g.finished() {
		return
	}

	var (
		cellsFlaggedTotal   int
		cellsFlaggedCorrect int

		cellsRevealed int
	)

	for _, c := range g.Cells {
		if c.State == CellFlagged {
			cellsFlaggedTotal++
			if c.ContainsMine {
				cellsFlaggedCorrect++
			}
		}
		if c.State == CellRevealed {
			cellsRevealed++
		}
	}

	if cellsFlaggedTotal == cellsFlaggedCorrect &&
		cellsFlaggedCorrect == g.Board.Mines {
		g.State = GameWon

	} else if cellsRevealed == g.Board.safeCells() {
		g.State = GameWon
	}
}

// reveals the Cell. if Cell is flagged, block. if Cell contains a mine, lose.
// if Cell has no neighbouring mines, reveal all neighbours.
func (g *Game) revealCell(ref CellRef) {
	c := g.Cells[ref]
	if c.State == CellRevealed {
		return
	}

	c.State = CellRevealed
	if c.ContainsMine {
		g.State = GameLost
		return
	}
	g.Cells[ref] = c

	// reveal neighbours if empty
	if c.NeighbouringMines == 0 {
		for r := -1; r <= 1; r++ {
			for col := -1; col <= 1; col++ {
				cr := CellRef{Row: ref.Row + r, Column: ref.Column + col}
				if c, ok := g.Cells[cr]; ok && c.State != CellRevealed {
					g.revealCell(cr) // recursive reveal
				}
			}
		}
	}
}

// UpdateCell tries to update the CellState of the Cell at CellRef to the
// provided one, returning a descriptive error if the operation fails.
//
// If the game is won or lost after this, Game.State will be set to GameWon or
// GameLost accordingly.
func (g *Game) UpdateCell(ref CellRef, s CellState) error {
	if g.finished() {
		return ErrGameFinished
	}

	if !g.checkBounds(ref) {
		return ErrOutOfBounds
	}

	switch s {
	case CellDefault, CellFlagged, CellQuestioned:
		// set cell to have new marking. if revealed, block.
		c := g.Cells[ref]
		if c.State == CellRevealed {
			return ErrRevealed
		}
		c.State = s
		g.Cells[ref] = c

	case CellRevealed:
		if g.Cells[ref].State == CellFlagged {
			return ErrFlagged
		}
		g.revealCell(ref)

	default:
		return fmt.Errorf("unknown action")
	}

	g.tryWin()
	return nil
}

func (g *Game) End() error {
	if g.finished() {
		return ErrGameFinished
	}
	g.State = GameResigned
	return nil
}

var (
	ErrGameNotFound = fmt.Errorf("game not found")
	ErrRevealed     = fmt.Errorf("cell is already revealed")
	ErrFlagged      = fmt.Errorf("cell is flagged")
	ErrOutOfBounds  = fmt.Errorf("selection is out of bounds")
	ErrGameFinished = fmt.Errorf("game is finished")
)
