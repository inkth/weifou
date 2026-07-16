// 订阅消息授权：访客请求「已回答」通知；主人在 inbox 请求「新提问 / 新线索」通知；学员请求「学习提醒」。
//
// 模板 ID 由服务端 /config/entries 下发（subscribeTmpls，真源=服务器 .env 的 WX_SUBSCRIBE_*_TMPL_ID）：
// 公众平台申请到模板后填服务器配置重启即全链路生效，前端无需发版。
// 未配置（空串/拉取失败）时全部静默降级：不弹授权、不报错，闭环退化为「进页刷新」。
const { loadEntries } = require('./entries');

let _tmpls = null;

// 拉取（并缓存）模板 ID 表：{ answered, newQuestion, lead, learnRemind }。
function loadTmpls() {
  if (_tmpls) return Promise.resolve(_tmpls);
  return loadEntries().then((e) => {
    _tmpls = (e && e.subscribeTmpls) || {};
    return _tmpls;
  });
}

// 某模板是否已配置（同步读缓存；须先有人 loadTmpls 过，否则按未配置处理）。
function tmplReady(name) {
  return !!(_tmpls && _tmpls[name]);
}

// 请求一次性订阅授权；空模板自动剔除，全空则直接 resolve（降级，不打扰用户）。
function requestSubscribe(tmplIds) {
  const ids = (tmplIds || []).filter(Boolean);
  if (!ids.length) return Promise.resolve({ skipped: true });
  return new Promise((resolve) => {
    wx.requestSubscribeMessage({
      tmplIds: ids,
      success: (res) => resolve(res),
      fail: () => resolve({ failed: true }),
    });
  });
}

// 按模板名请求授权，统一返回 { accepted, skipped }——调用方不再关心具体模板 ID。
function requestFor(name) {
  return loadTmpls().then((t) => {
    const id = t[name];
    if (!id) return { skipped: true, accepted: false };
    return requestSubscribe([id]).then((res) => ({ accepted: !!(res && res[id] === 'accept') }));
  });
}

// 访客：提问后授权「已回答」通知。
function requestQuestionNotify() {
  return requestFor('answered');
}

// 主人：授权「新提问」通知
function requestNewQuestionNotify() {
  return requestFor('newQuestion');
}

// 主人：授权「新访客线索」通知（免费线索召回）
function requestLeadNotify() {
  return requestFor('lead');
}

// 学员：课后「明天这个点叫你继续吗」的提醒承诺授权。res.accepted === true 表示用户点了允许。
function requestLearnRemind() {
  return requestFor('learnRemind');
}

module.exports = {
  loadTmpls,
  tmplReady,
  requestQuestionNotify,
  requestNewQuestionNotify,
  requestLeadNotify,
  requestLearnRemind,
};
