package mongoslow

import (
	"container/ring"
	"context"
	"encoding/json"
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

var (
	// length of history of slow queries to keep
	HistoryLen            int   = 1000    // number of items
	HistoryQueryThreshold int64 = 5000000 // microsecs, think this is 5s
)

// MongoSlow holds the state of slow queries, we have to keep state as we poll every x seconds and want to emit the
// cumulative slow query time for each user/connection/query.
type MongoSlow struct {
	ThresholdMicros   int
	QueryCounter      *prometheus.CounterVec   // prometheus counter, for running queries
	QueryHistogram    *prometheus.HistogramVec // prometheus histogram, for completed queries
	client            *mongo.Client
	runningQueryTimes map[int32]int64 // opid to microsecs_running map so we can measure how long something is running for
	runningQueries    map[int32]*Query
	history           *ring.Ring // history of slow queries
}

func New(ctx context.Context, uri, host, user, pass string, port int32) (*MongoSlow, error) {
	if uri == "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/?directConnection=true", user, pass, host, port)
	}
	clientOptions := options.Client().ApplyURI(uri)

	c, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Error().Str("uri", uri).Err(err).Msg("failed to connect to mongo")
		return nil, err
	}

	err = c.Ping(ctx, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to ping mongo")
		return nil, err
	}

	s := &MongoSlow{}
	s.runningQueryTimes = make(map[int32]int64)
	s.runningQueries = make(map[int32]*Query)
	s.history = ring.New(HistoryLen)
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

			lastMicrosecs, ok := s.runningQueryTimes[q.OperationID]
			if ok {
				q.DeltaMicros = q.RunningMicros - lastMicrosecs
				log.Info().
					Str("user", q.EffectiveUser).
					Str("op", q.Operation).
					Int32("opid", q.OperationID).
					Int64("last_microsecs_running", lastMicrosecs).
					Int64("microsecs_running", q.RunningMicros).
					Int64("delta", q.DeltaMicros).
					Msg("query still running")
			} else {
				log.Debug().Str("user", q.EffectiveUser).
					Str("op", q.Operation).
					Int32("opid", q.OperationID).Msg("new query started")
				q.DeltaMicros = q.RunningMicros
			}

			q.Inc(s.QueryCounter)

			s.runningQueryTimes[q.OperationID] = q.RunningMicros
			s.runningQueries[q.OperationID] = q
			currentQueryOpIDs[q.OperationID] = true
		}

		for opid, _ := range s.runningQueryTimes {
			_, ok := currentQueryOpIDs[opid]
			if !ok {
				log.Debug().Int32("opid", opid).Msg("query no longer running")
				// see if we should add it to the history
				microsecs := s.runningQueryTimes[opid]
				q := s.runningQueries[opid]
				q.Observe(s.QueryHistogram)
				if microsecs > HistoryQueryThreshold {
					if q.Namespace != "admin.$cmd" { // skip system queries in the history
						s.History(q)
					}
				}
				delete(s.runningQueryTimes, opid)
				delete(s.runningQueries, opid)
			}
		}

		time.Sleep(interval)
	}
}

func (s *MongoSlow) History(query *Query) {
	s.history.Value = query
	s.history = s.history.Next()
}

// Query object to hold current query details for feeding to Prometheus metrics
type Query struct {
	OperationID   int32       `json:"opid"`           // opid
	EffectiveUser string      `json:"effective_user"` // effectiveUsers:[map[db:admin user:auto-default-some-user-name-92c989781b97]]
	RunningMicros int64       `json:"running_micros"` // microseconds_running (with state to get delta)
	DeltaMicros   int64       `json:"delta_micros"`   // delta from last check in microseconds
	Operation     string      `json:"op"`             // op
	Namespace     string      `json:"ns"`             // ns
	Command       string      `json:"command"`        // string representation of the command
	Raw           primitive.M `json:"raw"`
}

// Observe updates the histogram with completed queries - use to get a view of slow completed queries
func (q *Query) Observe(histogram *prometheus.HistogramVec) {
	if q.RunningMicros > 500000 {
		histogram.WithLabelValues(q.EffectiveUser, q.Operation, q.Namespace).Observe(float64(q.RunningMicros) / 1000000)
	}
}

// Inc updates the query counter for running queries - use to get real time data on running slow queries
func (q *Query) Inc(counter *prometheus.CounterVec) {
	if q.DeltaMicros < 10000 { // if we are just picking up just executed queries, skip them
		return
	}
	counter.WithLabelValues(q.EffectiveUser, q.Operation, q.Namespace).Add(float64(q.DeltaMicros) / 1000) // change to milliseconds
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
	q.Raw = query

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
	q.EffectiveUser = trimRandomBytes(q.EffectiveUser)

	command, err := json.Marshal(query["command"])
	if err == nil {
		q.Command = string(command)
	}

	return q, nil
}
