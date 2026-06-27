// 调起微信支付：传入后端返回的 payParams
function requestPayment(payParams) {
  return new Promise((resolve, reject) => {
    wx.requestPayment({
      timeStamp: payParams.timeStamp,
      nonceStr: payParams.nonceStr,
      package: payParams.package,
      signType: payParams.signType || 'RSA',
      paySign: payParams.paySign,
      success: () => resolve(true),
      fail: (err) => {
        if (err.errMsg && err.errMsg.includes('cancel')) {
          reject({ code: 'PAY_CANCEL', message: '已取消支付' });
        } else {
          reject({ code: 'PAY_FAIL', message: err.errMsg || '支付失败' });
        }
      },
    });
  });
}

// 调起虚拟支付（wx.requestVirtualPayment）：传入后端 vpay-order 返回的 vpayParams。
// 虚拟商品（会员/工具 Agent）专用；iOS 自动走苹果 IAP，安卓走微信虚拟支付。
function requestVirtualPayment(vp) {
  return new Promise((resolve, reject) => {
    if (typeof wx.requestVirtualPayment !== 'function') {
      reject({ code: 'VPAY_UNSUPPORTED', message: '当前微信版本过低，请升级后再开通' });
      return;
    }
    wx.requestVirtualPayment({
      signData: vp.signData,
      paySig: vp.paySig,
      signature: vp.signature,
      mode: vp.mode,
      success: () => resolve(true),
      fail: (err) => {
        const msg = (err && err.errMsg) || '';
        if (msg.includes('cancel')) reject({ code: 'PAY_CANCEL', message: '已取消支付' });
        else reject({ code: 'PAY_FAIL', message: msg || '支付失败' });
      },
    });
  });
}

// 分 → 元 展示
function fenToYuan(fen) {
  return (fen / 100).toFixed(2).replace(/\.00$/, '');
}

module.exports = { requestPayment, requestVirtualPayment, fenToYuan };
