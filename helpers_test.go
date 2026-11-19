package dna

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

const (
	testResellerID = "00000000-0000-4000-8000-000000000001"
	testAPIKey     = "test-api-key"
)

// recorded is one request the test server received.
type recorded struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   []byte
}

// reply is one canned response.
type reply struct {
	Status int
	Body   string
	Header map[string]string
}

// fakeAPI is a test server that records requests and answers them in order
// with its replies (repeating the last one).
type fakeAPI struct {
	t       *testing.T
	mu      sync.Mutex
	replies []reply
	got     []recorded
	srv     *httptest.Server
}

func newFakeAPI(t *testing.T, replies ...reply) *fakeAPI {
	t.Helper()
	f := &fakeAPI{t: t, replies: replies}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.got = append(f.got, recorded{Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Header: r.Header.Clone(), Body: body})
		rep := reply{Status: http.StatusOK}
		if n := len(f.got) - 1; n < len(f.replies) {
			rep = f.replies[n]
		} else if len(f.replies) > 0 {
			rep = f.replies[len(f.replies)-1]
		}
		f.mu.Unlock()
		for k, v := range rep.Header {
			w.Header().Set(k, v)
		}
		if rep.Status == 0 {
			rep.Status = http.StatusOK
		}
		w.WriteHeader(rep.Status)
		io.WriteString(w, rep.Body)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// client returns a client for the fake API with pacing off.
func (f *fakeAPI) client(opts ...Option) *Client {
	f.t.Helper()
	c, err := NewClient(testResellerID, testAPIKey, append([]Option{WithBaseURL(f.srv.URL), WithPacing(false)}, opts...)...)
	if err != nil {
		f.t.Fatal(err)
	}
	return c
}

// requests returns the recorded requests.
func (f *fakeAPI) requests() []recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recorded(nil), f.got...)
}

// last returns the last recorded request.
func (f *fakeAPI) last() recorded {
	f.t.Helper()
	got := f.requests()
	if len(got) == 0 {
		f.t.Fatal("no request reached the server")
	}
	return got[len(got)-1]
}

func ok(body string) reply { return reply{Status: http.StatusOK, Body: body} }

func apiErr(status int, code, message string) reply {
	return reply{Status: status, Body: `{"error":{"code":"` + code + `","message":"` + message + `","details":null,"data":[],"validationErrors":null}}`}
}

// sampleContact is a complete, valid contact.
func sampleContact() Contact {
	return Contact{
		FirstName: "Ada", LastName: "Lovelace", CompanyName: "Example Ltd", Email: "ada@example.com",
		Address: "1 Example Street", City: "Istanbul", State: "Istanbul", PostalCode: "34000", Country: "TR",
		PhoneCountryCode: "90", Phone: "5551234567", FaxCountryCode: "90", Fax: "2121234567", DiscloseFlag: Bool(false),
	}
}

func sampleSSLContact() SSLContact {
	return SSLContact{
		Title: "IT", FirstName: "Ada", LastName: "Lovelace", CompanyName: "Example Ltd", Email: "ada@example.com",
		Address: "1 Example Street", City: "Istanbul", State: "Istanbul", PostalCode: "34000", Country: "TR",
		PhoneCountryCode: "90", Phone: "5551234567", FaxCountryCode: "90", Fax: "2121234567",
	}
}

func mustURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
