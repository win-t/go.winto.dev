# mainrun

[![Go Reference](https://pkg.go.dev/badge/go.winto.dev/mainrun.svg)](https://pkg.go.dev/go.winto.dev/mainrun)

Utility for main package

## How to use

```go
func main() { mainrun.Exec(run) }

func run(ctx context.Context) error {

  // ...

  return nil
}
```

The `ctx` passed to run will be cancelled if the program caught os signal
(graceful shutdown).

The returned `error` will be printed to `stderr` and the program will be exit
with exit code 1
