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

func New(driverName string, dataSourceName string) (*Manager, error) {
	return newManager(driverName, dataSourceName, nil, false)
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

func (m *Manager) AdminURL() string {
	return m.adminURL.String()
}

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

func (m *Manager) Destroy(conn string) {
	m.mu.Lock()
	_, ok := m.created[conn]
	m.mu.Unlock()
	if !ok {
		return
	}

	uri, err := url.Parse(conn)
	if err != nil {
		return
	}

	user := uri.User.Username()
	dbname := uri.EscapedPath()
	dbname = strings.TrimPrefix(dbname, "/")

	m.dropDB(dbname)
	m.dropUser(user)

	m.mu.Lock()
	delete(m.created, conn)
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
