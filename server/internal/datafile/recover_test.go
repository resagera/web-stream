package datafile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicWritesContentAndCleansTemp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := WriteAtomic(path, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("WriteAtomic() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("state content = %q", data)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat state: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("state mode = %v, want 0600", got)
	}

	matches, err := filepath.Glob(filepath.Join(dir, ".state.json.tmp-*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temp files left after WriteAtomic: %v", matches)
	}
}
