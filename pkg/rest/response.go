package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/context"
	"github.com/rs/zerolog/log"
)

// ResponseOptionFunc defines a function that will be run on all rest client
// responses to transform the response in various ways
type ResponseOptionFunc func(resp *http.Response, result interface{}) error

const bodyErrorStringLimit = 1024

// Error custom error message for json parsing error
func NewJSONError(code int, body []byte, err error) *Error {
	if len(body) > bodyErrorStringLimit {
		body = body[:bodyErrorStringLimit]
	}
	return &Error{StatusCode: code, Err: fmt.Errorf("failed to parse response [%v]: %v", string(body), err)}
}

// ResponseJSON turns a rest client response into JSON
func ResponseJSON(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()
	if result == nil {
		return nil
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return NewJSONError(resp.StatusCode, content, err)
	}
	if len(content) == 0 {
		return nil
	}
	if err := json.Unmarshal(content, result); err != nil {
		return NewJSONError(resp.StatusCode, content, err)
	}
	return nil
}

// ResponseText turns a rest client response into a text string
func ResponseText(resp *http.Response, result interface{}) error {
	defer resp.Body.Close()
	if result == nil {
		return nil
	}
	resultPtr, ok := result.(*string)
	if !ok {
		return fmt.Errorf("result is not a string pointer")
	}
	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	if n == 0 {
		return fmt.Errorf("response body is empty")
	}
	*resultPtr = buf.String() // nolint:ineffassign
	return nil
}

// RequestJSON turns a request body into a JSON object
func RequestJSON(r *http.Request, result interface{}) error {
	defer r.Body.Close()
	if result == nil {
		return nil
	}
	return json.NewDecoder(r.Body).Decode(result)
}

// ResponseDebug can be used to debug a rest call, it will dump the http
// status code and the raw body returned
func ResponseDebug(resp *http.Response, result interface{}) error {
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(body)
	resp.Body = ioutil.NopCloser(buffer)
	return nil
}

// Error is a rest error, encapsulates the status code
type Error struct {
	StatusCode int
	Response   *http.Response
	Err        error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%d - %v", e.StatusCode, e.Err)
}

// ResponseTimer logs the request duration
func ResponseTimer(resp *http.Response, result interface{}) error {
	start := context.Get(resp.Request, "start")
	log.Info().
		Str("request", resp.Request.URL.Path).
		Str("method", resp.Request.Method).
		Float64("duration_secs", time.Since(start.(time.Time)).Seconds()).
		Msg("client request")
	return nil
}

// ResponseOK checks the HTTP response code and treats each of the response codes passed in
// as a successful response
func ResponseOK(ok ...int) ResponseOptionFunc {
	return func(resp *http.Response, result interface{}) error {
		for _, ok := range ok {
			if resp.StatusCode == ok {
				return nil
			}
		}
		return &Error{StatusCode: resp.StatusCode, Err: fmt.Errorf("%+v", result)}
	}
}

// ResponseError checks the HTTP response code and treats each of the response codes passed
// as an error condition
func ResponseError(errors ...int) ResponseOptionFunc {
	return func(resp *http.Response, result interface{}) error {
		for _, err := range errors {
			if resp.StatusCode == err {
				return &Error{
					StatusCode: resp.StatusCode,
					Response:   resp,
					Err:        fmt.Errorf("%+v", result),
				}
			}
		}
		return nil
	}
}

// ResponseOnlyOK is a quick convenience function, these are used by the portal to indicate a successful
// call
func ResponseOnlyOK() ResponseOptionFunc {
	return ResponseOK(http.StatusOK, http.StatusCreated, http.StatusNoContent)
}
