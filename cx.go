package cx

import (
	"net/http"
	"sync"
)

//Context is the context type used by our handlers
type Context interface{}

//Handler takes context as a closure (curry)
type Handler func(context Context) http.Handler

//ServeHTTP default implementation for Handler, which uses a nil context
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h(nil).ServeHTTP(w, r)
}

//Handle handlers are either http.Handler or <this-package>.Handler
func Handle(contextFactory func() Context, handlers ...http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var once sync.Once //because in case there is no need for a context, we would not instantiate one
		var ctx Context

		for _, handler := range handlers {
			ctxHandler, ok := handler.(Handler)
			if ok {
				once.Do(func() {
					if contextFactory != nil {
						ctx = contextFactory()
					}
				})
				ctxHandler(ctx).ServeHTTP(w, r)
				continue
			}

			handler.ServeHTTP(w, r)
		}
	})
}

//Middleware inerface
type Middleware interface {
	Execute(http.Handler) http.Handler
}

//MiddlewareFunc middleware
type MiddlewareFunc func(http.Handler) http.Handler

//Execute implements Middleware inerface
func (mf MiddlewareFunc) Execute(next http.Handler) http.Handler {
	return mf(next)
}

//ContextProvider a middleware that accepts a context (default: nil)
type ContextProvider func(context Context) MiddlewareFunc

//Execute implements Middleware inerface
func (mfx ContextProvider) Execute(next http.Handler) http.Handler {
	return mfx(nil)(next)
}

//Plumb creates a pipeline if middlewares () either MiddlewareFunc or ContextProvider)
func Plumb(contextFactory func() Context, middlewares ...Middleware) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var once sync.Once //because in case there is no need for a context, we would not instantiate one
		var ctx Context
		var final http.Handler = http.DefaultServeMux

		for i := len(middlewares) - 1; i >= 0; i-- {
			ctxMiddleware, ok := middlewares[i].(ContextProvider)
			if ok {
				once.Do(func() {
					if contextFactory != nil {
						ctx = contextFactory()
					}
				})
				final = ctxMiddleware(ctx)(final)
				continue
			}

			final = middlewares[i].Execute(final)
		}

		final.ServeHTTP(w, r)
	})
}
