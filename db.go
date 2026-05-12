package main

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

type Database struct {
	db  *sql.DB
	mux sync.RWMutex
}

func NewDatabase(path string) (*Database, error) {
	d := &Database{}
	var err error
	d.db, err = sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err = d.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

func (d *Database) Close() error { return d.db.Close() }

func (d *Database) migrate() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS connections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			host TEXT NOT NULL,
			port INTEGER NOT NULL DEFAULT 22,
			username TEXT NOT NULL,
			auth_type TEXT NOT NULL DEFAULT 'password',
			password TEXT DEFAULT '',
			key_path TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
		);
		CREATE TABLE IF NOT EXISTS scripts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			content TEXT NOT NULL DEFAULT '',
			interpreter TEXT NOT NULL DEFAULT 'sh',
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
		);
		CREATE TABLE IF NOT EXISTS execution_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			connection_id INTEGER NOT NULL,
			connection_name TEXT NOT NULL DEFAULT '',
			script_id INTEGER NOT NULL,
			script_name TEXT NOT NULL DEFAULT '',
			interpreter TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'running',
			output TEXT DEFAULT '',
			error TEXT DEFAULT '',
			started_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			finished_at TEXT DEFAULT '',
			duration_ms INTEGER DEFAULT 0,
			FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE CASCADE,
			FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE
		);
		CREATE INDEX IF NOT EXISTS idx_history_started ON execution_history(started_at DESC);
	`)
	return err
}

// --- Connections ---

func (d *Database) GetConnections() ([]*Connection, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()
	rows, err := d.db.Query(
		"SELECT id, name, host, port, username, auth_type, password, key_path FROM connections ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []*Connection
	for rows.Next() {
		c := &Connection{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Host, &c.Port, &c.Username,
			&c.AuthType, &c.Password, &c.KeyPath); err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

func (d *Database) GetConnection(id int64) (*Connection, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()
	row := d.db.QueryRow(
		"SELECT id, name, host, port, username, auth_type, password, key_path FROM connections WHERE id=?", id)
	c := &Connection{}
	err := row.Scan(&c.ID, &c.Name, &c.Host, &c.Port, &c.Username, &c.AuthType, &c.Password, &c.KeyPath)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (d *Database) AddConnection(c *Connection) (int64, error) {
	d.mux.Lock()
	defer d.mux.Unlock()
	res, err := d.db.Exec(
		"INSERT INTO connections (name, host, port, username, auth_type, password, key_path) VALUES (?,?,?,?,?,?,?)",
		c.Name, c.Host, c.Port, c.Username, c.AuthType, c.Password, c.KeyPath)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *Database) UpdateConnection(c *Connection) error {
	d.mux.Lock()
	defer d.mux.Unlock()
	_, err := d.db.Exec(
		"UPDATE connections SET name=?, host=?, port=?, username=?, auth_type=?, password=?, key_path=?, updated_at=datetime('now','localtime') WHERE id=?",
		c.Name, c.Host, c.Port, c.Username, c.AuthType, c.Password, c.KeyPath, c.ID)
	return err
}

func (d *Database) DeleteConnection(id int64) error {
	d.mux.Lock()
	defer d.mux.Unlock()
	_, err := d.db.Exec("DELETE FROM connections WHERE id=?", id)
	return err
}

// --- Scripts ---

func (d *Database) GetScripts() ([]*Script, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()
	rows, err := d.db.Query(
		"SELECT id, name, content, interpreter FROM scripts ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scripts []*Script
	for rows.Next() {
		s := &Script{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Content, &s.Interpreter); err != nil {
			return nil, err
		}
		scripts = append(scripts, s)
	}
	return scripts, rows.Err()
}

func (d *Database) GetScript(id int64) (*Script, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()
	row := d.db.QueryRow("SELECT id, name, content, interpreter FROM scripts WHERE id=?", id)
	s := &Script{}
	err := row.Scan(&s.ID, &s.Name, &s.Content, &s.Interpreter)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (d *Database) AddScript(s *Script) (int64, error) {
	d.mux.Lock()
	defer d.mux.Unlock()
	res, err := d.db.Exec(
		"INSERT INTO scripts (name, content, interpreter) VALUES (?,?,?)",
		s.Name, s.Content, s.Interpreter)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *Database) UpdateScript(s *Script) error {
	d.mux.Lock()
	defer d.mux.Unlock()
	_, err := d.db.Exec(
		"UPDATE scripts SET name=?, content=?, interpreter=?, updated_at=datetime('now','localtime') WHERE id=?",
		s.Name, s.Content, s.Interpreter, s.ID)
	return err
}

func (d *Database) DeleteScript(id int64) error {
	d.mux.Lock()
	defer d.mux.Unlock()
	_, err := d.db.Exec("DELETE FROM scripts WHERE id=?", id)
	return err
}

// --- Execution History ---

func (d *Database) GetHistory(limit int) ([]ExecHistory, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()
	rows, err := d.db.Query(
		"SELECT id, connection_name, script_name, interpreter, status, started_at, finished_at, duration_ms "+
			"FROM execution_history ORDER BY id DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ExecHistory
	for rows.Next() {
		var h ExecHistory
		if err := rows.Scan(&h.ID, &h.ConnectionName, &h.ScriptName,
			&h.Interpreter, &h.Status, &h.StartedAt, &h.FinishedAt, &h.DurationMs); err != nil {
			return nil, err
		}
		items = append(items, h)
	}
	return items, rows.Err()
}

func (d *Database) GetHistoryDetail(id int64) (*ExecHistory, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()
	row := d.db.QueryRow(
		"SELECT id, connection_name, script_name, interpreter, status, output, error, started_at, finished_at, duration_ms "+
			"FROM execution_history WHERE id=?", id)
	h := &ExecHistory{}
	err := row.Scan(&h.ID, &h.ConnectionName, &h.ScriptName, &h.Interpreter,
		&h.Status, &h.Output, &h.Error, &h.StartedAt, &h.FinishedAt, &h.DurationMs)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (d *Database) AddHistory(h *ExecHistory) (int64, error) {
	d.mux.Lock()
	defer d.mux.Unlock()
	res, err := d.db.Exec(
		"INSERT INTO execution_history (connection_id, connection_name, script_id, script_name, interpreter, status) VALUES (?,?,?,?,?,?)",
		h.ConnectionID, h.ConnectionName, h.ScriptID, h.ScriptName, h.Interpreter, h.Status)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *Database) UpdateHistory(id int64, status, output, errMsg string, durationMs int) error {
	d.mux.Lock()
	defer d.mux.Unlock()
	_, err := d.db.Exec(
		"UPDATE execution_history SET status=?, output=?, error=?, finished_at=datetime('now','localtime'), duration_ms=? WHERE id=?",
		status, output, errMsg, durationMs, id)
	return err
}

func (d *Database) ClearHistory() error {
	d.mux.Lock()
	defer d.mux.Unlock()
	_, err := d.db.Exec("DELETE FROM execution_history")
	return err
}

func (d *Database) DeleteHistoryByID(id int64) error {
	d.mux.Lock()
	defer d.mux.Unlock()
	_, err := d.db.Exec("DELETE FROM execution_history WHERE id=?", id)
	return err
}
