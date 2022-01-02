package main

import (
	"context"
	"fmt"
	stdlog "log"
	"os"

	"github.com/jeks313/go-mongo-slow-queries/pkg/options"
	flags "github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/mongo/options"
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

	clientOptions := mongoOptions.Client().ApplyURI("mongodb://root:pass@localhost:27017")

	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to mongo")
		os.Exit(1)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to ping mongo")
		os.Exit(1)
	}

	var runningQueries bson.M
	cmd := bson.D{{Key: "currentOp", Value: 1}, {Key: "$all", Value: "true"}}

	r := client.Database("admin").RunCommand(context.TODO(), cmd)
	err = r.Decode(&runningQueries)
	if err != nil {
		log.Error().Err(err).Msg("failed to run query")
		os.Exit(1)
	}

	queries := runningQueries["inprog"].(primitive.A)

	for i, query := range queries {
		fmt.Println(i, query)
		q := query.(primitive.M)
		fmt.Println(q["active"])
	}

	os.Exit(0)
}
