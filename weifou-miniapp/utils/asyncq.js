// 异步提问（访客免费向本人提问，本人异步作答）的接口封装。
// AI 默认已对外作答；这是「让主人本人也回一句」的可选动作，一问一答闭环，不涉及支付。
const { request } = require('./request');
const { ensureLogin } = require('./auth');

// 问答箱：访客（对主人匿名）向 TA 的 AI 分身问一句，分身据画像即时作答。
// 成功返回 { id, answer, status }；同时入库供主人围观、补一句。
async function qaboxAsk(profileId, question) {
  await ensureLogin();
  return request({
    url: '/async-question/qabox',
    method: 'POST',
    data: { profileId, question },
  });
}

// 访客：把 AI 已答的问题升温为「请本人亲自回答」，强制通知主人。成功返回 { id, escalatedAt }。
function escalateQuestion(id) {
  return request({ url: `/async-question/${id}/escalate`, method: 'POST' });
}

// 主人作答。payload 可为字符串（纯文字，向后兼容）或 { answer, voiceUrl, voiceDuration }。
function answerQuestion(id, payload) {
  const data = typeof payload === 'string' ? { answer: payload } : (payload || {});
  return request({ url: `/async-question/${id}/answer`, method: 'POST', data });
}

// 主人：收到的提问列表（status 可选过滤 pending/answered）
function hostQuestions(status) {
  return request({ url: `/async-question/host${status ? '?status=' + status : ''}` });
}

// 访客：我发起的提问列表
function myQuestions() {
  return request({ url: '/async-question/mine' });
}

// 详情（host / asker 均可，返回 role）
function questionDetail(id) {
  return request({ url: `/async-question/detail/${id}` });
}

module.exports = { qaboxAsk, escalateQuestion, answerQuestion, hostQuestions, myQuestions, questionDetail };
