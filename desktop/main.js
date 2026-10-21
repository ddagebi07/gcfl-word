'use strict';
/**
 * GCFL 한글 편집기 — 데스크톱(오프라인) 셸
 *  - 모든 리소스는 app/ 폴더(웹 빌드)에서 읽는다. 네트워크 연결이 필요 없다.
 *  - 문서 저장은 내려받기를 가로채 Windows 표준 '저장' 대화상자로 보낸다.
 */
const { app, BrowserWindow, Menu, protocol, dialog, shell, session, ipcMain } = require('electron');
const path = require('path');
const fs = require('fs');
const { pathToFileURL } = require('url');

const APP_NAME = 'GCFL 한글 편집기';
const SCHEME = 'gcfl-app';
const APP_DIR = path.join(__dirname, 'app');
const ICON = path.join(__dirname, 'build', 'icon.ico');
const DOC_EXT = /\.(hwp|hwpx|hml)$/i;

protocol.registerSchemesAsPrivileged([
  { scheme: SCHEME, privileges: { standard: true, secure: true, supportFetchAPI: true, stream: true, corsEnabled: true } },
]);

const MIME = {
  '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8',
  '.wasm': 'application/wasm', '.png': 'image/png', '.svg': 'image/svg+xml', '.ico': 'image/x-icon',
  '.woff2': 'font/woff2', '.woff': 'font/woff', '.json': 'application/json', '.webmanifest': 'application/manifest+json',
};

// 명령줄/더블클릭으로 넘어온 문서를 ?url= 로 열기 위해 잠시 등록해 두는 경로 표
const pendingFiles = new Map();
let lastDir = null;

function docFromArgv(argv) {
  return argv.slice(1).find((a) => DOC_EXT.test(a) && fs.existsSync(a)) || null;
}

function registerProtocol() {
  protocol.handle(SCHEME, async (request) => {
    const url = new URL(request.url);
    if (url.pathname.startsWith('/__open/')) {
      const file = pendingFiles.get(url.pathname.slice('/__open/'.length));
      if (!file) return new Response('not found', { status: 404 });
      return new Response(await fs.promises.readFile(file), { headers: { 'content-type': 'application/octet-stream' } });
    }
    let rel = decodeURIComponent(url.pathname);
    if (rel === '/' || rel === '') rel = '/index.html';
    const full = path.normalize(path.join(APP_DIR, rel));
    if (!full.startsWith(APP_DIR + path.sep)) return new Response('forbidden', { status: 403 });
    try {
      const data = await fs.promises.readFile(full);
      return new Response(data, { headers: { 'content-type': MIME[path.extname(full).toLowerCase()] || 'application/octet-stream' } });
    } catch {
      return new Response('not found', { status: 404 });
    }
  });
}

function createWindow(docPath) {
  const win = new BrowserWindow({
    width: 1360, height: 880, minWidth: 900, minHeight: 600,
    title: APP_NAME, icon: ICON, backgroundColor: '#07090e', show: false,
    webPreferences: { contextIsolation: true, sandbox: true, spellcheck: false, preload: path.join(__dirname, 'preload.js') },
  });
  win.once('ready-to-show', () => win.show());
  win.setMenuBarVisibility(false);

  let query = '';
  if (docPath) {
    lastDir = path.dirname(docPath);
    const id = Math.random().toString(36).slice(2);
    pendingFiles.set(id, docPath);
    query = `?url=${encodeURIComponent(`${SCHEME}://app/__open/${id}`)}&filename=${encodeURIComponent(path.basename(docPath))}`;
  }
  win.loadURL(`${SCHEME}://app/index.html${query}`);

  // 외부 링크는 기본 브라우저로, 인쇄 창 등 앱 내부 창은 허용
  win.webContents.setWindowOpenHandler(({ url }) => {
    if (/^https?:/i.test(url)) { shell.openExternal(url); return { action: 'deny' }; }
    return { action: 'allow' };
  });
  win.webContents.on('will-navigate', (e, url) => {
    if (!url.startsWith(`${SCHEME}://`)) { e.preventDefault(); if (/^https?:/i.test(url)) shell.openExternal(url); }
  });
  // 저장하지 않은 변경이 있을 때 닫기 → 확인 대화상자
  win.webContents.on('will-prevent-unload', (e) => {
    const choice = dialog.showMessageBoxSync(win, {
      type: 'question', buttons: ['저장하지 않고 닫기', '취소'], defaultId: 1, cancelId: 1,
      title: APP_NAME, message: '저장하지 않은 변경 사항이 있어요.', detail: '그대로 닫으면 작성한 내용이 사라집니다.',
    });
    if (choice === 0) e.preventDefault();
  });
  return win;
}

function setupDownloads() {
  session.defaultSession.on('will-download', (event, item, wc) => {
    const win = BrowserWindow.fromWebContents(wc) || BrowserWindow.getFocusedWindow();
    const name = item.getFilename() || 'document.hwp';
    const ext = path.extname(name).slice(1).toLowerCase();
    const filters = {
      hwp: [{ name: '한글 문서 (*.hwp)', extensions: ['hwp'] }],
      hwpx: [{ name: '한글 문서 (*.hwpx)', extensions: ['hwpx'] }],
      doc: [{ name: 'Word 문서 (*.doc)', extensions: ['doc'] }],
      html: [{ name: '웹 문서 (*.html)', extensions: ['html'] }],
      pdf: [{ name: 'PDF 문서 (*.pdf)', extensions: ['pdf'] }],
    }[ext] || [];
    const dir = lastDir || app.getPath('documents');
    const target = dialog.showSaveDialogSync(win, {
      title: '저장', defaultPath: path.join(dir, name), filters: [...filters, { name: '모든 파일', extensions: ['*'] }],
    });
    if (!target) { item.cancel(); return; }
    lastDir = path.dirname(target);
    item.setSavePath(target);
    item.once('done', (_e, state) => {
      if (state === 'completed' && win && !win.isDestroyed()) win.setTitle(`${path.basename(target)} — ${APP_NAME}`);
      if (state === 'interrupted') dialog.showErrorBox(APP_NAME, '파일을 저장하지 못했어요. 저장 위치를 확인해 주세요.');
    });
  });
}

ipcMain.on('gcfl:opened', (_e, file) => {
  if (typeof file === 'string' && file) lastDir = path.dirname(file);
});

ipcMain.on('gcfl:default-apps', () => {
  // Windows 10 은 기본 앱을 프로그램이 직접 바꾸지 못하게 막으므로 설정 화면을 열어 준다.
  if (process.platform === 'win32') shell.openExternal('ms-settings:defaultapps');
});

const gotLock = app.requestSingleInstanceLock();
if (!gotLock) {
  app.quit();
} else {
  app.on('second-instance', (_e, argv) => {
    const doc = docFromArgv(argv);
    if (doc) createWindow(doc);
    else {
      const w = BrowserWindow.getAllWindows()[0];
      if (w) { if (w.isMinimized()) w.restore(); w.focus(); }
    }
  });
  app.whenReady().then(() => {
    app.setAppUserModelId('kr.co.gcflarchive.word');
    Menu.setApplicationMenu(null);
    registerProtocol();
    setupDownloads();
    createWindow(docFromArgv(process.argv));
  });
  app.on('window-all-closed', () => app.quit());
}
