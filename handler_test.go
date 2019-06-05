package mux_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.soquee.net/mux"
)

var notFoundTests = [...]struct {
	h    http.HandlerFunc
	code int
}{
	0: {
		h: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPaymentRequired)
		},
		code: http.StatusPaymentRequired,
	},
	1: {
		h: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusMultiStatus)
			_, err := w.Write([]byte("Test"))
			if err != nil {
				panic(err)
			}
		},
		code: http.StatusMultiStatus,
	},
	2: {
		h: func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte("Test"))
			if err != nil {
				panic(err)
			}
		},
		code: http.StatusNotFound,
	},
}

// NotFound handlers should always return a 404. If the underlying handler
// doesn't explicitly call WriteHeader, we should set a 404 as the default
// instead of a 200.
func TestNotFoundAlwaysSetsStatusCode(t *testing.T) {
	for i, tc := range notFoundTests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			m := mux.New(mux.NotFound(tc.h))
			rec := httptest.NewRecorder()
			m.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
			if rec.Code != tc.code {
				t.Errorf("Wrong status code from handler: want=%d, got=%d", tc.code, rec.Code)
			}
		})
	}
}
