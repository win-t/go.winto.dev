package httphandler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"go.winto.dev/httphandler"
)

func genMiddleware(id string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Before", id)
			next(w, r)
			w.Header().Add("After", id)
		}
	}
}

var testData = "testdata"

func check(t *testing.T, handler any) {
	h := httphandler.Chain(
		genMiddleware("1"),
		nil,
		[]any{
			genMiddleware("2"),
			genMiddleware("3"),
			[]any{
				genMiddleware("4"),
			},
			genMiddleware("5"),
		},
		genMiddleware("6"),
		handler,
		func(http.HandlerFunc) http.HandlerFunc {
			panic("should not go here")
		},
	)

	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h(res, req)

	if res.Body.String() != testData {
		t.Errorf("invalid res body")
	}
	if !reflect.DeepEqual(res.Header()["Before"], []string{"1", "2", "3", "4", "5", "6"}) {
		t.Errorf("invalid res header: Before")
	}
	if !reflect.DeepEqual(res.Header()["After"], []string{"6", "5", "4", "3", "2", "1"}) {
		t.Errorf("invalid res header: After")
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, testData) }

func TestChain(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", testHandler)

	// for http.HandlerFunc
	check(t, testHandler)

	// for http.Handler
	check(t, mux)

	// for func(*http.Request) http.HandlerFunc
	check(t, func(*http.Request) http.HandlerFunc { return testHandler })

	// for func(http.ResponseWriter, *http.Request) error
	check(t, func(w http.ResponseWriter, r *http.Request) error { testHandler(w, r); return nil })
}

func TestInvalidChain(t *testing.T) {
	// this test is just for code coverage

	gotPanic := false

	func() {
		defer func() { gotPanic = recover() != nil }()
		httphandler.Chain("invalid middleware")
	}()

	if !gotPanic {
		t.Errorf("should panic")
	}
}

func TestDoc(t *testing.T) {
	// this test is from documentation of Chain function
	var h http.HandlerFunc
	var m func(http.HandlerFunc) http.HandlerFunc
	var ms [2]func(http.HandlerFunc) http.HandlerFunc

	h = testHandler
	m = genMiddleware("1")
	ms = [2]func(http.HandlerFunc) http.HandlerFunc{genMiddleware("2"), genMiddleware("3")}

	req1 := httptest.NewRequest("GET", "/", nil)
	res1 := httptest.NewRecorder()
	all1 := m(ms[0](ms[1](h)))
	all1(res1, req1)

	req2 := httptest.NewRequest("GET", "/", nil)
	res2 := httptest.NewRecorder()
	all2 := httphandler.Chain(m, ms, h)
	all2(res2, req2)

	if !reflect.DeepEqual(res1, res2) {
		t.Errorf("Chain should have same behaviour as manual chaining by hand")
	}
}

func TestHandlerInMiddle(t *testing.T) {
	gotPanic := false

	identityMiddleware := func(next http.HandlerFunc) http.HandlerFunc { return next }
	panicMiddleware := func(http.HandlerFunc) http.HandlerFunc { panic("should not go here") }
	handler := func(http.ResponseWriter, *http.Request) {}
	func() {
		defer func() { gotPanic = recover() != nil }()
		httphandler.Chain(
			identityMiddleware,
			[]any{
				identityMiddleware,
				[]any{
					identityMiddleware,
					handler,
					panicMiddleware,
				},
				panicMiddleware,
			},
			panicMiddleware,
		)
	}()

	if gotPanic {
		t.Errorf("should not panic")
	}
}

func checkBody(t *testing.T, expected string, h http.HandlerFunc) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h(res, req)

	if res.Body.String() != expected {
		t.Errorf("invalid res body")
	}
}

func TestIfaceIface(t *testing.T) {
	h := httphandler.Chain(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "1")
				next.ServeHTTP(w, r)
				fmt.Fprint(w, "3")
			})
		},
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "2")
		},
	)

	checkBody(t, "123", h)
}

func TestFuncIface(t *testing.T) {
	h := httphandler.Chain(
		func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "1")
				next.ServeHTTP(w, r)
				fmt.Fprint(w, "3")
			})
		},
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "2")
		},
	)

	checkBody(t, "123", h)
}

func TestIfaceFunc(t *testing.T) {
	h := httphandler.Chain(
		func(next http.Handler) http.HandlerFunc {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "1")
				next.ServeHTTP(w, r)
				fmt.Fprint(w, "3")
			})
		},
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "2")
		},
	)

	checkBody(t, "123", h)
}

func TestHandlerFunc(t *testing.T) {
	h := httphandler.Chain(
		func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
			fmt.Fprint(w, "1")
			next(w, r)
			fmt.Fprint(w, "3")

		},
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "2")
		},
	)

	checkBody(t, "123", h)
}

func TestHandlerIface(t *testing.T) {
	h := httphandler.Chain(
		func(w http.ResponseWriter, r *http.Request, next http.Handler) {
			fmt.Fprint(w, "1")
			next.ServeHTTP(w, r)
			fmt.Fprint(w, "3")

		},
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "2")
		},
	)

	checkBody(t, "123", h)
}

func ExampleChain() {
	middleware1 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Before", "m1")
			next(w, r)
			w.Header().Add("After", "m1")
		}
	}

	middleware2 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Before", "m2")
			next(w, r)
			w.Header().Add("After", "m2")
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "x")
	}

	all := httphandler.Chain(
		middleware1,
		middleware2,
		handler,
	)

	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	all(res, req)

	fmt.Printf(
		"%s,%s,%s,%s,%s\n",
		res.Header().Values("Before")[0],
		res.Header().Values("Before")[1],
		res.Body.String(),
		res.Header().Values("After")[0],
		res.Header().Values("After")[1],
	)

	// Output: m1,m2,x,m2,m1
}
