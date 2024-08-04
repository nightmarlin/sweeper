package main

import (
	"context"
	"os"

	sweeperv1 "github.com/nightmarlin/sweeper/gen/sweeper/v1"
)

func Example_renderGame() {
	var g = &sweeperv1.Game{
		Id:    "0_example_game",
		State: sweeperv1.GameState_LOST,
		Board: &sweeperv1.Board{Height: 3, Width: 3, Mines: 2},
		Cells: []*sweeperv1.Cell{
			{
				Row:    0,
				Column: 0,
				State: &sweeperv1.Cell_Revealed{
					Revealed: &sweeperv1.RevealedCell{
						Value: &sweeperv1.RevealedCell_Clear{
							Clear: &sweeperv1.ClearRevealedCell{NeighbouringMines: 0},
						},
					},
				},
			},
			{Row: 0, Column: 1, State: &sweeperv1.Cell_Unrevealed{}},
			{Row: 0, Column: 2, State: &sweeperv1.Cell_Flagged{}},

			{
				Row:    1,
				Column: 0,
				State: &sweeperv1.Cell_Revealed{
					Revealed: &sweeperv1.RevealedCell{
						Value: &sweeperv1.RevealedCell_Clear{
							Clear: &sweeperv1.ClearRevealedCell{NeighbouringMines: 1},
						},
					},
				},
			},
			{Row: 1, Column: 1, State: &sweeperv1.Cell_Questioned{}},
			{Row: 1, Column: 2, State: &sweeperv1.Cell_Unrevealed{}},

			{
				Row:    2,
				Column: 0,
				State: &sweeperv1.Cell_Revealed{
					Revealed: &sweeperv1.RevealedCell{
						Value: &sweeperv1.RevealedCell_Mine{},
					},
				},
			},
			{Row: 2, Column: 1, State: &sweeperv1.Cell_Unrevealed{}},
			{Row: 2, Column: 2, State: &sweeperv1.Cell_Unrevealed{}},
		},
	}

	_ = renderGame(
		context.Background(),
		os.Stdout,
		g,
	)

	// Output: Game '0_example_game'
	// You lost.	• 1/2
	//    │ 1 │ 2 │ 3
	// ───┼───┼───┼───
	//  1 │ 0 ╎   ╎ !
	// ───┼╌╌╌•╌╌╌•╌╌╌
	//  2 │ 1 ╎ ? ╎
	// ───┼╌╌╌•╌╌╌•╌╌╌
	//  3 │ ╳ ╎   ╎
}
