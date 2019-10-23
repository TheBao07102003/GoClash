package clash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// This is the time format used by time fields -- we'll be using it to provide cleaner APIs.
var TimeLayout = "20060102T150405.000Z"

type logTimeFunc func(
	statusCode string,
	method string,
	hostname string,
	path string,
	elapsed time.Duration,
)

type Client struct {
	BaseURL     *url.URL
	UserAgent   string
	Bearer      string
	httpClient  http.Client
	logger      *log.Logger
	logTimeFunc logTimeFunc
}

// Base struct for paged queries.
type PagedQuery struct {
	Limit  int
	After  int
	Before int
}

// The error response sent by the API if 4xx/5xx status code.
type ErrorBody struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// APIError implements the error interface.
type APIError struct {
	Response *http.Response
	Body     *ErrorBody
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.Response.StatusCode, e.Body.Reason, e.Body.Message)
}

// Paging for pager responses. 'before' and 'after' may be empty if there are no more results to return.
type Paging struct {
	Cursors struct {
		Before string `json:"before"`
		After  string `json:"after"`
	} `json:"cursors"`
}

func NewClient(token string) *Client {
	base, _ := url.Parse("https://api.clashroyale.com")
	logger := log.New(os.Stdout, "(go-clash) ", 0)

	client := &Client{
		Bearer:  token,
		BaseURL: base,
		logger:  logger,
	}

	return client
}

func (c *Client) SetTimeout(duration time.Duration) {
	c.httpClient.Timeout = duration
}

func (c *Client) SetLogLatencyFunc(logTime logTimeFunc) {
	c.logTimeFunc = logTime
}

// make a new request object.
func (c *Client) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Bearer))
	req.Header.Set("User-Agent", c.UserAgent)
	return req, nil
}

// execute the request.
func (c *Client) Do(req *http.Request, v interface{}, label string) (*http.Response, error) {
	start := time.Now()

	c.logger.Println(req.Method, req.URL.String())
	resp, err := c.httpClient.Do(req)

	if err != nil {
		c.logTime(http.StatusInternalServerError, req.Method, label, start)
		c.logger.Println("Request error", err.Error())
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.logger.Println("Unexpected status code", resp.StatusCode)

		errorResponse := &ErrorBody{}
		err = json.NewDecoder(resp.Body).Decode(errorResponse)

		if err == nil {
			err = &APIError{resp, errorResponse}
		}
	} else {
		err = json.NewDecoder(resp.Body).Decode(v)
	}

	c.logTime(resp.StatusCode, req.Method, label, start)

	return resp, err
}
func (c *Client) logTime(statusCode int, method string, path string, start time.Time) {

	c.logTimeFunc(
		strconv.Itoa(statusCode),
		method,
		"api.clashroyale.com",
		path,
		time.Since(start),
	)
}

// make sure the tag is prefixed with a # if it doesn't have one
func NormaliseTag(tag string) string {
	if len(tag) > 0 && tag[0] == '#' {
		return tag
	}

	return "#" + tag
}
