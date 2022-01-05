package main

import (
	"context"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jeks313/go-mongo-slow-queries/internal/mongoslow"
	"github.com/jeks313/go-mongo-slow-queries/pkg/options"
	"github.com/jeks313/go-mongo-slow-queries/pkg/server"
	flags "github.com/jessevdk/go-flags"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
)

// MongoOpts is all the mongo specific connection options
type MongoOpts struct {
	User string `long:"mongo-user" env:"MONGO_USER" default:"" description:"mongo user name"`
	Pass string `long:"mongo-pass" env:"MONGO_PASS" default:"" description:"mongo password"`
	Host string `long:"mongo-host" env:"MONGO_HOST" default:"" description:"mongo hostname"`
	Port int32  `long:"mongo-port" env:"MONGO_PORT" default:"27017" description:"mongo port"`
	URI  string `long:"mongo-uri" env:"MONGO_URI" default:"" description:"instead of user,pass,host,port, pass a mongo URI to use directly"`
}

var opts struct {
	Port        int                        `long:"port" env:"PORT" default:"8172" description:"port number to listen on"`
	Application options.ApplicationOptions `group:"Default Application Options"`
	Mongo       MongoOpts                  `group:"Mongo Connection Options"`
}

var (
	slowQueries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "mongo",
			Name:      "slow_query_microsecs",
			Help:      "microseconds of slow query, according to db.currentOp()",
		},
		[]string{"user", "operation", "ns"},
	)
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	stdlog.SetFlags(0)
	stdlog.SetOutput(log)

	_, err := flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		log.Error().Err(err).Msg("failed to parse command line arguments")
		os.Exit(1)
	}

	if opts.Application.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if opts.Application.Version {
		options.LogVersion()
		os.Exit(0)
	}

	if opts.Mongo.URI == "" {
		if opts.Mongo.User == "" ||
			opts.Mongo.Pass == "" ||
			opts.Mongo.Host == "" {
			log.Error().Msg("pass in a mongo URI, or a user/pass/host/port combo")
			os.Exit(1)
		}
	}

	// router
	r := mux.NewRouter()
	r.Use(handlers.CompressHandler)

	// setup logging
	server.Log(r, log)

	// default end points
	server.Profiling(r, "/debug/pprof")

	// metrics
	server.Metrics(r, "/metrics")

	listen := fmt.Sprintf(":%d", opts.Port)

	srv := &http.Server{
		Handler:      r,
		Addr:         listen,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	defer func() {
		signal.Stop(c)
		cancel()
	}()

	go func() {
		select {
		case <-c:
			cancel()
			log.Info().Msg("interrupt, shutting down ...")
			srv.Shutdown(ctx)
		case <-ctx.Done():
		}
	}()

	go func(ctx context.Context, counter *prometheus.CounterVec) {
		slow, err := mongoslow.New(opts.Mongo.URI, opts.Mongo.User, opts.Mongo.Pass, opts.Mongo.Host, opts.Mongo.Port)
		if err != nil {
			log.Error().Err(err).Msg("failed to setup mongo")
			os.Exit(1)
		}
		slow.QueryCounter = counter
		err = slow.Run(5 * time.Second)
		if err != nil {
			log.Error().Err(err).Msg("run loop failed")
			cancel()
			srv.Shutdown(ctx)
		}
	}(ctx, slowQueries)

	log.Info().Int("port", opts.Port).Msg("started server ...")

	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("failed to start http server")
		os.Exit(1)
	}

	log.Info().Msg("stopped")

}
