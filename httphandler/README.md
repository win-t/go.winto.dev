# handler

[![GoDoc](https://pkg.go.dev/badge/go.winto.dev/httphandler)](https://pkg.go.dev/go.winto.dev/httphandler)

## Alternative handler signature

package httphandler provide new signature for handling http request.

stdlib handler signature is `func(http.ResponseWriter, *http.Request)`, and it is not convenience to write branching inside it.

For example:

```go
func h(w http.ResponseWriter, r *http.Request) {
    if ... {
        http.Error(w, "some error 1", 500)
        // it will be disaster if we forget this return
        return
    }

    ...

    if ... {
        http.Error(w, "some error 2", 500)
        // it will be disaster if we forget this return
        return
    }

    ...

    fmt.Fprintln(w, "some data")
}
```

we can rewrite it like this to force developer to return when the branch of the code end:

```go
func h(r *http.Request) http.HandlerFunc {
    if ... {
        return defresponse.Text(500, "some error 1")
    }

    ...


    if ... {
        return defresponse.Text(500, "some error 2")
    }

    ...

    // will compile error if we forget return
    return defresponse.Text(200, "some data")
}

func main() {
    http.ListenAndServe(":8080", httphandler.Of(h))
}
```

## Middleware helper

this package provide also `Chain` function to chain multiple middleware into single handler function

`Chain` multiple middleware into single handler function

Middleware is any value that have following type
```go
func(next http.HandlerFunc) http.HandlerFunc
func(next http.Handler) http.Handler
func(next http.HandlerFunc) http.Handler
func(next http.Handler) http.HandlerFunc
```

you can pass multiple `middleware`, slice/array of `middleware`, or combination of them

this function also accept following type as normal handler (last function in middleware chain)
```go
http.HandlerFunc
http.Handler
func(*http.Request) http.HandlerFunc
func(http.ResponseWriter, *http.Request) error
```

when you have following code
```go
var h http.HandlerFunc
var m func(http.HandlerFunc) http.HandlerFunc
var ms [2]func(http.HandlerFunc) http.HandlerFunc
```
then
```go
all := httphandler.Chain(m, ms, h)
```
will have same effect as
```go
all := m(ms[0](ms[1](h)))
```
