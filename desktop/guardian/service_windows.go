package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/ddagebi07/gcfl-word/desktop/guardian/internal/guard"
)

// guardService runs as SYSTEM from boot. It keeps the install folder locked down,
// puts back any file that was deleted or changed (from the backup made at install),
// and is the only one that can uninstall the program (after checking the
// administrator password).
type guardService struct {
	cfg      *guard.Config
	limiter  *guard.Limiter
	manifest guard.Manifest

	mu           sync.Mutex
	pausedUntil  time.Time
	uninstalling bool
}

func runService() error {
	return svc.Run(guard.ServiceName, &guardService{limiter: guard.NewLimiter()})
}

func (g *guardService) loadState() error {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	g.cfg = &guard.Config{}
	if err := json.Unmarshal(raw, g.cfg); err != nil {
		return err
	}
	return g.loadManifest()
}

func (g *guardService) loadManifest() error {
	raw, err := os.ReadFile(manifestFn)
	if err != nil {
		return err
	}
	m := guard.Manifest{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	g.mu.Lock()
	g.manifest = m
	g.mu.Unlock()
	return nil
}

func (g *guardService) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	if err := g.loadState(); err != nil {
		return false, 1 // not installed properly; the SCM will restart us, which is harmless
	}
	harden()

	ln, err := listenGuard()
	if err == nil {
		go serveGuard(ln, g.handle)
		defer ln.Close()
	}
	// Not accepting Stop: nobody, administrators included, can stop the service from outside.
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptShutdown}
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	ticks := 0
	for {
		select {
		case <-tick.C:
			ticks++
			g.tick(ticks%20 == 0)
		case c := <-requests:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Shutdown:
				return false, 0
			}
		}
		g.mu.Lock()
		gone := g.uninstalling
		g.mu.Unlock()
		if gone {
			return false, 0
		}
	}
}

func (g *guardService) tick(rehard bool) {
	g.mu.Lock()
	until, gone, m := g.pausedUntil, g.uninstalling, g.manifest
	g.mu.Unlock()
	if gone {
		return
	}
	if time.Now().Before(until) { // maintenance window: administrators may change files
		return
	}
	if !until.IsZero() { // it just ended: take the administrator's changes as the new normal
		g.mu.Lock()
		g.pausedUntil = time.Time{}
		g.mu.Unlock()
		g.resnapshot()
		return
	}
	if bad := m.Missing(installDir); len(bad) > 0 {
		if _, err := guard.Restore(backupZip, installDir, bad); err == nil {
			rehard = true
		}
	}
	if !fileExists(installDir) {
		rehard = false
	}
	if rehard {
		harden()
	}
}

// resnapshot records the install folder as it is now (manifest + backup) and locks it again.
func (g *guardService) resnapshot() {
	if err := snapshot(); err == nil {
		_ = g.loadManifest()
	}
	harden()
}

// snapshot writes the manifest and the backup zip for the current install folder.
func snapshot() error {
	m, err := guard.BuildManifest(installDir)
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(m)
	if err := os.WriteFile(manifestFn, raw, 0o600); err != nil {
		return err
	}
	return guard.WriteBackup(installDir, backupZip, nil)
}

func (g *guardService) handle(req guard.Request) guard.Response {
	switch req.Op {
	case "ping":
		return guard.Response{OK: true}
	case "resume": // ends a maintenance window early; only turns protection back on
		g.mu.Lock()
		g.pausedUntil = time.Now() // tick() sees an expired window and re-snapshots
		g.mu.Unlock()
		return guard.Response{OK: true}
	case "maintenance", "uninstall":
	default:
		return guard.Response{Error: "알 수 없는 요청이에요."}
	}
	if err := g.limiter.Allow(); err != nil {
		return guard.Response{Error: err.Error()}
	}
	ok := g.cfg.Verify(req.Password)
	if penalty := g.limiter.Record(ok); !ok {
		time.Sleep(penalty)
		return guard.Response{Error: "비밀번호가 맞지 않아요."}
	}
	switch req.Op {
	case "maintenance":
		minutes := min(max(req.Minutes, 1), 60)
		g.mu.Lock()
		g.pausedUntil = time.Now().Add(time.Duration(minutes) * time.Minute)
		g.mu.Unlock()
		openForMaintenance()
	case "uninstall":
		go g.uninstall()
	}
	return guard.Response{OK: true}
}

// uninstall removes everything. It runs inside the service because only SYSTEM may.
func (g *guardService) uninstall() {
	g.mu.Lock()
	g.uninstalling = true
	g.mu.Unlock()
	killApp()
	_ = setACL(installDir, windows.SE_FILE_OBJECT, "D:PAI(A;OICI;FA;;;SY)")
	if m, err := mgr.Connect(); err == nil {
		if s, err := m.OpenService(guard.ServiceName); err == nil {
			_ = s.Delete()
			s.Close()
		}
		m.Disconnect()
	}
	applySafeMode(false)
	removeIntegrations()
	// The files are in use until this process ends, so a detached command removes them a moment later.
	script := fmt.Sprintf(`ping -n 4 127.0.0.1 >nul & rd /s /q "%s" & rd /s /q "%s"`, installDir, dataDir)
	cmd := exec.Command("cmd.exe", "/C", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP}
	_ = cmd.Start()
}

// removeIntegrations deletes the shortcuts and file-association entries the installer made.
func removeIntegrations() {
	_ = os.RemoveAll(startMenuDir)
	_ = os.Remove(desktopLnk)
	_ = exec.Command("reg", "delete", `HKLM\SOFTWARE\Classes\GCFL.HangulDoc`, "/f").Run()
	_ = exec.Command("reg", "delete", `HKLM\SOFTWARE\Classes\.hwp\OpenWithProgids`, "/v", "GCFL.HangulDoc", "/f").Run()
	_ = exec.Command("reg", "delete", `HKLM\SOFTWARE\Classes\.hwpx\OpenWithProgids`, "/v", "GCFL.HangulDoc", "/f").Run()
}

// killApp closes every running copy of the editor so its files can be deleted.
func killApp() {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		if !strings.EqualFold(windows.UTF16ToString(e.ExeFile[:]), appExeName) {
			continue
		}
		if h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, e.ProcessID); err == nil {
			_ = windows.TerminateProcess(h, 0)
			windows.CloseHandle(h)
		}
	}
}
