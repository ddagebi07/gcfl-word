; GCFL 한글 편집기 — 오프라인 설치 프로그램 (32비트 x86 Windows 10 호환, 관리자 권한 불필요)
Unicode true
!include "MUI2.nsh"
!include "FileFunc.nsh"

!define APP_NAME "GCFL 한글 편집기"
!define APP_EXE "GCFL-Hangul-Editor.exe"
!define APP_VERSION "${VERSION}"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\GCFLHangulEditor"

Name "${APP_NAME}"
OutFile "${OUTFILE}"
InstallDir "$LOCALAPPDATA\Programs\GCFL Hangul Editor"
InstallDirRegKey HKCU "Software\GCFLHangulEditor" "InstallDir"
RequestExecutionLevel user
SetCompressor /SOLID lzma
SetCompressorDictSize 64
BrandingText "GCFL Archive"
VIProductVersion "${APP_VERSION}.0"
VIAddVersionKey /LANG=1042 "ProductName" "${APP_NAME}"
VIAddVersionKey /LANG=1042 "CompanyName" "GCFL Archive"
VIAddVersionKey /LANG=1042 "FileDescription" "${APP_NAME} 설치 프로그램"
VIAddVersionKey /LANG=1042 "FileVersion" "${APP_VERSION}"
VIAddVersionKey /LANG=1042 "LegalCopyright" "MIT License"

!define MUI_ICON "build\icon.ico"
!define MUI_UNICON "build\icon.ico"
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "${APP_NAME} 설치"
!define MUI_WELCOMEPAGE_TEXT "학교에서 쓰는 가벼운 한글(HWP/HWPX) 편집기입니다.$\r$\n$\r$\n인터넷 연결 없이 설치되며 관리자 권한이 필요하지 않아요.$\r$\n$\r$\nPowered by GCFL Archive"
!define MUI_FINISHPAGE_RUN "$INSTDIR\${APP_EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "지금 ${APP_NAME} 실행하기"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE-INSTALLER.txt"
!insertmacro MUI_PAGE_COMPONENTS
!define MUI_PAGE_CUSTOMFUNCTION_LEAVE DirLeave
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "Korean"

; 제거할 때 폴더째 지우므로, 선택한 폴더가 전용 하위 폴더가 아니면 하위 폴더를 붙인다.
Function DirLeave
  ${GetFileName} "$INSTDIR" $R0
  StrCmp $R0 "GCFL Hangul Editor" +2 0
    StrCpy $INSTDIR "$INSTDIR\GCFL Hangul Editor"
FunctionEnd

Section "${APP_NAME} (필수)" SecMain
  SectionIn RO
  SetOutPath "$INSTDIR"
  File /r "${SRCDIR}\*.*"
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE}" "" "$INSTDIR\${APP_EXE}" 0
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\제거.lnk" "$INSTDIR\Uninstall.exe"
  WriteRegStr HKCU "Software\GCFLHangulEditor" "InstallDir" "$INSTDIR"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "${APP_NAME}"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${APP_VERSION}"
  WriteRegStr HKCU "${UNINST_KEY}" "Publisher" "GCFL Archive"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${APP_EXE}"
  WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoRepair" 1
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKCU "${UNINST_KEY}" "EstimatedSize" "$0"
SectionEnd

Section "바탕화면 바로 가기" SecDesktop
  CreateShortcut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE}" "" "$INSTDIR\${APP_EXE}" 0
SectionEnd

Section /o "HWP·HWPX 파일의 '연결 프로그램' 목록에 추가" SecAssoc
  WriteRegStr HKCU "Software\Classes\GCFL.HangulDoc" "" "한글 문서"
  WriteRegStr HKCU "Software\Classes\GCFL.HangulDoc\DefaultIcon" "" "$INSTDIR\${APP_EXE},0"
  WriteRegStr HKCU "Software\Classes\GCFL.HangulDoc\shell\open" "FriendlyAppName" "${APP_NAME}"
  WriteRegStr HKCU "Software\Classes\GCFL.HangulDoc\shell\open\command" "" '"$INSTDIR\${APP_EXE}" "%1"'
  WriteRegStr HKCU "Software\Classes\.hwp\OpenWithProgids" "GCFL.HangulDoc" ""
  WriteRegStr HKCU "Software\Classes\.hwpx\OpenWithProgids" "GCFL.HangulDoc" ""
  WriteRegDWORD HKCU "${UNINST_KEY}" "GcflAssoc" 1
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecMain} "프로그램 본체입니다."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "바탕화면에서 바로 실행할 수 있어요."
  !insertmacro MUI_DESCRIPTION_TEXT ${SecAssoc} "파일을 마우스 오른쪽 → '연결 프로그램'에서 고를 수 있게 합니다. 기본 프로그램은 바뀌지 않아요."
!insertmacro MUI_FUNCTION_DESCRIPTION_END

Section "Uninstall"
  Delete "$DESKTOP\${APP_NAME}.lnk"
  RMDir /r "$SMPROGRAMS\${APP_NAME}"
  DeleteRegKey HKCU "Software\Classes\GCFL.HangulDoc"
  DeleteRegValue HKCU "Software\Classes\.hwp\OpenWithProgids" "GCFL.HangulDoc"
  DeleteRegValue HKCU "Software\Classes\.hwpx\OpenWithProgids" "GCFL.HangulDoc"
  DeleteRegKey HKCU "${UNINST_KEY}"
  DeleteRegKey HKCU "Software\GCFLHangulEditor"
  RMDir /r "$INSTDIR"
SectionEnd
