package pgtestutil_test

import (
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"go.winto.dev/pgtestutil"
)

func doTest(t *testing.T, driver string) {
	if !pgtestutil.DockerAvailable() {
		t.Skip("Docker is not available")
		return
	}

	m, err := pgtestutil.NewDocker(driver, 0)
	check(err)
	defer m.Close()

	m2, err := pgtestutil.New(driver, m.AdminDSN())
	check(err)
	defer m2.Close()

	c1, err := m2.Create()
	check(err)

	c2, err := m2.Create()
	check(err)

	db1, err := sql.Open(driver, c1)
	check(err)
	defer db1.Close()

	db2, err := sql.Open(driver, c2)
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

func TestDockerLibPQ(t *testing.T) {
	doTest(t, "postgres")
}

func TestDockerPGX(t *testing.T) {
	doTest(t, "pgx")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
