package mux_test

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"code.soquee.net/mux"
)

func Example_wild() {
	m := mux.New(
		mux.HandleFunc("GET", "/sha256/{wildcard wild}", func(w http.ResponseWriter, r *http.Request) {
			val, _ := mux.Param(r, "wildcard")
			sum := sha256.Sum256([]byte(val.Raw))
			fmt.Fprintf(w, "the hash of %q is %x", val.Raw, sum)
		}),
	)

	server := httptest.NewServer(m)
	resp, err := http.Get(server.URL + "/sha256/a/b")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)
	// Output:
	// the hash of "a/b" is c14cddc033f64b9dea80ea675cf280a015e672516090a5626781153dc68fea11
}

func Example_path() {
	m := mux.New(
		mux.HandleFunc("GET", "/files/{p path}/cover.png", func(w http.ResponseWriter, r *http.Request) {
			val, _ := mux.Param(r, "p")
			fmt.Fprintf(w, "downloading file %q…", val.Raw)
		}),
	)

	server := httptest.NewServer(m)
	resp, err := http.Get(server.URL + "/files/myalbum/cover.png")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)
	// Output:
	// downloading file "myalbum/cover.png"…
}

func Example_normalization() {
	m := mux.New(
		mux.HandleFunc("GET", "/profile/{username string}/personal", func(w http.ResponseWriter, r *http.Request) {
			username, _ := mux.Param(r, "username")
			// You probably want to use the Username Case Mapped profile from the
			// golang.org/x/text/secure/precis package instead of lowercasing.
			normalized := strings.ToLower(username.Raw)

			// If the username had any capital letters, redirect to the canonical
			// username.
			if normalized != username.Raw {
				newPath := r.URL.Path[:username.Offset] + normalized + r.URL.Path[username.Offset+uint(len(username.Raw)):]
				http.Redirect(w, r, newPath, http.StatusPermanentRedirect)
				return
			}

			// Show the users profile.
			fmt.Fprintf(w, "Profile for the user %q", username.Raw)
		}),
	)

	server := httptest.NewServer(m)
	resp, err := http.Get(server.URL + "/profile/Me/personal")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)
	// Output:
	// Profile for the user "me"
}
