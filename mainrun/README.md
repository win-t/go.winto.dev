# mainrun

[![Go Reference](https://pkg.go.dev/badge/go.winto.dev/mainrun.svg)](https://pkg.go.dev/go.winto.dev/mainrun)

Utility for main package

## How to use

```go
func main() { mainrun.Exec(realmain) }

func realmain(ctx context.Context) {

  // ...

}
```

The `ctx` passed to `realmain` will be cancelled if the program caught os signal for graceful shutdown.
