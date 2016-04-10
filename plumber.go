package plumber

import (
	"net/http"
	"sync"
)

// ContextFactory things

//ContextFactory makes (creates/provides) the context for using in the chain of handlers
type ContextFactory interface {
	Make(http.ResponseWriter, *http.Request) interface{}
}

//ContextFactoryFunc implements ContextFactory interface & provide an easy way to adopt out ContextFactories
type ContextFactoryFunc func(http.ResponseWriter, *http.Request) interface{}

//Make implementation of ContextFactoryFunc
func (factory ContextFactoryFunc) Make(rs http.ResponseWriter, rq *http.Request) interface{} {
	return factory(rs, rq)
}

// Handler things

//Handler takes context as a closure (curry)
type Handler func(context interface{}) http.Handler

//ServeHTTP default implementation for Handler, which uses a nil context
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h(nil).ServeHTTP(w, r)
}

//Handle handlers are either http.Handler or plumber.Handler; just executes them sequentially & provides the context if they are of type plumber.Handler
func Handle(contextFactory ContextFactory, handlers ...http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var once sync.Once //because in case there is no need for a context, we would not instantiate one
		var ctx interface{}

		for _, handler := range handlers {
			ctxHandler, ok := handler.(Handler)
			if ok {
				once.Do(func() {
					if contextFactory != nil {
						ctx = contextFactory.Make(w, r)
					}
				})
				ctxHandler(ctx).ServeHTTP(w, r)
				continue
			}

			handler.ServeHTTP(w, r)
		}
	})
}

//Middleware things

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
type ContextProvider func(context interface{}) MiddlewareFunc

//Execute implements Middleware inerface
func (mfx ContextProvider) Execute(next http.Handler) http.Handler {
	return mfx(nil)(next)
}

//Plumb creates a pipeline if middlewares () either MiddlewareFunc or ContextProvider)
func Plumb(contextFactory ContextFactory, middlewares ...Middleware) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var once sync.Once //because in case there is no need for a context, we would not instantiate one
		var ctx interface{}
		var final http.Handler = http.DefaultServeMux

		for i := len(middlewares) - 1; i >= 0; i-- {
			ctxMiddleware, ok := middlewares[i].(ContextProvider)
			if ok {
				once.Do(func() {
					if contextFactory != nil {
						ctx = contextFactory.Make(w, r)
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
