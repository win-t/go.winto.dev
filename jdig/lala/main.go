package main

import (
	"fmt"

	"go.winto.dev/jdig"
)

func main() {
	v := jdig.Unmarshal(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"echo-6889bcb44c-jhkdp","namespace":"win-test","ownerReferences":[{"apiVersion":"apps/v1","kind":"ReplicaSet","name":"echo-6889bcb44c","uid":"000277e5-5c62-44d8-8da9-29477b2a8e1f"}],"resourceVersion":"13598225388","uid":"a8e70ff0-dcd0-4b6d-b861-b3156554cdd7"},"spec":{},"status":{}}`)
	jdig.RecursiveDeleteKeyIfEmpty(v)
	fmt.Println(jdig.Marshal(v))
}
