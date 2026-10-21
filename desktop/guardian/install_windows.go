package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
	"golang.org/x/term"

	"github.com/ddagebi07/gcfl-word/desktop/guardian/internal/guard"
)

func isAdmin() bool { return windows.GetCurrentProcessToken().IsElevated() }

func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	return string(b), err
}

func serviceExists() bool {
	m, err := mgr.Connect()
	if err != nil {
		return false
	}
	defer m.Disconnect()
	s, err := m.OpenService(guard.ServiceName)
	if err != nil {
		return false
	}
	s.Close()
	return true
}

// cliInstall turns the extracted program folder into a protected installation.
// The installer runs it once per PC (as administrator) with GCFLWORD_PW set.
func cliInstall() int {
	if !isAdmin() {
		fmt.Println("관리자 권한이 필요해요.")
		return 1
	}
	if serviceExists() {
		fmt.Println("이미 설치되어 있어요. 다시 설치하려면 먼저 제거해 주세요.")
		return 1
	}
	if !fileExists(filepath.Join(installDir, appExeName)) {
		fmt.Println("이 폴더에서", appExeName, "를 찾지 못했어요. 설치 프로그램으로 설치해 주세요.")
		return 1
	}
	pw := os.Getenv(guardExeEnv)
	if pw == "" {
		var err error
		if pw, err = readPassword(fmt.Sprintf("관리자 비밀번호를 정해 주세요 (%d자 이상): ", guard.MinPasswordLength)); err != nil {
			fmt.Println("비밀번호를 읽지 못했어요.")
			return 1
		}
		again, _ := readPassword("한 번 더 입력해 주세요: ")
		if pw != again {
			fmt.Println("두 비밀번호가 달라요.")
			return 1
		}
	}
	_ = os.Unsetenv(guardExeEnv)
	cfg, err := guard.NewConfig(pw)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	step := func(what string, err error) bool {
		if err != nil {
			fmt.Printf("%s 실패: %v\n", what, err)
			return false
		}
		return true
	}
	if !step("폴더 만들기", os.MkdirAll(dataDir, 0o755)) {
		return 1
	}
	raw, _ := json.Marshal(cfg)
	if !step("설정 저장", os.WriteFile(configPath, raw, 0o600)) || !step("백업 만들기", snapshot()) {
		return 1
	}

	m, err := mgr.Connect()
	if !step("서비스 관리자 연결", err) {
		return 1
	}
	defer m.Disconnect()
	s, err := m.CreateService(guard.ServiceName, filepath.Join(installDir, filepath.Base(selfExe)), mgr.Config{
		DisplayName: guard.ServiceDisplayName, StartType: mgr.StartAutomatic,
		Description: appLabel + " 파일이 지워지거나 바뀌지 않게 지키고, 지워지면 되살립니다.",
	}, "service")
	if !step("서비스 만들기", err) {
		return 1
	}
	defer s.Close()
	_ = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: time.Second}, {Type: mgr.ServiceRestart, Delay: time.Second}, {Type: mgr.ServiceRestart, Delay: 5 * time.Second},
	}, 86400)
	_ = s.SetRecoveryActionsOnNonCrashFailures(true)
	if !step("서비스 시작", s.Start()) {
		return 1
	}
	time.Sleep(3 * time.Second) // the service applies the ACLs itself
	fmt.Println("보호 서비스를 설치했어요.")
	return 0
}

// cliUninstall removes the program after the service has checked the password.
func cliUninstall() int {
	if !serviceExists() {
		fmt.Println("설치되어 있지 않아요.")
		return 1
	}
	pw, err := readPassword("관리자 비밀번호: ")
	if err != nil {
		return 1
	}
	if err := askGuard(guard.Request{Op: "uninstall", Password: pw}); err != nil {
		fmt.Println(err)
		return 1
	}
	fmt.Println("제거하고 있어요. 잠시 뒤에 모두 사라져요.")
	return 0
}

// cliMaintenance lets the administrator change the program files for a while (e.g. to update it).
func cliMaintenance() int {
	pw, err := readPassword("관리자 비밀번호: ")
	if err != nil {
		return 1
	}
	if err := askGuard(guard.Request{Op: "maintenance", Password: pw, Minutes: 10}); err != nil {
		fmt.Println(err)
		return 1
	}
	fmt.Println("10분 동안 설치 폴더를 바꿀 수 있어요. 끝나면 자동으로 다시 보호해요. 일찍 끝내려면 resume 을 실행하세요.")
	return 0
}

// cliResume ends a maintenance window early (no password needed: it only turns protection back on).
func cliResume() int {
	if err := askGuard(guard.Request{Op: "resume"}); err != nil {
		fmt.Println(err)
		return 1
	}
	fmt.Println("보호를 다시 켰어요.")
	return 0
}
