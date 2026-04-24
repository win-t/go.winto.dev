// Package pgtestutil provides utilities for managing PostgreSQL databases in tests.
//
// The manager will maintain single admin connection to the database cluster and then later can create/destroy database on demand.
// All created database will be cleaned up when the manager is closed.
package pgtestutil

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
)

type Manager struct {
	adminURL *url.URL
	adminDB  *sql.DB

	skipCleanup bool
	closeHook   func()

	mu      sync.Mutex
	created map[string]struct{}
}

// New creates a new Manager with the given driver name and dsn.
// The parameters all equivalent to [database/sql.Open] function
func New(driver string, adminDSN string) (*Manager, error) {
	return newManager(driver, adminDSN, nil, false)
}

func newManager(driverName string, adminConn string, closeHook func(), skipCleanup bool) (*Manager, error) {
	adminURL, err := url.Parse(adminConn)
	if err != nil {
		return nil, err
	}

	adminDB, err := sql.Open(driverName, adminConn)
	if err != nil {
		return nil, err
	}
	adminDB.SetMaxOpenConns(2)

	return &Manager{
		adminURL:    adminURL,
		adminDB:     adminDB,
		closeHook:   closeHook,
		skipCleanup: skipCleanup,
		created:     make(map[string]struct{}),
	}, nil
}

func (m *Manager) AdminDSN() string {
	return m.adminURL.String()
}

// Close the manager and cleanup all create databases.
// This function must be called at the end of Manager lifetime to avoid resource leak.
func (m *Manager) Close() error {
	if !m.skipCleanup {
		for {
			var created []string

			m.mu.Lock()
			for conn := range m.created {
				created = append(created, conn)
			}
			m.mu.Unlock()

			if len(created) == 0 {
				break
			}

			for _, conn := range created {
				m.Destroy(conn)
			}
		}
	}

	m.adminDB.Close()
	if m.closeHook != nil {
		m.closeHook()
	}
	return nil
}

// Create a new database and will return the DSN for it.
func (m *Manager) Create() (string, error) {
	user := "u" + randomHex()
	pass := "p" + randomHex()
	dbname := "d" + randomHex()

	if err := m.createUser(user, pass); err != nil {
		return "", err
	}

	if err := m.createDB(user, dbname); err != nil {
		m.dropUser(user)
		return "", err
	}

	connURL := &url.URL{}
	*connURL = *m.adminURL
	connURL.User = url.UserPassword(user, pass)
	connURL.Path = "/" + dbname
	connURL.RawPath = ""

	conn := connURL.String()

	m.mu.Lock()
	m.created[conn] = struct{}{}
	m.mu.Unlock()

	return conn, nil
}

// Destroy the database created by [pgtestutil.Create].
func (m *Manager) Destroy(dsn string) {
	m.mu.Lock()
	_, ok := m.created[dsn]
	m.mu.Unlock()
	if !ok {
		return
	}

	uri, err := url.Parse(dsn)
	if err != nil {
		return
	}

	user := uri.User.Username()
	dbname := uri.EscapedPath()
	dbname = strings.TrimPrefix(dbname, "/")

	m.dropDB(dbname)
	m.dropUser(user)

	m.mu.Lock()
	delete(m.created, dsn)
	m.mu.Unlock()
}

func randomHex() string {
	data := [8]byte{}
	for {
		_, err := rand.Read(data[:])
		if err == nil {
			return hex.EncodeToString(data[:])
		}
	}
}

func (m *Manager) createUser(user, pass string) error {
	if _, err := m.adminDB.Exec(
		fmt.Sprintf("CREATE USER %s WITH ENCRYPTED PASSWORD '%s'",
			user, pass,
		),
	); err != nil {
		return err
	}
	return nil
}

func (m *Manager) dropUser(user string) error {
	if _, err := m.adminDB.Exec(
		fmt.Sprintf("DROP USER %s", user),
	); err != nil {
		return err
	}
	return nil
}

func (m *Manager) createDB(user, dbname string) error {
	if _, err := m.adminDB.Exec(
		fmt.Sprintf("CREATE DATABASE %s OWNER %s",
			dbname, user,
		),
	); err != nil {
		return err
	}
	return nil
}

func (m *Manager) dropDB(dbname string) error {
	if _, err := m.adminDB.Exec(
		fmt.Sprintf("DROP DATABASE %s", dbname),
	); err != nil {
		return err
	}
	return nil
}
