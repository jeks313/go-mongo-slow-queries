package rest

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// RequestOptionFunc is a function that is called on the request before the rest
// call is made
type RequestOptionFunc func(req *http.Request) error

// BasicAuth adds the username/password as basic auth headers to the request
func BasicAuth(user, pass string) RequestOptionFunc {
	return func(req *http.Request) error {
		req.SetBasicAuth(user, pass)
		return nil
	}
}

// Header adds the name: val header to the request
func Header(name, val string) RequestOptionFunc {
	return func(req *http.Request) error {
		req.Header.Add(name, val)
		return nil
	}
}

// Headers adds all the key: values in the map to the request headers
func Headers(headers map[string]string) RequestOptionFunc {
	return func(req *http.Request) error {
		for name, val := range headers {
			req.Header.Add(name, val)
		}
		return nil
	}
}

// Query adds query parameters to the request
func Query(query url.Values) RequestOptionFunc {
	return func(req *http.Request) error {
		req.URL.RawQuery = query.Encode()
		return nil
	}
}

// BodyJSON is a utility function that encodes the body passed in
// as JSON
func BodyJSON(obj interface{}) RequestOptionFunc {
	return func(req *http.Request) error {
		b := new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(obj); err != nil {
			return err
		}
		req.Header.Add("content-type", "application/json")
		req.Body = ioutil.NopCloser(b)
		return nil
	}
}

// BodyForm adds the data passed in as form variables to a request
func BodyForm(data url.Values) RequestOptionFunc {
	return func(req *http.Request) error {
		b := bytes.NewBufferString(data.Encode())
		req.Header.Add("content-type", "application/x-www-form-urlencoded")
		req.Body = ioutil.NopCloser(b)
		return nil
	}
}

// BodyReader sets the body via reader.
func BodyReader(body io.Reader) RequestOptionFunc {
	return func(req *http.Request) error {
		req.Body = ioutil.NopCloser(body)
		return nil
	}
}

// BodyBytes sets the body as bytes
func BodyBytes(data []byte) RequestOptionFunc {
	return func(req *http.Request) error {
		buf := bytes.NewBuffer(data)
		req.Body = ioutil.NopCloser(buf)
		return nil
	}
}

// BodyText is a utility function that encodes the body passed in
// as plain text
func BodyText(rawMessage string) RequestOptionFunc {
	return func(req *http.Request) error {
		b := bytes.NewBufferString(rawMessage)
		req.Header.Add("content-type", "text/plain")
		req.Body = ioutil.NopCloser(b)
		return nil
	}
}
