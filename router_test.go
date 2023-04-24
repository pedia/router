package router

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

type readWriter struct {
	net.Conn
	r bytes.Buffer
	w bytes.Buffer
}

var httpMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
	MethodWild,
	"CUSTOM",
}

func randomHTTPMethod() string {
	method := httpMethods[rand.Intn(len(httpMethods)-1)]

	for method == MethodWild {
		method = httpMethods[rand.Intn(len(httpMethods)-1)]
	}

	return method
}

func buildLocation(host, path string) string {
	// return fmt.Sprintf("http://%s%s", host, path)
	return path
}

var zeroTCPAddr = &net.TCPAddr{
	IP: net.IPv4zero,
}

func (rw *readWriter) Close() error {
	return nil
}

func (rw *readWriter) Read(b []byte) (int, error) {
	return rw.r.Read(b)
}

func (rw *readWriter) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func (rw *readWriter) RemoteAddr() net.Addr {
	return zeroTCPAddr
}

func (rw *readWriter) LocalAddr() net.Addr {
	return zeroTCPAddr
}

func (rw *readWriter) SetReadDeadline(t time.Time) error {
	return nil
}

func (rw *readWriter) SetWriteDeadline(t time.Time) error {
	return nil
}

type assertFn func(rw *readWriter)

func assertWithTestServer(t *testing.T, uri string, handler http.HandlerFunc, fn assertFn) {
	// s := &http.Server{
	// 	Handler: handler,
	// }

	// rw := &readWriter{}
	// ch := make(chan error)

	// rw.r.WriteString(uri)
	// go func() {
	// 	ch <- s.ServeConn(rw)
	// }()
	// select {
	// case err := <-ch:
	// 	if err != nil {
	// 		t.Fatalf("return error %s", err)
	// 	}
	// case <-time.After(500 * time.Millisecond):
	// 	t.Fatalf("timeout")
	// }

	// fn(rw)
}

func catchPanic(testFunc func()) (recv interface{}) {
	defer func() {
		recv = recover()
	}()

	testFunc()
	return
}

func TestRouter(t *testing.T) {
	router := New()

	routed := false
	router.Handle(http.MethodGet, "/user/{name}", func(w http.ResponseWriter, r *http.Request) {
		routed = true
		want := "gopher"

		param, ok := UserValue(r, "name").(string)

		if !ok {
			t.Fatalf("wrong wildcard values: param value is nil")
		}

		if param != want {
			t.Fatalf("wrong wildcard values: want %s, got %s", want, param)
		}
	})

	r := httptest.NewRequest("GET", "/user/gopher", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if !routed {
		t.Fatal("routing failed")
	}
}

func TestRouterAPI(t *testing.T) {
	var handled, get, head, post, put, patch, delete, connect, options, trace, any bool

	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		handled = true
	}

	router := New()
	router.GET("/GET", func(w http.ResponseWriter, r *http.Request) {
		get = true
	})
	router.HEAD("/HEAD", func(w http.ResponseWriter, r *http.Request) {
		head = true
	})
	router.POST("/POST", func(w http.ResponseWriter, r *http.Request) {
		post = true
	})
	router.PUT("/PUT", func(w http.ResponseWriter, r *http.Request) {
		put = true
	})
	router.PATCH("/PATCH", func(w http.ResponseWriter, r *http.Request) {
		patch = true
	})
	router.DELETE("/DELETE", func(w http.ResponseWriter, r *http.Request) {
		delete = true
	})
	router.CONNECT("/CONNECT", func(w http.ResponseWriter, r *http.Request) {
		connect = true
	})
	router.OPTIONS("/OPTIONS", func(w http.ResponseWriter, r *http.Request) {
		options = true
	})
	router.TRACE("/TRACE", func(w http.ResponseWriter, r *http.Request) {
		trace = true
	})
	router.ANY("/ANY", func(w http.ResponseWriter, r *http.Request) {
		any = true
	})
	router.Handle(http.MethodGet, "/Handler", httpHandler)

	var request = func(method, path string) {
		r := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
	}

	request(http.MethodGet, "/GET")
	if !get {
		t.Error("routing GET failed")
	}

	request(http.MethodHead, "/HEAD")
	if !head {
		t.Error("routing HEAD failed")
	}

	request(http.MethodPost, "/POST")
	if !post {
		t.Error("routing POST failed")
	}

	request(http.MethodPut, "/PUT")
	if !put {
		t.Error("routing PUT failed")
	}

	request(http.MethodPatch, "/PATCH")
	if !patch {
		t.Error("routing PATCH failed")
	}

	request(http.MethodDelete, "/DELETE")
	if !delete {
		t.Error("routing DELETE failed")
	}

	request(http.MethodConnect, "/CONNECT")
	if !connect {
		t.Error("routing CONNECT failed")
	}

	request(http.MethodOptions, "/OPTIONS")
	if !options {
		t.Error("routing OPTIONS failed")
	}

	request(http.MethodTrace, "/TRACE")
	if !trace {
		t.Error("routing TRACE failed")
	}

	request(http.MethodGet, "/Handler")
	if !handled {
		t.Error("routing Handler failed")
	}

	for _, method := range httpMethods {
		request(method, "/ANY")
		if !any {
			t.Errorf("routing ANY failed - Method: %s", method)
		}

		any = false
	}
}

func TestRouterInvalidInput(t *testing.T) {
	router := New()

	handle := func(w http.ResponseWriter, r *http.Request) {}

	recv := catchPanic(func() {
		router.Handle("", "/", handle)
	})
	if recv == nil {
		t.Fatal("registering empty method did not panic")
	}

	recv = catchPanic(func() {
		router.GET("", handle)
	})
	if recv == nil {
		t.Fatal("registering empty path did not panic")
	}

	recv = catchPanic(func() {
		router.GET("noSlashRoot", handle)
	})
	if recv == nil {
		t.Fatal("registering path not beginning with '/' did not panic")
	}

	recv = catchPanic(func() {
		router.GET("/", nil)
	})
	if recv == nil {
		t.Fatal("registering nil handler did not panic")
	}
}

func TestRouterRegexUserValues(t *testing.T) {
	mux := New()
	mux.GET("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	v4 := mux.Group("/v4")
	id := v4.Group("/{id:^[1-9]\\d*}")
	var v1 interface{}
	id.GET("/click", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		v1 = UserValue(r, "id")

	})

	r := httptest.NewRequest("GET", "/v4/123/click", nil)
	mux.ServeHTTP(httptest.NewRecorder(), r)

	if v1 != "123" {
		t.Fatalf(`expected "123" in user value, got %q`, v1)
	}

	r = httptest.NewRequest("GET", "/metrics", nil)
	mux.ServeHTTP(httptest.NewRecorder(), r)

	if v1 != "123" {
		t.Fatalf(`expected "123" in user value, got %q`, v1)
	}
}

func TestRouterChaining(t *testing.T) {
	router1 := New()
	router2 := New()
	router1.NotFound = router2.ServeHTTP

	fooHit := false
	router1.POST("/foo", func(w http.ResponseWriter, r *http.Request) {
		fooHit = true
		w.WriteHeader(http.StatusOK)
	})

	barHit := false
	router2.POST("/bar", func(w http.ResponseWriter, r *http.Request) {
		barHit = true
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("POST", "/foo", nil)
	w := httptest.NewRecorder()
	router1.ServeHTTP(w, r)

	if !(w.Code == http.StatusOK && fooHit) {
		t.Errorf("Regular routing failed with router chaining.")
		t.FailNow()
	}

	r = httptest.NewRequest("POST", "/bar", nil)
	w = httptest.NewRecorder()
	router1.ServeHTTP(w, r)

	if !(w.Code == http.StatusOK && barHit) {
		t.Errorf("Chained routing failed with router chaining.")
		t.FailNow()
	}

	r = httptest.NewRequest("POST", "/qax", nil)
	w = httptest.NewRecorder()
	router1.ServeHTTP(w, r)

	if !(w.Code == http.StatusNotFound) {
		t.Errorf("NotFound behavior failed with router chaining.")
		t.FailNow()
	}
}

func TestRouterMutable(t *testing.T) {
	handler1 := func(w http.ResponseWriter, r *http.Request) {}
	handler2 := func(w http.ResponseWriter, r *http.Request) {}

	router := New()
	router.Mutable(true)

	if !router.treeMutable {
		t.Errorf("Router.treesMutables is false")
	}

	for _, method := range httpMethods {
		router.Handle(method, "/", handler1)
	}

	for method := range router.trees {
		if !router.trees[method].Mutable {
			t.Errorf("Method %d - Mutable == %v, want %v", method, router.trees[method].Mutable, true)
		}
	}

	routes := []string{
		"/",
		"/api/{version}",
		"/{filepath:*}",
		"/user{user:.*}",
	}

	router = New()

	for _, route := range routes {
		for _, method := range httpMethods {
			router.Handle(method, route, handler1)
		}

		for _, method := range httpMethods {
			err := catchPanic(func() {
				router.Handle(method, route, handler2)
			})

			if err == nil {
				t.Errorf("Mutable 'false' - Method %s - Route %s - Expected panic", method, route)
			}

			h, _ := router.Lookup(method, route, nil)
			if reflect.ValueOf(h).Pointer() != reflect.ValueOf(handler1).Pointer() {
				t.Errorf("Mutable 'false' - Method %s - Route %s - Handler updated", method, route)
			}
		}

		router.Mutable(true)

		for _, method := range httpMethods {
			err := catchPanic(func() {
				router.Handle(method, route, handler2)
			})

			if err != nil {
				t.Errorf("Mutable 'true' - Method %s - Route %s - Unexpected panic: %v", method, route, err)
			}

			h, _ := router.Lookup(method, route, nil)
			if reflect.ValueOf(h).Pointer() != reflect.ValueOf(handler2).Pointer() {
				t.Errorf("Method %s - Route %s - Handler is not updated", method, route)
			}
		}

		router.Mutable(false)
	}

}

func TestRouterOPTIONS(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {}

	router := New()
	router.POST("/path", handlerFunc)

	var checkHandling = func(path, expectedAllowed string, expectedStatusCode int) {
		r := httptest.NewRequest(http.MethodOptions, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if !(w.Code == expectedStatusCode) {
			t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, "w.Header")
		} else if allow := w.Header().Get("Allow"); allow != expectedAllowed {
			t.Error("unexpected Allow header value: " + allow)
		}
	}

	// test not allowed
	// * (server)
	checkHandling("*", "OPTIONS, POST", http.StatusOK)

	// path
	checkHandling("/path", "OPTIONS, POST", http.StatusOK)

	r := httptest.NewRequest(http.MethodOptions, "/doesnotexist", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNotFound) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, r.Header)
	}

	// add another method
	router.GET("/path", handlerFunc)

	// set a global OPTIONS handler
	router.GlobalOPTIONS = func(w http.ResponseWriter, r *http.Request) {
		// Adjust status code to 204
		w.WriteHeader(http.StatusNoContent)
	}

	// test again
	// * (server)
	checkHandling("*", "GET, OPTIONS, POST", http.StatusNoContent)

	// path
	checkHandling("/path", "GET, OPTIONS, POST", http.StatusNoContent)

	// custom handler
	var custom bool
	router.OPTIONS("/path", func(w http.ResponseWriter, r *http.Request) {
		custom = true
	})

	// test again
	// * (server)
	checkHandling("*", "GET, OPTIONS, POST", http.StatusNoContent)
	if custom {
		t.Error("custom handler called on *")
	}

	// path
	r = httptest.NewRequest(http.MethodOptions, "/path", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d, Header=%v", w.Code, "w.Header")
	}
	if !custom {
		t.Error("custom handler not called")
	}
}

func TestRouterNotAllowed(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {}

	router := New()
	router.POST("/path", handlerFunc)

	var checkHandling = func(path, expectedAllowed string, expectedStatusCode int) {
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if !(w.Code == expectedStatusCode) {
			t.Errorf("NotAllowed handling failed:: Code=%d, Header=%v", w.Code, "w.Header")
		} else if allow := w.Header().Get("Allow"); allow != expectedAllowed {
			t.Error("unexpected Allow header value: " + allow)
		}
	}

	// test not allowed
	checkHandling("/path", "OPTIONS, POST", http.StatusMethodNotAllowed)

	// add another method
	router.DELETE("/path", handlerFunc)
	router.OPTIONS("/path", handlerFunc) // must be ignored

	// test again
	checkHandling("/path", "DELETE, OPTIONS, POST", http.StatusMethodNotAllowed)

	// test custom handler
	responseText := "custom method"
	router.MethodNotAllowed = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte(responseText))
	}

	r := httptest.NewRequest("foo", "/path", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if got := w.Body.String(); !(got == responseText) {
		t.Errorf("unexpected response got %q want %q", got, responseText)
	}
	if w.Code != http.StatusTeapot {
		t.Errorf("unexpected response code %d want %d", w.Code, http.StatusTeapot)
	}
	if allow := string(w.Header().Get("Allow")); allow != "DELETE, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}
}

func testRouterNotFoundByMethod(t *testing.T, method string) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {}
	host := "fast"

	router := New()
	router.Handle(method, "/path", handlerFunc)
	router.Handle(method, "/dir/", handlerFunc)
	router.Handle(method, "/", handlerFunc)
	router.Handle(method, "/{proc}/StaTus", handlerFunc)
	router.Handle(method, "/USERS/{name}/enTRies/", handlerFunc)
	router.Handle(method, "/static/{filepath:*}", handlerFunc)

	reqMethod := method
	if method == MethodWild {
		reqMethod = randomHTTPMethod()
	}

	// Moved Permanently, request with GET method
	expectedCode := http.StatusMovedPermanently
	switch {
	case reqMethod == http.MethodConnect:
		// CONNECT method does not allow redirects, so Not Found (404)
		expectedCode = http.StatusNotFound
	case reqMethod != http.MethodGet:
		// Permanent Redirect, request with same method
		expectedCode = http.StatusPermanentRedirect
	}

	type testRoute struct {
		route    string
		code     int
		location string
	}

	testRoutes := []testRoute{
		// {"", http.StatusOK, ""},                                  // TSR +/ (Not clean by router, this path is cleaned by fasthttp `ctx.Path()`)
		// {"/../path", expectedCode, buildLocation(host, "/path")}, // CleanPath (Not clean by router, this path is cleaned by fasthttp `ctx.Path()`)
		{"/nope", http.StatusNotFound, ""}, // NotFound
	}

	if method != http.MethodConnect {
		testRoutes = append(testRoutes, []testRoute{
			{"/path/", expectedCode, buildLocation(host, "/path")}, // TSR -/
			{"/dir", expectedCode, buildLocation(host, "/dir/")},   // TSR +/
			// {"/PATH", expectedCode, buildLocation(host, "/path")},                                    // Fixed Case
			// {"/DIR/", expectedCode, buildLocation(host, "/dir/")},                                    // Fixed Case
			// {"/PATH/", expectedCode, buildLocation(host, "/path")},                                   // Fixed Case -/
			// {"/DIR", expectedCode, buildLocation(host, "/dir/")},                                     // Fixed Case +/
			// {"/paTh/?name=foo", expectedCode, buildLocation(host, "/path?name=foo")},                 // Fixed Case With Query Params +/
			// {"/paTh?name=foo", expectedCode, buildLocation(host, "/path?name=foo")},                  // Fixed Case With Query Params +/
			// {"/sergio/status/", expectedCode, buildLocation(host, "/sergio/StaTus")},                 // Fixed Case With Params -/
			// {"/users/atreugo/eNtriEs", expectedCode, buildLocation(host, "/USERS/atreugo/enTRies/")}, // Fixed Case With Params +/
			// {"/STatiC/test.go", expectedCode, buildLocation(host, "/static/test.go")},                // Fixed Case Wildcard
		}...)
	}

	for _, tr := range testRoutes {
		r := httptest.NewRequest(reqMethod, tr.route, nil)
		r.Host = host
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		statusCode := w.Code
		location := string(w.Header().Get("Location"))

		if !(statusCode == tr.code && (statusCode == http.StatusNotFound || location == tr.location)) {
			fn := t.Errorf
			msg := "NotFound handling route '%s' failed: Method=%s, ReqMethod=%s, Code=%d, ExpectedCode=%d, Header=%v"

			if runtime.GOOS == "windows" && strings.HasPrefix(tr.route, "/../") {
				// See: https://github.com/valyala/fasthttp/issues/1226
				// Not fail, because it is a known issue.
				fn = t.Logf
				msg = "ERROR: " + msg
			}

			fn(msg, tr.route, method, reqMethod, statusCode, tr.code, location)
		}
	}

	// Test custom not found handler
	var notFound bool
	router.NotFound = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		notFound = true
	}

	r := httptest.NewRequest(reqMethod, "/nope", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if !(w.Code == http.StatusNotFound && notFound == true) {
		t.Errorf(
			"Custom NotFound handling failed: Method=%s, ReqMethod=%s, Code=%d, Header=%v",
			method, reqMethod, w.Code, "w.Header",
		)
	}
}

func TestRouterNotFound(t *testing.T) {
	for _, method := range httpMethods {
		testRouterNotFoundByMethod(t, method)
	}

	router := New()
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {}
	host := "fast"

	// Test other method than GET (want 308 instead of 301)
	router.PATCH("/path", handlerFunc)

	r := httptest.NewRequest(http.MethodPatch, "/path/?key=val", nil)
	r.Host = host
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusPermanentRedirect) &&
		w.Header().Get("Location") == buildLocation(host, "/path?key=val") {
		t.Errorf("Custom NotFound handler failed: Code=%d, Header=%v", w.Code, "w.Header")
	}

	// Test special case where no node for the prefix "/" exists
	router = New()
	router.GET("/a", handlerFunc)

	r = httptest.NewRequest(http.MethodPatch, "/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !(w.Code == http.StatusNotFound) {
		t.Errorf("NotFound handling route / failed: Code=%d", w.Code)
	}
}

func TestRouterNotFound_MethodWild(t *testing.T) {
	postFound, anyFound := false, false

	router := New()
	router.ANY("/{path:*}", func(w http.ResponseWriter, r *http.Request) { anyFound = true })
	router.POST("/specific", func(w http.ResponseWriter, r *http.Request) { postFound = true })

	for i := 0; i < 100; i++ {
		router.Handle(
			randomHTTPMethod(),
			fmt.Sprintf("/%d", rand.Int63()),
			func(w http.ResponseWriter, r *http.Request) {},
		)
	}

	var request = func(method, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}

	for _, method := range httpMethods {
		w := request(method, "/specific")

		if method == http.MethodPost {
			if !postFound {
				t.Errorf("Method '%s': not found", method)
			}
		} else {
			if !anyFound {
				t.Errorf("Method 'ANY' not found with request method %s", method)
			}
		}

		status := w.Code
		if status != http.StatusOK {
			t.Errorf("Response status code == %d, want %d", status, http.StatusOK)
		}

		postFound, anyFound = false, false
	}
}

func TestRouterPanicHandler(t *testing.T) {
	router := New()
	panicHandled := false

	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, p interface{}) {
		panicHandled = true
	}

	router.Handle(http.MethodPut, "/user/{name}", func(w http.ResponseWriter, r *http.Request) {
		panic("oops!")
	})

	r := httptest.NewRequest(http.MethodPut, "/user/gopher", nil)
	w := httptest.NewRecorder()

	defer func() {
		if rcv := recover(); rcv != nil {
			t.Fatal("handling panic failed")
		}
	}()

	router.ServeHTTP(w, r)

	if !panicHandled {
		t.Fatal("simulating failed")
	}
}

func testRouterLookupByMethod(t *testing.T, method string) {
	reqMethod := method
	if method == MethodWild {
		reqMethod = randomHTTPMethod()
	}

	routed := false
	wantHandle := func(w http.ResponseWriter, r *http.Request) {
		routed = true
	}
	wantParams := map[string]string{"name": "gopher"}

	r := httptest.NewRequest("GET", "/nope", nil)
	router := New()

	// try empty router first
	handle, tsr := router.Lookup(reqMethod, "/nope", r)
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}

	// insert route and try again
	router.Handle(method, "/user/{name}", wantHandle)
	handle, _ = router.Lookup(reqMethod, "/user/gopher", r)
	w := httptest.NewRecorder()
	if handle == nil {
		t.Fatal("Got no handle!")
	} else {
		handle(w, r)
		if !routed {
			t.Fatal("Routing failed!")
		}
	}

	for expectedKey, expectedVal := range wantParams {
		_ = expectedKey
		_ = expectedVal
		// if ctx.UserValue(expectedKey) != expectedVal {
		// 	t.Errorf("The values %s = %s is not save in context", expectedKey, expectedVal)
		// }
	}

	routed = false

	// route without param
	router.Handle(method, "/user", wantHandle)
	handle, _ = router.Lookup(reqMethod, "/user", r)
	if handle == nil {
		t.Fatal("Got no handle!")
	} else {
		handle(w, r)
		if !routed {
			t.Fatal("Routing failed!")
		}
	}

	for expectedKey, expectedVal := range wantParams {
		_ = expectedKey
		_ = expectedVal
		// if ctx.UserValue(expectedKey) != expectedVal {
		// 	t.Errorf("The values %s = %s is not save in context", expectedKey, expectedVal)
		// }
	}

	handle, tsr = router.Lookup(reqMethod, "/user/gopher/", r)
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if !tsr {
		t.Error("Got no TSR recommendation!")
	}

	handle, tsr = router.Lookup(reqMethod, "/nope", r)
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}
}

func TestRouterLookup(t *testing.T) {
	for _, method := range httpMethods {
		testRouterLookupByMethod(t, method)
	}
}

func TestRouterMatchedRoutePath(t *testing.T) {
	route1 := "/user/{name}"
	routed1 := false
	handle1 := func(w http.ResponseWriter, r *http.Request) {
		route := UserValue(r, MatchedRoutePathParam)
		if route != route1 {
			t.Fatalf("Wrong matched route: want %s, got %s", route1, route)
		}
		routed1 = true
	}

	route2 := "/user/{name}/details"
	routed2 := false
	handle2 := func(w http.ResponseWriter, r *http.Request) {
		route := UserValue(r, MatchedRoutePathParam)
		if route != route2 {
			t.Fatalf("Wrong matched route: want %s, got %s", route2, route)
		}
		routed2 = true
	}

	route3 := "/"
	routed3 := false
	handle3 := func(w http.ResponseWriter, r *http.Request) {
		route := UserValue(r, MatchedRoutePathParam)
		if route != route3 {
			t.Fatalf("Wrong matched route: want %s, got %s", route3, route)
		}
		routed3 = true
	}

	router := New()
	router.SaveMatchedRoutePath = true
	router.Handle(http.MethodGet, route1, handle1)
	router.Handle(http.MethodGet, route2, handle2)
	router.Handle(http.MethodGet, route3, handle3)

	r := httptest.NewRequest(http.MethodGet, "/user/gopher", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !routed1 || routed2 || routed3 {
		t.Fatal("Routing failed!")
	}

	r = httptest.NewRequest(http.MethodGet, "/user/gopher/details", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !routed2 || routed3 {
		t.Fatal("Routing failed!")
	}

	r = httptest.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if !routed3 {
		t.Fatal("Routing failed!")
	}
}

// func TestRouterServeFiles(t *testing.T) {
// 	r := New()

// 	recv := catchPanic(func() {
// 		r.ServeFiles("/noFilepath", os.TempDir())
// 	})
// 	if recv == nil {
// 		t.Fatal("registering path not ending with '{filepath:*}' did not panic")
// 	}
// 	body := []byte("fake ico")
// 	ioutil.WriteFile(os.TempDir()+"/favicon.ico", body, 0644)

// 	r.ServeFiles("/{filepath:*}", os.TempDir())

// 	assertWithTestServer(t, "GET /favicon.ico HTTP/1.1\r\n\r\n", r.Handler, func(rw *readWriter) {
// 		br := bufio.NewReader(&rw.w)
// 		var resp fasthttp.Response
// 		if err := resp.Read(br); err != nil {
// 			t.Fatalf("Unexpected error when reading response: %s", err)
// 		}
// 		if resp.Header.StatusCode() != 200 {
// 			t.Fatalf("Unexpected status code %d. Expected %d", resp.Header.StatusCode(), 200)
// 		}
// 		if !bytes.Equal(resp.Body(), body) {
// 			t.Fatalf("Unexpected body %q. Expected %q", resp.Body(), string(body))
// 		}
// 	})
// }

// func TestRouterServeFilesCustom(t *testing.T) {
// 	r := New()

// 	root := os.TempDir()

// 	fs := &fasthttp.FS{
// 		Root: root,
// 	}

// 	recv := catchPanic(func() {
// 		r.ServeFilesCustom("/noFilepath", fs)
// 	})
// 	if recv == nil {
// 		t.Fatal("registering path not ending with '{filepath:*}' did not panic")
// 	}
// 	body := []byte("fake ico")
// 	ioutil.WriteFile(root+"/favicon.ico", body, 0644)

// 	r.ServeFilesCustom("/{filepath:*}", fs)

// 	assertWithTestServer(t, "GET /favicon.ico HTTP/1.1\r\n\r\n", r.Handler, func(rw *readWriter) {
// 		br := bufio.NewReader(&rw.w)
// 		var resp fasthttp.Response
// 		if err := resp.Read(br); err != nil {
// 			t.Fatalf("Unexpected error when reading response: %s", err)
// 		}
// 		if resp.Header.StatusCode() != 200 {
// 			t.Fatalf("Unexpected status code %d. Expected %d", resp.Header.StatusCode(), 200)
// 		}
// 		if !bytes.Equal(resp.Body(), body) {
// 			t.Fatalf("Unexpected body %q. Expected %q", resp.Body(), string(body))
// 		}
// 	})
// }

func TestRouterList(t *testing.T) {
	expected := map[string][]string{
		"GET":    {"/bar"},
		"PATCH":  {"/foo"},
		"POST":   {"/v1/users/{name}/{surname?}"},
		"DELETE": {"/v1/users/{id?}"},
	}

	r := New()
	r.GET("/bar", func(w http.ResponseWriter, r *http.Request) {})
	r.PATCH("/foo", func(w http.ResponseWriter, r *http.Request) {})

	v1 := r.Group("/v1")
	v1.POST("/users/{name}/{surname?}", func(w http.ResponseWriter, r *http.Request) {})
	v1.DELETE("/users/{id?}", func(w http.ResponseWriter, r *http.Request) {})

	result := r.List()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Router.List() == %v, want %v", result, expected)
	}

}

func TestRouterSamePrefixParamRoute(t *testing.T) {
	var id1, id2, id3, pageSize, page, iid string
	var routed1, routed2, routed3 bool

	router := New()
	v1 := router.Group("/v1")
	v1.GET("/foo/{id}/{pageSize}/{page}", func(w http.ResponseWriter, r *http.Request) {
		id1 = UserValue(r, "id").(string)
		pageSize = UserValue(r, "pageSize").(string)
		page = UserValue(r, "page").(string)
		routed1 = true
	})
	v1.GET("/foo/{id}/{iid}", func(w http.ResponseWriter, r *http.Request) {
		id2 = UserValue(r, "id").(string)
		iid = UserValue(r, "iid").(string)
		routed2 = true
	})
	v1.GET("/foo/{id}", func(w http.ResponseWriter, r *http.Request) {
		id3 = UserValue(r, "id").(string)
		routed3 = true
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/foo/1/20/4", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/foo/2/3", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/foo/v3", nil))

	if !routed1 {
		t.Error("/foo/{id}/{pageSize}/{page} not routed.")
	}
	if !routed2 {
		t.Error("/foo/{id}/{iid} not routed")
	}

	if !routed3 {
		t.Error("/foo/{id} not routed")
	}

	if id1 != "1" {
		t.Errorf("/foo/{id}/{pageSize}/{page} id expect: 1 got %s", id1)
	}

	if pageSize != "20" {
		t.Errorf("/foo/{id}/{pageSize}/{page} pageSize expect: 20 got %s", pageSize)
	}

	if page != "4" {
		t.Errorf("/foo/{id}/{pageSize}/{page} page expect: 4 got %s", page)
	}

	if id2 != "2" {
		t.Errorf("/foo/{id}/{iid} id expect: 2 got %s", id2)
	}

	if iid != "3" {
		t.Errorf("/foo/{id}/{iid} iid expect: 3 got %s", iid)
	}

	if id3 != "v3" {
		t.Errorf("/foo/{id} id expect: v3 got %s", id3)
	}
}

func BenchmarkAllowed(b *testing.B) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {}

	router := New()
	router.POST("/path", handlerFunc)
	router.GET("/path", handlerFunc)

	b.Run("Global", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = router.allowed("*", http.MethodOptions)
		}
	})
	b.Run("Path", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = router.allowed("/path", http.MethodOptions)
		}
	})
}

func BenchmarkRouterGet(b *testing.B) {
	router := New()
	router.GET("/hello", func(w http.ResponseWriter, r *http.Request) {})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/hello", nil)

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, r)
	}
}

func BenchmarkRouterParams(b *testing.B) {
	r := New()
	r.GET("/{id}", func(w http.ResponseWriter, r *http.Request) {})

	w := httptest.NewRecorder()
	r0 := httptest.NewRequest(http.MethodGet, "/hello", nil)

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, r0)
	}
}

func BenchmarkRouterANY(b *testing.B) {
	r := New()
	r.GET("/data", func(w http.ResponseWriter, r *http.Request) {})
	r.ANY("/", func(w http.ResponseWriter, r *http.Request) {})

	w := httptest.NewRecorder()
	r0 := httptest.NewRequest(http.MethodGet, "/", nil)

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, r0)
	}
}

func BenchmarkRouterGet_ANY(b *testing.B) {
	resp := []byte("Bench GET")
	respANY := []byte("Bench GET (ANY)")

	r := New()
	r.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "text/plain")
		w.Write(resp)
	})
	r.ANY("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-type", "text/plain")
		w.Write(respANY)
	})

	w := httptest.NewRecorder()
	r0 := httptest.NewRequest("UNICORN", "/", nil)

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, r0)
	}
}

func BenchmarkRouterNotFound(b *testing.B) {
	r := New()
	r.GET("/bench", func(w http.ResponseWriter, r *http.Request) {})

	w := httptest.NewRecorder()
	r0 := httptest.NewRequest(http.MethodGet, "/notfound", nil)

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, r0)
	}
}

func BenchmarkRouterFindCaseInsensitive(b *testing.B) {
	r := New()
	r.GET("/bench", func(w http.ResponseWriter, r *http.Request) {})

	w := httptest.NewRecorder()
	r0 := httptest.NewRequest(http.MethodGet, "/BenCh/.", nil)

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, r0)
	}
}

func BenchmarkRouterRedirectTrailingSlash(b *testing.B) {
	r := New()
	r.GET("/bench/", func(w http.ResponseWriter, r *http.Request) {})

	w := httptest.NewRecorder()
	r0 := httptest.NewRequest(http.MethodGet, "/bench", nil)

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, r0)
	}
}

func Benchmark_Get(b *testing.B) {
	handler := func(w http.ResponseWriter, r *http.Request) {}

	r := New()

	r.GET("/", handler)
	r.GET("/plaintext", handler)
	r.GET("/json", handler)
	r.GET("/fortune", handler)
	r.GET("/fortune-quick", handler)
	r.GET("/db", handler)
	r.GET("/queries", handler)
	r.GET("/update", handler)

	w := httptest.NewRecorder()
	r0 := httptest.NewRequest(http.MethodGet, "/update", nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, r0)
	}
}
