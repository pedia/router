package router

import (
	"bufio"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

type routerGrouper interface {
	Group(string) *Group
	ServeFiles(path string, rootPath string)
	ServeFilesCustom(path string, fs http.FileSystem)
}

func assertGroup(t *testing.T, gs ...routerGrouper) {
	for i, g := range gs {
		g2 := g.Group("/")

		v1 := reflect.ValueOf(g)
		v2 := reflect.ValueOf(g2)

		if v1.String() != v2.String() { // router -> group
			if v1.Pointer() == v2.Pointer() {
				t.Errorf("[%d] equal pointers: %p == %p", i, g, g2)
			}
		} else { // group -> subgroup
			if v1.Pointer() != v2.Pointer() {
				t.Errorf("[%d] mismatch pointers: %p != %p", i, g, g2)
			}
		}

		if err := catchPanic(func() { g.Group("v999") }); err == nil {
			t.Error("an error was expected when a path does not begin with slash")
		}

		if err := catchPanic(func() { g.Group("/v999/") }); err == nil {
			t.Error("an error was expected when a path has a trailing slash")
		}

		if err := catchPanic(func() { g.Group("") }); err == nil {
			t.Error("an error was expected with an empty path")
		}

		if err := catchPanic(func() { g.ServeFiles("static/{filepath:*}", "./") }); err == nil {
			t.Error("an error was expected when a path does not begin with slash")
		}

		if err := catchPanic(func() {
			g.ServeFilesCustom("", http.Dir("./"))
		}); err == nil {
			t.Error("an error was expected with an empty path")
		}

	}
}

func TestGroup(t *testing.T) {
	t.Skip("TODO")

	r1 := New()
	r2 := r1.Group("/boo")
	r3 := r1.Group("/goo")
	r4 := r1.Group("/moo")
	r5 := r4.Group("/foo")
	r6 := r5.Group("/foo")

	assertGroup(t, r1, r2, r3, r4, r5, r6)

	hit := false

	r1.POST("/foo", func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	})
	r2.POST("/bar", func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	})
	r3.POST("/bar", func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	})
	r4.POST("/bar", func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	})
	r5.POST("/bar", func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	})
	r6.POST("/bar", func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	})
	r6.ServeFiles("/static/{filepath:*}", "./")
	r6.ServeFilesCustom("/custom/static/{filepath:*}", http.Dir("./"))

	uris := []string{
		"POST /foo HTTP/1.1\r\n\r\n",
		// testing router group - r2 (grouped from r1)
		"POST /boo/bar HTTP/1.1\r\n\r\n",
		// testing multiple router group - r3 (grouped from r1)
		"POST /goo/bar HTTP/1.1\r\n\r\n",
		// testing multiple router group - r4 (grouped from r1)
		"POST /moo/bar HTTP/1.1\r\n\r\n",
		// testing sub-router group - r5 (grouped from r4)
		"POST /moo/foo/bar HTTP/1.1\r\n\r\n",
		// testing multiple sub-router group - r6 (grouped from r5)
		"POST /moo/foo/foo/bar HTTP/1.1\r\n\r\n",
		// testing multiple sub-router group - r6 (grouped from r5) to serve files
		"GET /moo/foo/foo/static/router.go HTTP/1.1\r\n\r\n",
		// testing multiple sub-router group - r6 (grouped from r5) to serve files with custom settings
		"GET /moo/foo/foo/custom/static/router.go HTTP/1.1\r\n\r\n",
	}

	for _, uri := range uris {
		hit = false

		assertWithTestServer(t, uri, r1.ServeHTTP, func(rw *readWriter) {
			br := bufio.NewReader(&rw.w)
			var resp fasthttp.Response
			if err := resp.Read(br); err != nil {
				t.Fatalf("Unexpected error when reading response: %s", err)
			}
			if !(resp.Header.StatusCode() == http.StatusOK) {
				t.Fatalf("Status code %d, want %d", resp.Header.StatusCode(), http.StatusOK)
			}
			if !strings.Contains(uri, "static") && !hit {
				t.Fatalf("Regular routing failed with router chaining. %s", uri)
			}
		})
	}

	assertWithTestServer(t, "POST /qax HTTP/1.1\r\n\r\n", r1.ServeHTTP, func(rw *readWriter) {
		br := bufio.NewReader(&rw.w)
		var resp fasthttp.Response
		if err := resp.Read(br); err != nil {
			t.Fatalf("Unexpected error when reading response: %s", err)
		}
		if !(resp.Header.StatusCode() == http.StatusNotFound) {
			t.Errorf("NotFound behavior failed with router chaining.")
			t.FailNow()
		}
	})
}

func TestGroup_shortcutsAndHandle(t *testing.T) {
	r := New()
	g := r.Group("/v1")

	shortcuts := []func(path string, handler http.HandlerFunc){
		g.GET,
		g.HEAD,
		g.POST,
		g.PUT,
		g.PATCH,
		g.DELETE,
		g.CONNECT,
		g.OPTIONS,
		g.TRACE,
		g.ANY,
	}

	for _, fn := range shortcuts {
		fn("/bar", func(w http.ResponseWriter, r *http.Request) {})

		if err := catchPanic(func() { fn("buzz", func(w http.ResponseWriter, r *http.Request) {}) }); err == nil {
			t.Error("an error was expected when a path does not begin with slash")
		}

		if err := catchPanic(func() { fn("", func(w http.ResponseWriter, r *http.Request) {}) }); err == nil {
			t.Error("an error was expected with an empty path")
		}
	}

	methods := httpMethods[:len(httpMethods)-1] // Avoid customs methods
	for _, method := range methods {
		h, _ := r.Lookup(method, "/v1/bar", nil)
		if h == nil {
			t.Errorf("Bad shorcurt")
		}
	}

	g2 := g.Group("/foo")

	for _, method := range httpMethods {
		g2.Handle(method, "/bar", func(w http.ResponseWriter, r *http.Request) {})

		if err := catchPanic(func() { g2.Handle(method, "buzz", func(w http.ResponseWriter, r *http.Request) {}) }); err == nil {
			t.Error("an error was expected when a path does not begin with slash")
		}

		if err := catchPanic(func() { g2.Handle(method, "", func(w http.ResponseWriter, r *http.Request) {}) }); err == nil {
			t.Error("an error was expected with an empty path")
		}

		h, _ := r.Lookup(method, "/v1/foo/bar", nil)
		if h == nil {
			t.Errorf("Bad shorcurt")
		}
	}
}
