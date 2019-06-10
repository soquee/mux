package mux_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"

	"code.soquee.net/mux"
)

var paramsTests = [...]struct {
	routes  []string
	path    string
	params  []mux.ParamInfo
	noMatch bool
	panics  bool
}{
	0: {
		routes: []string{"/user/{account uint}/{user int}/{name string}/{f float}"},
		path:   "/user/123/-11/me/1.123",
		params: []mux.ParamInfo{
			mux.ParamInfo{
				Value:  uint64(123),
				Raw:    "123",
				Name:   "account",
				Type:   "uint",
				Offset: 6,
			},
			mux.ParamInfo{
				Value:  int64(-11),
				Raw:    "-11",
				Name:   "user",
				Type:   "int",
				Offset: 10,
			},
			mux.ParamInfo{
				Value:  "me",
				Raw:    "me",
				Name:   "name",
				Type:   "string",
				Offset: 14,
			},
			mux.ParamInfo{
				Value:  float64(1.123),
				Raw:    "1.123",
				Name:   "f",
				Type:   "float",
				Offset: 17,
			},
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
		params: []mux.ParamInfo{
			mux.ParamInfo{
				Value:  "two/three",
				Raw:    "two/three",
				Name:   "other",
				Type:   "path",
				Offset: 5,
			},
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
	5: {
		routes: []string{"/{badtype}"},
		panics: true,
	},
	6: {
		routes: []string{"not/rooted"},
		panics: true,
	},
	7: {
		routes: []string{"unclean/./path"},
		panics: true,
	},
	8: {
		routes: []string{"unclean/../path"},
		panics: true,
	},
}

// Used as an HTTP status code code to make sure the test path matches at
// least one of the routes. This is just a sanity check on the tests
// themselves.
const (
	testStatusCode     = 42
	notFoundStatusCode = 43
)

func paramsHandler(t *testing.T, params []mux.ParamInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(testStatusCode)
		for _, v := range params {
			pinfo, ok := mux.Param(r, v.Name)
			if !ok {
				t.Errorf("No such parameter found %q", v.Name)
				continue
			}
			if !reflect.DeepEqual(pinfo, v) {
				t.Errorf("Param do not match: want=%+v, got=%+v)", v, pinfo)
			}
		}
	}
}

func TestParams(t *testing.T) {
	for i, tc := range paramsTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if tc.panics {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected test to panic")
					}
				}()
			}

			var opts []mux.Option
			for _, route := range tc.routes {
				opts = append(opts, mux.HandleFunc("GET", route, paramsHandler(t, tc.params)))
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

func TestParamNotFound(t *testing.T) {
	pinfo, ok := mux.Param(httptest.NewRequest("GET", "/", nil), "badparam")
	if ok {
		t.Errorf("Did not expect to find param but got %+v", pinfo)
	}
}
