// 异步提问（访客免费向本人提问，本人异步作答）的接口封装。
// AI 默认已对外作答；这是「让主人本人也回一句」的可选动作，一问一答闭环，不涉及支付。
const { request } = require('./request');
const { ensureLogin } = require('./auth');

// 访客免费提问。成功返回 { id, status }。失败抛 { code, message }。
async function askQuestion(profileId, question) {
  await ensureLogin();
  return request({
    url: '/async-question',
    method: 'POST',
    data: { profileId, question },
  });
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

module.exports = { askQuestion, answerQuestion, hostQuestions, myQuestions, questionDetail };
