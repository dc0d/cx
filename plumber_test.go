package plumber

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	var md ContextProvider = mdl
	chain := Plumb(ContextFactoryFunc(func() interface{} {
		return &AppContext{}
	}), md, md, ContextProvider(check(t)))

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chain.ServeHTTP(w, r)
}

func check(t *testing.T) func(context interface{}) MiddlewareFunc {
	f := func(context interface{}) MiddlewareFunc {
		var mid MiddlewareFunc = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				appx, ok := context.(*AppContext)
				if ok {
					assert.Equal(t, appx.Count, 2)
				}
				next.ServeHTTP(w, r)
			})
		}

		return mid
	}

	return f
}

func mdl(context interface{}) MiddlewareFunc {
	var mid MiddlewareFunc = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			appx, ok := context.(*AppContext)
			if ok {
				appx.Count = appx.Count + 1
			}
			next.ServeHTTP(w, r)
		})
	}

	return mid
}

type AppContext struct {
	Name  string
	Count int
}

func Test2(t *testing.T) {
	var c1 MiddlewareFunc = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("1"))
			h.ServeHTTP(w, r)
		})
	}
	var c2 MiddlewareFunc = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("2"))
			h.ServeHTTP(w, r)
		})
	}
	var c3 ContextProvider = func(context interface{}) MiddlewareFunc {
		return c2
	}

	slice := []Middleware{c1, c2, c3}

	chain := Plumb(nil, slice...)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chain.ServeHTTP(w, r)

	assert.Equal(t, w.Body.String()[:3], "122")
}

func TestHandlersOrders(t *testing.T) {
	t1 := tagMiddleware("1")
	t2 := tagMiddleware("2")
	t3 := tagMiddleware("3")

	chained := Plumb(nil, t1, t2, t3)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)
	output := w.Body.String()[:3]
	assert.Equal(t, output, "123")
}

func tagMiddleware(tag string) Middleware {
	var mid MiddlewareFunc = func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(tag))
			h.ServeHTTP(w, r)
		})
	}

	return mid
}
