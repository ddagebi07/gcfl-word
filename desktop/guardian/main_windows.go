// Command gcfl-guard is the protection service of GCFL 한글 편집기. It keeps the
// installed program from being deleted, changed or uninstalled by accident.
package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/svc"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if isService, _ := svc.IsWindowsService(); isService {
		if err := runService(); err != nil {
			return 1
		}
		return 0
	}
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "install":
		return cliInstall()
	case "uninstall":
		return cliUninstall()
	case "maintenance":
		return cliMaintenance()
	case "resume":
		return cliResume()
	default:
		fmt.Println("사용법: gcfl-guard.exe [install | uninstall | maintenance | resume]")
		return 2
	}
}
