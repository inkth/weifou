// 信任徽章文案：社会证明全部派生自既有交易数据（profile 接口的 trust 字段，无新埋点）。
// 冷启动护栏：人数 / 样本过小（<3）时返回空串——宁可不展示，也不要"0 人咨询"的反效果，
// 此时页面只保留"答不上全额退"等保障文案。
// who: 'asked' → "付费问过 TA"（对话场景）；其余 → "付费咨询过"（主页 / 付费页）。
function buildTrustLine(trust, who) {
  if (!trust) return '';
  const verb = who === 'asked' ? '付费问过 TA' : '付费咨询过';
  const parts = [];
  if (trust.consultedPeople >= 3) parts.push(`已有 ${trust.consultedPeople} 人${verb}`);
  if (trust.answeredCount >= 3 && trust.avgAnswerHours > 0) {
    const h = trust.avgAnswerHours;
    const t = h < 1 ? `${Math.max(1, Math.round(h * 60))} 分钟` : `${Math.round(h)} 小时`;
    parts.push(`平均 ${t}内回答`);
  }
  return parts.join(' · ');
}

module.exports = { buildTrustLine };
