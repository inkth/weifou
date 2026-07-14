// AI 工具 Agent（平台预设）接口封装。
// 解锁靠会员(见 utils/membership)，非会员每个 Agent 有几次免费体验。
// 一人一 Agent 支持多会话（ChatGPT 式）：历史会话列表 + 按会话取消息 + 续聊/新开。
const { request } = require('./request');

// 目录（含 freeTrialRemaining）
function listAgents() {
  return request({ url: '/agents' });
}

function myAgents() {
  return request({ url: '/agents/mine' });
}

function agentDetail(id) {
  return request({ url: `/agents/detail/${id}` });
}

// 我与该 Agent 的历史会话列表（最近活动倒序，含 title/lastMessage/updatedAt）
function agentSessions(agentId) {
  return request({ url: `/agents/sessions/${agentId}` });
}

// 取指定会话的消息流（按 sessionId）
function sessionMessages(sessionId) {
  return request({ url: `/agents/messages/${sessionId}` });
}

// 我在某「学习型」Agent（如英语陪练）的三维段位档案。
// 非学习型 Agent 返回 { enabled: false }；学习型返回 { enabled, level, levelName, fluency, accuracy, expression, assessed, note }。
function agentSkill(agentId) {
  return request({ url: `/agents/skill/${agentId}` });
}

// 我在某「概念型」学习 Agent（如学心理/会说话）的点亮进度。
// 非概念型返回 { enabled: false }；概念型返回 { enabled, total, lit, mastered, due,
// tiers:[{tier, lit, total, concepts:[{slug, name, blurb, hook, note, level, theme}]}] }（note=本课战报）。
function agentConcepts(agentId) {
  return request({ url: `/agents/concepts/${agentId}` });
}

// 对话。sessionId 续聊指定会话，空 = 新开一段；返回里带 sessionId（新建时回传新 id）。
// mode='review' = 复习挑战（概念型专用：快问快答已点亮概念，不开新课）。
// concept=<slug> = 从闯关地图点选指定关卡开课（概念型专用）。
// 会员畅用;非会员扣免费体验,耗尽抛 { code: 'MEMBERSHIP_REQUIRED' }。
function chatAgent(agentId, content, sessionId, mode, concept) {
  return request({
    url: `/agents/${agentId}/chat`,
    method: 'POST',
    data: { content, sessionId: sessionId || undefined, mode: mode || undefined, concept: concept || undefined },
  });
}

// 连续学习天数（跨学习 Agent 全局一条）。
function learnStreak() {
  return request({ url: '/agents/streak' });
}

// “我的”页一次性学习摘要：最近课程、连续天数、累计掌握与已学课程数。
function learningSummary() {
  return request({ url: '/agents/learning-summary' });
}

// 学习提醒承诺：订阅消息授权成功后落账，服务端明天这个点发一条提醒。
function remindLearn(agentId) {
  return request({ url: `/agents/${agentId}/remind`, method: 'POST', data: {} });
}

// 添加 / 移除「首页」（我的小队）。
function pinAgent(agentId) {
  return request({ url: '/home/agents/pin', method: 'POST', data: { agentId } });
}
function unpinAgent(agentId) {
  return request({ url: `/home/agents/pin/${agentId}`, method: 'DELETE' });
}

// 谁看过我（访客列表）。
function listVisitors() {
  return request({ url: '/visit/visitors' });
}

module.exports = {
  listAgents, myAgents, agentDetail, agentSessions, sessionMessages, agentSkill, agentConcepts,
  chatAgent, learnStreak, learningSummary, remindLearn, pinAgent, unpinAgent, listVisitors,
};
