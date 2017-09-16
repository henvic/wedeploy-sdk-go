package wedeploy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wedeploy/api-go/query"
	"github.com/wedeploy/api-go/urilib"
)

const (
	// Version of Go API Client for WeDeploy Project
	Version = "master"
	// UserAgent of the WeDeploy api-go client
	UserAgent = "WeDeploy/" + Version + " (+https://wedeploy.com)"
)

var (
	// Client is the HTTP Client to use with WeDeploy
	Client = &http.Client{}
	// ErrUnexpectedResponse is used when an unexpected response happens
	ErrUnexpectedResponse = errors.New("Unexpected response")
)

// WeDeploy is the structure for a WeDeploy query
type WeDeploy struct {
	ID            int
	URL           string
	Time          time.Time
	Query         *query.Builder
	FormValues    *url.Values
	Headers       http.Header
	RequestBody   io.Reader
	Request       *http.Request
	Response      *http.Response
	context       context.Context
	cancelTimeout *context.CancelFunc
	httpClient    *http.Client
	timeout       *time.Duration
}

// URL creates a new request object
func URL(uri string, paths ...string) *WeDeploy {
	var time = time.Now()
	rand.Seed(time.UTC().UnixNano())
	uri = urilib.ResolvePath(uri, urilib.ResolvePath(paths...))

	var w = &WeDeploy{
		ID:         rand.Int(),
		Time:       time,
		URL:        uri,
		httpClient: Client,
	}

	w.Headers = http.Header{}

	w.Headers.Set("User-Agent", UserAgent)
	w.Headers.Set("Content-Type", "application/json")

	return w
}

// Aggregate adds an Aggregate query to the request
func (w *WeDeploy) Aggregate(ai ...interface{}) *WeDeploy {
	w.getOrCreateQuery().Aggregate(ai...)
	return w
}

// Auth sets HTTP basic auth headers
func (w *WeDeploy) Auth(args ...string) *WeDeploy {
	switch len(args) {
	case 1:
		w.Header("Authorization", "Bearer "+args[0])
	default:
		w.Header("Authorization", "Basic "+basicAuth(args[0], args[1]))
	}
	return w
}

// Body sets the body for the request
func (w *WeDeploy) Body(body io.Reader) *WeDeploy {
	w.RequestBody = body
	return w
}

// Count adds a Count query to the request
func (w *WeDeploy) Count() *WeDeploy {
	w.getOrCreateQuery().Count()
	return w
}

// DecodeJSON decodes a JSON response
func (w *WeDeploy) DecodeJSON(class interface{}) error {
	return json.NewDecoder(w.Response.Body).Decode(class)
}

// Delete method
func (w *WeDeploy) Delete() error {
	return w.action("DELETE")
}

// Filter adds a Filter query to the request
func (w *WeDeploy) Filter(ai ...interface{}) *WeDeploy {
	w.getOrCreateQuery().Filter(ai...)
	return w
}

// Form adds a Form query to the request
func (w *WeDeploy) Form(key, value string) *WeDeploy {
	w.getOrCreateForm().Add(key, value)

	return w
}

// Get method
func (w *WeDeploy) Get() error {
	return w.action("GET")
}

// Head method
func (w *WeDeploy) Head() error {
	return w.action("HEAD")
}

// Header adds a new header to the request
func (w *WeDeploy) Header(key, value string) *WeDeploy {
	w.Headers.Add(key, value)
	return w
}

// Highlight adds a Highlight query to the request
func (w *WeDeploy) Highlight(field string) *WeDeploy {
	w.getOrCreateQuery().Highlight(field)
	return w
}

// Limit adds a Limit query to the request
func (w *WeDeploy) Limit(limit int) *WeDeploy {
	w.getOrCreateQuery().Limit(limit)
	return w
}

// Offset adds an Offset query to the request
func (w *WeDeploy) Offset(offset int) *WeDeploy {
	w.getOrCreateQuery().Offset(offset)
	return w
}

// Param sets a query string param to the Request URL
// Check TestParamParsingErrorSilentFailure if you find unexpected result
func (w *WeDeploy) Param(key, value string) *WeDeploy {
	var u, err = url.Parse(w.URL)

	if err == nil {
		var query = u.Query()
		query.Set(key, value)
		u.RawQuery = query.Encode()
		w.URL = u.String()
	}

	return w
}

// Params gets the params from the Request URL
// Check TestParamsParsingErrorSilentFailure if you find unexpected result
func (w *WeDeploy) Params() url.Values {
	var u, err = url.Parse(w.URL)

	if err == nil {
		return u.Query()
	}

	return nil
}

// Patch method
func (w *WeDeploy) Patch() error {
	return w.action("PATCH")
}

// Path creates a new WeDeploy object composing paths
func (w *WeDeploy) Path(paths ...string) *WeDeploy {
	return URL(w.URL, paths...)
}

// Post method
func (w *WeDeploy) Post() error {
	return w.action("POST")
}

// Put method
func (w *WeDeploy) Put() error {
	return w.action("PUT")
}

// SetContext for the request
func (w *WeDeploy) SetContext(ctx context.Context) {
	w.context = ctx
}

// Sort adds a Sort query to the request
func (w *WeDeploy) Sort(field string, direction ...string) *WeDeploy {
	w.getOrCreateQuery().Sort(field, direction...)
	return w
}

// Timeout for the request
func (w *WeDeploy) Timeout(timeout time.Duration) {
	w.timeout = &timeout
}

// basicAuth creates the basic auth parameter
// extracted from golang/go/src/net/http/client.go
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (w *WeDeploy) getOrCreateQuery() *query.Builder {
	if w.Query == nil {
		w.Query = query.New()
	}
	return w.Query
}

func (w *WeDeploy) getOrCreateForm() *url.Values {
	if w.FormValues == nil {
		w.FormValues = &url.Values{}
	}
	return w.FormValues
}

func (w *WeDeploy) action(method string) (err error) {
	err = w.setupAction(method)

	if err != nil {
		w.cancelRemainingTimeout()
		return err
	}

	var bb *bytes.Buffer

	switch w.RequestBody.(type) {
	case *bytes.Buffer:
		bb = bytes.NewBuffer(w.RequestBody.(*bytes.Buffer).Bytes())
	}

	w.Response, err = w.httpClient.Do(w.Request)
	w.cancelRemainingTimeout()

	if bb != nil {
		w.RequestBody = bb
	}

	if err == nil && w.Response.StatusCode >= 400 {
		err = ErrUnexpectedResponse
	}

	return err
}

func (w *WeDeploy) setupContext() {
	if w.context == nil {
		w.context = context.Background()
	}

	w.Request = w.Request.WithContext(w.context)
}

func (w *WeDeploy) setupRequestTimeout() {
	if w.timeout != nil && *w.timeout != time.Duration(0) {
		requestCtx := w.Request.Context()
		var c context.CancelFunc
		w.context, c = context.WithTimeout(requestCtx, *w.timeout)
		w.cancelTimeout = &c
		w.Request = w.Request.WithContext(w.context)
	}
}

func (w *WeDeploy) setupAction(method string) (err error) {
	if w.FormValues != nil {
		w.RequestBody = strings.NewReader(w.FormValues.Encode())
		w.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if w.Query != nil {
		bin, err := json.Marshal(w.Query)

		if err != nil {
			return err
		}

		w.RequestBody = bytes.NewReader(bin)
	}

	if w.Request, err = http.NewRequest(method, w.URL, w.RequestBody); err != nil {
		return err
	}

	w.setupContext()
	w.setupRequestTimeout()
	w.Request.Header = w.Headers

	return err
}

func (w *WeDeploy) cancelRemainingTimeout() {
	if w.cancelTimeout != nil {
		(*w.cancelTimeout)()
	}
}
