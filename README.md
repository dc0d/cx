# plumber
Another try in middleware plumbing in Go (inspired by [restiful](https://github.com/laicosly/restiful), [alice](https://github.com/justinas/alice) and some experiences). I does not forces you to change the way you code classic handlers and middlewares.

It achieves this by employing _**function currying**_ and _**closures**_ - and Go has one of the sanest implementations of closures.

A chain of middlewares is simply a slice of `Middleware`. That's the basic concept here. Having three middlewares `c1`, `c2` and `c3` we can simply chain them and serve requests like this:
```go
chain := Plumb(nil, c1, c2, c3)
chain.ServeHTTP(w, r)
```
The first parameter is a `ContextFactory` which will get called for each request, if provided. It just creates a new instance of your context, once. So we could write:
```go
chain := Plumb(ContextFactoryFunc(func(http.ResponseWriter, *http.Request) interface{} {
  return &AppContext{}
}), middleware1, middleware2, app)
```
and `middleware1`, `middleware2` and `app` are of type `ContextInjector` which has this signature `func(context interface{}) MiddlewareFunc` (or `func(context interface{}) ActionFunc`) which makes it super easy to introduce a request context to classic middlewares.

Since we just use slices of middlewares, creating groups is very easy too. For example we can add a logger and a recovery middleware to all groups like this:
```go
commonGroup := []Middleware{MiddlewareFunc(reqLogger), MiddlewareFunc(recoverPlumbing)}
var counterMd ContextInjector = counterMiddleware

apiGroup1 := commonGroup
apiGroup1 = append(apiGroup1, counterMd, counterMd, checkCount(t, 2))

apiGroup2 := commonGroup
apiGroup2 = append(apiGroup2, counterMd, counterMd, counterMd, checkCount(t, 3))
```
API polished and changed a bit - should still be used in my projects a bit longer to get more refined & close to my taste. 

Current stable version is [gopkg.in/dc0d/plumber.v2](http://gopkg.in/dc0d/plumber.v2). For previous stable version use [gopkg.in/dc0d/plumber.v1](http://gopkg.in/dc0d/plumber.v1).
