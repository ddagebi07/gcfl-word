package guard

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPasswordAndLimiter(t *testing.T) {
	if _, err := NewConfig("short"); err == nil {
		t.Fatal("short password accepted")
	}
	c, err := NewConfig("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !c.Verify("correct horse") || c.Verify("wrong") {
		t.Fatal("verify")
	}
	now := time.Unix(0, 0)
	l := &Limiter{MaxFails: 2, Lockout: time.Minute, Now: func() time.Time { return now }}
	l.Record(false)
	l.Record(false)
	if l.Allow() == nil {
		t.Fatal("should be locked")
	}
	now = now.Add(2 * time.Minute)
	if l.Allow() != nil {
		t.Fatal("lock should end")
	}
}

func TestManifestBackupRestore(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, s string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("app.exe", "binary-bytes")
	write("resources/app/main.js", "console.log(1)")
	m, err := BuildManifest(dir)
	if err != nil || len(m) != 2 {
		t.Fatalf("manifest %v %v", m, err)
	}
	bak := filepath.Join(t.TempDir(), "b", "backup.zip")
	if err := WriteBackup(dir, bak, nil); err != nil {
		t.Fatal(err)
	}
	if len(m.Missing(dir)) != 0 {
		t.Fatal("nothing should be missing")
	}
	os.Remove(filepath.Join(dir, "app.exe"))
	write("resources/app/main.js", "tampered!!")
	bad := m.Missing(dir)
	if len(bad) != 2 {
		t.Fatalf("bad=%v", bad)
	}
	n, err := Restore(bak, dir, bad)
	if err != nil || n != 2 {
		t.Fatalf("restore %d %v", n, err)
	}
	if len(m.Missing(dir)) != 0 {
		t.Fatal("still missing after restore")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "resources/app/main.js"))
	if string(b) != "console.log(1)" {
		t.Fatal("content")
	}
	if _, err := Restore(bak, dir, []string{"../evil"}); err == nil {
		t.Fatal("path escape should not restore")
	}
}
