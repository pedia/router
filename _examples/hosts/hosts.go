package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/pedia/router"
)

// Index is the index handler
func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome!\n")
}

// Hello is the Hello handler
func Hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello, %s!\n", router.UserValue(r, "name"))
}

// HostSwitch is the host-handler map
// We need an object that implements the http.Handler interface.
// We just use a map here, in which we map host names (with port) to http.Handlers
type HostSwitch map[string]http.HandlerFunc

// CheckHost Implement a CheckHost method on our new type
func (hs HostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if a http.Handler is registered for the given host.
	// If yes, use it to handle the request.
	if handler := hs[string(r.Host)]; handler != nil {
		handler(w, r)
	} else {
		// Handle host names for which no handler is registered
		w.WriteHeader(403) // Or Redirect?
	}
}

func main() {
	// Initialize a router as usual
	r := router.New()
	r.GET("/", Index)
	r.GET("/hello/{name}", Hello)

	// Make a new HostSwitch and insert the router (our http handler)
	// for example.com and port 12345
	hs := make(HostSwitch)
	hs["example.com:12345"] = r.ServeHTTP

	// Use the HostSwitch to listen and serve on port 12345
	log.Fatal(http.ListenAndServe(":12345", hs))

	// curl -vs -H "Host: example.com:12345" http://127.0.0.1:12345/
	// curl -vs  http://127.0.0.1:12345/
}
