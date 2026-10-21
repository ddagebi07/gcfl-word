import { defineConfig } from 'vite';
import { resolve, extname, join } from 'path';
import { readFileSync, readFile } from 'fs';

const configDir = import.meta.dirname;
const pkg = JSON.parse(readFileSync(resolve(configDir, 'package.json'), 'utf-8'));
const subsecondWasmDir = resolve(
  configDir,
  '..',
  'target',
  'rhwp-subsecond-vite',
);
const useSubsecondWasm = process.env.RHWP_SUBSECOND === '1';
/**
 * hwpctrl 플러그인 포함 여부.
 *
 * `RHWP_WITHOUT_HWPCTRL=1` 로 빌드하면 studio 는 `npm/hwpctrl-ocx` 에 **빌드 시점에도** 묶이지
 * 않는다 — `main.ts` 의 동적 import 가 상수 분기 안에 있어 통째로 tree-shake 되고, 산출물에
 * `studio-plugin` 청크 자체가 남지 않는다. studio 만 떼어 배포할 때 쓴다.
 */
const withHwpctrl = process.env.RHWP_WITHOUT_HWPCTRL !== '1';

export default defineConfig({
  base: './',
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
    // 셀프 호스팅 빌드에서 외부(CDN) 웹폰트 로드를 빌드 시점에 끈다.
    // 확장 storage 설정(disableExternalWebFonts)이 있으면 그 값이 우선한다.
    __RHWP_DISABLE_EXTERNAL_WEBFONTS__: JSON.stringify(
      process.env.RHWP_DISABLE_EXTERNAL_WEBFONTS === '1',
    ),
    __RHWP_HWPCTRL__: JSON.stringify(withHwpctrl),
  },
  resolve: {
    alias: {
      '@': resolve(configDir, 'src'),
      '@wasm/rhwp.js': useSubsecondWasm
        ? resolve(subsecondWasmDir, 'rhwp-subsecond.js')
        : resolve(configDir, '..', 'pkg', 'rhwp.js'),
      '@wasm': resolve(configDir, '..', 'pkg'),
      // 플러그인 패키지 — 동적 import 로만 들어오므로 별도 청크가 된다.
      // 올리지 않으면 코드도 로드되지 않는다(플러그인 없는 studio 의 초기 비용 0).
      '@rhwp/hwpctrl/studio-plugin': resolve(configDir, '..', 'npm', 'hwpctrl-ocx', 'src', 'studio-plugin.mjs'),
    },
  },
  server: {
    host: '127.0.0.1',
    port: 7700,
    proxy: useSubsecondWasm ? {
      '/_dioxus': {
        target: 'http://127.0.0.1:7711',
        ws: true,
      },
      '/wasm': {
        target: 'http://127.0.0.1:7711',
      },
    } : undefined,
    fs: {
      // [Task #741 후속] 외부 file path 그림 영역 영역 samples/ dir 영역 영역 fetch 가능 영역.
      allow: [
        configDir,
        resolve(configDir, '..', 'pkg'),
        subsecondWasmDir,
        resolve(configDir, '..', 'samples'),
        resolve(configDir, '..', 'npm', 'editor'),
      ],
    },
    watch: {
      ignored: ['**/librhwp-subsecond-patch-*.wasm'],
    },
  },
  plugins: [
    {
      name: 'ignore-subsecond-patch-artifacts',
      handleHotUpdate(context) {
        if (/librhwp-subsecond-patch-\d+\.wasm$/.test(context.file)) {
          return [];
        }
      },
    },
    // [Task #741 후속] dev 서버 영역 영역 /samples/* 경로 영역 영역 parent samples/ dir 영역
    // 영역 정적 serve 영역 — wasm-bridge.ts 영역 영역 외부 image fetch 영역 영역 영역.
    {
      name: 'serve-samples-dir',
      configureServer(server) {
        const samplesDir = resolve(configDir, '..', 'samples');
        server.middlewares.use('/samples', (req, res, next) => {
          if (!req.url) return next();
          // URL decode + sanitize (path traversal 차단)
          const reqPath = decodeURIComponent(req.url.split('?')[0]);
          const relPath = reqPath.replace(/^\/+/, '');
          if (relPath.includes('..')) { res.statusCode = 403; return res.end(); }
          const full = join(samplesDir, relPath);
          if (!full.startsWith(samplesDir)) { res.statusCode = 403; return res.end(); }
          readFile(full, (err: NodeJS.ErrnoException | null, data: Buffer) => {
            if (err) { res.statusCode = 404; return res.end(); }
            const ext = extname(full).toLowerCase();
            const mime: Record<string, string> = {
              '.gif': 'image/gif', '.jpg': 'image/jpeg', '.jpeg': 'image/jpeg',
              '.png': 'image/png', '.bmp': 'image/bmp', '.webp': 'image/webp',
            };
            res.setHeader('Content-Type', mime[ext] ?? 'application/octet-stream');
            // [Task #741 후속] OS 영역 절대 경로 영역 영역 response header 영역 노출 — JS
            // 영역 영역 dialog 영역 영역 한컴 viewer 정합 (D:\\... 영역 영역 영역 의 영역 영역) 영역.
            res.setHeader('X-File-Path', encodeURI(full));
            res.setHeader('Access-Control-Expose-Headers', 'X-File-Path');
            res.end(data);
          });
        });
      },
    },
  ],
});
