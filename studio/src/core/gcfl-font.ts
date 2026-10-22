/**
 * GCFL 한글 편집기 — 글꼴은 Pretendard GOV(굵기 9가지)만 쓴다.
 *
 * - 화면 표시·측정: 캔버스 글꼴 설정이 어떤 이름으로 들어와도 Pretendard GOV 로 바꾼다(wasm-bridge).
 * - 글꼴 목록: 아래 9개 굵기만 보여 준다(toolbar).
 * - 글꼴 파일은 fonts/PretendardGOV-*.woff2 (OFL). 쓰는 굵기만 그때 내려받는다.
 */

export interface GovWeight {
  /** 문서에 저장되는 글꼴 이름 */
  face: string;
  /** 파일 이름 조각 */
  file: string;
  weight: number;
  label: string;
}

export const GOV_FAMILY = 'Pretendard GOV';

export const GOV_WEIGHTS: readonly GovWeight[] = [
  { face: 'Pretendard GOV Thin', file: 'Thin', weight: 100, label: 'Thin · 100' },
  { face: 'Pretendard GOV ExtraLight', file: 'ExtraLight', weight: 200, label: 'ExtraLight · 200' },
  { face: 'Pretendard GOV Light', file: 'Light', weight: 300, label: 'Light · 300' },
  { face: 'Pretendard GOV', file: 'Regular', weight: 400, label: 'Regular · 400' },
  { face: 'Pretendard GOV Medium', file: 'Medium', weight: 500, label: 'Medium · 500' },
  { face: 'Pretendard GOV SemiBold', file: 'SemiBold', weight: 600, label: 'SemiBold · 600' },
  { face: 'Pretendard GOV Bold', file: 'Bold', weight: 700, label: 'Bold · 700' },
  { face: 'Pretendard GOV ExtraBold', file: 'ExtraBold', weight: 800, label: 'ExtraBold · 800' },
  { face: 'Pretendard GOV Black', file: 'Black', weight: 900, label: 'Black · 900' },
];

export const GOV_FACE_NAMES: readonly string[] = GOV_WEIGHTS.map(w => w.face);

const REGULAR = GOV_WEIGHTS[3];
const BOLD = GOV_WEIGHTS[6];
const byLowerName = new Map(GOV_WEIGHTS.map(w => [w.face.toLowerCase(), w]));

function unquote(name: string): string {
  return name.trim().replace(/^["']|["']$/g, '').trim();
}

/** 문서가 요구한 글꼴 이름을 9개 굵기 중 하나로 정한다. 모르는 이름은 보통(400), 굵게면 Bold(700). */
export function resolveGovWeight(name: string, bold = false): GovWeight {
  const known = byLowerName.get(unquote(name).toLowerCase());
  if (known) return known;
  return bold ? BOLD : REGULAR;
}

/** 글꼴 선택 목록에 보일 이름(어떤 이름이든 9개 중 하나로). */
export function govFaceOf(name: string): string {
  return resolveGovWeight(name).face;
}

/**
 * 캔버스 CSS font 문자열(`italic bold 14.5px "함초롬바탕", serif`)을 Pretendard GOV 로 고쳐 쓴다.
 * 굵게는 굵은 파일(Bold)로 대신하므로 합성 굵게가 겹치지 않게 bold 토큰을 뺀다.
 */
export function forceGovCssFont(cssFont: string): string {
  const pxIdx = cssFont.indexOf('px ');
  if (pxIdx < 0) return cssFont;
  let prefix = cssFont.substring(0, pxIdx + 3);
  const familyPart = cssFont.substring(pxIdx + 3);
  const first = familyPart.split(',')[0] ?? '';
  const bold = /(^|\s)(bold|bolder|[6-9]00)(\s|$)/i.test(prefix);
  const w = resolveGovWeight(first, bold);
  prefix = prefix.replace(/(^|\s)(bold|bolder|[1-9]00|normal)(?=\s)/gi, '$1');
  const lastSpace = prefix.lastIndexOf(' ', prefix.length - 2);
  const head = prefix.substring(0, lastSpace + 1).replace(/\s+/g, ' ');
  const size = prefix.substring(lastSpace + 1);
  return `${head}${size}"${w.face}", "${GOV_FAMILY}", sans-serif`;
}

const loading = new Map<string, Promise<void>>();

function fontUrl(file: string): string {
  return new URL(`fonts/PretendardGOV-${file}.woff2`, document.baseURI).href;
}

function ensureStyle(): void {
  const id = 'gcfl-gov-font-faces';
  if (document.getElementById(id)) return;
  const style = document.createElement('style');
  style.id = id;
  const faces = GOV_WEIGHTS.map(w =>
    `@font-face { font-family: "${w.face}"; src: url("${fontUrl(w.file)}") format("woff2"); font-weight: normal; font-style: normal; font-display: block; }`);
  // 화면(UI)용 — 굵기 100~900 을 한 가족 이름으로
  const ui = GOV_WEIGHTS.map(w =>
    `@font-face { font-family: "${GOV_FAMILY} UI"; src: url("${fontUrl(w.file)}") format("woff2"); font-weight: ${w.weight}; font-style: normal; font-display: swap; }`);
  style.textContent = [...faces, ...ui].join('\n');
  document.head.appendChild(style);
}

/** 글꼴 파일을 내려받아 캔버스에서 바로 쓸 수 있게 한다. */
export function loadGovFaces(names: readonly string[]): Promise<void> {
  ensureStyle();
  const wanted = new Set<string>([REGULAR.face, BOLD.face]);
  for (const n of names) wanted.add(resolveGovWeight(n).face);
  return Promise.all([...wanted].map(face => {
    let p = loading.get(face);
    if (!p) {
      p = document.fonts.load(`16px "${face}"`, '가나다ABC abc123').then(() => undefined, () => undefined);
      loading.set(face, p);
    }
    return p;
  })).then(() => undefined);
}
