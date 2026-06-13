Page({
  data: {
    tipText: '理解你的背景中…',
  },
  onLoad() {
    const tips = [
      '理解你的背景中…',
      '提炼你的一句话介绍…',
      '挑选你的人格标签…',
      '完成主页排版…',
    ];
    let i = 0;
    this.timer = setInterval(() => {
      i = (i + 1) % tips.length;
      this.setData({ tipText: tips[i] });
    }, 1800);
  },
  onUnload() {
    if (this.timer) clearInterval(this.timer);
  },
});
