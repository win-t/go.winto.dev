package pgtestutil_test

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"go.winto.dev/pgtestutil"
)

func TestDocker(t *testing.T) {
	if !pgtestutil.DockerAvailable() {
		t.Skip("Docker is not available")
		return
	}

	m, err := pgtestutil.NewDocker(0)
	check(err)
	defer m.Close()

	m2, err := pgtestutil.New(m.AdminURL())
	check(err)
	defer m2.Close()

	c1, err := m2.Create()
	check(err)

	c2, err := m2.Create()
	check(err)

	db1, err := sql.Open("postgres", c1)
	check(err)
	defer db1.Close()

	db2, err := sql.Open("postgres", c2)
	check(err)
	defer db2.Close()

	var result int
	err = db1.QueryRow("select 1").Scan(&result)
	check(err)
	if result != 1 {
		t.Fatalf("unexpected result: %d", result)
	}

	err = db2.QueryRow("select 2").Scan(&result)
	check(err)
	if result != 2 {
		t.Fatalf("unexpected result: %d", result)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
