package launchpad

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/launchpad-project/api.go/query"
	"github.com/launchpad-project/api.go/urilib"
)

const (
	// Version of Go API Client for Launchpad Project
	Version = "master"
	// UserAgent of the Launchpad api.go client
	UserAgent = "Launchpad/" + Version + " (+https://launchpad.io)"
)

var (
	// Client is the HTTP Client to use with Launchpad
	Client = &http.Client{}
	// ErrUnexpectedResponse is used when an unexpected response happens
	ErrUnexpectedResponse = errors.New("Unexpected response")
)

// Launchpad is the structure for a Launchpad query
type Launchpad struct {
	ID          int
	URL         string
	Time        time.Time
	Query       *query.Builder
	FormValues  *url.Values
	Headers     http.Header
	RequestBody io.Reader
	Request     *http.Request
	Response    *http.Response
	httpClient  *http.Client
}

// URL creates a new request object
func URL(uri string, paths ...string) *Launchpad {
	var time = time.Now()
	rand.Seed(time.UTC().UnixNano())
	uri = urilib.ResolvePath(uri, urilib.ResolvePath(paths...))

	var l = &Launchpad{
		ID:         rand.Int(),
		Time:       time,
		URL:        uri,
		httpClient: Client,
	}

	l.Headers = http.Header{}

	l.Headers.Set("User-Agent", UserAgent)
	l.Headers.Set("Content-Type", "application/json")

	return l
}

// Aggregate adds an Aggregate query to the request
func (l *Launchpad) Aggregate(ai ...interface{}) *Launchpad {
	l.getOrCreateQuery().Aggregate(ai...)
	return l
}

// Auth sets HTTP basic auth headers
func (l *Launchpad) Auth(args ...string) *Launchpad {
	switch len(args) {
	case 1:
		l.Header("Authorization", "Bearer "+args[0])
	default:
		l.Header("Authorization", "Basic "+basicAuth(args[0], args[1]))
	}
	return l
}

// Body sets the body for the request
func (l *Launchpad) Body(body io.Reader) *Launchpad {
	l.RequestBody = body
	return l
}

// Count adds a Count query to the request
func (l *Launchpad) Count() *Launchpad {
	l.getOrCreateQuery().Count()
	return l
}

// DecodeJSON decodes a JSON response
func (l *Launchpad) DecodeJSON(class interface{}) error {
	return json.NewDecoder(l.Response.Body).Decode(class)
}

// Delete method
func (l *Launchpad) Delete() error {
	return l.action("DELETE")
}

// Filter adds a Filter query to the request
func (l *Launchpad) Filter(ai ...interface{}) *Launchpad {
	l.getOrCreateQuery().Filter(ai...)
	return l
}

// Form adds a Form query to the request
func (l *Launchpad) Form(key, value string) *Launchpad {
	l.getOrCreateForm().Add(key, value)

	return l
}

// Get method
func (l *Launchpad) Get() error {
	return l.action("GET")
}

// Head method
func (l *Launchpad) Head() error {
	return l.action("HEAD")
}

// Header adds a new header to the request
func (l *Launchpad) Header(key, value string) *Launchpad {
	l.Headers.Add(key, value)
	return l
}

// Highlight adds a Highlight query to the request
func (l *Launchpad) Highlight(field string) *Launchpad {
	l.getOrCreateQuery().Highlight(field)
	return l
}

// Limit adds a Limit query to the request
func (l *Launchpad) Limit(limit int) *Launchpad {
	l.getOrCreateQuery().Limit(limit)
	return l
}

// Offset adds an Offset query to the request
func (l *Launchpad) Offset(offset int) *Launchpad {
	l.getOrCreateQuery().Offset(offset)
	return l
}

// Patch method
func (l *Launchpad) Patch() error {
	return l.action("PATCH")
}

// Path creates a new Launchpad object composing paths
func (l *Launchpad) Path(paths ...string) *Launchpad {
	return URL(l.URL, paths...)
}

// Post method
func (l *Launchpad) Post() error {
	return l.action("POST")
}

// Put method
func (l *Launchpad) Put() error {
	return l.action("PUT")
}

// Sort adds a Sort query to the request
func (l *Launchpad) Sort(field string, direction ...string) *Launchpad {
	l.getOrCreateQuery().Sort(field, direction...)
	return l
}

// basicAuth creates the basic auth parameter
// extracted from golang/go/src/net/http/client.go
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (l *Launchpad) getOrCreateQuery() *query.Builder {
	if l.Query == nil {
		l.Query = query.New()
	}
	return l.Query
}

func (l *Launchpad) getOrCreateForm() *url.Values {
	if l.FormValues == nil {
		l.FormValues = &url.Values{}
	}
	return l.FormValues
}

func (l *Launchpad) action(method string) (err error) {
	err = l.setupAction(method)

	if err != nil {
		return err
	}

	l.Response, err = l.httpClient.Do(l.Request)

	if err == nil && l.Response.StatusCode >= 400 {
		err = ErrUnexpectedResponse
	}

	return err
}

func (l *Launchpad) setupAction(method string) (err error) {
	if l.FormValues != nil {
		l.RequestBody = strings.NewReader(l.FormValues.Encode())
		l.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if l.Query != nil {
		bin, err := json.Marshal(l.Query)

		if err != nil {
			return err
		}

		l.RequestBody = bytes.NewReader(bin)
	}

	req, err := http.NewRequest(method, l.URL, l.RequestBody)

	if err != nil {
		return err
	}

	req.Header = l.Headers
	l.Request = req

	return err
}
