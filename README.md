# plumber
Another try in middleware plumbing in Go (inspired by [restiful](https://github.com/laicosly/restiful), [alice](https://github.com/justinas/alice) and some experiences).

A chain of middlewares is simply a slice of `Middleware`. That's the basic concept here. Having three middlewares `c1`, `c2` and `c3` we can simply chain them and serve requests like this:
```go
chain := Plumb(nil, c1, c2, c3)
chain.ServeHTTP(w, r)
```
The first parameter is a `ContextFactory` which will get called for each request, if provided. It just creates a new instance of your context. So we could write:
```go
chain := Plumb(ContextFactoryFunc(func(http.ResponseWriter, *http.Request) interface{} {
  return &AppContext{}
}), middleware1, middleware2, app)
```
and `middleware1`, `middleware2` and `app` are of type `ContextProvider` which has this signature `func(context interface{}) MiddlewareFunc` which makes it super easy to introduce a request context to classic middlewares.

Since we just use slices of middlewares, creating groups is vary easy too. For example we can add a logger and a recovery middleware to all groups like this:
```go
commonGroup := []Middleware{MiddlewareFunc(reqLogger), MiddlewareFunc(recoverPlumbing)}

apiGroup1 := commonGroup
apiGroup1 = append(apiGroup1, counterMd, counterMd, ContextProvider(checkCount(t, 2)))

apiGroup2 := commonGroup
apiGroup2 = append(apiGroup2, counterMd, counterMd, counterMd, ContextProvider(checkCount(t, 3)))
```
