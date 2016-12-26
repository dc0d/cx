package plumber

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestType1(t *testing.T) {
	md := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testValue))
	}
	typeTester(t, md)
}

func TestType2(t *testing.T) {
	md := func(next http.Handler) http.Handler {
		var fh http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(testValue))
			if next == nil {
				return
			}
			next.ServeHTTP(w, r)
		}

		return fh
	}
	typeTester(t, md)
}

func TestType3(t *testing.T) {
	md := func() http.Handler {
		var fh http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(testValue))
		}

		return fh
	}
	typeTester(t, md)
}

func TestType4(t *testing.T) {
	md := func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		w.Write([]byte(testValue))
		if next == nil {
			return
		}
		next.ServeHTTP(w, r)
	}
	typeTester(t, md)
}

func TestType5(t *testing.T) {
	var md http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testValue))
	}
	typeTester(t, md)
}

func TestType6(t *testing.T) {
	var md http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testValue))
	})
	typeTester(t, md)
}

func typeTester(t *testing.T, md interface{}) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chain := Plumb(md)

	chain.ServeHTTP(w, r)
	output := w.Body.String()
	if !strings.Contains(output, testValue) {
		t.Fail()
	}
}

const testValue = `ONE`

//-----------------------------------------------------------------------------

func type1(http.ResponseWriter, *http.Request) {
	callCounter++
}

func type2(next http.Handler) http.Handler {
	var fh http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		callCounter++
		if next == nil {
			return
		}
		next.ServeHTTP(w, r)
	}

	return fh
}

func type3() http.Handler {
	callCounter++
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
}

func type4(w http.ResponseWriter, r *http.Request, next http.Handler) {
	callCounter++
	if next == nil {
		return
	}
	next.ServeHTTP(w, r)
}

func type5() http.HandlerFunc {
	return func(http.ResponseWriter, *http.Request) {
		callCounter++
	}
}

func type6() http.Handler {
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		callCounter++
	})
}

var callCounter int

func TestCallCount(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained := Plumb(type1, type4, type6(), type2, type3, type2, type5(), type3, type4)
	chained.ServeHTTP(w, r)

	if callCounter != 9 {
		t.Fail()
	}
}

//-----------------------------------------------------------------------------

func action1(s string) func() http.Handler {
	return func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(s))
		})
	}
}

func mw1(state string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), contextKey("state"), state)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func mw2(t *testing.T, expectedState string) func() http.Handler {
	return func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			val := r.Context().Value(contextKey("state"))
			sval := fmt.Sprintf("%v", val)
			assert.Equal(t, sval, expectedState)
		})
	}
}

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
	output := w.Body.String()
	if !strings.Contains(output, "123") {
		t.Fail()
	}
}

//-----------------------------------------------------------------------------

func TestMiddlewareChain(t *testing.T) {
	t.SkipNow()

	var chain = []interface{}{
		action1("1"),
		action1("2"),
		action1("3"),
	}

	chained := Plumb(nil, action1("5"), nil, nil, chain, action1("4"), nil)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	chained.ServeHTTP(w, r)
	s := w.Body.String()

	if !strings.Contains(s, "51234") {
		t.Fail()
	}
}

//-----------------------------------------------------------------------------

func TestContext(t *testing.T) {
	var apiGroup = []interface{}{mw1("14"), mw1("12"), mw2(t, "12"), action1("4")}

	chain := Plumb(apiGroup...)

	//one does this using her/his router of choice [or is there a mux just for testing?]
	w1 := httptest.NewRecorder()
	r1, err := http.NewRequest("GET", "/api/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	chain.ServeHTTP(w1, r1)
}

func TestContext2(t *testing.T) {
	var apiGroup = []interface{}{reqState, index}

	chain := Plumb(apiGroup...)

	w1 := httptest.NewRecorder()
	r1, err := http.NewRequest("GET", "/api/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	chain.ServeHTTP(w1, r1)

	s := w1.Body.String()
	if !strings.Contains(s, "123") {
		t.Fail()
	}
}

func TestContext3(t *testing.T) {
	var commonGroup = []interface{}{recoverPlumbing, reqLogger}
	var apiGroup = []interface{}{reqState, index}

	chain := Plumb(commonGroup, apiGroup)

	w1 := httptest.NewRecorder()
	r1, err := http.NewRequest("GET", "/api/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	chain.ServeHTTP(w1, r1)

	s := w1.Body.String()
	if !strings.Contains(s, "123") {
		t.Fail()
	}
}

//-----------------------------------------------------------------------------

func TestGroups(t *testing.T) {
	var commonGroup = []interface{}{recoverPlumbing, reqLogger}

	var apiGroup1 = append(commonGroup, mw1("14"), mw1("12"), mw2(t, "12"), action1("4"))

	var apiGroup2 = []interface{}{action1("4"), action1("5")}

	chain1 := Plumb(apiGroup1)
	chain2 := Plumb(commonGroup, apiGroup2)

	//one does this using her/his router of choice [or is there a mux just for testing?]
	w1 := httptest.NewRecorder()
	r1, err := http.NewRequest("GET", "/api/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if chain1 == nil {
		t.Fail()
		return
	}
	chain1.ServeHTTP(w1, r1)

	w2 := httptest.NewRecorder()
	r2, err := http.NewRequest("GET", "/api/v2", nil)
	if err != nil {
		t.Fatal(err)
	}
	chain2.ServeHTTP(w2, r2)

	s1 := w1.Body.String()
	if !strings.Contains(s1, "4") {
		t.Fail()
	}

	s2 := w2.Body.String()
	if !strings.Contains(s2, "45") {
		t.Fail()
	}
}

//-----------------------------------------------------------------------------

func recoverPlumbing(next http.Handler) http.Handler {
	var fh http.HandlerFunc = func(res http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("error: %v\n", err)
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

//-----------------------------------------------------------------------------

const viewStateContextKey contextKey = `viewStateContextKey.key`

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

//-----------------------------------------------------------------------------

type contextKey string
