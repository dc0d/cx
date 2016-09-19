package plumber

import "net/http"

//-----------------------------------------------------------------------------

// Adapt is a helper adapter that allows a function with
//    func(http.ResponseWriter, *http.Request)
// signature be used as an ActionFunc which implements Middleware interface and a function with
//    func(http.ResponseWriter, *http.Request, http.Handler)
// signature be used as a MiddlewareFunc which also implements Middleware interface.
func Adapt(f interface{}) Middleware {
	switch tf := f.(type) {
	case func(http.ResponseWriter, *http.Request):
		return adaptAction(tf)
	case func(http.ResponseWriter, *http.Request, http.Handler):
		return adaptMiddleware(tf)
	}

	return nil
}

func adaptMiddleware(mw func(http.ResponseWriter, *http.Request, http.Handler)) Middleware {
	return MiddlewareFunc(func(nextHandler http.Handler) http.Handler {
		var fh http.HandlerFunc = func(res http.ResponseWriter, req *http.Request) {
			mw(res, req, nextHandler)
		}

		return fh
	})
}

func adaptAction(action func(http.ResponseWriter, *http.Request)) ActionFunc {
	return func() http.Handler {
		return http.HandlerFunc(action)
	}
}

//-----------------------------------------------------------------------------
