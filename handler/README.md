# handler

[![GoDoc](https://pkg.go.dev/badge/go.winto.dev/handler)](https://pkg.go.dev/go.winto.dev/handler)

Package handler provide new signature for handling http request.

stdlib Handler signature is `func(http.ResponseWriter, *http.Request)`, and it is not convenience to write branching inside it.

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
```
