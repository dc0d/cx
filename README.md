# plumber
Another try in middleware plumbing in Go (inspired by [restiful](https://github.com/laicosly/restiful), [alice](https://github.com/justinas/alice) and some experiences). It does not forces you to change the way you code classic handlers and middlewares.

> Adapted Go 1.7 context pattern, so the previous API is not longer needed. Though they are available at [gopkg.in/dc0d/plumber.v2](http://gopkg.in/dc0d/plumber.v2) and [gopkg.in/dc0d/plumber.v1](http://gopkg.in/dc0d/plumber.v1).  

A chain of middlewares is simply a slice of `Middleware`. That's the basic concept here. Having three middlewares `c1`, `c2` and `c3` we can simply chain them and serve requests like this:
```go
chain := Plumb(c1, c2, c3)
chain.ServeHTTP(w, r)
```
For samples on how to use this package see tests - for example creating groups and reuse common middlewares. Also functions with signatures of either `func(http.ResponseWriter, *http.Request)` or `func(http.ResponseWriter, *http.Request, http.Handler)` can be used as an action or middleware using the `Adapt` function.