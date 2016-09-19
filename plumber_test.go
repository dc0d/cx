package plumber

import (
	"context"
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

//-----------------------------------------------------------------------------

func TestHandlersOrders(t *testing.T) {
	t1 := action1("1")
	t2 := action1("2")
	t3 := action1("3")

	chained := Plumb(t1, t2, t3)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)
	output := w.Body.String()[:3]
	assert.Equal(t, output, "123")
}

//-----------------------------------------------------------------------------

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

const (
	//XRealIP +
	XRealIP = "X-Real-IP"

	//XForwardedFor +
	XForwardedFor = "X-Forwarded-For"
)

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
		_ = logMsg
		if hasError {
			// log.Printf("WARN: %s\n", logMsg)
		} else {
			// log.Printf("LOG : %s\n", logMsg)
		}

	}

	return fh
}

func action1(s string) ActionFunc {
	return func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(s))
		})
	}
}

func mw1(state string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "state", state)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func mw2(t *testing.T, expectedState string) ActionFunc {
	return func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			val := r.Context().Value("state")
			sval := fmt.Sprintf("%v", val)
			assert.Equal(t, sval, expectedState)
		})
	}
}

//-----------------------------------------------------------------------------

func TestMiddlewareChain(t *testing.T) {
	var chain MiddlewareChain = []Middleware{
		action1("1"),
		action1("2"),
		action1("3"),
	}

	chained := Plumb(action1("5"), nil, nil, chain, action1("4"), nil)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)
	s := w.Body.String()

	output := s[:5]
	assert.Equal(t, output, "51234")
}

//-----------------------------------------------------------------------------

func TestGroups(t *testing.T) {
	var commonGroup MiddlewareChain = []Middleware{MiddlewareFunc(reqLogger), MiddlewareFunc(recoverPlumbing)}

	var apiGroup1 = append(commonGroup, mw1("14"), mw1("12"), mw2(t, "12"), action1("4"))

	var apiGroup2 MiddlewareChain = []Middleware{action1("4"), action1("5")}

	chain1 := Plumb(apiGroup1)
	chain2 := Plumb(commonGroup, apiGroup2)

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

//-----------------------------------------------------------------------------

func TestContext(t *testing.T) {
	var apiGroup = []Middleware{mw1("14"), mw1("12"), mw2(t, "12"), action1("4")}

	chain := Plumb(apiGroup...)

	//one does this using her/his router of choice [or is there a mux just for testing?]
	w1 := httptest.NewRecorder()
	r1, err := http.NewRequest("GET", "/api/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	chain.ServeHTTP(w1, r1)
}

//-----------------------------------------------------------------------------

const viewStateContextKey = `viewStateContextKey.key`

type viewState struct {
	Message string
}

func reqState(res http.ResponseWriter, req *http.Request, next http.Handler) {
	st := new(viewState)
	st.Message = `123`

	ctx := context.WithValue(req.Context(), viewStateContextKey, st)
	next.ServeHTTP(res, req.WithContext(ctx))
}

func index(w http.ResponseWriter, r *http.Request) {
	st := r.Context().Value(viewStateContextKey).(*viewState)
	w.Write([]byte(st.Message))
}

func TestAdapt(t *testing.T) {
	chained := Plumb(Adapt(reqState), Adapt(index))

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)
	s := w.Body.String()

	output := s[:3]
	assert.Equal(t, output, "123")
}
