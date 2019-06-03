package mux_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"

	"code.soquee.net/mux"
)

func TestInvalidType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected an invalid type to cause a panic")
		}
	}()
	mux.New(mux.Handle("/{badtype}", failHandler(t)))
}

var paramsTests = [...]struct {
	routes  []string
	path    string
	params  map[string]interface{}
	noMatch bool
}{
	0: {
		routes: []string{"/user/{account uint}/{user int}/{name string}/{f float}"},
		path:   "/user/123/-11/me/1.123",
		params: map[string]interface{}{
			"account": uint64(123),
			"user":    int64(-11),
			"name":    "me",
			"f":       float64(1.123),
		},
	},
	1: {
		routes:  []string{"/{bad float}"},
		path:    "/notfloat",
		noMatch: true,
	},
	2: {
		routes: []string{"/one/{other path}"},
		path:   "/one/two/three",
		params: map[string]interface{}{
			"other": "two/three",
		},
	},
	3: {
		routes:  []string{"/a"},
		path:    "/b",
		noMatch: true,
	},
	4: {
		routes: []string{"/{}"},
		path:   "/b",
	},
}

// Used as an HTTP status code code to make sure the test path matches at
// least one of the routes. This is just a sanity check on the tests
// themselves.
const (
	testStatusCode     = 42
	notFoundStatusCode = 43
)

func paramsHandler(t *testing.T, params map[string]interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(testStatusCode)
		for k, v := range params {
			val := mux.Param(r, k)
			if !reflect.DeepEqual(val, v) {
				t.Errorf("Params has wrong type for %[1]q, want=%[2]T(%[2]v), got=%[3]T(%[3]v)", k, v, val)
			}
		}
	}
}

func TestParams(t *testing.T) {
	for i, tc := range paramsTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var opts []mux.Option
			for _, route := range tc.routes {
				opts = append(opts, mux.HandleFunc(route, paramsHandler(t, tc.params)))
			}
			opts = append(opts, mux.NotFound(codeHandler(t, notFoundStatusCode)))

			m := mux.New(opts...)
			rec := httptest.NewRecorder()
			m.ServeHTTP(rec, httptest.NewRequest("GET", tc.path, nil))
			switch {
			case tc.noMatch && rec.Code != notFoundStatusCode:
				t.Fatalf("Expected path to not be found, got code %d", rec.Code)
			case !tc.noMatch && rec.Code != testStatusCode:
				t.Fatalf("Test path (%q) did not match any route!", tc.path)
			}
		})
	}
}
