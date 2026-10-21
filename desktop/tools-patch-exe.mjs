// electron.exe 에 GCFL 아이콘/제품 정보를 심는다 (Linux 에서도 동작하는 순수 JS 구현).
import * as ResEdit from 'resedit';
import fs from 'fs';

const [exePath, icoPath, outPath, version] = process.argv.slice(2);
const exe = ResEdit.NtExecutable.from(fs.readFileSync(exePath));
const res = ResEdit.NtExecutableResource.from(exe);

const ico = ResEdit.Data.IconFile.from(fs.readFileSync(icoPath));
const groups = ResEdit.Resource.IconGroupEntry.fromEntries(res.entries);
const groupId = groups.length ? groups[0].id : 1;
const lang = groups.length ? groups[0].lang : 1033;
ResEdit.Resource.IconGroupEntry.replaceIconsForResource(
  res.entries, groupId, lang, ico.icons.map((i) => i.data),
);

const vi = ResEdit.Resource.VersionInfo.fromEntries(res.entries)[0];
const ver = (version || '0.1.0').split('.').map(Number);
vi.setFileVersion(ver[0] || 0, ver[1] || 0, ver[2] || 0, 0, lang);
vi.setProductVersion(ver[0] || 0, ver[1] || 0, ver[2] || 0, 0, lang);
vi.setStringValues({ lang, codepage: 1200 }, {
  FileDescription: 'GCFL 한글 편집기', ProductName: 'GCFL 한글 편집기',
  CompanyName: 'GCFL Archive', LegalCopyright: 'MIT License — GCFL Archive / Edward Kim (rhwp)',
  OriginalFilename: 'GCFL-Hangul-Editor.exe', InternalName: 'GCFL-Hangul-Editor',
});
vi.outputToResourceEntries(res.entries);
res.outputResource(exe);
fs.writeFileSync(outPath, Buffer.from(exe.generate()));
console.log('patched', outPath);
