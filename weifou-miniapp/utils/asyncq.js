// 付费异步咨询（付费向本人提问，本人异步作答）的接口封装。
// 付费购买的是「主人本人作答」（真人服务），与 AI 回答无关。
const { request } = require('./request');
const { ensureLogin } = require('./auth');
const { requestPayment } = require('./pay');

// 访客付费提问并完成支付。source: 'profile' / 'chat_card'。
// 成功返回订单详情（含 asyncQuestionId）。失败抛 { code, message }，PAY_CANCEL=用户取消。
async function askQuestion(profileId, question, source) {
  await ensureLogin();
  const data = await request({
    url: '/async-question',
    method: 'POST',
    data: { profileId, question, source },
  });
  await requestPayment(data.payParams);
  return request({ url: `/payment/orders/${data.orderId}` });
}

// 定价（含 asyncEnabled / asyncPrice），复用 consult pricing 接口。
function fetchPricing(profileId) {
  return request({ url: `/consult/pricing/${profileId}` });
}

// 主人作答
function answerQuestion(id, answer) {
  return request({ url: `/async-question/${id}/answer`, method: 'POST', data: { answer } });
}

// 主人：收到的付费提问列表（status 可选过滤 paid/answered/refunded）
function hostQuestions(status) {
  return request({ url: `/async-question/host${status ? '?status=' + status : ''}` });
}

// 访客：我发起的付费提问列表
function myQuestions() {
  return request({ url: '/async-question/mine' });
}

// 详情（host / asker 均可，返回 role）
function questionDetail(id) {
  return request({ url: `/async-question/detail/${id}` });
}

module.exports = { askQuestion, fetchPricing, answerQuestion, hostQuestions, myQuestions, questionDetail };
