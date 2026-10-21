package guard

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Manifest maps a slash-separated relative path to the size the file had at install time.
type Manifest map[string]int64

// BuildManifest lists every file under dir.
func BuildManifest(dir string) (Manifest, error) {
	m := Manifest{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		m[filepath.ToSlash(rel)] = info.Size()
		return nil
	})
	return m, err
}

// Missing returns the files that are gone or have a different size than the manifest says.
func (m Manifest) Missing(dir string) []string {
	var bad []string
	for rel, size := range m {
		fi, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || fi.IsDir() || fi.Size() != size {
			bad = append(bad, rel)
		}
	}
	return bad
}

// WriteBackup zips dir into dst (a temp file is renamed into place).
func WriteBackup(dir, dst string, skip func(rel string) bool) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".new"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(out)
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if skip != nil && skip(rel) {
			return nil
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: rel, Method: zip.Deflate})
		if err != nil {
			return err
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(w, in)
		return err
	})
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// Restore puts the named files back from the backup zip. Files that cannot be
// written (for instance the running service executable) are skipped, not fatal.
func Restore(backup, dir string, rels []string) (restored int, err error) {
	if len(rels) == 0 {
		return 0, nil
	}
	want := make(map[string]bool, len(rels))
	for _, r := range rels {
		want[r] = true
	}
	zr, err := zip.OpenReader(backup)
	if err != nil {
		return 0, err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if !want[f.Name] {
			continue
		}
		target := filepath.Join(dir, filepath.FromSlash(f.Name))
		if !strings.HasPrefix(target, filepath.Clean(dir)+string(filepath.Separator)) {
			continue // never write outside dir
		}
		if err := extractOne(f, target); err == nil {
			restored++
		}
	}
	if restored == 0 {
		return 0, errors.New("백업에서 되살린 파일이 없어요")
	}
	return restored, nil
}

func extractOne(f *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	in, err := f.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := target + ".restore"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	_ = os.Remove(target)
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("restore %s: %w", f.Name, err)
	}
	return nil
}
