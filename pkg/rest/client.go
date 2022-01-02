package rest

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/context"
)

// Client represents a REST client, has a host, http client, request and response options
type Client struct {
	Host            string
	Client          *http.Client
	ResponseOptions []ResponseOptionFunc
	RequestOptions  []RequestOptionFunc
}

// setup some reasonable client timeouts here, note this does not stop
// the entire request from hanging, that needs to be handled in the reader
var (
	// DefaultClient is a default http client used by the rest calls, sets up
	// timeouts, keepalive, TLS timeout, response timeout
	DefaultClient = &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}
)

// NewClient creates a new rest client with some standard configured
// response options:
// Response:
// - Timing of requests
// - JSON decoding
// - HTTP response parsing, treating 200, 201, and 204 as good responses
func NewClient(host string) *Client {
	return &Client{
		Host:           host,
		Client:         DefaultClient,
		RequestOptions: []RequestOptionFunc{},
		ResponseOptions: []ResponseOptionFunc{
			ResponseTimer,
			ResponseJSON,
			ResponseOnlyOK()},
	}
}

// AddRequestOptions adds to the configured request options
func (c *Client) AddRequestOptions(options ...RequestOptionFunc) {
	// append
	c.RequestOptions = append(c.RequestOptions, options...)
}

func (c *Client) Timeout(d time.Duration) {
	c.Client.Transport.(*http.Transport).ResponseHeaderTimeout = d
}

// AddResponseOptions adds to the configured response options
func (c *Client) AddResponseOptions(options ...ResponseOptionFunc) {
	// prepend
	c.ResponseOptions = append(options, c.ResponseOptions...)
}

// Do does a HTTP REST request, applying all request options and applying all response options
func (c *Client) Do(method, path string, result interface{}, options ...RequestOptionFunc) error {
	url := fmt.Sprintf("%s%s", c.Host, path)

	requestOptions := c.RequestOptions
	if options != nil && len(options) > 0 {
		requestOptions = append(requestOptions, options...)
	}

	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return err
	}

	context.Set(req, "start", time.Now())
	defer context.Clear(req)

	for _, option := range requestOptions {
		err := option(req)
		if err != nil {
			return err
		}
	}

	resp, err := c.Client.Do(req)

	if err != nil {
		return err
	}

	for _, option := range c.ResponseOptions {
		err := option(resp, result)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get do a REST GET request
func (c *Client) Get(path string, result interface{}, options ...RequestOptionFunc) error {
	return c.Do("GET", path, result, options...)
}

// Post does a REST POST request
func (c *Client) Post(path string, result interface{}, options ...RequestOptionFunc) error {
	return c.Do("POST", path, result, options...)
}

// Delete does a REST DELETE request
func (c *Client) Delete(path string, result interface{}, options ...RequestOptionFunc) error {
	return c.Do("DELETE", path, result, options...)
}

// Head does a REST HEAD request
func (c *Client) Head(path string, result interface{}, options ...RequestOptionFunc) error {
	return c.Do("HEAD", path, result, options...)
}

// Patch does a REST PATCH request
func (c *Client) Patch(path string, result interface{}, options ...RequestOptionFunc) error {
	return c.Do("PATCH", path, result, options...)
}

// Put does a REST PUT request
func (c *Client) Put(path string, result interface{}, options ...RequestOptionFunc) error {
	return c.Do("PUT", path, result, options...)
}
