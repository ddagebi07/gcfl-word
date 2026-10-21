package main

import (
	"os"
	"path/filepath"
)

const (
	appExeName  = "GCFL-Hangul-Editor.exe" // the editor itself; its presence marks a valid install folder
	appLabel    = "GCFL 한글 편집기"
	systemSID   = "S-1-5-18"
	adminsSID   = "S-1-5-32-544"
	guardExeEnv = "GCFLWORD_PW" // lets the installer pass the administrator password without a console prompt
)

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

var (
	selfExe, _ = os.Executable()
	installDir = filepath.Dir(selfExe)
	dataDir    = filepath.Join(envOr("ProgramData", `C:\ProgramData`), "GCFLHangulEditor")
	configPath = filepath.Join(dataDir, "guard.json")
	backupZip  = filepath.Join(dataDir, "backup.zip")
	manifestFn = filepath.Join(dataDir, "manifest.json")
	// Shortcuts and file-association keys the installer creates for all users.
	startMenuDir = filepath.Join(envOr("ProgramData", `C:\ProgramData`), `Microsoft\Windows\Start Menu\Programs`, appLabel)
	desktopLnk   = filepath.Join(envOr("PUBLIC", `C:\Users\Public`), "Desktop", appLabel+".lnk")
)
