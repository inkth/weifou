// 会员（一价解锁全部工具 Agent）接口封装。
// 安卓:小程序内微信支付开通;iOS:不能在此开通,改为留意向(intent)+ 站外 H5（后续）。
const { request } = require('./request');
const { ensureLogin } = require('./auth');
const { requestPayment } = require('./pay');

// 我的会员状态 + 可购买套餐：{ isMember, expiresAt, plans:[{id,slug,name,days,price}] }
function status() {
  return request({ url: '/membership/status' });
}

// 开通/续费会员（安卓）。iOS 后端会拒(IOS_VIRTUAL_PAY_BLOCKED)。
async function buyMembership(planId) {
  await ensureLogin();
  const data = await request({ url: '/membership/buy', method: 'POST', data: { planId } });
  await requestPayment(data.payParams);
  return request({ url: `/payment/orders/${data.orderId}` });
}

// 留意向（多为 iOS 用户）：当下不能在小程序内开通,记录意向便于后续触达。
function leaveIntent() {
  return request({ url: '/membership/intent', method: 'POST' });
}

// 换取"在浏览器开通"链接（带短时交接令牌）。用户复制到外部 Safari 打开即可开通,会员入同一账号。
function h5Link() {
  return request({ url: '/membership/h5-link', method: 'POST' });
}

module.exports = { status, buyMembership, leaveIntent, h5Link };
