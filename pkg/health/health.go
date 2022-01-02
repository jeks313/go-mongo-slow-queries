package health

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	// Dependencies holds all registered concrete dependencies.
	Dependencies = map[string]*Dependency{}
)

// RegisterDependencies registers one or more Dependencies. When setting up metrics please also use duration_seconds not duration_ms
func RegisterDependencies(dependencies ...*Dependency) {
	for _, dependency := range dependencies {
		logger := log.With().Interface("dependency", dependency).Logger()

		// Validate Name as required.
		if dependency.Name == "" {
			logger.Panic().Msg("Dependency's Name is required")
		}

		// Validate Item as required.
		if dependency.Item == nil {
			logger.Panic().Msg("Dependency's Item is required")
		}

		// Validate dependency.Name as unique.
		dependency.key = strings.ToLower(dependency.Name)

		if _, found := Dependencies[dependency.key]; found {
			logger.Panic().Str("key", dependency.key).
				Msg("Dependencies must be unique by Name")
		}

		if _, err := json.Marshal(dependency.Item); err != nil {
			logger.Panic().Err(err).
				Msg("Failed to marshal Dependency's Item (Depender)")
		}

		Dependencies[dependency.key] = dependency
	}
}

var (
	// Served indicates whether health has been served.
	Served bool
)

// Serve sets up and serves, forking checker.
func Serve() {
	if len(Dependencies) == 0 {
		log.Warn().Msg("no health dependencies detected, use health.Register")
	}

	go StartChecker()

	Served = true
}

var (
	// Config holds the health configuration.
	Config = &struct {
		StatusUnhealthy int           `json:"status_unhealthy"`  // Status code for an unhealthy state (at least one dependency with error).
		CheckInterval   time.Duration `json:"check_interval"`    // How often dependencies must be checked.
		CheckMaxTimeout time.Duration `json:"check_max_timeout"` // Maximum timeout for each dependency check.

		LogChecks               bool          `json:"log_checks"`                // Log check infos.
		MinimumCheckInterval    time.Duration `json:"min_check_interval"`        // Minimum duration to wait between health checks.
		CheckIntervalSubtrahend time.Duration `json:"check_interval_subtrahend"` // Time to subtract from CheckInterval in order to apply timeouts.
	}{
		StatusUnhealthy: http.StatusServiceUnavailable,
		CheckInterval:   15 * time.Second,
		CheckMaxTimeout: 14 * time.Second,

		MinimumCheckInterval:    2 * time.Second,
		CheckIntervalSubtrahend: 500 * time.Millisecond,
	}

	// Health holds the status of all checked dependencies.
	Health = struct {
		Version      map[string]string `json:"version"`      // Map for version/build info.
		Dependencies *SyncMap          `json:"dependencies"` // Map for all dependencies.
		Status       *SyncMap          `json:"status"`       // Map for single health state.
	}{
		Dependencies: NewSyncMap(),
		Status:       NewSyncMap(),
	}

	// Stats contains health statistics.
	Stats = struct {
		Total uint64 `json:"total"`
		Fails uint64 `json:"fails"`

		TotalRequests uint64 `json:"total_requests"` // @ WebHandler.
		TotalChecks   uint64 `json:"total_checks"`   // @ StartChecker's loop.

		CheckDurationMS int64 `json:"check_duration_ms"`
	}{}

	// Errors.
	errUnhealthyDefault = errors.New("starting (unhealthy by default)")
	errMsgCheckTimeout  = "Health dependency check has timed out after %v"
)

type depCheck struct {
	dependency *Dependency
	duration   time.Duration
	state      map[string]interface{}
	err        error
}

// StartChecker loops every interval (CheckInterval) to update status of dependencies.
func StartChecker() {
	// Validate minimum interval.
	if Config.CheckInterval < Config.MinimumCheckInterval {
		log.Panic().Dur("interval", Config.CheckInterval).
			Dur("min", Config.MinimumCheckInterval).
			Msg("Invalid interval - too short")
	}

	// Initialize all as unhealthy.
	for _, dependency := range Dependencies {
		setDep(depCheck{
			dependency: dependency,
			err:        errUnhealthyDefault,
		})
	}

	// Started must be set AFTER initialization above,
	// used for overall healthy status in WebHandler.
	started := time.Now()
	Health.Status.Store("started", started)

	timeout := Config.CheckInterval - Config.CheckIntervalSubtrahend
	if timeout > Config.CheckMaxTimeout {
		timeout = Config.CheckMaxTimeout
	}
	Health.Status.Store("config_interval", fmt.Sprintf("%v", Config.CheckInterval))
	Health.Status.Store("config_timeout", fmt.Sprintf("%v", timeout))

	// Infinite loop.
	for {
		atomic.AddUint64(&Stats.TotalChecks, 1)

		for _, dependency := range Dependencies {
			go func(dependency *Dependency) {
				chChecked := make(chan time.Duration, 1) // buffer=1 to avoid goroutine leak

				go func(dependency *Dependency) {
					dependencyStart := time.Now()
					state, err := dependency.Item.Check()
					elapsedDuration := time.Since(dependencyStart)
					setDep(depCheck{
						dependency: dependency,
						duration:   elapsedDuration,
						state:      state,
						err:        err,
					})
					chChecked <- elapsedDuration
				}(dependency)

				// Watch timeout.
				select {
				case elapsedDuration := <-chChecked:
					if Config.LogChecks {
						log.Info().Interface("dependency", dependency).
							Dur("duration", elapsedDuration).
							Msg("health dependency check completed")
					}
				case <-time.After(timeout):
					emsg := fmt.Sprintf(errMsgCheckTimeout, timeout)
					log.Warn().Interface("dependency", dependency).
						Dur("timeout", timeout).Msg(emsg)
					setDep(depCheck{
						dependency: dependency,
						duration:   timeout,
						err:        errors.New(emsg),
					})
				}
			}(dependency)
		}

		last := time.Now()
		Health.Status.Store("last", last)

		Stats.CheckDurationMS = ElapsedMillis(started, last)
		Health.Status.Store("duration_seconds", last.Sub(started).Seconds())

		time.Sleep(Config.CheckInterval)
	}
}

func setDep(dc depCheck) {

	dv := map[string]interface{}{
		"dependency": dc.dependency,
	}
	dv["duration_seconds"] = dc.duration.Seconds()

	if dc.state != nil {
		dv["state"] = dc.state
	}

	ready := dc.err == nil
	dv["ready"] = ready
	atomic.AddUint64(&Stats.Total, 1)

	if !ready {
		atomic.AddUint64(&Stats.Fails, 1)
		dv["error"] = dc.err.Error()

		if dc.err != errUnhealthyDefault {
			log.Error().Interface("dependency", dv).Msg("unhealthy dependency")
		}
	}

	Health.Dependencies.Store(dc.dependency.key, dv)
}

const (
	// StatusHealthy defines the status code for a healthy state.
	StatusHealthy = http.StatusOK

	stateHealthy   = "healthy"
	stateUnhealthy = "unhealthy"

	errMsgUnhealthy     = "Unhealthy"
	errMsgFailedMarshal = "Failed to marshal Health"
	errMsgFailedWrite   = "Failed to write Health response"
)

func setStatus(status int) int {
	var headerStatusCode int
	hstate := stateHealthy

	ready := status == StatusHealthy
	Health.Status.Store("ready", ready)

	if !ready {
		// Do NOT WriteHeader here, first check other potential errors (e.g. json marshal).
		headerStatusCode = status
		hstate = stateUnhealthy
	}
	Health.Status.Store("status", status)
	Health.Status.Store("state", hstate)

	return headerStatusCode
}

func handleError(w http.ResponseWriter, err error, msg string) {
	log.Error().Err(err).Str("msg", msg).Interface("health", Health).
		Msg("error during health web handler")

	errStatus := http.StatusInternalServerError
	setStatus(errStatus)

	w.WriteHeader(errStatus)
}

var (
	errCheckerNotStarted = errors.New("checker NOT yet started")
)

// WebHandler provides web handler.
func WebHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&Stats.TotalRequests, 1)

		var headerStatusCode int

		// Unhealthy if not yet started.
		if _, found := Health.Status.Load("started"); !found {
			handleError(w, errCheckerNotStarted, errMsgUnhealthy)
			return
		}

		// Unhealthy if any dependency contains error.
		healthy := true

		Health.Dependencies.Range(func(_, d interface{}) bool {
			hdep := d.(map[string]interface{})

			if _, found := hdep["error"]; found {
				// Even if unhealthy, do NOT fail and return, but instead
				// let it generate the usual json contents BUT with unhealthy header.
				log.Info().Interface("dependency", hdep).
					Msg("unhealthy dependencies (breaking on first)")
				setStatus(Config.StatusUnhealthy)
				healthy = false
				return false
			}

			return true
		})

		if healthy {
			headerStatusCode = setStatus(StatusHealthy)
		}

		healthInfo, err := json.Marshal(Health)
		if err != nil {
			handleError(w, err, errMsgFailedMarshal)
			return
		}
		Health.Status.Delete("status")
		Health.Status.Delete("state")

		// No marshal errors, so write this header BEFORE WriteHeader below.
		w.Header().Set("Content-Type", "application/json")

		if headerStatusCode != 0 {
			w.WriteHeader(headerStatusCode)
		}

		if _, err = w.Write(healthInfo); err != nil {
			handleError(w, err, errMsgFailedWrite)
			return
		}
	})
}
