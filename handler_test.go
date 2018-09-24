package pathmux_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"soquee.net/pathmux"
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
	routes func(t *testing.T) []pathmux.Option
	expect []expected
}{
	0: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{pathmux.Handle("", failHandler(t))}
	}},
	1: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{pathmux.Handle("user/{uid int}", failHandler(t))}
	}},
	2: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{
			pathmux.Handle("/user", failHandler(t)),
			pathmux.Handle("/user", failHandler(t)),
		}
	}},
	3: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{
			pathmux.Handle("/{uint}", failHandler(t)),
			pathmux.Handle("/{uint}", failHandler(t)),
		}
	}},
	4: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{
			pathmux.Handle("/{int}", failHandler(t)),
			pathmux.Handle("/{string}", failHandler(t)),
		}
	}},
	5: {
		routes: func(t *testing.T) []pathmux.Option {
			return []pathmux.Option{pathmux.Handle("/{int}", codeHandler(t, 5))}
		},
		expect: []expected{
			{path: "/1", code: 5},
			{path: "/-1", code: 5},
			{path: "/nope", code: 404},
		},
	},
	6: {
		routes: func(t *testing.T) []pathmux.Option {
			return []pathmux.Option{pathmux.Handle("/{u uint}", codeHandler(t, 5))}
		},
		expect: []expected{
			{path: "/1", code: 5},
			{path: "/-1", code: 404},
			{path: "/nope", code: 404},
		},
	},
	7: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{pathmux.Handle("/{*}/user", failHandler(t))}
	}},
	8: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{pathmux.Handle("/{*named}/user", failHandler(t))}
	}},
	9: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{
			pathmux.Handle("/", failHandler(t)),
			pathmux.Handle("/", failHandler(t)),
		}
	}},
	10: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{pathmux.Handle("/{badtyp}", failHandler(t))}
	}},
	11: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{pathmux.Handle("/{name badtyp}", failHandler(t))}
	}},
	12: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{
			pathmux.Handle("/bad/{}", failHandler(t)),
			pathmux.Handle("/bad/{*}", failHandler(t)),
		}
	}},
	13: {panics: true, routes: func(t *testing.T) []pathmux.Option {
		return []pathmux.Option{
			pathmux.Handle("/bad/{}/end", failHandler(t)),
			pathmux.Handle("/bad/{*}/end", failHandler(t)),
		}
	}},
	14: {
		routes: func(t *testing.T) []pathmux.Option {
			return []pathmux.Option{
				pathmux.Handle("/good/one", codeHandler(t, 1)),
				pathmux.Handle("/good/two", codeHandler(t, 2)),
			}
		},
		expect: []expected{
			{path: "/good", code: 404},
			{path: "/good/one", code: 1},
			{path: "/good/two", code: 2},
		},
	},
	15: {
		routes: func(t *testing.T) []pathmux.Option {
			return []pathmux.Option{
				pathmux.Handle("/a/b", codeHandler(t, 1)),
				pathmux.Handle("/a", codeHandler(t, 2)),
			}
		},
		expect: []expected{
			{path: "/a", code: 2},
			{path: "/a/b", code: 1},
			{path: "/a/c", code: 404},
		},
	},
	16: {
		routes: func(t *testing.T) []pathmux.Option {
			return []pathmux.Option{
				pathmux.Handle("/a", codeHandler(t, 2)),
				pathmux.Handle("/a/b", codeHandler(t, 1)),
			}
		},
		expect: []expected{
			{path: "/a", code: 2},
			{path: "/a/b", code: 1},
			{path: "/a/c", code: 404},
		},
	},
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
			mux := pathmux.NewServeMux(tc.routes(t)...)

			for _, e := range tc.expect {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, httptest.NewRequest("GET", e.path, nil))

				if rec.Code != e.code {
					t.Errorf("Got unexpected response code: want=%d, got=%d", e.code, rec.Code)
				}
			}
		})
	}
}
