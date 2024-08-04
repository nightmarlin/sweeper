package main

import (
	"context"
	"fmt"
	"io"
	"strconv"

	sweeperv1 "github.com/nightmarlin/sweeper/gen/sweeper/v1"
)

const (
	renderCellEmpty    = ' '
	renderCellFlagged  = '!'
	renderCellQuestion = '?'
	renderCellMine     = '╳'
	renderCellUnknown  = '¿'

	renderHeaderDividerVertical     = '│'
	renderHeaderDividerHorizontal   = '─'
	renderHeaderDividerJoint        = '┼'
	renderStandardDividerVertical   = '╎'
	renderStandardDividerHorizontal = '╌'
	renderStandardDividerJoint      = '•'
)

func runeFromInt(i int32) rune {
	if 0 <= i && i <= 9 {
		return '0' + i
	}
	return renderCellUnknown
}

func renderGame(
	ctx context.Context,
	w io.Writer,
	g *sweeperv1.Game,
) error {
	cells := make([][]rune, g.Board.Height)
	for i := range cells {
		cells[i] = make([]rune, g.Board.Width)
	}

	// accumulate cells into slice for render
	var flaggedCells int

	for _, c := range g.Cells {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		r := renderCellUnknown

		switch s := c.State.(type) {
		case *sweeperv1.Cell_Unrevealed:
			r = renderCellEmpty

		case *sweeperv1.Cell_Questioned:
			r = renderCellQuestion

		case *sweeperv1.Cell_Flagged:
			r = renderCellFlagged
			flaggedCells++

		case *sweeperv1.Cell_Revealed:
			switch v := s.Revealed.Value.(type) {
			case *sweeperv1.RevealedCell_Clear:
				r = runeFromInt(v.Clear.NeighbouringMines)

			case *sweeperv1.RevealedCell_Mine:
				r = renderCellMine
			}
		}

		cells[c.Row][c.Column] = r
	}

	// render game header
	if _, err := fmt.Fprintf(
		w, "Game '%s'\n%s\t%d/%d\n",
		g.Id, gameStateToString(g.State), flaggedCells, g.Board.Mines,
	); err != nil {
		return err
	}

	// render column titles
	rowNameWidth := len(strconv.Itoa(int(g.Board.Height) + 1))
	colWidth := len(strconv.Itoa(int(g.Board.Width) + 1))
	if _, err := fmt.Fprintf(w, ` %s`, pad(" ", rowNameWidth)); err != nil {
		return err
	}
	for col := range g.Board.Width {
		if _, err := fmt.Fprintf(
			w, " %c %s",
			renderHeaderDividerVertical, pad(strconv.Itoa(int(col)+1), colWidth),
		); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	// render slice to board
	for rowNum, row := range cells {
		// print header for row number
		if _, err := fmt.Fprintf(w, printN(renderHeaderDividerHorizontal, rowNameWidth+2)); err != nil {
			return err
		}

		// if first row, render solid top dividers, else render dotted top dividers
		var (
			horizDiv = renderStandardDividerHorizontal
			join     = renderStandardDividerJoint
		)
		if rowNum == 0 {
			horizDiv = renderHeaderDividerHorizontal
			join = renderHeaderDividerJoint
		}
		for colNum := range g.Board.Width {
			join := join
			if colNum == 0 {
				join = renderHeaderDividerJoint
			}

			if _, err := fmt.Fprintf(w, "%c%s", join, printN(horizDiv, colWidth+2)); err != nil {
				return err
			}
		}

		// render row number
		if _, err := fmt.Fprintf(w, "\n %s", pad(strconv.Itoa(rowNum+1), rowNameWidth)); err != nil {
			return err
		}

		// render cells
		for colNum, cell := range row {
			// if first column render solid vertical divider, else dotted
			div := renderStandardDividerVertical
			if colNum == 0 {
				div = renderHeaderDividerVertical
			}

			if _, err := fmt.Fprintf(
				w, ` %c %s`,
				div, pad(fmt.Sprintf("%c", cell), colWidth),
			); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func gameStateToString(s sweeperv1.GameState) string {
	switch s {
	case sweeperv1.GameState_ONGOING:
		return "Ongoing."
	case sweeperv1.GameState_WON:
		return "You won!"
	case sweeperv1.GameState_LOST:
		return "You lost."
	case sweeperv1.GameState_RESIGNED:
		return "You resigned."
	default:
		return "Unknown State"
	}
}

func printN(c rune, n int) (s string) {
	for range n {
		s = fmt.Sprintf("%s%c", s, c)
	}
	return
}

func pad(s string, l int) string {
	for len(s) < l {
		s = fmt.Sprintf(" %s", s)
	}
	return s

}
