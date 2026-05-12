package main

import (
	"testing"
)

func TestParseLSOutput_StandardFormat(t *testing.T) {
	output := `total 64
-rw-r--r-- 1 root root   123 Jan 15 10:30 README.md
drwxr-xr-x 2 root root  4096 Jan 15 10:30 bin
-rwxr-xr-x 1 root root 12345 Jan 15 10:30 run.sh
lrwxrwxrwx 1 root root    12 Jan 15 10:30 link -> target.txt
-rw-r--r-- 1 root root  1024 Feb 28 23:59 config.yaml
drwx------ 2 root root  4096 Mar 1 00:00 secret`

	files := parseLSOutput(output)

	if len(files) != 5 {
		t.Fatalf("expected 5 files, got %d: %+v", len(files), files)
	}

	// Check README.md
	if files[0].Name != "README.md" {
		t.Errorf("expected README.md, got %s", files[0].Name)
	}
	if files[0].IsDir {
		t.Errorf("README.md should not be a directory")
	}
	if files[0].Size != "123 B" {
		t.Errorf("expected 123 B, got %s", files[0].Size)
	}
	if files[0].Permission != "-rw-r--r--" {
		t.Errorf("expected -rw-r--r--, got %s", files[0].Permission)
	}

	// Check bin directory
	if files[1].Name != "bin" {
		t.Errorf("expected bin, got %s", files[1].Name)
	}
	if !files[1].IsDir {
		t.Errorf("bin should be a directory")
	}

	// Check run.sh
	if files[2].Name != "run.sh" {
		t.Errorf("expected run.sh, got %s", files[2].Name)
	}
	if files[2].Size != "12.1 KB" {
		t.Errorf("expected 12.1 KB, got %s", files[2].Size)
	}

	// Check symlink (should resolve to link name only)
	if files[3].Name != "link" {
		t.Errorf("expected link, got %s", files[3].Name)
	}

	// Check config.yaml
	if files[4].Name != "config.yaml" {
		t.Errorf("expected config.yaml, got %s", files[4].Name)
	}
	if files[4].Size != "1.0 KB" {
		t.Errorf("expected 1.0 KB, got %s", files[4].Size)
	}
}

func TestParseLSOutput_LongISOTimeFormat(t *testing.T) {
	output := `total 12
-rw-r--r-- 1 user group 2048 2024-01-15 10:30:00 app.log
drwxr-xr-x 2 user group 4096 2024-03-20 14:22:00 data`

	files := parseLSOutput(output)

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	if files[0].Name != "app.log" {
		t.Errorf("expected app.log, got %s", files[0].Name)
	}
	if files[0].IsDir {
		t.Errorf("app.log should not be a directory")
	}
	if files[0].Size != "2.0 KB" {
		t.Errorf("expected 2.0 KB, got %s", files[0].Size)
	}

	if files[1].Name != "data" {
		t.Errorf("expected data, got %s", files[1].Name)
	}
	if !files[1].IsDir {
		t.Errorf("data should be a directory")
	}
}

func TestParseLSOutput_FilenameWithSpaces(t *testing.T) {
	output := `total 8
-rw-r--r-- 1 user group 100 Jan 15 10:30 my file with spaces.txt
-rw-r--r-- 1 user group 200 Jan 15 10:30 normal.txt`

	files := parseLSOutput(output)

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	if files[0].Name != "my file with spaces.txt" {
		t.Errorf("expected 'my file with spaces.txt', got '%s'", files[0].Name)
	}
	if files[1].Name != "normal.txt" {
		t.Errorf("expected 'normal.txt', got '%s'", files[1].Name)
	}
}

func TestParseLSOutput_SkipsHiddenFiles(t *testing.T) {
	output := `total 12
drwxr-xr-x 2 user group 4096 Jan 15 10:30 .
drwxr-xr-x 3 user group 4096 Jan 15 10:30 ..
-rw-r--r-- 1 user group 100 Jan 15 10:30 .hidden
-rw-r--r-- 1 user group 200 Jan 15 10:30 visible.txt`

	files := parseLSOutput(output)

	if len(files) != 1 {
		t.Fatalf("expected 1 file (only visible.txt), got %d", len(files))
	}
	if files[0].Name != "visible.txt" {
		t.Errorf("expected visible.txt, got %s", files[0].Name)
	}
}

func TestParseLSOutput_Empty(t *testing.T) {
	files := parseLSOutput("")
	if len(files) != 0 {
		t.Errorf("expected 0 files for empty input, got %d", len(files))
	}

	files = parseLSOutput("total 0\n")
	if len(files) != 0 {
		t.Errorf("expected 0 files for empty listing, got %d", len(files))
	}
}

func TestParseLSOutput_Garbage(t *testing.T) {
	output := `not a valid ls output
this should be skipped
also garbage`
	files := parseLSOutput(output)
	if len(files) != 0 {
		t.Errorf("expected 0 files for garbage, got %d", len(files))
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0", "0 B"},
		{"512", "512 B"},
		{"1023", "1023 B"},
		{"1024", "1.0 KB"},
		{"1536", "1.5 KB"},
		{"1048576", "1.0 MB"},
		{"1073741824", "1.0 GB"},
		{"notanumber", "notanumber"},
	}

	for _, tt := range tests {
		got := formatSize(tt.input)
		if got != tt.want {
			t.Errorf("formatSize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/", "'/'"},
		{"/var/log", "'/var/log'"},
		{"/path/with spaces", "'/path/with spaces'"},
		{"/path/with'quote", "'/path/with'\\''quote'"},
	}

	for _, tt := range tests {
		got := escapePath(tt.input)
		if got != tt.want {
			t.Errorf("escapePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
