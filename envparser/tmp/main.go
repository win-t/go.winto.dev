package main

import (
	"fmt"

	"go.winto.dev/envparser"
)

func main() {
	var config struct {
		AA int `env:",required"`
		BB int `env:"bb,required"`
	}
	err := envparser.UnmarshalWithPrefix(&config, "PREFIX_")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(config)
}
