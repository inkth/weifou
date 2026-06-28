// 找对象 · 择偶测试接口封装（B 形态：我 × 平台预设原型 → 匹配度 + 择偶画像）。
// AI 动态出题、LLM 直接打分；产出的择偶画像由后端回写进主人分身的知识库。
const { request } = require('./request');
const { ensureLogin } = require('./auth');

// 开始测试：AI 动态出题。返回 { quizId, questions:[{id,text,options:[{key,label}]}] }。
async function startQuiz() {
  await ensureLogin();
  return request({ url: '/dating/start', method: 'POST' });
}

// 提交答案：返回 { profile, matches:[{archetype,score,reason}] }。
// answers: [{ questionId, key }]
async function submitQuiz(quizId, answers) {
  await ensureLogin();
  return request({ url: '/dating/submit', method: 'POST', data: { quizId, answers } });
}

// 最近一次结果（用于回看）。无结果返回 null。
async function latestResult() {
  await ensureLogin();
  return request({ url: '/dating/result' });
}

module.exports = { startQuiz, submitQuiz, latestResult };
