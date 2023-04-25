package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"

	scrypt "github.com/elithrar/simple-scrypt"
	"github.com/pedia/router"
)

// basicAuth returns the username and password provided in the request's
// Authorization header, if the request uses HTTP Basic Authentication.
// See RFC 2617, Section 2.
func basicAuth(r *http.Request) (username, password string, ok bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return
	}
	return parseBasicAuth(string(auth))
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

// BasicAuth is the basic auth handler
func BasicAuth(h http.HandlerFunc, requiredUser string, requiredPasswordHash []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the Basic Authentication credentials
		user, password, hasAuth := basicAuth(r)

		// WARNING:
		// DO NOT use plain-text passwords for real apps.
		// A simple string comparison using == is vulnerable to a timing attack.
		// Instead, use the hash comparison function found in your hash library.
		// This example uses scrypt, which is a solid choice for secure hashing:
		//   go get -u github.com/elithrar/simple-scrypt

		if hasAuth && user == requiredUser {

			// Uses the parameters from the existing derived key. Return an error if they don't match.
			err := scrypt.CompareHashAndPassword(requiredPasswordHash, []byte(password))

			if err != nil {

				// log error and request Basic Authentication again below.
				log.Fatal(err)

			} else {

				// Delegate request to the given handle
				h(w, r)
				return

			}

		}

		// Request Basic Authentication otherwise
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
	}
}

// Index is the index handler
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Not protected!\n")
}

// Protected is the Protected handler
func Protected(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Protected!\n")
}

func main() {
	user := "gordon"
	pass := "secret!"

	// generate a hashed password from the password above:
	hashedPassword, err := scrypt.GenerateFromPassword([]byte(pass), scrypt.DefaultParams)
	if err != nil {
		log.Fatal(err)
	}

	r := router.New()
	r.GET("/", Index)
	r.GET("/protected/", BasicAuth(Protected, user, hashedPassword))

	s := http.Server{Addr: ":8080", Handler: r}
	log.Fatal(http.ListenAndServe(":8080", r))

	// curl -vs --user gordon:secret! http://127.0.0.1:8080/protected/
}
