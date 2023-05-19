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

// MultiParams is the multi params handler
func MultiParams(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hi, %s, %s!\n", router.UserValue(r, "name"), router.UserValue(r, "word"))
}

// RegexParams is the params handler with regex validation
func RegexParams(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hi, %s\n", router.UserValue(r, "name"))
}

// QueryArgs is used for uri query args test #11:
// if the req uri is /ping?name=foo, output: Pong! foo
// if the req uri is /piNg?name=foo, redirect to /ping, output: Pong!
func QueryArgs(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	fmt.Fprintf(w, "Pong! %s\n", name)
}

func main() {
	r := router.New()
	r.GET("/", Index)
	r.GET("/hello/{name}", Hello)
	r.GET("/multi/{name}/{word}", MultiParams)
	r.GET("/regex/{name:[a-zA-Z]+}/test", RegexParams)
	r.GET("/optional/{name?:[a-zA-Z]+}/{word?}", MultiParams)
	r.GET("/ping", QueryArgs)

	log.Fatal(http.ListenAndServe(":8080", r))
}
