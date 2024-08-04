package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	randv2 "math/rand/v2"
	"net/http"
	"os"
	"os/signal"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/nightmarlin/sweeper"
	"github.com/nightmarlin/sweeper/gen/sweeper/v1/sweeperv1connect"
	"github.com/nightmarlin/sweeper/handlers"
	"github.com/nightmarlin/sweeper/infra/memory"
)

var (
	port = flag.String("port", "34567", "port to listen on")
)

func main() {
	flag.Parse()

	var (
		log         = slog.New(slog.NewTextHandler(os.Stderr, nil))
		mux         = http.NewServeMux()
		srv         = &http.Server{Addr: fmt.Sprintf(":%s", *port), Handler: mux}
		ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	)

	defer cancel()
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(ctx)
	}()

	mux.Handle(
		sweeperv1connect.NewSweeperServiceHandler(
			handlers.NewConnect(sweeper.NewService(memory.NewStore(), uuid.New, randv2.IntN)),
			connect.WithInterceptors(LoggingInterceptor{logger: log}),
		),
	)

	log.Info("listening for connections", slog.String("port", *port))

	err := srv.ListenAndServe()
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		log.Info("exiting...")
	} else {
		log.Error("unexpected server shutdown", slog.String("error", err.Error()))
	}
}

type LoggingInterceptor struct {
	logger *slog.Logger
}

func (li LoggingInterceptor) WrapUnary(in connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		res, err := in(ctx, req)

		log := li.logger.Info
		result := "success"
		if err != nil {
			result = err.Error()
			log = li.logger.Error
		}

		log(
			"handled request",
			slog.String("method", req.Spec().Procedure),
			slog.String("result", result),
		)

		return res, err
	}
}
func (li LoggingInterceptor) WrapStreamingClient(in connect.StreamingClientFunc) connect.StreamingClientFunc {
	return in
}
func (li LoggingInterceptor) WrapStreamingHandler(in connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return in
}
