package plumber

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGroups(t *testing.T) {
	commonGroup := []Middleware{MiddlewareFunc(reqLogger), MiddlewareFunc(recoverPlumbing)}
	var md ContextProvider = mdl

	apiGroup1 := commonGroup
	apiGroup1 = append(apiGroup1, md, md, ContextProvider(check(t)))

	apiGroup2 := commonGroup
	apiGroup2 = append(apiGroup2, md, md, md, ContextProvider(checkCount(t, 3)))

	ctxFactory := ContextFactoryFunc(func(http.ResponseWriter, *http.Request) interface{} {
		return &AppContext{}
	})

	chain1 := Plumb(ctxFactory, apiGroup1...)
	chain2 := Plumb(ctxFactory, apiGroup2...)

	//one does this using her/his router of choice [or is there a mux just for testing?]
	w1 := httptest.NewRecorder()
	r1, err := http.NewRequest("GET", "/api/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	chain1.ServeHTTP(w1, r1)

	w2 := httptest.NewRecorder()
	r2, err := http.NewRequest("GET", "/api/v2", nil)
	if err != nil {
		t.Fatal(err)
	}
	chain2.ServeHTTP(w2, r2)
}

func checkCount(t *testing.T, count int) func(context interface{}) MiddlewareFunc {
	f := func(context interface{}) MiddlewareFunc {
		var mid MiddlewareFunc = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				appx, ok := context.(*AppContext)
				if ok {
					assert.Equal(t, appx.Count, count)
				}
				next.ServeHTTP(w, r)
			})
		}

		return mid
	}

	return f
}

func recoverPlumbing(next http.Handler) http.Handler {
	var fh http.HandlerFunc = func(res http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				trace := make([]byte, 1<<16)
				n := runtime.Stack(trace, true)
				errMsg := fmt.Errorf("panic recover\n %v\n stack trace %d bytes\n %s",
					err, n, trace[:n])
				log.Printf("ERR : %v\n", errMsg)
			}
		}()

		next.ServeHTTP(res, req)
	}

	return fh
}

func reqLogger(next http.Handler) http.Handler {
	var fh http.HandlerFunc = func(res http.ResponseWriter, req *http.Request) {
		remoteAddr := req.RemoteAddr
		if ip := req.Header.Get(XRealIP); ip != "" {
			remoteAddr = ip
		} else if ip = req.Header.Get(XForwardedFor); ip != "" {
			remoteAddr = ip
		} else {
			remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
		}

		hasError := false

		start := time.Now()

		next.ServeHTTP(res, req)

		elapsedTime := time.Since(start)
		method := req.Method
		path := req.URL.Path
		if path == "" {
			path = "/"
		}

		logMsg := fmt.Sprintf("%v %v %v %v", remoteAddr, method, path, elapsedTime)
		if hasError {
			log.Printf("WARN: %s\n", logMsg)
		} else {
			log.Printf("LOG : %s\n", logMsg)
		}

	}

	return fh
}

const (
	//XRealIP +
	XRealIP = "X-Real-IP"

	//XForwardedFor +
	XForwardedFor = "X-Forwarded-For"
)

func TestContext(t *testing.T) {
	var md ContextProvider = mdl
	chain := Plumb(ContextFactoryFunc(func(http.ResponseWriter, *http.Request) interface{} {
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
