package launchpad

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/launchpad-project/api.go/aggregation"
	"github.com/launchpad-project/api.go/jsonlib"
)

var mux *http.ServeMux
var server *httptest.Server

func TestAuthBasic(t *testing.T) {
	r := URL("http://localhost/")
	r.Auth("admin", "safe")

	var want = "Basic YWRtaW46c2FmZQ==" // admin:safe in base64
	var got = r.Headers.Get("Authorization")

	if want != got {
		t.Errorf("Wrong auth header. Expected %s, got %s instead", want, got)
	}
}

func TestAuthBasicRequestParam(t *testing.T) {
	r := URL("http://localhost/")
	r.Auth("admin", "safe")

	err := r.setupAction("GET")

	if err != nil {
		t.Error(err)
	}

	var username, password, ok = r.Request.BasicAuth()

	if username != "admin" || password != "safe" || ok != true {
		t.Errorf("Wrong user credentials")
	}
}

func TestAuthOAuth(t *testing.T) {
	var want = "Bearer myToken"
	r := URL("http://localhost/")

	r.Auth("myToken")
	got := r.Headers.Get("Authorization")

	if want != got {
		t.Errorf("Wrong OAuth token. Wanted Bearer %s, got %s instead", want, got)
	}
}

func TestHeader(t *testing.T) {
	key := "X-Custom"
	value := "foo"
	req := URL("https://example.com/")
	req.Header(key, value)

	got := req.Headers.Get(key)

	if got != value {
		t.Errorf("Expected header %s=%s not found, got %s instead", key, value, got)
	}

}

func TestBodyRequest(t *testing.T) {
	setupServer()
	defer teardownServer()

	wantContentType := "text/plain"
	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `"body"`)

		gotContentType := r.Header.Get("Content-Type")
		assertTextualBody(t, "foo", r.Body)

		if gotContentType != wantContentType {
			t.Errorf("Expected content type %s, got %s instead",
				wantContentType,
				gotContentType)
		}
	})

	req := URL("http://example.com/url")

	req.Headers.Set("Content-Type", wantContentType)
	req.Body(bytes.NewBufferString("foo"))

	if err := req.Post(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "POST", req.Request.Method)
}

func TestDecodeJSON(t *testing.T) {
	setupServer()
	defer teardownServer()

	var wantTitle = "body"

	setupDefaultMux(`{"title": "body"}`)

	req := URL("http://example.com/url")

	if err := req.Get(); err != nil {
		t.Error(err)
	}

	assertURI(t, "http://example.com/url", req.Request.URL.String())
	assertMethod(t, "GET", req.Request.Method)
	assertStatusCode(t, 200, req.Response.StatusCode)

	var content struct {
		Title string `json:"title"`
	}

	err := req.DecodeJSON(&content)

	if err != nil {
		t.Error(err)
	}

	if content.Title != wantTitle {
		t.Errorf("Expected title %s, got %s instead", wantTitle, content.Title)
	}
}

func TestDeleteRequest(t *testing.T) {
	setupServer()
	defer teardownServer()
	setupDefaultMux(`"body"`)

	req := URL("http://example.com/url")

	if err := req.Delete(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "DELETE", req.Request.Method)
}

func TestErrorStatusCode404(t *testing.T) {
	setupServer()
	defer teardownServer()

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	req := URL("http://example.com/url")

	if err := req.Get(); err != ErrUnexpectedResponse {
		t.Errorf("Missing error %s", ErrUnexpectedResponse)
	}

	assertTextualBody(t, "", req.Response.Body)
	assertStatusCode(t, 404, req.Response.StatusCode)
}

func TestGetRequest(t *testing.T) {
	setupServer()
	defer teardownServer()
	setupDefaultMux(`"body"`)

	req := URL("http://example.com/url")

	if err := req.Get(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "GET", req.Request.Method)
}

func TestHeadRequest(t *testing.T) {
	setupServer()
	defer teardownServer()

	wantHeader := "foo"

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Foo", wantHeader)
	})

	req := URL("http://example.com/url")

	if err := req.Head(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, "", req.Response.Body)
	assertMethod(t, "HEAD", req.Request.Method)

	gotHeader := req.Response.Header.Get("X-Foo")

	if wantHeader != gotHeader {
		t.Errorf("Want header X-Foo=%s, got %s instead", wantHeader, gotHeader)
	}
}

func TestPath(t *testing.T) {
	books := URL("https://example.com/books")
	book1 := books.Path("/1", "/2", "3")

	if books == book1 {
		t.Errorf("books and books1 should not be equal")
	}

	if books.URL != "https://example.com/books" {
		t.Error("books url is wrong")
	}

	want := "https://example.com/books/1/2/3"

	if book1.URL != want {
		t.Errorf("Unexpected book URL %s instead of %s", book1.URL, want)
	}
}

func TestUserAgent(t *testing.T) {
	r := URL("http://localhost/foo")
	err := r.setupAction("GET")

	if err != nil {
		t.Error(err)
	}

	var actual = r.Request.Header.Get("User-Agent")
	var expected = "Launchpad/master (+https://launchpad.io)"

	if actual != expected {
		t.Errorf("Expected User-Agent %s doesn't match with %s", actual, expected)
	}
}

func TestURL(t *testing.T) {
	r := URL("https://example.com/foo/bah")

	if err := r.setupAction("GET"); err != nil {
		t.Error(err)
	}

	assertURI(t, "https://example.com/foo/bah", r.Request.URL.String())
}

func TestURLErrorDueToInvalidURI(t *testing.T) {
	setupServer()
	defer teardownServer()
	r := URL("://example.com/foo/bah")
	err := r.Get()
	kind := reflect.TypeOf(err).String()

	if kind != "*url.Error" {
		t.Errorf("Expected error *url.Error, got %s instead", kind)
	}
}

func TestParam(t *testing.T) {
	var req = URL("http://example.com/xyz?keep=this")

	req.Param("x", "i")
	req.Param("y", "j")
	req.Param("z", "k")

	var want = "http://example.com/xyz?keep=this&x=i&y=j&z=k"

	if req.URL != want {
		t.Errorf("Wanted url %v, got %v instead", want, req.URL)
	}
}

func TestParamOverwrite(t *testing.T) {
	var req = URL("http://example.com/xyz")

	req.Param("foo", "bar")
	req.Param("foo", "bar2")
	req.Param("foo", "bar3")

	var want = "http://example.com/xyz?foo=bar3"

	if req.URL != want {
		t.Errorf("Wanted url %v, got %v instead", want, req.URL)
	}
}

func TestParamParsingErrorSilentFailure(t *testing.T) {
	// Silently ignoring errors from parsing for now.
	// This test describes what happens when parsing errors exists.
	// Reason: API simplicity.

	// Any error triggered here should be triggered as soon as a REST action
	// such as Get() or Post() is called.
	// Never even worry about it. Never say never.

	// See also TestParamsParsingErrorSilentFailure

	var req = URL(":wrong-schema")

	req.Param("foo", "bar")

	var want = ":wrong-schema"

	if req.URL != want {
		t.Errorf("Wanted invalid url %v, got %v instead", want, req.URL)
	}
}

func TestParams(t *testing.T) {
	var want = url.Values{
		"q":    []string{"foo"},
		"page": []string{"2"},
	}

	var req = URL("http://google.com/?q=foo&page=2")
	var got = req.Params()

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Params doesn't match:\n%s", pretty.Compare(want, got))
	}
}

func TestParamsParsingErrorSilentFailure(t *testing.T) {
	// See also TestParamParsingErrorSilentFailure
	var req = URL(":wrong-schema")
	var got = req.Params()

	if got != nil {
		t.Errorf("Params should be null, got %v instead", got)
	}
}

func TestPatchRequest(t *testing.T) {
	setupServer()
	defer teardownServer()
	setupDefaultMux(`"body"`)

	var req = URL("http://example.com/url")

	if err := req.Patch(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "PATCH", req.Request.Method)
}

func TestPostRequest(t *testing.T) {
	setupServer()
	defer teardownServer()
	setupDefaultMux(`"body"`)

	var req = URL("http://example.com/url")

	if err := req.Post(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "POST", req.Request.Method)
}

func TestPostFormRequest(t *testing.T) {
	setupServer()
	defer teardownServer()

	wantContentType := "application/x-www-form-urlencoded"
	wantTitle := "foo"

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `"body"`)

		var gotContentType = r.Header.Get("Content-Type")
		var gotTitle = r.PostFormValue("title")

		if gotContentType != wantContentType {
			t.Errorf("Expected content type %s, got %s instead",
				wantContentType,
				gotContentType)
		}

		if gotTitle != wantTitle {
			t.Errorf("Expected title %s, got %s instead", wantTitle, gotTitle)
		}
	})

	req := URL("http://example.com/url")

	req.Form("title", wantTitle)

	if err := req.Post(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "POST", req.Request.Method)
}

func TestPutRequest(t *testing.T) {
	setupServer()
	defer teardownServer()
	setupDefaultMux(`"body"`)

	var req = URL("http://example.com/url")

	if err := req.Put(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "PUT", req.Request.Method)
}

func TestQueryNull(t *testing.T) {
	req := URL("http://example.com/foo/bah")

	if req.Query != nil {
		t.Errorf("Expected empty query, found %v instead", req.Query)
	}
}

func TestQueryString(t *testing.T) {
	setupServer()
	defer teardownServer()

	var want = "http://example.com/url?foo=bar"

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		var got = r.URL.String()
		if got != want {
			t.Errorf("Wanted URL %v, got %v instead", want, got)
		}
	})

	req := URL("http://example.com/url")

	req.Param("foo", "bar")
	req.Param("foo", "bar")

	if err := req.Get(); err != nil {
		t.Error(err)
	}
}

func TestQueryAggregate(t *testing.T) {
	setupServer()
	defer teardownServer()

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		assertBody(t, `{"aggregation":[{"bah":{"name":"foo"}}]}`, r.Body)
		fmt.Fprintf(w, `"body"`)
	})

	req := URL("http://example.com/url")
	req.Aggregate("foo", "bah")

	if err := req.Post(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "POST", req.Request.Method)
}

func TestQueryCount(t *testing.T) {
	setupServer()
	defer teardownServer()

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		assertBody(t, `{"type":"count"}`, r.Body)
		fmt.Fprintf(w, `"body"`)
	})

	req := URL("http://example.com/url").Count()

	if req.Query.Type != "count" {
		t.Errorf("Expected count type, found %s instead", req.Query.Type)
	}

	if err := req.Get(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "GET", req.Request.Method)
}

func TestQueryFilter(t *testing.T) {
	setupServer()
	defer teardownServer()

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		assertBody(t, `{
    "filter": [
        {
            "foo": {
                "operator": "not",
                "value": "bah"
            }
        }
    ]
}`, r.Body)
		fmt.Fprintf(w, `"body"`)
	})

	req := URL("http://example.com/url").Filter("foo", "not", "bah")

	if err := req.Post(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "POST", req.Request.Method)
}

func TestQueryHighlight(t *testing.T) {
	setupServer()
	defer teardownServer()

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		assertBody(t, `{"highlight":["xyz"]}`, r.Body)
		fmt.Fprintf(w, `"body"`)
	})

	req := URL("http://example.com/url").Highlight("xyz")

	if err := req.Post(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "POST", req.Request.Method)
}

func TestQueryLimit(t *testing.T) {
	setupServer()
	defer teardownServer()

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		assertBody(t, `{"limit":10}`, r.Body)
		fmt.Fprintf(w, `"body"`)
	})

	req := URL("http://example.com/url").Limit(10)

	if err := req.Post(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "POST", req.Request.Method)
}

func TestQueryOffset(t *testing.T) {
	setupServer()
	defer teardownServer()

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		assertBody(t, `{"offset":0}`, r.Body)
		fmt.Fprintf(w, `"body"`)
	})

	req := URL("http://example.com/url").Offset(0)

	if err := req.Post(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "POST", req.Request.Method)
}

func TestQuerySort(t *testing.T) {
	setupServer()
	defer teardownServer()

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		assertBody(t, `{"sort":[{"id":"desc"}]}`, r.Body)
		fmt.Fprintf(w, `"body"`)
	})

	req := URL("http://example.com/url").Sort("id", "desc")

	if err := req.Post(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "POST", req.Request.Method)
}

func TestQuerySortAndAggregate(t *testing.T) {
	setupServer()
	defer teardownServer()

	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		assertBody(t, `{
    "sort": [
        {
            "field2": "asc"
        }
    ],
    "aggregation": [
        {
            "f": {
                "operator": "min",
                "name": "a"
            }
        },
        {
            "f": {
                "operator": "missing",
                "name": "m"
            }
        }
    ]
}`, r.Body)
		fmt.Fprintf(w, `"body"`)
	})

	req := URL("http://example.com/url")
	req.Sort("field2").Aggregate("a", "f", "min")
	req.Aggregate(aggregation.Missing("m", "f"))

	if err := req.Post(); err != nil {
		t.Error(err)
	}

	assertTextualBody(t, `"body"`, req.Response.Body)
	assertMethod(t, "POST", req.Request.Method)
}

func assertBody(t *testing.T, want string, body io.ReadCloser) {
	bin, err := ioutil.ReadAll(body)

	if err != nil {
		t.Error(err)
	}

	got := make(map[string]interface{})
	err = json.Unmarshal(bin, &got)

	if err != nil {
		t.Error(err)
	}

	jsonlib.AssertJSONMarshal(t, want, got)
}

func assertStatusCode(t *testing.T, want int, got int) {
	if got != want {
		t.Errorf("Expected status code %d, got %d", want, got)
	}
}

func assertURI(t *testing.T, want, got string) {
	if got != want {
		t.Errorf("Expected URL %s, got %s", want, got)
	}
}

func assertMethod(t *testing.T, want, got string) {
	if got != want {
		t.Errorf("%s method expected, found %s instead", want, got)
	}
}

func assertTextualBody(t *testing.T, want string, got io.ReadCloser) {
	body, err := ioutil.ReadAll(got)

	if err != nil {
		t.Error(err)
	}

	var bString = string(body)

	if bString != want {
		t.Errorf("Expected body with %s, got %s instead", want, bString)
	}
}

func setupDefaultMux(content string) {
	mux.HandleFunc("/url", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, content)
	})
}

func setupServer() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}
	Client = &http.Client{Transport: transport}
}

func teardownServer() {
	Client = &http.Client{}
	server.Close()
}
