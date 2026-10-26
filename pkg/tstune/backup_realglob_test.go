package tstune

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetBackupsRealGlob exercises getBackups with the real filepath.Glob
// instead of the mocked filepathGlobFn used by TestGetBackups. The mock
// returns paths built with the same separator as the code under test, which
// hides separator mismatches between os.TempDir and filepath.Glob.
func TestGetBackupsRealGlob(t *testing.T) {
	tmpDir := t.TempDir()
	// os.TempDir consults TMPDIR on POSIX and TMP/TEMP on Windows.
	t.Setenv("TMPDIR", tmpDir)
	t.Setenv("TMP", tmpDir)
	t.Setenv("TEMP", tmpDir)

	oldFilepathGlobFn := filepathGlobFn
	filepathGlobFn = filepath.Glob
	defer func() { filepathGlobFn = oldFilepathGlobFn }()

	want := filepath.Join(tmpDir, backupFilePrefix+"201901181122")
	if err := os.WriteFile(want, []byte("foo\n"), 0600); err != nil {
		t.Fatalf("could not create backup file: %v", err)
	}
	invalid := filepath.Join(tmpDir, backupFilePrefix+"notadate")
	if err := os.WriteFile(invalid, []byte("foo\n"), 0600); err != nil {
		t.Fatalf("could not create invalid backup file: %v", err)
	}

	files, err := getBackups()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("incorrect number of backups: got %d want 1 (%v)", len(files), files)
	}
	if files[0] != want {
		t.Errorf("incorrect backup path: got\n%s\nwant\n%s", files[0], want)
	}
}
