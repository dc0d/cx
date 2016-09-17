package plumber

import "net/http"

//-----------------------------------------------------------------------------

//MiddlewareFunc middleware
type MiddlewareFunc func(http.Handler) http.Handler

//Execute implements Middleware inerface
func (mf MiddlewareFunc) Execute(next http.Handler) http.Handler {
	return mf(next)
}

//-----------------------------------------------------------------------------

//ActionFunc automatically calls next middleware, if not nil. Useful for time you are not concerned with the next middleware. Can also be used as last action.
type ActionFunc func() http.Handler

//Execute implements Middleware inerface
func (act ActionFunc) Execute(next http.Handler) http.Handler {
	var h http.HandlerFunc = func(res http.ResponseWriter, req *http.Request) {
		act().ServeHTTP(res, req)
		if next != nil {
			next.ServeHTTP(res, req)
		}
	}

	return h
}

//-----------------------------------------------------------------------------

// MiddlewareChain /
type MiddlewareChain []Middleware

//Execute implements Middleware inerface
func (mc MiddlewareChain) Execute(next http.Handler) http.Handler {
	piled := mc
	if next != nil {
		var same ActionFunc = func() http.Handler { return next }
		piled = append(mc, same)
	}

	return Plumb(piled...)
}

//-----------------------------------------------------------------------------

//Plumb creates a pipeline if middlewares
func Plumb(middlewares ...Middleware) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var final http.Handler

		for i := len(middlewares) - 1; i >= 0; i-- {
			mw := middlewares[i]
			if mw == nil {
				continue
			}

			final = mw.Execute(final)
		}

		final.ServeHTTP(w, r)
	})
}

//-----------------------------------------------------------------------------

//Middleware inerface
type Middleware interface {
	Execute(http.Handler) http.Handler
}

//-----------------------------------------------------------------------------
