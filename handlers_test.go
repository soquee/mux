package mux_test

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"

	"code.soquee.net/mux"
)

const (
	testBody = "Test"
	testCode = 123
)

func successHandler(writeCode, writeBody bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if writeCode {
			w.WriteHeader(testCode)
		}
		if writeBody {
			_, err := w.Write([]byte(testBody))
			if err != nil {
				panic(err)
			}
		}
	}
}

var handlerTests = [...]struct {
	opts     func(t *testing.T) []mux.Option
	method   string
	req      string
	code     int
	respBody string
	header   http.Header
}{
	0: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.NotFound(successHandler(true, false)),
				mux.Options(nil),
			}
		},
		code: testCode,
	},
	1: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.NotFound(successHandler(true, true)),
				mux.Options(nil),
			}
		},
		code:     testCode,
		respBody: testBody,
	},
	2: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.NotFound(successHandler(false, true)),
				mux.Options(nil),
			}
		},
		code:     http.StatusNotFound,
		respBody: testBody,
	},
	3: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Handle("GET", "/", failHandler(t)),
				mux.Handle("POST", "/", failHandler(t)),
				mux.Handle("PUT", "/test", failHandler(t)),
			}
		},
		method: http.MethodOptions,
		code:   http.StatusOK,
		header: map[string][]string{
			"Allow": []string{"GET,POST"},
		},
	},
	4: {
		method: http.MethodOptions,
		code:   http.StatusOK,
		header: map[string][]string{
			"Allow": []string{""},
		},
	},
	5: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Options(func([]string) http.Handler {
					return successHandler(true, true)
				}),
			}
		},
		method:   http.MethodOptions,
		code:     testCode,
		respBody: testBody,
	},
	6: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Options(nil),
				mux.NotFound(successHandler(false, true)),
			}
		},
		method:   http.MethodOptions,
		code:     http.StatusNotFound,
		respBody: testBody,
	},
	7: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Options(func([]string) http.Handler {
					return failHandler(t)
				}),
				mux.Handle(http.MethodOptions, "/", successHandler(true, false)),
			}
		},
		method: http.MethodOptions,
		code:   testCode,
	},
	8: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Handle(http.MethodGet, "/", failHandler(t)),
				mux.Options(nil),
			}
		},
		method:   http.MethodPost,
		code:     http.StatusMethodNotAllowed,
		respBody: http.StatusText(http.StatusMethodNotAllowed) + "\n",
	},
	9: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Handle(http.MethodGet, "/", failHandler(t)),
				mux.Options(nil),
				mux.MethodNotAllowed(nil),
				mux.NotFound(successHandler(true, false)),
			}
		},
		method: http.MethodPost,
		code:   testCode,
	},
	10: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.MethodNotAllowed(failHandler(t)),
				mux.Options(nil),
				mux.NotFound(successHandler(false, true)),
			}
		},
		method:   http.MethodPost,
		code:     http.StatusNotFound,
		respBody: testBody,
	},
	11: {
		opts: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.MethodNotAllowed(nil),
				mux.Options(nil),
				mux.NotFound(successHandler(true, false)),
			}
		},
		method: http.MethodPost,
		code:   testCode,
	},
}

func TestHandlers(t *testing.T) {
	for i, tc := range handlerTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if tc.opts == nil {
				tc.opts = func(*testing.T) []mux.Option { return []mux.Option{} }
			}
			m := mux.New(tc.opts(t)...)
			rec := httptest.NewRecorder()
			if tc.req == "" {
				tc.req = "/"
			}
			if tc.method == "" {
				tc.method = http.MethodGet
			}
			m.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.req, nil))
			if rec.Code != tc.code {
				t.Errorf("Unexpected status code: want=%d, got=%d", tc.code, rec.Code)
			}
			if s := rec.Body.String(); s != tc.respBody {
				t.Errorf("Unexpected response body: want=%q, got=%q", tc.respBody, s)
			}
			for k, _ := range tc.header {
				var v, vv string
				if k == "Allow" {
					// Sort "Allow" headers as a special case so that we don't have to do
					// a sort or anything in the actual handler.
					methods := strings.Split(tc.header.Get(k), ",")
					sort.Strings(methods)
					v = strings.Join(methods, ",")

					methods = strings.Split(rec.HeaderMap.Get(k), ",")
					sort.Strings(methods)
					vv = strings.Join(methods, ",")
				} else {
					v = tc.header.Get(k)
					vv = rec.HeaderMap.Get(k)
				}
				if vv != v {
					t.Errorf("Unexpected value for header %q: want=%q, got=%q", k, v, vv)
				}
			}
		})
	}
}
