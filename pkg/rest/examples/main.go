/*
Package main provides web app/service functionality.
*/
package main

import (
	"fmt"
	"net/url"

	"github.com/jeks313/go-mongo-slow-queries/pkg/rest"
	"github.com/rs/zerolog/log"
)

func main() {
	client := rest.NewClient("http://httpbin.org")

	// test some basic options
	client.AddRequestOptions(
		rest.Header("X-Test", "blah"),
		rest.BasicAuth("batman", "cave"))

	var result map[string]interface{}

	result = make(map[string]interface{})
	err := client.Get("/get", &result)

	if err != nil {
		log.Error().Str("method", "GET").Err(err).Msg("failed")
	} else {
		log.Info().Str("method", "GET").Err(err).Msgf("%v", result)
	}

	// test a POST request with arbitrary object
	payload := make(map[string]string)
	payload["one"] = "two"
	payload["three"] = "four"

	result = make(map[string]interface{})
	err = client.Post("/post", &result, rest.BodyJSON(payload))

	if err != nil {
		log.Error().Err(err).Msg("failed")
	} else {
		log.Info().Msgf("%v", result)
	}

	// test the basic auth works
	result = make(map[string]interface{})
	err = client.Get(fmt.Sprintf("/basic-auth/%s/%s", "batman", "cave"), &result, rest.BodyJSON(payload))

	if err != nil {
		log.Error().Str("method", "POST").Err(err).Msg("failed")
	} else {
		log.Info().Msgf("%v", result)
	}

	// test that gzip responses are handled
	result = make(map[string]interface{})
	err = client.Get("/gzip", &result)

	if err != nil {
		log.Error().Err(err).Str("method", "GET").Msg("failed")
	} else {
		log.Info().Msgf("%v", result)
	}

	// query parameters
	var query url.Values
	query = url.Values{}
	query.Set("cus_id", "1234")

	// test that gzip responses are handled
	result = make(map[string]interface{})
	err = client.Get("/get", &result, rest.Query(query))

	if err != nil {
		log.Error().Err(err).Str("method", "GET").Msg("failed")
	} else {
		log.Info().Msgf("%v", result)
	}
}
