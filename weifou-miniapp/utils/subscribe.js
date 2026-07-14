// 订阅消息授权：访客请求「已回答」通知；主人在 inbox 请求「新提问 / 新线索」通知。
//
// ⚠️ 需先在公众平台「功能→订阅消息」申请一次性模板，把模板 ID 填到下面四个常量
//   （并与服务端 .env 的 WX_SUBSCRIBE_*_TMPL_ID 对齐）。
//   未配置时全部静默降级：不弹授权、不报错，闭环退化为「进页刷新」。
const ANSWERED_TMPL_ID = ''; // 访客：你的提问已回答
const NEW_QUESTION_TMPL_ID = ''; // 主人：有人问了你的问答箱（召回）
const LEAD_TMPL_ID = ''; // 主人：有新的访客线索
// 学员：学习提醒（明天叫你继续）。建议申请「学习提醒/上课提醒」类目模板，
// 字段顺序：课程(thing1)+提醒内容(thing2)+时间(time3)，与服务端 WX_SUBSCRIBE_LEARN_REMIND_TMPL_ID 对齐。
const LEARN_REMIND_TMPL_ID = '';

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

// 访客：提问后授权「已回答」通知。
function requestQuestionNotify() {
  return requestSubscribe([ANSWERED_TMPL_ID]);
}

// 主人：授权「新提问」通知
function requestNewQuestionNotify() {
  return requestSubscribe([NEW_QUESTION_TMPL_ID]);
}

// 主人：授权「新访客线索」通知（免费线索召回）
function requestLeadNotify() {
  return requestSubscribe([LEAD_TMPL_ID]);
}

// 学员：课后「明天这个点叫你继续吗」的提醒承诺授权。
// 返回 res[LEARN_REMIND_TMPL_ID] === 'accept' 表示用户点了允许。
function requestLearnRemind() {
  return requestSubscribe([LEARN_REMIND_TMPL_ID]);
}

module.exports = {
  requestQuestionNotify,
  requestNewQuestionNotify,
  requestLeadNotify,
  requestLearnRemind,
  LEARN_REMIND_TMPL_ID,
  ANSWERED_TMPL_ID,
  NEW_QUESTION_TMPL_ID,
  LEAD_TMPL_ID,
};
