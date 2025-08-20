package sqlmigrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestNormal(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmts := []string{
		"create table t1(k integer primary key, v integer)",
		"create table t2(k integer primary key, v integer)",
	}

	// first run
	err = Exec(context.Background(), db, stmts, nil)
	if err != nil {
		t.Fatal(err)
	}

	// second run should be nop
	err = Exec(context.Background(), db, stmts, nil)
	if err != nil {
		t.Fatal(err)
	}

	var tableCount int
	db.QueryRow("select count(*) from sqlite_master").Scan(&tableCount)
	if tableCount != 3 {
		t.Fatalf("expected 3 tables, got %d", tableCount)
	}
}

func TestDiffStmtMustFail(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// first run
	err = Exec(context.Background(), db, []string{
		"create table t1(k integer primary key, v integer)",
		"create table t2(k integer primary key, v integer)",
		"create table t3(k integer primary key, v integer)",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// second run should be nop
	err = Exec(context.Background(), db, []string{
		"create table t1(k integer primary key, v integer)",
		"create table t2_diff(k integer primary key, v integer)",
		"create table t3(k integer primary key, v integer)",
	}, nil)
	realErr := (*ChecksumMismatchError)(nil)
	if !errors.As(err, &realErr) {
		t.Fatal(err)
	}
	if realErr.StmtIndex != 1 {
		t.Fatal("statement index 1 should fail")
	}
	fmt.Fprintln(os.Stderr, "TEST DEBUG", realErr.Error())

	var tableCount int
	db.QueryRow("select count(*) from sqlite_master").Scan(&tableCount)
	if tableCount != 4 {
		t.Fatalf("expected 4 tables, got %d", tableCount)
	}
}

func TestUpToFailStmt(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = Exec(context.Background(), db, []string{
		"create table t1(k integer primary key, v integer)",
		"create table t2(k integer primary key, v integer)",
		"create table t2(k integer primary key, v integer)",
		"create table t3(k integer primary key, v integer)",
	}, nil)
	realErr := (*StmtExecError)(nil)
	if !errors.As(err, &realErr) {
		t.Fatal(err)
	}
	if realErr.StmtIndex != 2 {
		t.Fatal("statement index 2 should fail")
	}
	if errors.Unwrap(err) == nil {
		t.Fatal("should wrap original error")
	}
	fmt.Fprintln(os.Stderr, "TEST DEBUG", err.Error())

	var tableCount int
	db.QueryRow("select count(*) from sqlite_master").Scan(&tableCount)
	if tableCount != 3 {
		t.Fatalf("expected 3 tables, got %d", tableCount)
	}

	err = Exec(context.Background(), db, []string{
		"create table t1(k integer primary key, v integer)",
		"create table t2(k integer primary key, v integer)",
		"create table t3(k integer primary key, v integer)",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	db.QueryRow("select count(*) from sqlite_master").Scan(&tableCount)
	if tableCount != 4 {
		t.Fatalf("expected 4 tables, got %d", tableCount)
	}
}

func TestLeaderGiveUp(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = Exec(context.Background(), db, []string{
		"create table t1(k integer primary key, v integer)",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var giveUpDone atomic.Bool

	_, err = db.Exec("insert into go_winto_dev_sqlmigrate (i, c) values (2, null)")
	check(err)
	go func() {
		time.Sleep(500 * time.Millisecond)
		_, err := db.Exec("delete from go_winto_dev_sqlmigrate where i = 2")
		check(err)
		giveUpDone.Store(true)
	}()

	err = Exec(context.Background(), db, []string{
		"create table t1(k integer primary key, v integer)",
		"create table t2(k integer primary key, v integer)",
		"create table t3(k integer primary key, v integer)",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !giveUpDone.Load() {
		t.Fatal("leader should give up")
	}

	var tableCount int
	db.QueryRow("select count(*) from sqlite_master").Scan(&tableCount)
	if tableCount != 4 {
		t.Fatalf("expected 4 tables, got %d", tableCount)
	}
}

func TestCB(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	count := 0
	stmts := []string{
		// 0
		"select 1",
	}
	cb := map[int]func(context.Context, DB) error{
		0: func(ctx context.Context, db DB) error {
			count++
			return nil
		},
	}

	err = Exec(context.Background(), db, stmts, cb)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("callback not called")
	}

	stmts = append(stmts, "select 2")
	err = Exec(context.Background(), db, stmts, cb)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("callback is called twice")
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
