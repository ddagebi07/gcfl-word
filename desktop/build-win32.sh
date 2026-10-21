#!/usr/bin/env bash
# GCFL 한글 편집기 — Windows 10 (32비트 x86) 오프라인 설치 프로그램 빌드 (Linux/macOS 에서 실행)
# 필요: node 18+, npm, unzip, curl, zip, makensis(NSIS 3)
set -euo pipefail
cd "$(dirname "$0")"
ELECTRON_VERSION="$(node -p "require('./package.json').devDependencies.electron")"
VERSION="$(node -p "require('./package.json').version")"
CACHE="${CACHE_DIR:-.cache}"; OUT="out"
mkdir -p "$CACHE" "$OUT"

echo "==> 1/5 웹 앱 빌드"
(cd ../studio && npm install --no-audit --no-fund >/dev/null && npm run build)
rm -rf "${PWD:?}/app" && cp -r ../studio/dist app

echo "==> 2/5 Electron ${ELECTRON_VERSION} (win32-ia32) 준비"
ZIP="$CACHE/electron-v${ELECTRON_VERSION}-win32-ia32.zip"
[ -f "$ZIP" ] || curl -fL -o "$ZIP" "https://github.com/electron/electron/releases/download/v${ELECTRON_VERSION}/electron-v${ELECTRON_VERSION}-win32-ia32.zip"
rm -rf "${OUT:?}/unpacked" && mkdir -p "$OUT/unpacked" && unzip -q "$ZIP" -d "$OUT/unpacked"
# 불필요한 언어 파일 제거 (한국어/영어만 유지)
find "${OUT:?}/unpacked/locales" -name '*.pak' ! -name 'ko.pak' ! -name 'en-US.pak' -delete
rm -rf "${OUT:?}/unpacked/resources/default_app.asar"

echo "==> 3/5 앱 배치 + 아이콘/제품 정보"
APPDIR="$OUT/unpacked/resources/app"; mkdir -p "$APPDIR/build"
cp main.js preload.js "$APPDIR/"; cp -r app "$APPDIR/app"; cp build/icon.ico build/icon.png "$APPDIR/build/"
node -e "const p=require('./package.json');delete p.devDependencies;require('fs').writeFileSync('$APPDIR/package.json',JSON.stringify(p,null,2))"
[ -d node_modules/resedit ] || npm install --no-audit --no-fund >/dev/null
node tools-patch-exe.mjs "$OUT/unpacked/electron.exe" build/icon.ico "$OUT/unpacked/GCFL-Hangul-Editor.exe" "$VERSION"
rm "${OUT:?}/unpacked/electron.exe"
cp ../LICENSE-RHWP "$OUT/unpacked/LICENSE-RHWP.txt"; cp LICENSE-INSTALLER.txt "$OUT/unpacked/LICENSE-GCFL.txt"

echo "==> 4/5 설치 프로그램 (NSIS)"
SETUP="GCFL-Hangul-Editor-Setup-${VERSION}-win32-x86.exe"
makensis -V2 -DVERSION="$VERSION" -DOUTFILE="$OUT/$SETUP" -DSRCDIR="$OUT/unpacked" installer.nsi

echo "==> 5/5 포터블 zip (설치 없이 압축만 풀어 실행)"
PORTABLE="GCFL-Hangul-Editor-Portable-${VERSION}-win32-x86.zip"
rm -f "${OUT:?}/${PORTABLE:?}"
(cd "$OUT" && mv unpacked "GCFL-Hangul-Editor" && zip -qr -9 "$PORTABLE" "GCFL-Hangul-Editor" && mv "GCFL-Hangul-Editor" unpacked)
ls -la "$OUT"/*.exe "$OUT"/*.zip
