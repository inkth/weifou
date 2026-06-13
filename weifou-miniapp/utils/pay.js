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

// 分 → 元 展示
function fenToYuan(fen) {
  return (fen / 100).toFixed(2).replace(/\.00$/, '');
}

module.exports = { requestPayment, fenToYuan };
