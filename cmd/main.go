package main

import (
	"context"
	stdlog "log"
	"os"

	"github.com/go-mongo-slow-queries/pkg/options"
	flags "github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var opts struct {
	Application options.ApplicationOptions `group:"Default Application Options"`
}

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

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Err(err).Error("failed to connect to mongo")
		os.Exit(1)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Err(err).Error("failed to ping mongo")
		os.Exit(1)
	}

	os.Exit(0)
}
