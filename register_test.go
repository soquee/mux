package mux_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.soquee.net/mux"
)

func failHandler(t *testing.T) http.HandlerFunc {
	return func(http.ResponseWriter, *http.Request) {
		t.Error("Handler was called unexpectedly")
	}
}

func codeHandler(t *testing.T, code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	}
}

type expected struct {
	path string
	code int
}

var registerTests = [...]struct {
	panics bool
	routes func(t *testing.T) []mux.Option
	expect []expected
}{
	0: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("GET", "", failHandler(t))}
	}},
	1: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("GET", "user/{uid int}", failHandler(t))}
	}},
	2: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/user", failHandler(t)),
			mux.Handle("GET", "/user", failHandler(t)),
		}
	}},
	3: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/{uint}", failHandler(t)),
			mux.Handle("GET", "/{uint}", failHandler(t)),
		}
	}},
	4: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/{int}", failHandler(t)),
			mux.Handle("GET", "/{}", failHandler(t)),
		}
	}},
	5: {
		routes: func(t *testing.T) []mux.Option {
			return []mux.Option{mux.Handle("GET", "/{int}", codeHandler(t, 5))}
		},
		expect: []expected{
			{path: "/1", code: 5},
			{path: "/-1", code: 5},
			{path: "/nope", code: 404},
		},
	},
	6: {
		routes: func(t *testing.T) []mux.Option {
			return []mux.Option{mux.Handle("GET", "/{u uint}", codeHandler(t, 5))}
		},
		expect: []expected{
			{path: "/1", code: 5},
			{path: "/-1", code: 404},
			{path: "/nope", code: 404},
		},
	},
	7: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("GET", "/{path}/user", failHandler(t))}
	}},
	8: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("GET", "/{named path}/user", failHandler(t))}
	}},
	9: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/", failHandler(t)),
			mux.Handle("GET", "/", failHandler(t)),
		}
	}},
	10: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("GET", "/{badtyp}", failHandler(t))}
	}},
	11: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("GET", "/{name badtyp}", failHandler(t))}
	}},
	12: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/bad/{int}", failHandler(t)),
			mux.Handle("GET", "/bad/{path}", failHandler(t)),
		}
	}},
	13: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/bad/{int}/end", failHandler(t)),
			mux.Handle("GET", "/bad/{path}/end", failHandler(t)),
		}
	}},
	14: {
		routes: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Handle("GET", "/good/one", codeHandler(t, 1)),
				mux.Handle("GET", "/good/two", codeHandler(t, 2)),
			}
		},
		expect: []expected{
			{path: "/good", code: 404},
			{path: "/good/one", code: 1},
			{path: "/good/two", code: 2},
		},
	},
	15: {
		routes: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Handle("GET", "/a/b", codeHandler(t, 1)),
				mux.Handle("GET", "/a", codeHandler(t, 2)),
			}
		},
		expect: []expected{
			{path: "/a", code: 2},
			{path: "/a/b", code: 1},
			{path: "/a/c", code: 404},
		},
	},
	16: {
		routes: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Handle("GET", "/a", codeHandler(t, 2)),
				mux.Handle("GET", "/a/b", codeHandler(t, 1)),
			}
		},
		expect: []expected{
			{path: "/a", code: 2},
			{path: "/a/b", code: 1},
			{path: "/a/c", code: 404},
		},
	},
	17: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/{uint}", failHandler(t)),
			mux.Handle("GET", "/me", failHandler(t)),
		}
	}},
	18: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/{static}", failHandler(t)),
		}
	}},
	19: {
		routes: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Handle("GET", "/", codeHandler(t, 2)),
			}
		},
		expect: []expected{
			{path: "/", code: 2},
		},
	},
	20: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/{a int}/a", failHandler(t)),
			mux.Handle("GET", "/{b int}/b", failHandler(t)),
		}
	}},
	21: {routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/user", failHandler(t)),
			mux.Handle("POST", "/user", failHandler(t)),
		}
	}},
	22: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "//user", failHandler(t)),
		}
	}},
	23: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/../user", failHandler(t)),
		}
	}},
	24: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "/./user", failHandler(t)),
		}
	}},
	25: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("GET", "test", failHandler(t)),
		}
	}},
}

func TestRegisterRoutes(t *testing.T) {
	for i, tc := range registerTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			defer func() {
				r := recover()
				switch {
				case tc.panics && r == nil:
					t.Errorf("Expected bad route to panic")
				case !tc.panics && r != nil:
					t.Errorf("Did not expect panic, got=%q", r)
				}
			}()
			m := mux.New(tc.routes(t)...)

			for _, e := range tc.expect {
				rec := httptest.NewRecorder()
				m.ServeHTTP(rec, httptest.NewRequest("GET", e.path, nil))

				if rec.Code != e.code {
					t.Errorf("Got unexpected response code: want=%d, got=%d", e.code, rec.Code)
				}
			}
		})
	}
}
