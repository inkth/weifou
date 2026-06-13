// 付费咨询 / 打赏的下单+支付封装，供 profile 页与 chat 对话内成交卡片复用。
const { request } = require('./request');
const { ensureLogin } = require('./auth');
const { requestPayment } = require('./pay');

// 预约档期并完成支付。source 标记成交来源（'profile' / 'chat_card'）。
// 成功后返回订单详情（含 consultSessionId）。失败抛出 { code, message }，
// 其中 code === 'PAY_CANCEL' 表示用户主动取消支付。
async function bookConsult(profileId, slotId, source) {
  await ensureLogin();
  const data = await request({
    url: '/payment/consult',
    method: 'POST',
    data: { profileId, slotId, source },
  });
  await requestPayment(data.payParams);
  return request({ url: `/payment/orders/${data.orderId}` });
}

// 打赏并完成支付。amountFen 为分。source 同上。
async function sendTip(profileId, amountFen, message, source) {
  await ensureLogin();
  const data = await request({
    url: '/payment/tip',
    method: 'POST',
    data: { profileId, amount: amountFen, message: message || undefined, source },
  });
  await requestPayment(data.payParams);
}

module.exports = { bookConsult, sendTip };
