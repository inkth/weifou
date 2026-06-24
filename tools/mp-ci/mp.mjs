#!/usr/bin/env node
// 微否小程序命令行上传 / 预览（基于官方 miniprogram-ci）。
// 编译设置从 weifou-miniapp/project.config.json 读取并对齐，appid 同源。
//
//   node mp.mjs check                       预检：校验工程 / appid / 密钥（不上传）
//   node mp.mjs upload [--ver x] [--desc y]  上传开发版到微信（之后到公众平台设体验版 / 提交审核）
//   node mp.mjs preview [--desc y]           生成预览二维码（终端 + preview-qr.jpg）
//
// 密钥路径默认 tools/mp-ci/private.<appid>.key，可用环境变量 WX_PRIVATE_KEY_PATH 覆盖。
import ci from 'miniprogram-ci';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { execSync } from 'node:child_process';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const PROJECT_PATH = path.resolve(__dirname, '../../weifou-miniapp');
const args = process.argv.slice(2);
const cmd = args[0] || 'check';

const flag = (name, def) => { const i = args.indexOf(`--${name}`); return i >= 0 ? args[i + 1] : def; };
const git = (c, def = '') => { try { return execSync(c, { cwd: __dirname }).toString().trim(); } catch { return def; } };
const die = (msg) => { console.error(`\n\x1b[31m✗\x1b[0m ${msg}\n`); process.exit(1); };
const ok = (msg) => console.log(`\x1b[32m✓\x1b[0m ${msg}`);

function keyHelp() {
  return `还差上传密钥：${KEY_PATH}
  1) 打开 mp.weixin.qq.com → 开发管理 → 开发设置 →「小程序代码上传」
  2) 生成并下载「上传密钥」，重命名为 private.${APPID}.key，放进 tools/mp-ci/ 或仓库根（与 server.pem 同处，均已 gitignore，脚本自动识别）
  3) 同页把你的公网出口 IP 加进「IP 白名单」，否则上传会被微信拒绝
     查公网 IP： curl ifconfig.me`;
}

// —— 工程 / 配置（appid + 编译设置都从 project.config.json 取，单一真源）——
const confPath = path.join(PROJECT_PATH, 'project.config.json');
if (!fs.existsSync(confPath)) die(`找不到 project.config.json：${confPath}`);
const conf = JSON.parse(fs.readFileSync(confPath, 'utf8'));
const APPID = conf.appid;
if (!APPID) die('project.config.json 缺少 appid');

// 密钥位置：环境变量 WX_PRIVATE_KEY_PATH > tools/mp-ci/ > 仓库根（与 server.pem 同处）。取第一个存在的。
const DEFAULT_KEY = path.join(__dirname, `private.${APPID}.key`);
const KEY_CANDIDATES = [process.env.WX_PRIVATE_KEY_PATH, DEFAULT_KEY, path.resolve(__dirname, '../..', `private.${APPID}.key`)].filter(Boolean);
const KEY_PATH = KEY_CANDIDATES.find((p) => fs.existsSync(p)) || DEFAULT_KEY;

const s = conf.setting || {};
const setting = {
  es6: s.es6 ?? true,
  es7: s.enhance ?? true,
  minify: s.minified ?? true,
  minifyJS: s.minifyJS ?? s.minified ?? true,
  minifyWXML: s.minifyWXML ?? s.minified ?? true,
  minifyWXSS: s.minifyWXSS ?? s.minified ?? true,
  autoPrefixWXSS: s.postcss ?? true,
  ignoreUploadUnusedFiles: s.ignoreUploadUnusedFiles ?? true,
};

const now = new Date();
const stamp = `${now.getFullYear()}.${String(now.getMonth() + 1).padStart(2, '0')}.${String(now.getDate()).padStart(2, '0')}`;
const version = flag('ver', `${stamp}-${git('git rev-parse --short HEAD', 'dev')}`);
const desc = flag('desc', git('git log -1 --pretty=%s', `CLI 上传 ${stamp}`));
const robot = Number(flag('robot', '1'));

console.log(`命令    ${cmd}`);
console.log(`appid   ${APPID}`);
console.log(`工程    ${PROJECT_PATH}`);
console.log(`密钥    ${KEY_PATH}${fs.existsSync(KEY_PATH) ? '' : '  \x1b[31m(缺失)\x1b[0m'}`);
console.log(`版本    ${version}`);
console.log(`描述    ${desc}`);
console.log('');

if (cmd === 'check') {
  if (fs.existsSync(KEY_PATH)) ok('全部就位：工程 / appid / 密钥均在，可以 `npm run upload` 了。');
  else { console.log('\x1b[33m!\x1b[0m ' + keyHelp()); }
  process.exit(0);
}

if (cmd !== 'upload' && cmd !== 'preview') die(`未知命令：${cmd}（可用：check | upload | preview）`);
if (!fs.existsSync(KEY_PATH)) die(keyHelp());

const project = new ci.Project({
  appid: APPID,
  type: 'miniProgram',
  projectPath: PROJECT_PATH,
  privateKeyPath: KEY_PATH,
  ignores: ['node_modules/**'],
});

if (cmd === 'upload') {
  await ci.upload({ project, version, desc, setting, robot, onProgressUpdate: () => {} });
  ok('上传成功！到 mp.weixin.qq.com → 管理 → 版本管理 → 开发版本 查看；可设为「体验版」或「提交审核」→ 发布。');
} else {
  const out = path.join(__dirname, 'preview-qr.jpg');
  await ci.preview({ project, version, desc, setting, robot, qrcodeFormat: 'terminal', qrcodeOutputDest: out, pagePath: '', onProgressUpdate: () => {} });
  ok(`预览二维码已打印（终端）并存到 ${out}，用微信扫码真机预览。`);
}
