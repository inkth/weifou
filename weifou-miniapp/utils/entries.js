// 入口可见性（iOS 虚拟支付红线的前端真源）。
// 后端 GET /config/entries 据 X-Platform 下发各入口在当前端是否可见；
// 拉取失败或未拉取时，用本地平台兜底（iOS 一律隐藏虚拟商品 / 工具 Agent）。
const { request } = require('./request');

let _entries = null;
let _loading = null;

function loadEntries(force) {
  if (_entries && !force) return Promise.resolve(_entries);
  if (_loading) return _loading;
  _loading = request({ url: '/config/entries' })
    .then((e) => {
      _entries = e || {};
      _loading = null;
      return _entries;
    })
    .catch(() => {
      _entries = {};
      _loading = null;
      return _entries;
    });
  return _loading;
}

// 通用入口可见性；未拉到该键时回退 fallback。
function entryVisible(name, fallback) {
  const fb = fallback === undefined ? true : fallback;
  if (_entries && _entries[name] !== undefined) return !!_entries[name];
  return fb;
}

// AI 工具 Agent（虚拟商品）：服务端明确则以服务端为准。
// 2026-04 起小程序虚拟支付双端合规，兜底放开（拉取失败也不再隐藏）。
function agentVisible() {
  if (_entries && _entries.agent !== undefined) return !!_entries.agent;
  return true;
}

module.exports = { loadEntries, entryVisible, agentVisible };
