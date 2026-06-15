// 订阅消息授权：访客付费提问后请求「已回答」通知；主人在 inbox 请求「新提问」通知。
//
// ⚠️ 需先在公众平台「功能→订阅消息」申请一次性模板，把模板 ID 填到下面两个常量
//   （并与服务端 .env 的 WX_SUBSCRIBE_ANSWERED_TMPL_ID / WX_SUBSCRIBE_NEW_QUESTION_TMPL_ID 对齐）。
//   未配置时全部静默降级：不弹授权、不报错，异步闭环退化为「进页刷新」。
const ANSWERED_TMPL_ID = ''; // 访客：你的提问已回答
const NEW_QUESTION_TMPL_ID = ''; // 主人：有新的付费提问

// 请求一次性订阅授权；模板为空时直接 resolve（降级，不打扰用户）。
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

// 访客：付费提问成功后授权「已回答」通知
function requestAnsweredNotify() {
  return requestSubscribe([ANSWERED_TMPL_ID]);
}

// 主人：授权「新提问」通知
function requestNewQuestionNotify() {
  return requestSubscribe([NEW_QUESTION_TMPL_ID]);
}

module.exports = {
  requestAnsweredNotify,
  requestNewQuestionNotify,
  ANSWERED_TMPL_ID,
  NEW_QUESTION_TMPL_ID,
};
