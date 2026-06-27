// 会员（一价解锁全部工具 Agent）接口封装。
// 安卓:小程序内微信支付开通;iOS:不能在此开通,改为留意向(intent)+ 站外 H5（后续）。
const { request } = require('./request');
const { ensureLogin } = require('./auth');
const { requestPayment, requestVirtualPayment } = require('./pay');

// 我的会员状态 + 可购买套餐：{ isMember, expiresAt, plans:[{id,slug,name,days,price}] }
function status() {
  return request({ url: '/membership/status' });
}

// wx.login 取新鲜 code（虚拟支付下单需后端用它实时换 session_key 做用户态签名）。
function wxLogin() {
  return new Promise((resolve, reject) => {
    wx.login({
      success: (r) => (r && r.code ? resolve(r.code) : reject({ message: '登录失败，请重试' })),
      fail: () => reject({ message: '登录失败，请重试' }),
    });
  });
}

// 开通/续费会员（虚拟支付，iOS/安卓统一；2026-04 起虚拟商品合规通道）。
// 异步发货：requestVirtualPayment 成功仅代表支付完成，会员到账靠服务端发货回调（秒级），
// 调用方应在成功后稍候刷新 status。
async function openMembership(planId) {
  await ensureLogin();
  const code = await wxLogin();
  const data = await request({ url: '/membership/vpay-order', method: 'POST', data: { planId, code } });
  await requestVirtualPayment(data.vpayParams);
  return data.orderId;
}

// 开通/续费会员（旧通道：安卓微信支付 JSAPI）。
// ⚠️ DEPRECATED：虚拟商品 2026-04 起必须走虚拟支付（openMembership），此通道已违规，仅暂留过渡。
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

module.exports = { status, openMembership, buyMembership, leaveIntent, h5Link };
