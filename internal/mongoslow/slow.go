package mongoslow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoSlow holds the state of slow queries, we have to keep state as we poll every x seconds and want to emit the
// cumulative slow query time for each user/connection/query.
type MongoSlow struct {
	client            *mongo.Client
	QueryCounter      *prometheus.CounterVec // prometheus counter
	runningQueryTimes map[int32]int64        // opid to microsecs_running map so we can measure how long something is running for
}

func New(uri, host, user, pass string, port int32) (*MongoSlow, error) {
	if uri == "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/?directConnection=true", user, pass, host, port)
	}
	clientOptions := options.Client().ApplyURI(uri)

	c, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Error().Str("uri", uri).Err(err).Msg("failed to connect to mongo")
		return nil, err
	}

	err = c.Ping(context.TODO(), nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to ping mongo")
		return nil, err
	}

	s := &MongoSlow{}
	s.runningQueryTimes = make(map[int32]int64)
	s.client = c
	return s, nil
}

func (s *MongoSlow) Run(interval time.Duration) error {
	var runningQueries bson.M

	cmd := bson.D{{Key: "currentOp", Value: 1}, {Key: "$all", Value: true}}

	for {
		r := s.client.Database("admin").RunCommand(context.TODO(), cmd)
		err := r.Decode(&runningQueries)
		if err != nil {
			log.Error().Err(err).Msg("failed to run query")
			return err
		}

		queries := runningQueries["inprog"].(primitive.A)
		currentQueryOpIDs := make(map[int32]bool)

		for _, query := range queries {
			q, err := Parse(query.(primitive.M))
			if err != nil {
				log.Debug().Err(err).Interface("query", query).Msg("failed to parse query")
				continue
			}

			q.EffectiveUser = trimRandomBytes(q.EffectiveUser)

			lastMicrosecs, ok := s.runningQueryTimes[q.OperationID]
			if ok {
				var delta int64
				delta = q.RunningMicros - lastMicrosecs
				log.Info().
					Str("user", q.EffectiveUser).
					Str("op", q.Operation).
					Int32("opid", q.OperationID).
					Int64("last_microsecs_running", lastMicrosecs).
					Int64("microsecs_running", q.RunningMicros).
					Int64("delta", delta).
					Msg("query still running")
				q.Inc(s.QueryCounter)
			} else {
				log.Info().Str("user", q.EffectiveUser).
					Str("op", q.Operation).
					Int32("opid", q.OperationID).Msg("new query started")
				q.DeltaMicros = q.RunningMicros
				q.Inc(s.QueryCounter)
			}

			s.runningQueryTimes[q.OperationID] = q.RunningMicros
			currentQueryOpIDs[q.OperationID] = true
		}

		for opid, _ := range s.runningQueryTimes {
			_, ok := currentQueryOpIDs[opid]
			if !ok {
				log.Info().Int32("opid", opid).Msg("query no longer running")
				delete(s.runningQueryTimes, opid)
			}
		}

		time.Sleep(interval)
	}
}

// Query object to hold current query details for feeding to Prometheus metrics
type Query struct {
	OperationID   int32  // opid
	EffectiveUser string // effectiveUsers:[map[db:admin user:auto-default-some-user-name-92c989781b97]]
	RunningMicros int64  // microseconds_running (with state to get delta)
	DeltaMicros   int64  // delta from last check in microseconds
	Operation     string // op:
	Namespace     string // ns:
}

func (q *Query) Inc(counter *prometheus.CounterVec) {
	counter.WithLabelValues(q.EffectiveUser, q.Operation, q.Namespace).Add(float64(q.DeltaMicros))
}

func trimRandomBytes(user string) string {
	last := strings.LastIndex(user, "-")
	if last < 0 {
		return user
	}
	return user[:last]
}

func Parse(query primitive.M) (*Query, error) {
	q := &Query{}
	opid, ok := query["opid"]
	if !ok {
		return nil, errors.New("missing opid field")
	}
	q.OperationID = opid.(int32)

	microSecsRunning, ok := query["microsecs_running"]
	if !ok {
		return nil, errors.New("missing microseconds_running")
	}
	q.RunningMicros = microSecsRunning.(int64)

	op, ok := query["op"]
	if !ok {
		return nil, errors.New("missing op")
	}
	q.Operation = op.(string)

	ns, ok := query["ns"]
	if !ok {
		return nil, errors.New("missing ns")
	}
	q.Namespace = ns.(string)

	effectiveUsers, ok := query["effectiveUsers"]
	if !ok {
		return nil, errors.New("missing effective user field")
	}
	user := effectiveUsers.(primitive.A)[0]
	q.EffectiveUser = user.(primitive.M)["user"].(string)

	return q, nil
}
