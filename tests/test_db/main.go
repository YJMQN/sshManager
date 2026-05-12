// test_db — Minimal SQLite test (console app, writes to log)
// Compile: go build -o test_db.exe .
// Run from cmd: test_db.exe
package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

func logToFile(msg string) {
	exe, _ := os.Executable()
	logPath := filepath.Join(filepath.Dir(exe), "test_db.log")
	f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		fmt.Fprintln(f, time.Now().Format("15:04:05"), msg)
	}
}

func main() {
	logToFile("=== test_db START ===")
	fmt.Println("[test_db] Starting...")

	// Test 1: Open in-memory database
	logToFile("Test 1: opening in-memory DB...")
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		logToFile(fmt.Sprintf("Test 1 FAILED: %v", err))
		fmt.Printf("[FAIL] Open memory: %v\n", err)
		os.Exit(1)
	}
	logToFile("Test 1 OK: in-memory DB opened")

	// Test 2: Create table
	logToFile("Test 2: creating table...")
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		logToFile(fmt.Sprintf("Test 2 FAILED: %v", err))
		fmt.Printf("[FAIL] Create table: %v\n", err)
		db.Close()
		os.Exit(1)
	}
	logToFile("Test 2 OK: table created")

	// Test 3: Insert and query
	logToFile("Test 3: insert and query...")
	_, err = db.Exec("INSERT INTO test (name) VALUES (?)", "hello world")
	if err != nil {
		logToFile(fmt.Sprintf("Test 3 FAILED: %v", err))
		fmt.Printf("[FAIL] Insert: %v\n", err)
		db.Close()
		os.Exit(1)
	}
	var name string
	db.QueryRow("SELECT name FROM test WHERE id = 1").Scan(&name)
	logToFile(fmt.Sprintf("Test 3 OK: queried name = %q", name))

	// Test 4: Open file database
	logToFile("Test 4: opening file DB...")
	exe, _ := os.Executable()
	dbPath := filepath.Join(filepath.Dir(exe), "test.db")
	db2, err := sql.Open("sqlite", dbPath)
	if err != nil {
		logToFile(fmt.Sprintf("Test 4 FAILED: %v", err))
		fmt.Printf("[FAIL] Open file: %v\n", err)
		db.Close()
		os.Exit(1)
	}
	_, err = db2.Exec("CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		logToFile(fmt.Sprintf("Test 4 FAILED: create table: %v", err))
	} else {
		logToFile("Test 4 OK: file DB works")
	}
	db2.Close()
	db.Close()

	logToFile("=== ALL TESTS PASSED ===")
	fmt.Println("[test_db] All tests passed!")
}
