package plumber

import (
	"log"
	"net/http"
)

//-----------------------------------------------------------------------------

//Plumb creates a pipeline if middlewares
func Plumb(middlewares ...interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(middlewares) == 0 {
			return
		}

		middlewares = flatten(middlewares)

		var final http.Handler

		for i := len(middlewares) - 1; i >= 0; i-- {
			mid := middlewares[i]

			if mid == nil {
				continue
			}

			switch current := mid.(type) {
			case func(http.ResponseWriter, *http.Request):
				next := final
				var h http.HandlerFunc = func(res http.ResponseWriter, req *http.Request) {
					current(res, req)
					if next == nil {
						return
					}
					next.ServeHTTP(res, req)
				}
				final = h
			case func(http.Handler) http.Handler:
				if next := current(final); next != nil {
					final = next
				}
			case func() http.Handler:
				next := final
				var h http.HandlerFunc = func(res http.ResponseWriter, req *http.Request) {
					current().ServeHTTP(res, req)
					if next == nil {
						return
					}
					next.ServeHTTP(res, req)
				}
				final = h
			case func(http.ResponseWriter, *http.Request, http.Handler):
				next := final
				var h http.HandlerFunc = func(res http.ResponseWriter, req *http.Request) {
					current(res, req, next)
				}
				final = h

			default:
				log.Printf("warn: not supported type %T as handler/middleware;\n"+`supported types are:
func(http.ResponseWriter, *http.Request)
func(http.Handler) http.Handler
func() http.Handler
func(http.ResponseWriter, *http.Request, http.Handler)`, mid)
			}
		}

		if final == nil {
			final = http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {})
		}
		final.ServeHTTP(w, r)
	})
}

//-----------------------------------------------------------------------------

func flatten(list ...interface{}) []interface{} {
	var result []interface{}
	for _, v := range list {
		v := v
		switch item := v.(type) {
		case []interface{}:
			if len(item) == 0 {
				continue
			}
			result = append(result, flatten(item...)...)
		default:
			result = append(result, item)
		}
	}
	return result
}
