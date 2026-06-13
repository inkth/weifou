// 漏斗埋点：fire-and-forget，失败静默。
// 事件类型白名单与服务端 internal/visit/visit.go 的 allowedEventTypes 同步维护。
const { request } = require('./request');

function track(type, profileId, meta) {
  request({
    url: '/visit/event/track',
    method: 'POST',
    data: { type, profileId: profileId || '', meta: meta || '' },
  }).catch(() => {});
}

module.exports = { track };
