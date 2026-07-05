// 底栏红点 · 主人待办计数（收件箱信号）。
// 待办 = 知识缺口(open) + 未跟进线索(new) + 待回答提问(pending，有人在等你回一句)。
// 与 pages/index 的 loadTodo 同源；抽到此处供自定义底栏「我的」红点 + 各页复用。
// 三个接口各自兜底，任一失败不影响其余；未登录/无名片时全 catch→0，自然不亮。
const { request } = require('./request');
const { hostQuestions } = require('./asyncq');

// 轻缓存：切 tab 会频繁触发，15s 内复用，避免每次都打三个接口。
let _cache = { count: 0, at: 0 };
const TTL = 15000;

async function fetchTodoCount({ force = false } = {}) {
  const now = Date.now();
  if (!force && now - _cache.at < TTL) return _cache.count;
  try {
    const [gaps, leads, pending] = await Promise.all([
      request({ url: '/profile/gaps' }).catch(() => []),
      request({ url: '/profile/leads' }).catch(() => []),
      hostQuestions('pending').catch(() => []),
    ]);
    const newLeads = (leads || []).filter((l) => l.status === 'new').length;
    const count = (gaps || []).length + newLeads + (pending || []).length;
    _cache = { count, at: now };
    return count;
  } catch (e) {
    return _cache.count;
  }
}

// 主人在某处清了待办后主动作废缓存，让下次 fetch 立刻拉新值。
function invalidateTodoCount() { _cache.at = 0; }

module.exports = { fetchTodoCount, invalidateTodoCount };
