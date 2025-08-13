// package sqlmigrate provide poor-man sql migration tool.
//
// the [Exec] function will try to handle the concurrency, but it is possible that the migration stuck
// because other process is crash without proper clean up. You need to manually inspect the database state
// and remove the row in `go_winto_dev_sqlmigrate` table.
package sqlmigrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/crc32"
	"strings"
	"time"
)

// Indicating that the already executed statement is different with the one passed to Exec.
type ChecksumMismatchError struct {
	StmtIndex int
	Expected  int32
	Actual    int32
}

func (e *ChecksumMismatchError) Error() string {
	return fmt.Sprintf("checksum mismatch for statement %d: expected %d, got %d", e.StmtIndex, e.Expected, e.Actual)
}

type StmtExecError struct {
	StmtIndex int
	Err       error
}

// Indicating execution error on a specific statement.
func (e *StmtExecError) Error() string {
	return fmt.Sprintf("error executing statement %d: %v", e.StmtIndex, e.Err)
}

func (e *StmtExecError) Unwrap() error { return e.Err }

type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Exec execute the stmts in order. It will skip if the stmt already executed.
func Exec(ctx context.Context, db DB, stmts []string) error {
	// attempt to create the table and ignore the error
	db.ExecContext(ctx, "create table go_winto_dev_sqlmigrate (i integer primary key, c integer)")

next_stmt:
	for i, stmt := range stmts {
		checksum := checksum(stmt)
	next_attempt:
		for {
			// race the row insertion for leader election
			_, err := db.ExecContext(ctx, fmt.Sprintf("insert into go_winto_dev_sqlmigrate (i, c) values (%d, null)", i))
			if err == nil {
				// we won the election, execute the statement
				if err := run(ctx, db, i, stmt, checksum); err != nil {
					return err
				}
				continue next_stmt
			}

			// we lost the election, wait for the checksum
		next_check:
			for {
				var storedChecksum sql.NullInt32
				err := db.QueryRowContext(ctx, fmt.Sprintf("select c from go_winto_dev_sqlmigrate where i = %d", i)).Scan(&storedChecksum)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) { // the leader give up
						continue next_attempt
					}
					return err
				}
				if !storedChecksum.Valid {
					select {
					case <-ctx.Done():
						return fmt.Errorf("context cancelled while waiting for checksum %d: %w", i, ctx.Err())
					case <-time.After(250 * time.Millisecond):
						continue next_check
					}
				}
				if storedChecksum.Int32 == checksum {
					continue next_stmt
				}
				return &ChecksumMismatchError{
					StmtIndex: i,
					Expected:  checksum,
					Actual:    storedChecksum.Int32,
				}
			}
		}
	}
	return nil
}

// when this function return, the checksum on the row must be filled or the row must be deleted
func run(ctx context.Context, db DB, i int, stmt string, checksum int32) (err error) {
	success := false
	defer func() {
		if !success {
			_, delErr := db.ExecContext(context.Background(), fmt.Sprintf("delete from go_winto_dev_sqlmigrate where i = %d", i))
			if delErr != nil {
				err = errors.Join(err, delErr)
			}
		}
	}()

	_, err = db.ExecContext(ctx, stmt)
	if err != nil {
		return &StmtExecError{
			StmtIndex: i,
			Err:       err,
		}
	}
	success = true

	_, err = db.ExecContext(ctx, fmt.Sprintf("update go_winto_dev_sqlmigrate set c = %d where i = %d", checksum, i))
	if err != nil {
		return err
	}

	return nil
}

func checksum(stmt string) int32 {
	stmt = strings.ReplaceAll(stmt, " ", "")
	stmt = strings.ReplaceAll(stmt, "\n", "")
	stmt = strings.ReplaceAll(stmt, "\t", "")
	stmt = strings.ToLower(stmt)
	return int32(crc32.ChecksumIEEE([]byte(stmt)))
}
