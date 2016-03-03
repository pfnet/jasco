# jasco: Compact JSON API Server library version 1

jasco is a compact library to build a JSON API server. It's written in Go and
built on github.com/gocraft/web.

## Requirements

jasco requires Go 1.4.2 or later.

## Installation

```
$ go get gopkg.in/pfnet/jasco.v1
```

## Example

`example.go`:
```go
package main

import (
    "net/http"

    "github.com/gocraft/web"
    "gopkg.in/pfnet/jasco.v1"
)

type Context struct {
    *jasco.Context
}

func (c *Context) Get(rw web.ResponseWriter, req *web.Request) {
    c.Render(map[string]interface{}{
        "hello": "world",
        "num":   10,
        "obj": map[string]interface{}{
            "a": 1,
            "b": "2",
            "c": []interface{}{3.4, "5"},
        },
    })
}

func (c *Context) Echo(rw web.ResponseWriter, req *web.Request) {
    var js map[string]interface{}
    if err := c.ParseBody(&js); err != nil {
        c.ErrLog(err.Err).Error("Cannot parse the request body")
        c.RenderError(err)
        return
    }
    c.Render(js)
}

func main() {
    root := jasco.New("", nil)
    router := root.Subrouter(Context{}, "/")
    router.Get("/", (*Context).Get)
    router.Post("/", (*Context).Echo)
    http.ListenAndServe(":10080", root)
}

```

Run the example:
```
$ go run sample.go
```

Test:
```
$ curl http://localhost:10080/
{"hello":"world","num":10,"obj":{"a":1,"b":"2","c":[3.4,"5"]}}
$ curl -XPOST -d'{"test":"data"}' http://localhost:10080/
{"test":"data"}
```

## Logging

jasco uses github.com/Sirupsen/logrus as a logger. Pass a logger that the
application uses to `jasco.New`. By default, the default logger of logrus will
be used.

`Context.AddLogField("field", value)` adds a field to be logged by logrus.
Because gocraft/web creates a context for each request, a field added to
the context is only available while the request is being processed.
