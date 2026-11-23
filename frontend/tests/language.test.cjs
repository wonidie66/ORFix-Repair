'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const sourcePath = path.join(__dirname, '..', 'dist', 'language.js');
const source = fs.readFileSync(sourcePath, 'utf8');

const windowStub = {
  prompt: message => message,
  dispatchEvent: () => true
};
const documentStub = { documentElement: {}, title: '' };
const localStorageStub = {
  getItem: key => key === 'orfix-language' ? 'en' : null,
  setItem: () => {}
};
const ReactStub = {
  createElement: (type, props, ...children) => ({ type, props: props || {}, children })
};

vm.runInNewContext(source, {
  window: windowStub,
  document: documentStub,
  localStorage: localStorageStub,
  React: ReactStub,
  Event: class { constructor(type) { this.type = type; } }
}, { filename: 'language.js' });

const i18n = windowStub.ORFixI18n;
assert.ok(i18n, 'i18n API should be initialized in English mode');

const paths = [
  'D:\\modPlayer\\xxmi\\GIMI\\Mods\\菲谢尔',
  'C:/Games/GIMI/Mods/菲谢尔',
  '\\\\NAS\\GIMI\\Mods\\菲谢尔',
  '/home/player/GIMI/Mods/菲谢尔',
  './Mods/菲谢尔',
  '../Mods/菲谢尔'
];

for (const folderPath of paths) {
  assert.equal(i18n.translateText(folderPath), folderPath,
    'filesystem paths containing CJK characters must be preserved verbatim');

  const element = i18n.createElement('span', { title: folderPath }, folderPath);
  assert.equal(element.props.title, folderPath, 'path title attribute must be preserved');
  assert.equal(element.children[0], folderPath, 'path text node must be preserved');
}

assert.equal(i18n.translateText('浏览目录'), 'Browse Folder',
  'ordinary UI labels should still be translated');
const fourSlotLog = 'TextureOverrideAsterBody [NNFix] 分支内贴图已匹配，绘制前插入 NNFix';
assert.match(i18n.translateText(fourSlotLog), /Branch texture layout matched; inserted before drawing/,
  'the generic four-slot repair log should be translated without changing technical identifiers');
assert.match(i18n.translateText('这是未收录的诊断信息'), /Additional diagnostic needs manual review/,
  'unknown Chinese diagnostics should retain the existing safe fallback');

console.log('PASS: CJK folder paths are preserved; UI labels and diagnostic fallback still work.');
