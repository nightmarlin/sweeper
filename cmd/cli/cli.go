// Command CLI provides CLI interactivity for the sweeper server.
//
// Usage
//
//	cli [-host=<host>] [-port=<port>] start <height> <width> <mines>
//	cli [-host=<host>] [-port=<port>] view <game-id>
//	cli [-host=<host>] [-port=<port>] play <game-id> <reset|flag|question|reveal> <row> <col>
//	cli [-host=<host>] [-port=<port>] end <game-id>
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	sweeperv1 "github.com/nightmarlin/sweeper/gen/sweeper/v1"
	"github.com/nightmarlin/sweeper/gen/sweeper/v1/sweeperv1connect"
)

var (
	host = flag.String("host", "http://localhost", "server hostname or ip address")
	port = flag.String("port", "34567", "server port")
)

func main() {
	flag.Parse()
	var (
		log         = slog.New(slog.NewTextHandler(os.Stderr, nil))
		ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		c           = client{
			c: sweeperv1connect.NewSweeperServiceClient(
				http.DefaultClient,
				fmt.Sprintf("%s:%s", *host, *port),
			),
		}
	)
	defer cancel()

	args := flag.Args()
	if len(args) == 0 {
		log.Error("at least one argument is required")
		return
	}

	var (
		g   *sweeperv1.Game
		err error
	)

	switch args[0] {
	case "start":
		if len(args) != 4 {
			log.Error("usage: start <height> <width> <mines>")
			return
		}
		g, err = c.start(ctx, args[1], args[2], args[3])

	case "view":
		if len(args) != 2 {
			log.Error("usage: view <game-id>")
			return
		}
		g, err = c.view(ctx, args[1])

	case "play":
		if len(args) != 5 {
			log.Error("usage: play <game-id> <command> <row> <column>")
			return
		}
		g, err = c.play(ctx, args[1], args[2], args[3], args[4])

	case "end":
		if len(args) != 2 {
			log.Error("usage: end <game-id>")
			return
		}
		g, err = c.end(ctx, args[1])
	}

	if err != nil {
		log.Error(fmt.Sprintf("failed to %s game", args[0]), slog.String("error", err.Error()))
		return
	}

	if err := renderGame(ctx, os.Stdout, g); err != nil {
		log.Error("failed to render game state", slog.String("error", err.Error()))
	}
}

type client struct {
	c sweeperv1connect.SweeperServiceClient
}

func (c client) start(ctx context.Context, h, w, m string) (*sweeperv1.Game, error) {
	hInt, err := strconv.ParseInt(h, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parsing height: %w", err)
	}
	wInt, err := strconv.ParseInt(w, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parsing width: %w", err)
	}
	mInt, err := strconv.ParseInt(m, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parsing mine count: %w", err)
	}

	res, err := c.c.StartGame(
		ctx,
		&connect.Request[sweeperv1.StartGameRequest]{
			Msg: &sweeperv1.StartGameRequest{
				Board: &sweeperv1.Board{
					Height: int32(hInt),
					Width:  int32(wInt),
					Mines:  int32(mInt),
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return res.Msg.Game, nil
}

func (c client) view(ctx context.Context, id string) (*sweeperv1.Game, error) {
	res, err := c.c.GetGame(
		ctx,
		&connect.Request[sweeperv1.GetGameRequest]{
			Msg: &sweeperv1.GetGameRequest{GameId: id},
		},
	)
	if err != nil {
		return nil, err
	}
	return res.Msg.Game, nil
}

func (c client) end(ctx context.Context, id string) (*sweeperv1.Game, error) {
	res, err := c.c.MakeMove(
		ctx,
		&connect.Request[sweeperv1.MakeMoveRequest]{
			Msg: &sweeperv1.MakeMoveRequest{
				GameId: id,
				Move:   &sweeperv1.MakeMoveRequest_End{End: &emptypb.Empty{}},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return res.Msg.Game, nil
}

func (c client) play(
	ctx context.Context,
	id string,
	action string,
	row, col string,
) (*sweeperv1.Game, error) {
	rInt, err := strconv.ParseInt(row, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parsing row: %w", err)
	}
	rInt -= 1

	cInt, err := strconv.ParseInt(col, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parsing column: %w", err)
	}
	cInt -= 1

	var a sweeperv1.CellMoveAction
	switch action {
	case "clear", "c", "reset":
		a = sweeperv1.CellMoveAction_CLEAR
	case "flag", "f":
		a = sweeperv1.CellMoveAction_FLAG
	case "question", "q":
		a = sweeperv1.CellMoveAction_QUESTION
	case "reveal", "r":
		a = sweeperv1.CellMoveAction_REVEAL
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}

	res, err := c.c.MakeMove(
		ctx,
		&connect.Request[sweeperv1.MakeMoveRequest]{
			Msg: &sweeperv1.MakeMoveRequest{
				GameId: id,
				Move: &sweeperv1.MakeMoveRequest_Cell{
					Cell: &sweeperv1.CellMove{
						Row:    int32(rInt),
						Column: int32(cInt),
						Action: a,
					},
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return res.Msg.Game, nil
}
