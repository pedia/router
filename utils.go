package router

import (
	"net/http"
	"strings"

	"github.com/pedia/router/radix"
)

func validatePath(path string) {
	switch {
	case len(path) == 0 || !strings.HasPrefix(path, "/"):
		panic("path must begin with '/' in path '" + path + "'")
	}
}

func UserValue(r *http.Request, key string) string {
	if m := radix.UserValues(r); m != nil {
		return m[key]
	}
	return ""
}

func UserValues(r *http.Request) map[string]string {
	return radix.UserValues(r)
}
