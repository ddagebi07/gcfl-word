# GCFL 한글 편집기

> Powered by **GCFL Archive** — 학교에서 쓰는 가볍고 직관적인 HWP/HWPX 워드프로세서

[rhwp](https://github.com/edwardkim/rhwp) (Edward Kim, MIT) 의 렌더링·편집 엔진 위에, 기초 기능만 남기고 화면을 새로 설계했습니다.

## 기능
- HWP / HWPX 열기, 새 문서, 텍스트 편집
- 글꼴·글자 크기·줄 간격·굵게·기울임·밑줄·취소선·글자색·문단 정렬
- 사진 넣기(캐럿 위치에 바로 삽입) / 모서리 점을 끌어 크기 조절 / `사진 크기` 로 정확한 크기 지정
- 저장(Ctrl+S) · 내보내기(HWP, HWPX, PDF, Word .doc, HTML) · 인쇄(Ctrl+P)

## 구성
| 폴더 | 내용 |
|---|---|
| `studio/` | 웹 앱 소스 (Vite + TypeScript). `npm run build` → `studio/dist` |
| `pkg/` | rhwp WebAssembly 코어 (`@rhwp/core` 0.8.7) |
| `desktop/` | 오프라인 설치형(Electron, Windows 10 32비트 x86) 소스·빌드 스크립트 |

## 빌드
```bash
# 웹
cd studio && npm install && npm run build        # 결과: studio/dist (상대 경로라 어느 하위 폴더에도 올릴 수 있음)

# Windows 10 32비트 설치 프로그램 + 포터블 zip (Linux/macOS 에서 실행; node, unzip, zip, curl, makensis 필요)
cd desktop && ./build-win32.sh                   # 결과: desktop/out/*.exe, *.zip
```
웹 빌드는 외부 네트워크(CDN)를 쓰지 않습니다. 글꼴은 `studio/public/fonts` 에 포함된 오픈 라이선스 글꼴만 사용하며,
함초롬 등 한컴 글꼴은 포함되지 않아 대체 글꼴로 그려집니다(PC에 설치된 글꼴은 그대로 사용).

## 라이선스
MIT. `LICENSE`, `LICENSE-RHWP`, `THIRD_PARTY_LICENSES.md` 참고. 앱의 `ⓘ` 버튼에도 고지가 표시됩니다.
