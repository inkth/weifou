// 打赏的下单+支付封装，供 profile 页与 chat 复用。
const { request } = require('./request');
const { ensureLogin } = require('./auth');
const { requestPayment } = require('./pay');

// 打赏并完成支付。amountFen 为分。source 标记成交来源。
async function sendTip(profileId, amountFen, message, source) {
  await ensureLogin();
  const data = await request({
    url: '/payment/tip',
    method: 'POST',
    data: { profileId, amount: amountFen, message: message || undefined, source },
  });
  await requestPayment(data.payParams);
}

module.exports = { sendTip };
