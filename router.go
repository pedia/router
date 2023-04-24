package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pedia/router/radix"
	"github.com/savsgio/gotils/bytes"
	"github.com/valyala/bytebufferpool"
)

// MethodWild wild HTTP method
const MethodWild = "*"

var (
	defaultContentType = []byte("text/plain; charset=utf-8")
	questionMark       = byte('?')

	// MatchedRoutePathParam is the param name under which the path of the matched
	// route is stored, if Router.SaveMatchedRoutePath is set.
	MatchedRoutePathParam = fmt.Sprintf("__matchedRoutePath::%s__", bytes.Rand(make([]byte, 15)))
)

// New returns a new router.
// Path auto-correction, including trailing slashes, is enabled by default.
func New() *Router {
	return &Router{
		trees:                  make([]*radix.Tree, 10),
		customMethodsIndex:     make(map[string]int),
		registeredPaths:        make(map[string][]string),
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
	}
}

// Group returns a new group.
// Path auto-correction, including trailing slashes, is enabled by default.
func (router *Router) Group(path string) *Group {
	validatePath(path)

	if path != "/" && strings.HasSuffix(path, "/") {
		panic("group path must not end with a trailing slash")
	}

	return &Group{
		router: router,
		prefix: path,
	}
}

func (router *Router) saveMatchedRoutePath(path string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ctx.SetUserValue(MatchedRoutePathParam, path)
		r = radix.AddRequestValue(r, MatchedRoutePathParam, path)
		handler(w, r)
	}
}

func (router *Router) methodIndexOf(method string) int {
	switch method {
	case http.MethodGet:
		return 0
	case http.MethodHead:
		return 1
	case http.MethodPost:
		return 2
	case http.MethodPut:
		return 3
	case http.MethodPatch:
		return 4
	case http.MethodDelete:
		return 5
	case http.MethodConnect:
		return 6
	case http.MethodOptions:
		return 7
	case http.MethodTrace:
		return 8
	case MethodWild:
		return 9
	}

	if i, ok := router.customMethodsIndex[method]; ok {
		return i
	}

	return -1
}

// Mutable allows updating the route handler
//
// # It's disabled by default
//
// WARNING: Use with care. It could generate unexpected behaviours
func (router *Router) Mutable(v bool) {
	router.treeMutable = v

	for i := range router.trees {
		tree := router.trees[i]

		if tree != nil {
			tree.Mutable = v
		}
	}
}

// List returns all registered routes grouped by method
func (router *Router) List() map[string][]string {
	return router.registeredPaths
}

// GET is a shortcut for router.Handle(http.MethodGet, path, handler)
func (router *Router) GET(path string, handler http.HandlerFunc) {
	router.Handle(http.MethodGet, path, handler)
}

// HEAD is a shortcut for router.Handle(http.MethodHead, path, handler)
func (router *Router) HEAD(path string, handler http.HandlerFunc) {
	router.Handle(http.MethodHead, path, handler)
}

// POST is a shortcut for router.Handle(http.MethodPost, path, handler)
func (router *Router) POST(path string, handler http.HandlerFunc) {
	router.Handle(http.MethodPost, path, handler)
}

// PUT is a shortcut for router.Handle(http.MethodPut, path, handler)
func (router *Router) PUT(path string, handler http.HandlerFunc) {
	router.Handle(http.MethodPut, path, handler)
}

// PATCH is a shortcut for router.Handle(http.MethodPatch, path, handler)
func (router *Router) PATCH(path string, handler http.HandlerFunc) {
	router.Handle(http.MethodPatch, path, handler)
}

// DELETE is a shortcut for router.Handle(http.MethodDelete, path, handler)
func (router *Router) DELETE(path string, handler http.HandlerFunc) {
	router.Handle(http.MethodDelete, path, handler)
}

// CONNECT is a shortcut for router.Handle(http.MethodConnect, path, handler)
func (router *Router) CONNECT(path string, handler http.HandlerFunc) {
	router.Handle(http.MethodConnect, path, handler)
}

// OPTIONS is a shortcut for router.Handle(http.MethodOptions, path, handler)
func (router *Router) OPTIONS(path string, handler http.HandlerFunc) {
	router.Handle(http.MethodOptions, path, handler)
}

// TRACE is a shortcut for router.Handle(http.MethodTrace, path, handler)
func (router *Router) TRACE(path string, handler http.HandlerFunc) {
	router.Handle(http.MethodTrace, path, handler)
}

// ANY is a shortcut for router.Handle(router.MethodWild, path, handler)
//
// WARNING: Use only for routes where the request method is not important
func (router *Router) ANY(path string, handler http.HandlerFunc) {
	router.Handle(MethodWild, path, handler)
}

// ServeFiles serves files from the given file system root.
// The path must end with "/{filepath:*}", files are then served from the local
// path /defined/root/dir/{filepath:*}.
// For example if root is "/etc" and {filepath:*} is "passwd", the local file
// "/etc/passwd" would be served.
// Internally a fasthttp.FSHandler is used, therefore fasthttp.NotFound is used instead
// Use:
//
//	router.ServeFiles("/src/{filepath:*}", "./")
func (router *Router) ServeFiles(path string, rootPath string) {
	router.ServeFilesCustom(path, http.Dir(rootPath))
	// {
	// 	Root:               rootPath,
	// 	IndexNames:         []string{"index.html"},
	// 	GenerateIndexPages: true,
	// 	AcceptByteRange:    true,
	// })
}

// ServeFilesCustom serves files from the given file system settings.
// The path must end with "/{filepath:*}", files are then served from the local
// path /defined/root/dir/{filepath:*}.
// For example if root is "/etc" and {filepath:*} is "passwd", the local file
// "/etc/passwd" would be served.
// Internally a fasthttp.FSHandler is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// Use:
//
//	router.ServeFilesCustom("/src/{filepath:*}", *customFS)
func (router *Router) ServeFilesCustom(path string, fs http.FileSystem) {
	suffix := "/{filepath:*}"

	if !strings.HasSuffix(path, suffix) {
		panic("path must end with " + suffix + " in path '" + path + "'")
	}

	// TODO:
	// prefix := path[:len(path)-len(suffix)]
	// stripSlashes := strings.Count(prefix, "/")

	// if fs.PathRewrite == nil && stripSlashes > 0 {
	// 	fs.PathRewrite = fasthttp.NewPathSlashesStripper(stripSlashes)
	// }
	fileServer := http.FileServer(fs)

	router.GET(path, fileServer.ServeHTTP)
}

// Handle registers a new request handler with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (router *Router) Handle(method, path string, handler http.HandlerFunc) {
	switch {
	case len(method) == 0:
		panic("method must not be empty")
	case handler == nil:
		panic("handler must not be nil")
	default:
		validatePath(path)
	}

	router.registeredPaths[method] = append(router.registeredPaths[method], path)

	methodIndex := router.methodIndexOf(method)
	if methodIndex == -1 {
		tree := radix.New()
		tree.Mutable = router.treeMutable

		router.trees = append(router.trees, tree)
		methodIndex = len(router.trees) - 1
		router.customMethodsIndex[method] = methodIndex
	}

	tree := router.trees[methodIndex]
	if tree == nil {
		tree = radix.New()
		tree.Mutable = router.treeMutable

		router.trees[methodIndex] = tree
		router.globalAllowed = router.allowed("*", "")
	}

	if router.SaveMatchedRoutePath {
		handler = router.saveMatchedRoutePath(path, handler)
	}

	optionalPaths := getOptionalPaths(path)

	// if not has optional paths, adds the original
	if len(optionalPaths) == 0 {
		tree.Add(path, handler)
	} else {
		for _, p := range optionalPaths {
			tree.Add(p, handler)
		}
	}
}

// Lookup allows the manual lookup of a method + path combo.
// This is e.g. useful to build a framework around this router.
// If the path was found, it returns the handler function.
// Otherwise the second return value indicates whether a redirection to
// the same path with an extra / without the trailing slash should be performed.
func (router *Router) Lookup(method, path string, r *http.Request) (http.HandlerFunc, bool) {
	methodIndex := router.methodIndexOf(method)
	if methodIndex == -1 {
		return nil, false
	}

	if tree := router.trees[methodIndex]; tree != nil {
		handler, tsr := tree.Get(path, r)
		if handler != nil || tsr {
			return handler, tsr
		}
	}

	if tree := router.trees[router.methodIndexOf(MethodWild)]; tree != nil {
		return tree.Get(path, r)
	}

	return nil, false
}

func (router *Router) recv(w http.ResponseWriter, r *http.Request) {
	if rcv := recover(); rcv != nil {
		router.PanicHandler(w, r, rcv)
	}
}

func (router *Router) allowed(path, reqMethod string) (allow string) {
	allowed := make([]string, 0, 9)

	if path == "*" || path == "/*" { // server-wide{ // server-wide
		// empty method is used for internal calls to refresh the cache
		if reqMethod == "" {
			for method := range router.registeredPaths {
				if method == http.MethodOptions {
					continue
				}
				// Add request method to list of allowed methods
				allowed = append(allowed, method)
			}
		} else {
			return router.globalAllowed
		}
	} else { // specific path
		for method := range router.registeredPaths {
			// Skip the requested method - we already tried this one
			if method == reqMethod || method == http.MethodOptions {
				continue
			}

			handle, _ := router.trees[router.methodIndexOf(method)].Get(path, nil)
			if handle != nil {
				// Add request method to list of allowed methods
				allowed = append(allowed, method)
			}
		}
	}

	if len(allowed) > 0 {
		// Add request method to list of allowed methods
		allowed = append(allowed, http.MethodOptions)

		// Sort allowed methods.
		// sort.Strings(allowed) unfortunately causes unnecessary allocations
		// due to allowed being moved to the heap and interface conversion
		for i, l := 1, len(allowed); i < l; i++ {
			for j := i; j > 0 && allowed[j] < allowed[j-1]; j-- {
				allowed[j], allowed[j-1] = allowed[j-1], allowed[j]
			}
		}

		// return as comma separated list
		return strings.Join(allowed, ", ")
	}
	return
}

func (router *Router) tryRedirect(w http.ResponseWriter, r *http.Request, tree *radix.Tree, tsr bool, method, path string) bool {
	// Moved Permanently, request with GET method
	code := http.StatusMovedPermanently
	if method != http.MethodGet {
		// Permanent Redirect, request with same method
		code = http.StatusPermanentRedirect
	}

	if tsr && router.RedirectTrailingSlash {
		uri := bytebufferpool.Get()

		if len(path) > 1 && path[len(path)-1] == '/' {
			uri.SetString(path[:len(path)-1])
		} else {
			uri.SetString(path)
			uri.WriteByte('/')
		}

		if queryBuf := r.URL.RawQuery; len(queryBuf) > 0 {
			uri.WriteByte(questionMark)
			uri.Write([]byte(queryBuf))
		}

		http.Redirect(w, r, uri.String(), code)
		// ctx.Redirect(uri.String(), code)
		bytebufferpool.Put(uri)

		return true
	}

	// Try to fix the request path
	if router.RedirectFixedPath {
		path2 := r.URL.RawPath

		uri := bytebufferpool.Get()
		found := tree.FindCaseInsensitivePath(
			cleanPath(path2),
			router.RedirectTrailingSlash,
			uri,
		)

		if found {
			if queryBuf := r.URL.RawQuery; len(queryBuf) > 0 {
				uri.WriteByte(questionMark)
				uri.Write([]byte(queryBuf))
			}

			// ctx.Redirect(uri.String(), code)
			http.Redirect(w, r, uri.String(), code)
			bytebufferpool.Put(uri)

			return true
		}

		bytebufferpool.Put(uri)
	}

	return false
}

// Handler makes the router implement the http.Handler interface.
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if router.PanicHandler != nil {
		defer router.recv(w, r)
	}

	path := r.URL.Path
	method := r.Method
	methodIndex := router.methodIndexOf(method)

	if methodIndex > -1 {
		if tree := router.trees[methodIndex]; tree != nil {
			if handler, tsr := tree.Get(path, r); handler != nil {
				handler.ServeHTTP(w, r)
				return
			} else if method != http.MethodConnect && path != "/" {
				if ok := router.tryRedirect(w, r, tree, tsr, method, path); ok {
					return
				}
			}
		}
	}

	// Try to search in the wild method tree
	if tree := router.trees[router.methodIndexOf(MethodWild)]; tree != nil {
		if handler, tsr := tree.Get(path, r); handler != nil {
			handler.ServeHTTP(w, r)
			return
		} else if method != http.MethodConnect && path != "/" {
			if ok := router.tryRedirect(w, r, tree, tsr, method, path); ok {
				return
			}
		}
	}

	if router.HandleOPTIONS && method == http.MethodOptions {
		// Handle OPTIONS requests

		if allow := router.allowed(path, http.MethodOptions); allow != "" {
			w.Header().Set("Allow", allow)
			if router.GlobalOPTIONS != nil {
				router.GlobalOPTIONS.ServeHTTP(w, r)
			}
			return
		}
	} else if router.HandleMethodNotAllowed {
		// Handle 405

		if allow := router.allowed(path, method); allow != "" {
			w.Header().Set("Allow", allow)
			if router.MethodNotAllowed != nil {
				router.MethodNotAllowed.ServeHTTP(w, r)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}
	}

	// Handle 404
	if router.NotFound != nil {
		router.NotFound.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
