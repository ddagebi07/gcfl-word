'use strict';
const { contextBridge, ipcRenderer, webFrame, webUtils } = require('electron');

// 파일 선택기(File System Access)는 쓰지 않고, 입력창 열기 + 저장 대화상자 흐름으로 통일한다.
try {
  webFrame.executeJavaScript('try{delete window.showOpenFilePicker;delete window.showSaveFilePicker;delete window.showDirectoryPicker;}catch(e){}');
} catch { /* ignore */ }

// 사용자가 연 파일의 폴더를 기억해 두었다가 저장 대화상자의 기본 위치로 쓴다.
window.addEventListener('change', (e) => {
  const t = e.target;
  if (t && t.tagName === 'INPUT' && t.type === 'file' && t.files && t.files[0]) {
    try { ipcRenderer.send('gcfl:opened', webUtils.getPathForFile(t.files[0])); } catch { /* ignore */ }
  }
}, true);
window.addEventListener('drop', (e) => {
  const f = e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files[0];
  if (f) { try { ipcRenderer.send('gcfl:opened', webUtils.getPathForFile(f)); } catch { /* ignore */ } }
}, true);

contextBridge.exposeInMainWorld('gcflDesktop', { platform: process.platform });
