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

		wrapped := wrap(middlewares...)
		nop := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {})
		var final http.Handler = nop

		for i := len(wrapped) - 1; i >= 0; i-- {
			mid := wrapped[i]

			if mid == nil {
				continue
			}

			if next := mid(final); next != nil {
				final = next
			} else {
				final = nop
			}
		}

		if final == nil {
			final = nop
		}
		final.ServeHTTP(w, r)
	})
}

//-----------------------------------------------------------------------------

func wrap(middlewares ...interface{}) (result []func(http.Handler) http.Handler) {
	for _, mid := range middlewares {
		if mid == nil {
			continue
		}

		var hm func(http.Handler) http.Handler

		switch current := mid.(type) {
		case func(http.ResponseWriter, *http.Request):
			hm = func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					current(w, r)
					if next == nil {
						return
					}
					next.ServeHTTP(w, r)
				})
			}
		case func(http.Handler) http.Handler:
			hm = current
		case func() http.Handler:
			hm = func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					current().ServeHTTP(w, r)
					if next == nil {
						return
					}
					next.ServeHTTP(w, r)
				})
			}
		case func(http.ResponseWriter, *http.Request, http.Handler):
			hm = func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					current(w, r, next)
				})
			}
		case http.HandlerFunc:
			hm = func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					current(w, r)
					if next == nil {
						return
					}
					next.ServeHTTP(w, r)
				})
			}
		case http.Handler:
			hm = func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					current.ServeHTTP(w, r)
					if next == nil {
						return
					}
					next.ServeHTTP(w, r)
				})
			}
		default:
			log.Printf("warn: not supported type %T as handler/middleware;\n"+`supported types are:
func(http.ResponseWriter, *http.Request)
func(http.Handler) http.Handler
func() http.Handler
func(http.ResponseWriter, *http.Request, http.Handler)
http.HandlerFunc
http.Handler`, mid)
		}

		if hm != nil {
			result = append(result, hm)
		}
	}

	return
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
