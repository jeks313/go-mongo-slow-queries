package health

import (
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/syncmap"
)

// SyncMap provides concurrent map @ golang.org/x/sync/syncmap.
type SyncMap struct {
	*syncmap.Map
}

// NewSyncMap constructs SyncMap instance for concurrent maps.
func NewSyncMap() *SyncMap {
	return &SyncMap{
		Map: &syncmap.Map{},
	}
}

// MarshalJSON generates json with concurrent protection.
func (m *SyncMap) MarshalJSON() ([]byte, error) {
	o := map[string]interface{}{}
	m.Range(func(k, v interface{}) bool {
		o[k.(string)] = v
		return true
	})
	return json.Marshal(o)
}

// ElapsedMillis returns the elapsed time between start and end in milliseconds.
// Use variadic to NOT require end, in which case "now" is assumed.
func ElapsedMillis(start time.Time, ends ...time.Time) int64 {
	var end time.Time
	switch len(ends) {
	case 0:
		end = time.Now()
	case 1:
		end = ends[0]
	default:
		log.Panic().Time("start", start).Interface("ends", ends).
			Msg("Invalid multiple ends for ElapsedMillis")
	}
	return DurationMillis(end.Sub(start))
}

// DurationMillis returns the milliseconds value for the provided duration.
func DurationMillis(dur time.Duration) int64 {
	return dur.Nanoseconds() / 1e6
}
