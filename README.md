# plumber
Another try in middleware plumbing in Go (inspired by [restiful](https://github.com/laicosly/restiful), [alice](https://github.com/justinas/alice) and some experiences). It does not forces you to change the way you code classic handlers and middlewares.


A chain of middlewares is simply a slice of `interface{}`. That's the basic concept here. Having three middlewares `c1`, `c2` and `c3` we can simply chain them and serve requests like this:

```go
chain := Plumb(c1, c2, c3)
chain.ServeHTTP(w, r)
```

For samples on how to use this package see tests - for example creating groups and reuse common middlewares. 

```go
// they are just Go slices!
var commonGroup = []interface{}{recoverPlumbing, reqLogger}
var apiGroup1 = append(commonGroup, mw1("14"), mw1("12"), mw2(t, "12"), action1("4"))
var apiGroup2 = []interface{}{action1("4"), action1("5")}

chain1 := Plumb(apiGroup1)
chain2 := Plumb(commonGroup, apiGroup2)

// ... 
chain1.ServeHTTP(w, r)

// ...
chain2.ServeHTTP(w, r)
```

Functions with signatures of: 

* `func(http.ResponseWriter, *http.Request)`
* `func(http.Handler) http.Handler`
* `func() http.Handler`
* `func(http.ResponseWriter, *http.Request, http.Handler)` 

can be used as an action or middleware.