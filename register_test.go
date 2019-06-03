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
		return []mux.Option{mux.Handle("", failHandler(t))}
	}},
	1: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("user/{uid int}", failHandler(t))}
	}},
	2: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("/user", failHandler(t)),
			mux.Handle("/user", failHandler(t)),
		}
	}},
	3: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("/{uint}", failHandler(t)),
			mux.Handle("/{uint}", failHandler(t)),
		}
	}},
	4: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("/{int}", failHandler(t)),
			mux.Handle("/{string}", failHandler(t)),
		}
	}},
	5: {
		routes: func(t *testing.T) []mux.Option {
			return []mux.Option{mux.Handle("/{int}", codeHandler(t, 5))}
		},
		expect: []expected{
			{path: "/1", code: 5},
			{path: "/-1", code: 5},
			{path: "/nope", code: 404},
		},
	},
	6: {
		routes: func(t *testing.T) []mux.Option {
			return []mux.Option{mux.Handle("/{u uint}", codeHandler(t, 5))}
		},
		expect: []expected{
			{path: "/1", code: 5},
			{path: "/-1", code: 404},
			{path: "/nope", code: 404},
		},
	},
	7: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("/{*}/user", failHandler(t))}
	}},
	8: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("/{*named}/user", failHandler(t))}
	}},
	9: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("/", failHandler(t)),
			mux.Handle("/", failHandler(t)),
		}
	}},
	10: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("/{badtyp}", failHandler(t))}
	}},
	11: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{mux.Handle("/{name badtyp}", failHandler(t))}
	}},
	12: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("/bad/{}", failHandler(t)),
			mux.Handle("/bad/{*}", failHandler(t)),
		}
	}},
	13: {panics: true, routes: func(t *testing.T) []mux.Option {
		return []mux.Option{
			mux.Handle("/bad/{}/end", failHandler(t)),
			mux.Handle("/bad/{*}/end", failHandler(t)),
		}
	}},
	14: {
		routes: func(t *testing.T) []mux.Option {
			return []mux.Option{
				mux.Handle("/good/one", codeHandler(t, 1)),
				mux.Handle("/good/two", codeHandler(t, 2)),
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
				mux.Handle("/a/b", codeHandler(t, 1)),
				mux.Handle("/a", codeHandler(t, 2)),
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
				mux.Handle("/a", codeHandler(t, 2)),
				mux.Handle("/a/b", codeHandler(t, 1)),
			}
		},
		expect: []expected{
			{path: "/a", code: 2},
			{path: "/a/b", code: 1},
			{path: "/a/c", code: 404},
		},
	},
	//17: {panics: true, routes: func(t *testing.T) []mux.Option {
	//	return []mux.Option{
	//		mux.Handle("/{uint}", failHandler(t)),
	//		mux.Handle("/me", failHandler(t)),
	//	}
	//}},
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
			mux := mux.New(tc.routes(t)...)

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
