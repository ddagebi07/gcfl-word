package main

import (
	"os"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/ddagebi07/gcfl-word/desktop/guardian/internal/guard"
)

// The protection works in layers; none of them is perfect alone, and an
// administrator who knows what they are doing can undo them with enough effort
// (taking ownership, running as SYSTEM, booting another OS). The aim is that a
// student cannot delete or break the program by accident or in a minute.
const (
	// SYSTEM only. Not even administrators can read the password hash or the backup.
	sddlDataDir = "D:PAI(A;OICI;FA;;;SY)"
	// SYSTEM may do everything; administrators and users may only read and run.
	sddlInstallDir = "D:PAI(A;OICI;FA;;;SY)(A;OICI;0x1200a9;;;BA)(A;OICI;0x1200a9;;;BU)"
	// During a maintenance window administrators may also change the files.
	sddlInstallDirOpen = "D:PAI(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)"
	// The service cannot be stopped, paused, reconfigured or deleted by anyone but itself.
	sddlService = "D:(A;;GA;;;SY)(A;;CCLCSWLOCRRC;;;BA)(A;;CCLCSWLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)"
	// Its registry key is read-only for administrators (no "Start=4", no deleting it).
	sddlServiceKey = "D:PAI(A;CI;KA;;;SY)(A;CI;KR;;;BA)(A;CI;KR;;;BU)"
)

func setACL(name string, typ windows.SE_OBJECT_TYPE, sddl string) error {
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	system, err := windows.StringToSid(systemSID)
	if err != nil {
		return err
	}
	// The owner may always rewrite the ACL, so SYSTEM must own it too.
	_ = windows.SetNamedSecurityInfo(name, typ, windows.OWNER_SECURITY_INFORMATION, system, nil, nil, nil)
	return windows.SetNamedSecurityInfo(name, typ, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
}

// applySafeMode keeps the service running in safe mode (on) or removes that entry (off).
func applySafeMode(on bool) {
	for _, set := range []string{"Minimal", "Network"} {
		path := `SYSTEM\CurrentControlSet\Control\SafeBoot\` + set + `\` + guard.ServiceName
		if on {
			if k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, path, registry.SET_VALUE); err == nil {
				_ = k.SetStringValue("", "Service")
				k.Close()
			}
		} else {
			_ = registry.DeleteKey(registry.LOCAL_MACHINE, path)
		}
	}
}

// harden (re)applies the ACLs and the safe-mode entry.
func harden() {
	_ = setACL(dataDir, windows.SE_FILE_OBJECT, sddlDataDir)
	_ = setACL(installDir, windows.SE_FILE_OBJECT, sddlInstallDir)
	_ = setACL(guard.ServiceName, windows.SE_SERVICE, sddlService)
	_ = setACL(`MACHINE\SYSTEM\CurrentControlSet\Services\`+guard.ServiceName, windows.SE_REGISTRY_KEY, sddlServiceKey)
	applySafeMode(true)
}

// openForMaintenance lets administrators change the install folder for a while.
func openForMaintenance() { _ = setACL(installDir, windows.SE_FILE_OBJECT, sddlInstallDirOpen) }

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }
