// Package toolagent — boss_cards.go：章末知识卡片内容表。
//
// 每门课章末的「综合关/Boss关」配一句策展知识结论 + 引用来源，通关时前端弹卡片展示（见 conceptProgressView
// 的 takeaway/source 字段、SeedConcepts 里按 slug 查表写入 AgentConcept）。learn-marketing 50 关平铺无章节
// 边界，不参与，通关仍走通用庆祝浮层。
package toolagent

var bossCardContent = map[string]bossCard{
	// ---- spoken-english（4）----
	"boss-daily-life":    {Takeaway: "救场的从不是背好的稿子，是接得住的听力", Source: "TBLT任务型教学"},
	"boss-travel-storm":  {Takeaway: "翻盘靠的是脱口而出的应急语块，不是现翻现译", Source: "语块Chunk理论"},
	"boss-work-sprint":   {Takeaway: "听不懂就问清楚，装懂才是职场英语的坑", Source: "CEFR can-do框架"},
	"boss-business-deal": {Takeaway: "撬动成交的是条件交换，不是一味降价", Source: "TBLT任务型教学"},

	// ---- learn-logic（6）----
	"boss-argument-autopsy": {Takeaway: "论证最先塌的那根柱子，往往是没说出口的前提", Source: "Toulmin论证模型"},
	"boss-comment-brawl":    {Takeaway: "攻击说话的人，从来驳不倒他说的道理", Source: "非形式谬误研究"},
	"boss-groupchat-debate": {Takeaway: "把复杂问题逼成非此即彼，本身就是一种耍赖", Source: "Bergstrom识谎术"},
	"boss-family-rumor":     {Takeaway: "数字听着吓人，先查它是哪来的，再信", Source: "SHEG横向阅读法"},
	"boss-research-claims":  {Takeaway: "相关性、个例、机制，三个都凑不成因果证据", Source: "Hill因果关系标准"},
	"boss-final-debate":     {Takeaway: "抓到对方犯规，不代表我这边就自动获胜", Source: "《Think Again》"},

	// ---- learn-psychology（10）----
	"bias-checkup":            {Takeaway: "确信感和正确性无关，越确信反而越该核查证据", Source: "Kahneman《思考快与慢》"},
	"emotion-sos":             {Takeaway: "情绪是身体发出的信号，读懂比压制更管用", Source: "Barrett情绪颗粒度"},
	"thought-court":           {Takeaway: "让你难受的不是事件本身，是你对事件的解读", Source: "Beck认知行为疗法"},
	"self-portrait":           {Takeaway: "自我价值不必靠比较撑起，善待自己更持久", Source: "Neff自我关怀"},
	"change-blueprint":        {Takeaway: "改变靠的是环境设计，不是发一次狠心的决心", Source: "Duhigg习惯回路"},
	"misunderstanding-repair": {Takeaway: "误会往往是拿别人的处境，当成了别人的人品", Source: "Rosenberg非暴力沟通"},
	"relationship-checkup":    {Takeaway: "关系好不好不看吵不吵架，看接不接得住对方的示好", Source: "Gottman实验室"},
	"crowd-immunity":          {Takeaway: "身边人越多越一致时，你的判断反而最不独立", Source: "Asch/Milgram实验"},
	"script-teardown":         {Takeaway: "话术能说动你，靠的不是道理，是踩中了你本能的按钮", Source: "Cialdini《影响力》"},
	"happiness-portfolio":     {Takeaway: "什么让人真正幸福，答案不是钱和成就，是关系的质量", Source: "哈佛成人发展研究"},

	// ---- learn-ai（4）----
	"rescue-prompt":    {Takeaway: "指令的好坏取决于你补全了多少缺失信息", Source: "GIGO原则"},
	"daily-boss":       {Takeaway: "AI跑腿好不好，取决于你给的约束够不够具体", Source: "具体性原则"},
	"project-boss":     {Takeaway: "复杂任务要拆解迭代，一次要求给不出好答案", Source: "分而治之"},
	"bs-detector-boss": {Takeaway: "答案越自信流畅，越可能藏着编造的细节", Source: "流畅性谬误"},

	// ---- learn-speaking（4）----
	"refuse-gauntlet":  {Takeaway: "干脆拒绝但话别说绝，才能拒事不拒人", Source: "把不说死"},
	"hard-talk-boss":   {Takeaway: "硬话要先说结论或先认错，别用铺垫拖延冲击", Source: "第一句见底"},
	"ask-gauntlet":     {Takeaway: "要东西时把苦劳换成价值，拒绝要换成具体承诺", Source: "DEAR MAN法"},
	"banquet-gauntlet": {Takeaway: "场面话要具体到细节，套话再多也不算真诚", Source: "上细节"},

	// ---- daodejing-full（9）----
	"boss1": {Takeaway: "反者道动不是宿命，而是提醒你盛极就该收手", Source: "《老子》第40章"},
	"boss2": {Takeaway: "无为不是躺平，是日日做减法、别把自己作死", Source: "《老子》第48章"},
	"boss3": {Takeaway: "治世无为不是撒手不管，是不折腾、防患未然", Source: "《老子》第64章"},
	"boss4": {Takeaway: "不争不是怂，是靠慈俭退让换来长久的成事底气", Source: "《老子》第67章"},
	"boss5": {Takeaway: "柔弱不是没底线，是能扛委屈、留有余地才活久", Source: "《老子》第76章"},
	"boss6": {Takeaway: "有无相生不必讲玄，事办妥了就该功成身退", Source: "《老子》第9章"},
	"boss7": {Takeaway: "虚静不是发呆，是让情绪沉淀后看清规律再动", Source: "《老子》第16章"},
	"boss8": {Takeaway: "道法自然不是随大流，是不硬拗本性、稳得住", Source: "《老子》第25章"},
	"boss9": {Takeaway: "全书终归不妄为，不是躺平，是知止而后成事", Source: "《老子》第37章"},

	// ---- learn-happiness（3）----
	"happiness-boss-check": {Takeaway: "别信预测：抵达、比较、花钱都会让幸福打折", Source: "Gilbert情感预测"},
	"happiness-boss-tools": {Takeaway: "幸福靠训练注意力，不是压抑负面情绪", Source: "走神47%研究"},
	"happiness-boss-plan":  {Takeaway: "幸福是可练的技能，靠具体行动不是靠运气", Source: "哈佛/耶鲁幸福课"},

	// ---- learn-habits（3）----
	"habits-boss-check":   {Takeaway: "习惯好坏是设计问题，不是自律或人品问题", Source: "Fogg行为公式"},
	"habits-boss-install": {Takeaway: "装习惯靠五件套设计，不靠目标和意志力", Source: "Fogg微习惯法"},
	"habits-boss-system":  {Takeaway: "习惯改变靠改系统设计，不靠意志力变强", Source: "Clear掌控习惯"},

	// ---- learn-dating（4）----
	"dating-boss-portrait": {Takeaway: "认清自己带进恋爱的常量，才不会带着老问题换新人", Source: "Marriage 101"},
	"dating-boss-judge":    {Takeaway: "判断心动看行为不看话语，尊重节奏最有分量", Source: "Logan Ury"},
	"dating-boss-date":     {Takeaway: "初见全程给对方接得住退得起的空间才是真本事", Source: "Logan Ury"},
	"dating-boss-decide":   {Takeaway: "清醒判断和真诚投入不冲突，这才是懂得爱", Source: "Logan Ury"},

	// ---- learn-business（3）----
	"biz-boss-account": {Takeaway: "开店前先算保本线、现金跑道和单笔账，三关不过别冲", Source: "Personal MBA"},
	"biz-boss-moat":    {Takeaway: "生意靠三问验证：选你的理由、抄不走的城河、真正的对手", Source: "巴菲特/芒格"},
	"biz-boss-system":  {Takeaway: "生意的本质是系统，算清账守住河才能放大杠杆", Source: "Naval杠杆论"},

	// ---- learn-negotiation（3）----
	"nego-boss-prep":  {Takeaway: "底牌、利益、筹码想清楚，谈判已赢九成", Source: "哈佛谈判项目"},
	"nego-boss-raise": {Takeaway: "谈判高手武装对手，而不是击败对手", Source: "沃顿谈判课·对方的世界"},
	"nego-boss-deal":  {Takeaway: "手握底牌敢拒绝，诚实到底才是真赢家", Source: "FBI谈判专家Voss"},

	// ---- learn-love（3）----
	"love-boss-eyes":  {Takeaway: "先看清自己的脚本，再用行为而非感觉识人", Source: "西北Marriage101"},
	"love-boss-fight": {Takeaway: "软启动开场、防洪水暂停、接住修复，才是体面吵架", Source: "Gottman夫妻实验室"},
	"love-boss-team":  {Takeaway: "关系质量决定终身幸福，把TA当队友而非对手", Source: "哈佛85年研究"},

	// ---- learn-learning（3）----
	"learning-boss-check":  {Takeaway: "学得爽不等于学得会，检索和间隔才是真正的学习", Source: "《认知天性》"},
	"learning-boss-engine": {Takeaway: "顺着大脑的构造做安排，好方法比死磕意志力管用", Source: "Oakley《学习之道》"},
	"learning-boss-system": {Takeaway: "读了听了收藏了都不算，能提取能迁移才算真学会", Source: "Ericsson刻意练习"},

	// ---- learn-lifedesign（3）----
	"checkup-now":  {Takeaway: "用仪表盘罗盘日志诚实画现状，不急着开药方", Source: "斯坦福人生设计课"},
	"odyssey-boss": {Takeaway: "先量产点子画三版奥德赛，再用访谈体验小成本验证", Source: "奥德赛计划(DYL)"},
	"launch-boss":  {Takeaway: "选择的质量不在结果，而在好奇行动反馈的循环里", Source: "人生设计·反思循环"},

	// ---- learn-meditation（3）----
	"mind-boss-start": {Takeaway: "冥想不是清空大脑，走神一次就是一次练习", Source: "卡巴金 MBSR"},
	"mind-boss-storm": {Takeaway: "情绪来了先停一拍贴标签，不是硬压下去", Source: "UCLA情绪标签实验"},
	"mind-boss-map":   {Takeaway: "练习不必增加时间，把在场感揉进已有的日子", Source: "MBSR非正式练习"},

	// ---- learn-writing（3）----
	"writing-boss-court": {Takeaway: "写作是为改变读者，结论要放在第一句", Source: "芝加哥大学McEnerney"},
	"writing-boss-clean": {Takeaway: "先删空话再补事实，干净只是及格线", Source: "Zinsser删废话"},
	"writing-boss-send":  {Takeaway: "坏消息第一句说透，方案占七成再定下一步", Source: "麦肯锡金字塔原理"},
}
