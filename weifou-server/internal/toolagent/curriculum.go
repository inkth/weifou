// Package toolagent — curriculum.go：概念型学习 Agent 的「核心概念」课程表与点亮引擎。
//
// 学心理 / 学经济 / 学哲学 各装载一份 curated 的 ~30 个核心概念（人工策展骨架，不让模型现编 = 这条线的护城河）。
// 分「入门 / 进阶」两档：成就感按档给（完成一档就庆祝），避免一条 X/100 的大分母进度条把人劝退。
// 用户在对话中「点亮」概念：每轮一问一答交给 DeepSeek 判定本轮真实涉及了哪些概念、是否已展现真正理解，
// 把概念档位往上推（0 未触及 / 1 已点亮 / 2 已掌握，只升不降）。整体照搬 skill.go 的范式。
// 砍下来的其余概念留作将来的「进阶层」，不进当前完成分母。
package toolagent

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"weifou-server/internal/deepseek"
	"weifou-server/internal/idgen"
	"weifou-server/internal/models"
)

// seedConcept 是课程表的一行（seed 用）。Slug 在单个 Agent 内稳定唯一。Tier：1 入门 / 2 进阶。
type seedConcept struct {
	Slug, Theme, Name, Blurb string
	Tier                     int
}

// 档位标签与展示顺序。
var tierLabels = map[int]string{1: "入门", 2: "进阶"}
var tierOrder = []int{1, 2}

// curricula：agent slug → 该领域核心概念。SeedConcepts 据此幂等写入 agent_concepts。
// spoken-english 的「概念」是真实场景关卡：点亮 = 用英语开口把这个场合的核心任务办成。
var curricula = map[string][]seedConcept{
	"learn-psychology": psychologyConcepts,
	"learn-economics":  economicsConcepts,
	"learn-philosophy": philosophyConcepts,
	"learn-ideas":      ideasConcepts,
	"learn-logic":      logicConcepts,
	"learn-science":    scienceConcepts,
	"learn-aesthetics": aestheticsConcepts,
	"learn-marketing":  marketingConcepts,
	"spoken-english":   englishScenarios,
}

// hookCheck 人工精编内容（开课钩子 + 检验题）。与 seedConcept 分开存：
// 只有旗舰课做精编，7 份未精编课程表的字面量不必全改；SeedConcepts 时按 slug 合并写入。
type hookCheck struct{ Hook, Check string }

// curatedContent：agent slug → concept slug → 精编内容。学心理 + 英语陪练（两门旗舰课）。
// 这是这条产品线的护城河：钩子制造好奇缺口、检验题考迁移应用——不是模型现编能稳定给出的品质。
var curatedContent = map[string]map[string]hookCheck{
	"learn-psychology": psychologyContent,
	"spoken-english":   englishContent,
}

// ============================ 英语陪练 · 场景关卡课程表 ============================
// 「概念」在这门课里 = 真实场合：点亮 = 用英语开口把这个场合的核心任务办成（哪怕磕巴）；
// 掌握 = 用上目标句式，且导师抛出变体/突发状况时仍能用英语接住。
// Tier1 生活/旅行（先能活下来），Tier2 职场/面试（再谈得成事）。

var englishScenarios = []seedConcept{
	// —— 生活（Tier 1）——
	{Slug: "cafe-order", Theme: "生活", Tier: 1, Name: "咖啡馆点单", Blurb: "定制一杯咖啡并买单"},
	{Slug: "restaurant-order", Theme: "生活", Tier: 1, Name: "餐厅点餐", Blurb: "问推荐、说忌口、结账"},
	{Slug: "first-smalltalk", Theme: "生活", Tier: 1, Name: "初次寒暄", Blurb: "自我介绍与破冰三句"},
	{Slug: "shopping-clothes", Theme: "生活", Tier: 1, Name: "买衣服", Blurb: "问尺码、试穿与退换"},
	{Slug: "ask-directions", Theme: "生活", Tier: 1, Name: "问路指路", Blurb: "问清路线并听懂指引"},
	{Slug: "phone-booking", Theme: "生活", Tier: 1, Name: "电话预约", Blurb: "电话订位与确认信息"},
	{Slug: "see-doctor", Theme: "生活", Tier: 1, Name: "看医生", Blurb: "描述症状、听懂医嘱"},
	// —— 旅行（Tier 1）——
	{Slug: "flight-checkin", Theme: "旅行", Tier: 1, Name: "机场值机", Blurb: "值机托运、选个好座位"},
	{Slug: "customs-qa", Theme: "旅行", Tier: 1, Name: "过关问答", Blurb: "从容答上海关三连问"},
	{Slug: "hotel-checkin", Theme: "旅行", Tier: 1, Name: "酒店入住", Blurb: "办理入住、提出换房要求"},
	{Slug: "taxi-ride", Theme: "旅行", Tier: 1, Name: "打车出行", Blurb: "说清目的地与路线偏好"},
	{Slug: "attraction-tickets", Theme: "旅行", Tier: 1, Name: "景点购票", Blurb: "问票价、时间与优惠"},
	{Slug: "lost-luggage", Theme: "旅行", Tier: 1, Name: "行李丢失", Blurb: "挂失并描述你的行李"},
	{Slug: "travel-help", Theme: "旅行", Tier: 1, Name: "旅途求助", Blurb: "丢了护照怎么开口求助"},
	// —— 职场（Tier 2）——
	{Slug: "work-self-intro", Theme: "职场", Tier: 2, Name: "同事初见", Blurb: "新团队里的自我介绍"},
	{Slug: "meeting-opinion", Theme: "职场", Tier: 2, Name: "会议表态", Blurb: "赞成、反对与补充意见"},
	{Slug: "task-handover", Theme: "职场", Tier: 2, Name: "任务交代", Blurb: "口头布置任务并确认理解"},
	{Slug: "project-present", Theme: "职场", Tier: 2, Name: "项目汇报", Blurb: "开场、过渡与收尾句式"},
	{Slug: "price-negotiation", Theme: "职场", Tier: 2, Name: "谈判议价", Blurb: "跟供应商体面地砍价"},
	{Slug: "pantry-smalltalk", Theme: "职场", Tier: 2, Name: "茶水间闲聊", Blurb: "周末、天气与项目近况"},
	{Slug: "video-call-clarify", Theme: "职场", Tier: 2, Name: "跨国视频会", Blurb: "听不清时优雅地追问"},
	// —— 面试（Tier 2）——
	{Slug: "interview-self-intro", Theme: "面试", Tier: 2, Name: "面试自我介绍", Blurb: "答好 Tell me about yourself"},
	{Slug: "strengths-weaknesses", Theme: "面试", Tier: 2, Name: "优缺点问答", Blurb: "把弱点讲成成长故事"},
	{Slug: "star-story", Theme: "面试", Tier: 2, Name: "讲一个经历", Blurb: "用 STAR 结构讲成就"},
	{Slug: "why-us", Theme: "面试", Tier: 2, Name: "为什么选我们", Blurb: "动机题答得真诚不谄媚"},
	{Slug: "salary-talk", Theme: "面试", Tier: 2, Name: "谈薪资", Blurb: "不吃亏也不失礼地谈钱"},
	{Slug: "ask-interviewer", Theme: "面试", Tier: 2, Name: "反问面试官", Blurb: "问出水平的三个问题"},
	{Slug: "mock-full-interview", Theme: "面试", Tier: 2, Name: "全英模拟面", Blurb: "把前面学的串成一整场"},
}

// englishContent：英语陪练精编 Hook/Check。先精编 6 个代表关；其余留空走兜底（模型现拟场景任务），逐步补齐。
var englishContent = map[string]hookCheck{
	"cafe-order": {
		Hook:  "你在纽约一家咖啡馆，店员冲你一笑：\"Hi! What can I get started for you?\"——你想要一杯少糖的燕麦拿铁。别用中文，开口试试？",
		Check: "换个情况：店员把你的单做错了（拿铁做成了美式）。用英语礼貌地指出问题并要求重做。",
	},
	"flight-checkin": {
		Hook:  "值机柜台前，地勤问你：\"Any bags to check?\"——你有一个托运箱、想要靠窗座位，用英语把这两件事都办成。",
		Check: "现在航班超售，地勤问你愿不愿意改签下一班换 200 美元代金券。听懂并用英语讨价还价一句。",
	},
	"first-smalltalk": {
		Hook:  "朋友聚会上，一个外国朋友伸出手：\"Hey, I don't think we've met.\"——用三句话完成自我介绍，并把话题抛回给对方。",
		Check: "对方说 \"I'm from Melbourne.\"——不冷场，用一个跟进问题让对话继续下去。",
	},
	"meeting-opinion": {
		Hook:  "例会上老板问：\"Any thoughts on the new plan?\"——你部分同意、但担心排期，用英语先肯定再提出顾虑。",
		Check: "同事的方案你完全不同意。用英语不伤和气地反对，并给出你的替代建议。",
	},
	"interview-self-intro": {
		Hook:  "面试官微笑着说：\"So, tell me about yourself.\"——你有 60 秒，用『现在-过去-未来』结构讲一版英文自我介绍。",
		Check: "同一个开场，面试官追问：\"Why are you leaving your current job?\"——不抱怨前司，用英语给出得体版本。",
	},
	"salary-talk": {
		Hook:  "HR 问：\"What are your salary expectations?\"——直接报数容易吃亏，用英语先反问薪资范围，再给一个带弹性的回答。",
		Check: "对方开的数低于你的预期。用英语礼貌地往上谈，至少用一个 \"Based on my experience…\" 式的理由。",
	},
}

// ============================ 学心理 · 精编钩子与检验题 ============================
// 钩子＝可直接开口的生活场景问题（制造好奇缺口，不剧透答案）；
// 检验题＝应用/迁移型（换一个场景让学员自己用，答对才配得上「点亮」）。

var psychologyContent = map[string]hookCheck{
	"loss-aversion": {
		Hook:  "丢 100 块的难受，和捡到 100 块的开心，哪个感觉更强烈？为什么几乎所有人都答前者？",
		Check: "商家说「不买就亏了」和「买了就赚了」，哪句更戳人？用损失厌恶解释给我听。",
	},
	"confirmation-bias": {
		Hook:  "有没有发现：越刷手机，越觉得自己原来的看法是对的？这不是巧合。",
		Check: "如果你想验证「TA 是不是不喜欢我」，该怎么找证据才能避开确认偏误的坑？",
	},
	"anchoring": {
		Hook:  "标价 999 划掉改 399，你会觉得捡了便宜——哪怕它本来就只值 300。为什么？",
		Check: "谈工资时先开口报数字是吃亏还是占便宜？用锚定效应分析一下。",
	},
	"fundamental-attribution-error": {
		Hook:  "同事迟到你觉得 TA 散漫，自己迟到你怪地铁——发现这个双标了吗？",
		Check: "看到有人开车加塞，除了「素质差」，试着给出两个处境层面的解释。",
	},
	"cognitive-dissonance": {
		Hook:  "为什么排了三小时队才吃上的餐厅，你更容易说它好吃——哪怕其实一般？",
		Check: "一个天天熬夜的人说「睡太多反而伤身」——他心里发生了什么？他在化解什么？",
	},
	"emotional-regulation": {
		Hook:  "情绪上头的时候，你是硬压下去、直接发出来，还是有第三条路？",
		Check: "朋友气到想发飙，除了说「冷静点」，你能给出两个真正有用的调节动作吗？",
	},
	"cognitive-reappraisal": {
		Hook:  "同一场大雨，有人觉得倒霉，有人觉得浪漫——差别不在雨，在哪？",
		Check: "「演讲前心跳加速」可以怎么重新解读，让紧张变成助力而不是拖累？",
	},
	"maslow-hierarchy": {
		Hook:  "为什么有人拿着高薪却觉得空虚，有人温饱刚够却干劲十足？",
		Check: "一个不缺钱的人辞职去山区支教——用需求层次分析，TA 在满足哪一层？",
	},
	"growth-mindset": {
		Hook:  "「我数学不行」和「我数学还没学明白」——这两句话会长出两种完全不同的人生。",
		Check: "孩子考好了，夸「你真聪明」和夸「你方法用得好」，哪个长期更伤人？为什么？",
	},
	"attachment-styles": {
		Hook:  "有人一恋爱就患得患失，有人一亲密就想逃——这套模式其实童年就写好了草稿。",
		Check: "「对方三小时没回消息」——安全型、焦虑型、回避型各会怎么反应？",
	},
	"self-esteem": {
		Hook:  "有人被批评一句就崩溃，有人却能笑着说「有道理」——底下垫着的东西不一样。",
		Check: "高自尊和自恋有什么区别？举一个能把两者分辨开的场景。",
	},
	"defense-mechanisms": {
		Hook:  "「我才没生气！」——说这话的人在干嘛？心理有一整套自我保护的花招。",
		Check: "吃不到葡萄说葡萄酸，是哪种防御机制？再从你身边举一个别的例子。",
	},
	"sunk-cost": {
		Hook:  "电影很难看，但票钱已经花了，你会看完吗？看完的那一刻，你其实亏了两次。",
		Check: "「都谈了五年了，分了多可惜」——用沉没成本帮这位朋友算笔账。",
	},
	"dunning-kruger": {
		Hook:  "为什么刚学会一点的人最敢说「这很简单」，真正的高手反而处处谨慎？",
		Check: "一个人自信爆棚，能说明他很懂吗？怎么用这个效应判断该信谁？",
	},
	"availability-heuristic": {
		Hook:  "坐飞机和坐汽车哪个更危险？大多数人的直觉都答错了——因为空难总上新闻。",
		Check: "刷到几条裁员视频就觉得「经济要完」，这个思维捷径出了什么问题？",
	},
	"rumination": {
		Hook:  "半夜回想白天说错的那句话，翻来覆去放了二十遍——这不是反思，是反刍。",
		Check: "反思和反刍的分界线在哪？给一个能把反刍拉回反思的具体做法。",
	},
	"flow": {
		Hook:  "有没有过一抬头三小时没了的体验？那种忘我状态，其实是可以被设计出来的。",
		Check: "玩游戏容易忘我、写报告却很难——用「挑战×技能」解释，并给报告一个改造建议。",
	},
	"hedonic-adaptation": {
		Hook:  "换新手机的快乐能撑几天？涨薪呢？为什么好事带来的快乐总是撑不久？",
		Check: "既然快乐会被适应磨平，怎么花钱才能买到更持久的快乐？给两个符合原理的策略。",
	},
	"intrinsic-motivation": {
		Hook:  "给爱画画的孩子发奖金，TA 反而不爱画了——奖励怎么会杀死热情？",
		Check: "想让自己坚持跑步，怎么设计才能保护内在动机，而不是靠打卡奖励硬撑？",
	},
	"delayed-gratification": {
		Hook:  "一颗现在就吃的糖，和两颗十五分钟后的糖——这个实验据说预测了孩子们的人生？",
		Check: "延迟满足只是天生自控力强吗？环境可以怎么帮一个人「忍得住」？",
	},
	"self-efficacy": {
		Hook:  "两个能力差不多的人，一个觉得「我能行」一个觉得「我不行」——结果真的会不一样。",
		Check: "自我效能和盲目自信有什么区别？怎么帮一个总觉得自己不行的人真实地建立它？",
	},
	"anxious-attachment": {
		Hook:  "消息三分钟没回就开始脑补分手大戏——这不是作，是有来处的。",
		Check: "焦虑型的「反复求确认」为什么常把对方越推越远？这个循环怎么破？",
	},
	"avoidant-attachment": {
		Hook:  "明明喜欢，却在关系升温时突然冷下来想逃——TA 到底在怕什么？",
		Check: "和回避型的人相处，「追问你到底怎么了」为什么适得其反？换个什么做法更好？",
	},
	"social-proof": {
		Hook:  "两家奶茶店，一家排长队一家空着，你会选哪家——哪怕根本不知道哪家好喝？",
		Check: "直播间挂「已售 10 万单」在利用什么心理？什么时候跟着别人走反而危险？",
	},
	"conformity": {
		Hook:  "一道明显答错的题，前面五个人都答错时，你还敢说出正确答案吗？实验里很多人不敢。",
		Check: "从众和虚心听取意见的边界在哪？各举一个「该顶住」和「该听劝」的场景。",
	},
	"self-serving-bias": {
		Hook:  "考好了是我聪明，考砸了是题太偏——这本账人人都在记。",
		Check: "团队复盘时怎么设计流程，才能不让每个人的自利偏误把责任推来推去？",
	},
	"impostor-syndrome": {
		Hook:  "「其实我没那么行，只是运气好还没被拆穿」——越优秀的人越容易这么想，为什么？",
		Check: "朋友升职了却说「我配不上」——除了安慰，用这个概念帮 TA 看清发生了什么。",
	},
	"self-compassion": {
		Hook:  "朋友搞砸了你会安慰，自己搞砸了你会骂自己——为什么双标的方向反了？",
		Check: "自我关怀会不会让人放纵摆烂？说说它和「对自己没要求」的区别。",
	},
	"boundaries": {
		Hook:  "「你怎么这么见外」——有没有发现，这句话总出现在你想拒绝的时候？",
		Check: "同事总把活推给你，怎么立边界既守住自己又不撕破脸？给一句可以直接说出口的话。",
	},
	"gaslighting": {
		Hook:  "「你太敏感了，我根本没那么说过」——这种话听多了，人真的会怀疑自己的记忆。",
		Check: "煤气灯操纵和普通的记忆分歧怎么区分？给两个识别信号。",
	},
}

// ============================ 学心理 · 30 概念 ============================

var psychologyConcepts = []seedConcept{
	// —— 入门 12 ——
	{"loss-aversion", "认知偏误", "损失厌恶", "亏的痛远大于赚的爽", 1},
	{"confirmation-bias", "认知偏误", "确认偏误", "只看支持自己的证据", 1},
	{"anchoring", "认知偏误", "锚定效应", "先入的印象拖住后续判断", 1},
	{"fundamental-attribution-error", "认知偏误", "基本归因错误", "怪人品而非怪处境", 1},
	{"cognitive-dissonance", "社会心理", "认知失调", "言行矛盾带来的别扭", 1},
	{"emotional-regulation", "情绪与调节", "情绪调节", "管理情绪的能力与策略", 1},
	{"cognitive-reappraisal", "情绪与调节", "认知重评", "换个解读就换了感受", 1},
	{"maslow-hierarchy", "动机与需求", "马斯洛需求层次", "从温饱到自我实现", 1},
	{"growth-mindset", "动机与需求", "成长型思维", "能力可练而非天定", 1},
	{"attachment-styles", "发展与依恋", "依恋类型", "亲密关系的底层模式", 1},
	{"self-esteem", "自我与认同", "自尊", "对自我价值的整体评价", 1},
	{"defense-mechanisms", "心理健康", "防御机制", "保护自己的心理花招", 1},
	// —— 进阶 18 ——
	{"sunk-cost", "认知偏误", "沉没成本谬误", "为已花掉的继续将错就错", 2},
	{"dunning-kruger", "认知偏误", "邓宁-克鲁格效应", "越不懂越觉得自己懂", 2},
	{"availability-heuristic", "认知偏误", "可得性启发", "好想起的就觉得更常见", 2},
	{"rumination", "情绪与调节", "反刍思维", "反复咀嚼负面念头", 2},
	{"flow", "情绪与调节", "心流", "全神贯注忘我的状态", 2},
	{"hedonic-adaptation", "情绪与调节", "享乐适应", "好坏都会慢慢习惯", 2},
	{"intrinsic-motivation", "动机与需求", "内在动机", "为兴趣本身而做", 2},
	{"delayed-gratification", "动机与需求", "延迟满足", "为更大回报忍住眼前", 2},
	{"self-efficacy", "动机与需求", "自我效能", "我能办成的信念", 2},
	{"anxious-attachment", "发展与依恋", "焦虑型依恋", "怕被抛弃、求确认", 2},
	{"avoidant-attachment", "发展与依恋", "回避型依恋", "近了就想逃", 2},
	{"social-proof", "社会心理", "社会认同", "别人都这样我也跟", 2},
	{"conformity", "社会心理", "从众", "随大流改变自己", 2},
	{"self-serving-bias", "自我与认同", "自利偏误", "成功归己失败赖外", 2},
	{"impostor-syndrome", "自我与认同", "冒名顶替综合征", "总觉得自己是侥幸蒙的", 2},
	{"self-compassion", "自我与认同", "自我关怀", "像对朋友一样待自己", 2},
	{"boundaries", "心理健康", "心理边界", "分清你我的责任与空间", 2},
	{"gaslighting", "心理健康", "煤气灯效应", "让你怀疑自己的记忆与判断", 2},
}

// ============================ 学经济 · 30 概念 ============================

var economicsConcepts = []seedConcept{
	// —— 入门 12 ——
	{"opportunity-cost", "微观基础", "机会成本", "选这个就放弃了那个", 1},
	{"marginal-thinking", "微观基础", "边际思维", "只看多一单位的得失", 1},
	{"supply-demand", "微观基础", "供求关系", "价由供需两端决定", 1},
	{"incentives", "微观基础", "激励", "人对好处与代价做反应", 1},
	{"scarcity", "微观基础", "稀缺性", "资源有限欲望无限", 1},
	{"sunk-cost-econ", "微观基础", "沉没成本", "收不回的就别再算", 1},
	{"comparative-advantage", "微观基础", "比较优势", "各干相对最擅长的", 1},
	{"inflation", "宏观经济", "通货膨胀", "钱变毛、物价普涨", 1},
	{"interest-rate", "货币与金融", "利率", "钱的价格", 1},
	{"compound-interest", "个人理财", "复利", "利滚利的雪球", 1},
	{"loss-aversion-econ", "行为经济学", "损失厌恶", "亏的痛大于赚的爽", 1},
	{"externalities", "供需与市场", "外部性", "代价或好处溢给了别人", 1},
	// —— 进阶 18 ——
	{"elasticity", "微观基础", "弹性", "价格变需求变多少", 2},
	{"diminishing-utility", "微观基础", "边际效用递减", "越多越不稀罕", 2},
	{"equilibrium-price", "供需与市场", "均衡价格", "供给等于需求那点", 2},
	{"monopoly", "供需与市场", "垄断", "一家独大定价", 2},
	{"price-discrimination", "供需与市场", "价格歧视", "对不同人收不同价", 2},
	{"mental-accounting", "行为经济学", "心理账户", "钱被分进不同心理口袋", 2},
	{"nudge", "行为经济学", "助推", "不强制地轻推你选择", 2},
	{"gdp", "宏观经济", "国内生产总值", "一国一年的总产出", 2},
	{"unemployment", "宏观经济", "失业率", "想工作却没工作的比例", 2},
	{"monetary-policy", "货币与金融", "货币政策", "央行用利率调经济", 2},
	{"exchange-rate", "货币与金融", "汇率", "两国货币的兑换比", 2},
	{"prisoners-dilemma", "博弈与激励", "囚徒困境", "各自理性却集体糟糕", 2},
	{"nash-equilibrium", "博弈与激励", "纳什均衡", "谁都不想单方改变", 2},
	{"moral-hazard", "博弈与激励", "道德风险", "有兜底就乱来", 2},
	{"network-effect", "博弈与激励", "网络效应", "用的人越多越值钱", 2},
	{"diversification", "个人理财", "分散投资", "别把蛋放一个篮子", 2},
	{"risk-return", "个人理财", "风险与收益", "想多赚就得担多险", 2},
	{"creative-destruction", "制度与增长", "创造性破坏", "新的淘汰旧的", 2},
}

// ============================ 学哲学 · 30 概念 ============================

var philosophyConcepts = []seedConcept{
	// —— 入门 12 ——
	{"free-will", "形而上学", "自由意志", "我们真能自主选择吗", 1},
	{"determinism", "形而上学", "决定论", "一切早被因果定好", 1},
	{"utilitarianism", "伦理学", "功利主义", "最大多数的最大幸福", 1},
	{"deontology", "伦理学", "义务论", "有些事本身就该做或不该", 1},
	{"virtue-ethics", "伦理学", "德性伦理学", "重在成为怎样的人", 1},
	{"trolley-problem", "思想实验", "电车难题", "牺牲一人救五人吗", 1},
	{"skepticism", "知识论", "怀疑论", "我们到底能确知什么", 1},
	{"empiricism", "知识论", "经验主义", "知识源于感官经验", 1},
	{"mind-body-problem", "心灵哲学", "身心问题", "意识与大脑怎么连", 1},
	{"existentialism", "人生意义", "存在主义", "人先存在再自定本质", 1},
	{"meaning-of-life", "人生意义", "人生意义", "意义是寻得还是造出", 1},
	{"social-contract", "政治哲学", "社会契约", "权力源于人们的约定", 1},
	// —— 进阶 18 ——
	{"categorical-imperative", "伦理学", "绝对命令", "能否成为普遍法则", 2},
	{"moral-relativism", "伦理学", "道德相对主义", "对错随文化而变", 2},
	{"is-ought", "伦理学", "是与应当", "事实推不出价值", 2},
	{"compatibilism", "形而上学", "相容论", "自由与决定可并存", 2},
	{"personal-identity", "形而上学", "人格同一性", "是什么让你还是你", 2},
	{"consciousness", "心灵哲学", "意识", "主观体验之谜", 2},
	{"qualia", "心灵哲学", "感受质", "红之为红的那种感觉", 2},
	{"chinese-room", "心灵哲学", "中文房间", "会处理符号≠真懂", 2},
	{"gettier-problem", "知识论", "葛梯尔问题", "确证真信念也未必是知识", 2},
	{"problem-of-induction", "知识论", "归纳问题", "凭过去不能保证未来", 2},
	{"occams-razor", "逻辑与论证", "奥卡姆剃刀", "如无必要勿增假设", 2},
	{"logical-fallacy", "逻辑与论证", "逻辑谬误", "看似有理的推理漏洞", 2},
	{"veil-of-ignorance", "政治哲学", "无知之幕", "不知自己处境来定规则", 2},
	{"absurdism", "人生意义", "荒诞主义", "人求意义而世界沉默", 2},
	{"nihilism", "人生意义", "虚无主义", "一切皆无意义", 2},
	{"ship-of-theseus", "思想实验", "忒修斯之船", "全换零件还是原物吗", 2},
	{"experience-machine", "思想实验", "体验机器", "愿接入完美幻觉吗", 2},
	{"stoicism", "东方与德性", "斯多葛主义", "修内心、应外境", 2},
}

// ============================ 学思想 · 30 概念（观念史，保持中立、不站队）============================

var ideasConcepts = []seedConcept{
	// —— 入门 12 ——
	{"enlightenment", "近代转折", "启蒙运动", "用理性照亮世界", 1},
	{"scientific-revolution", "近代转折", "科学革命", "用实验重估一切", 1},
	{"evolution-idea", "改变世界的观念", "进化论", "物种由自然选择而来", 1},
	{"invisible-hand", "改变世界的观念", "看不见的手", "自利汇成市场秩序", 1},
	{"humanism", "改变世界的观念", "人文主义", "以人而非神为中心", 1},
	{"liberalism-idea", "主要思潮", "自由主义", "以个人自由为核心", 1},
	{"marxism", "主要思潮", "马克思主义", "从阶级与资本看历史", 1},
	{"romanticism", "主要思潮", "浪漫主义", "情感与自然的反拨", 1},
	{"feminism", "主要思潮", "女性主义", "性别平等的思潮", 1},
	{"psychoanalysis", "改变世界的观念", "精神分析", "无意识在暗中支配我们", 1},
	{"democracy-idea", "主要思潮", "民主观念", "主权在民", 1},
	{"capitalism", "主要思潮", "资本主义", "市场与私有的体系", 1},
	// —— 进阶 18 ——
	{"dialectics", "思考方式", "辩证法", "正反合的运动", 2},
	{"nationalism", "主要思潮", "民族主义", "民族作为认同来源", 2},
	{"structuralism", "20世纪思潮", "结构主义", "深层结构决定意义", 2},
	{"postmodernism", "20世纪思潮", "后现代主义", "怀疑一切宏大叙事", 2},
	{"pragmatism", "思考方式", "实用主义", "有用即真理", 2},
	{"social-darwinism", "警示性观念", "社会达尔文主义", "把优胜劣汰搬进社会", 2},
	{"utopia", "改变世界的观念", "乌托邦", "对理想社会的构想", 2},
	{"secularization", "近代转折", "世俗化", "宗教退出公共中心", 2},
	{"modernity", "20世纪思潮", "现代性", "现代社会的独特气质", 2},
	{"the-other", "20世纪思潮", "他者", "通过对立定义自我", 2},
	{"cultural-relativism", "思考方式", "文化相对主义", "对错随文化而变", 2},
	{"individualism", "主要思潮", "个人主义", "个人先于集体", 2},
	{"collectivism", "主要思潮", "集体主义", "集体先于个人", 2},
	{"progress-idea", "改变世界的观念", "进步观念", "历史向好发展", 2},
	{"alienation", "20世纪思潮", "异化", "人与劳动和自我相疏离", 2},
	{"meritocracy", "当代议题", "绩效/精英主义", "凭本事论高下", 2},
	{"environmentalism", "当代议题", "环保主义", "重估人与自然的关系", 2},
	{"globalization-idea", "当代议题", "全球化", "世界连成一体的进程", 2},
}

// ============================ 学逻辑 · 30 概念 ============================

var logicConcepts = []seedConcept{
	// —— 入门 12 ——
	{"argument-structure", "论证基础", "论证结构", "前提如何撑起结论", 1},
	{"premise-conclusion", "论证基础", "前提与结论", "分清主张与理由", 1},
	{"deductive-inductive", "论证基础", "演绎与归纳", "必然 vs 或然", 1},
	{"validity", "论证基础", "有效性", "结构对不对", 1},
	{"soundness", "论证基础", "可靠性", "结构对且前提真", 1},
	{"correlation-causation", "因果思辨", "相关不等于因果", "同时出现≠谁导致谁", 1},
	{"ad-hominem", "常见谬误", "人身攻击", "攻击人而非观点", 1},
	{"straw-man", "常见谬误", "稻草人", "歪曲对方再打", 1},
	{"false-dilemma", "常见谬误", "非黑即白", "硬塞成只有两选", 1},
	{"slippery-slope", "常见谬误", "滑坡谬误", "一步就滑到极端", 1},
	{"burden-of-proof", "论证基础", "举证责任", "谁主张谁举证", 1},
	{"occams-razor-logic", "思维工具", "奥卡姆剃刀", "如无必要勿增假设", 1},
	// —— 进阶 18 ——
	{"circular-reasoning", "常见谬误", "循环论证", "用结论证明结论", 2},
	{"appeal-to-authority", "常见谬误", "诉诸权威", "他说的所以对", 2},
	{"appeal-to-emotion", "常见谬误", "诉诸情感", "煽情代替讲理", 2},
	{"hasty-generalization", "常见谬误", "以偏概全", "少数例子推全体", 2},
	{"survivorship-bias", "统计陷阱", "幸存者偏差", "只看到活下来的", 2},
	{"base-rate-fallacy", "统计陷阱", "基率谬误", "忽略了背景概率", 2},
	{"equivocation", "常见谬误", "偷换概念", "同一词悄悄换义", 2},
	{"begging-the-question", "常见谬误", "预设结论", "把待证的当已知", 2},
	{"necessary-sufficient-logic", "思维工具", "必要与充分", "缺它不行 vs 有它就行", 2},
	{"counterexample", "思维工具", "反例法", "一个反例即可推翻", 2},
	{"reductio", "思维工具", "归谬法", "顺着推到荒谬", 2},
	{"bayesian-thinking", "概率思维", "贝叶斯思维", "用新证据更新信念", 2},
	{"falsifiability-logic", "思维工具", "可证伪", "不能被证伪就没内容", 2},
	{"steelman", "论证素养", "钢铁人", "先把对方论证做到最强", 2},
	{"analogy-argument", "论证素养", "类比论证", "相似处能推多远", 2},
	{"gambler-fallacy", "概率思维", "赌徒谬误", "以为该轮到我了", 2},
	{"moving-goalposts", "常见谬误", "移动球门", "被驳倒就改标准", 2},
	{"cherry-picking", "常见谬误", "选择性举证", "只挑对自己有利的", 2},
}

// ============================ 学科学 · 30 概念（理解版，不做题）============================

var scienceConcepts = []seedConcept{
	// —— 入门 12 ——
	{"scientific-method", "科学怎么工作", "科学方法", "大胆假设小心求证", 1},
	{"gravity", "物理与宇宙", "引力", "万物相互吸引", 1},
	{"energy-conservation", "物理与宇宙", "能量守恒", "能量只转移不消失", 1},
	{"entropy", "物理与宇宙", "熵", "无序总在增加", 1},
	{"evolution", "生命", "演化", "自然选择塑造生命", 1},
	{"dna", "生命", "DNA", "生命的信息编码", 1},
	{"atoms", "物质", "原子", "万物的基本砖块", 1},
	{"cells", "生命", "细胞", "生命的基本单位", 1},
	{"relativity", "物理与宇宙", "相对论", "时空会弯曲", 1},
	{"big-bang", "物理与宇宙", "大爆炸", "宇宙有个起点", 1},
	{"photosynthesis", "生命", "光合作用", "把光变成能量", 1},
	{"plate-tectonics", "地球", "板块构造", "大陆在缓缓漂移", 1},
	// —— 进阶 18 ——
	{"quantum", "物理与宇宙", "量子力学", "微观世界靠概率", 2},
	{"thermodynamics", "物理与宇宙", "热力学", "热与能的规律", 2},
	{"electromagnetism", "物理与宇宙", "电磁", "光也是电磁波", 2},
	{"periodic-table", "物质", "元素周期", "万物由元素排布而成", 2},
	{"genes-heredity", "生命", "遗传", "性状如何传给后代", 2},
	{"neurons", "生命", "神经元", "大脑靠它传信号", 2},
	{"immune-system", "生命", "免疫系统", "身体的防御部队", 2},
	{"microbiome", "生命", "微生物组", "体内的微生物生态", 2},
	{"ecosystem", "地球", "生态系统", "生物与环境的网络", 2},
	{"climate-system", "地球", "气候系统", "碳与温度的平衡", 2},
	{"speed-of-light", "物理与宇宙", "光速极限", "没有更快的了", 2},
	{"black-holes", "物理与宇宙", "黑洞", "连光都逃不出", 2},
	{"dark-matter", "物理与宇宙", "暗物质", "看不见却撑起星系", 2},
	{"chemistry-bonds", "物质", "化学键", "原子如何结合成物", 2},
	{"probability-science", "科学怎么工作", "概率与统计", "用数据说话", 2},
	{"emergence-science", "复杂性", "涌现", "简单规则生出复杂", 2},
	{"chaos", "复杂性", "混沌", "蝴蝶效应", 2},
	{"scale-of-universe", "物理与宇宙", "宇宙尺度", "从原子到星系的量级", 2},
}

// ============================ 学审美 · 30 概念 ============================

var aestheticsConcepts = []seedConcept{
	// —— 入门 12 ——
	{"composition", "视觉语言", "构图", "画面如何安排", 1},
	{"color-theory", "视觉语言", "色彩", "冷暖与搭配", 1},
	{"light-shadow", "视觉语言", "光影", "明暗塑造体积", 1},
	{"contrast", "视觉语言", "对比", "差异制造重点", 1},
	{"balance", "视觉语言", "平衡", "视觉的稳与险", 1},
	{"negative-space", "视觉语言", "留白", "空也是内容", 1},
	{"focal-point", "视觉语言", "视觉焦点", "眼睛先落在哪", 1},
	{"perspective", "视觉语言", "透视", "平面造出深度", 1},
	{"rhythm-visual", "视觉语言", "节奏", "重复与变化", 1},
	{"proportion", "视觉语言", "比例", "黄金分割式的舒服", 1},
	{"symmetry", "视觉语言", "对称与不对称", "秩序 vs 张力", 1},
	{"mood-tone", "鉴赏入门", "基调", "作品的情绪底色", 1},
	// —— 进阶 18 ——
	{"renaissance-art", "艺术流派", "文艺复兴", "对人与真实的重拾", 2},
	{"baroque", "艺术流派", "巴洛克", "繁复与戏剧性", 2},
	{"impressionism", "艺术流派", "印象派", "抓住光与瞬间", 2},
	{"modern-art", "艺术流派", "现代艺术", "为何看不懂也是艺术", 2},
	{"abstraction", "艺术流派", "抽象", "脱离具象的表达", 2},
	{"minimalism", "艺术流派", "极简主义", "少即是多", 2},
	{"mise-en-scene", "电影语言", "场面调度", "一个画面里塞了什么", 2},
	{"montage", "电影语言", "蒙太奇", "剪辑创造新意义", 2},
	{"cinematography", "电影语言", "摄影影调", "镜头怎么讲情绪", 2},
	{"framing-shot", "电影语言", "景别", "远近如何影响感受", 2},
	{"leitmotif", "鉴赏进阶", "主题动机", "贯穿全片的母题", 2},
	{"typography", "设计", "字体排印", "字也是表情", 2},
	{"gestalt", "鉴赏进阶", "格式塔", "整体大于部分之和", 2},
	{"wabi-sabi", "东方美学", "侘寂", "残缺与素朴之美", 2},
	{"sublime", "美的类型", "崇高", "敬畏中的美", 2},
	{"kitsch", "美的类型", "媚俗", "廉价而讨好的感动", 2},
	{"taste-canon", "鉴赏进阶", "经典/正典", "什么被奉为好", 2},
	{"form-content", "鉴赏进阶", "形式与内容", "怎么说 vs 说什么", 2},
}

// ============================ 学营销 · 30 概念 ============================

var marketingConcepts = []seedConcept{
	// —— 入门 12 ——
	{"positioning", "定位与品牌", "定位", "在用户心智占一个词", 1},
	{"target-audience", "用户与需求", "目标人群", "你到底为谁做", 1},
	{"value-proposition", "定位与品牌", "价值主张", "一句话说清凭啥选你", 1},
	{"brand", "定位与品牌", "品牌", "名字之外的信任与联想", 1},
	{"marketing-mix-4p", "营销基础", "4P 组合", "产品·价格·渠道·推广", 1},
	{"customer-needs", "用户与需求", "用户需求", "卖解决方案不卖产品", 1},
	{"funnel", "增长与转化", "营销漏斗", "从看见到成交层层流失", 1},
	{"differentiation", "定位与品牌", "差异化", "凭什么和别人不一样", 1},
	{"word-of-mouth", "传播", "口碑", "最便宜也最贵的渠道", 1},
	{"call-to-action", "增长与转化", "行动号召", "明确告诉用户下一步", 1},
	{"segmentation", "用户与需求", "市场细分", "别想通吃所有人", 1},
	{"pricing-strategy", "增长与转化", "定价策略", "价格本身在传递信号", 1},
	// —— 进阶 18 ——
	{"usp", "定位与品牌", "独特卖点", "一个记得住的买它理由", 2},
	{"customer-journey", "用户与需求", "用户旅程", "从陌生到复购的全程", 2},
	{"aida", "传播", "AIDA 模型", "注意·兴趣·欲望·行动", 2},
	{"conversion-rate", "增长与转化", "转化率", "有多少看的变成买的", 2},
	{"cac-ltv", "增长与转化", "获客成本与终身价值", "拉一个人花多少·赚多少", 2},
	{"retention", "增长与转化", "留存", "比拉新更值钱", 2},
	{"viral-loop", "增长与转化", "裂变/病毒循环", "让用户带来用户", 2},
	{"content-marketing", "传播", "内容营销", "用有用有趣的内容获客", 2},
	{"storytelling", "传播", "品牌故事", "让人记住并愿意传", 2},
	{"social-proof-mkt", "说服心理", "社会认同", "别人都在用所以我也", 2},
	{"scarcity-urgency", "说服心理", "稀缺与紧迫", "限时限量催下单", 2},
	{"anchoring-price", "说服心理", "价格锚点", "先给个贵的做参照", 2},
	{"kol-marketing", "渠道", "达人/KOL 营销", "借他人信任带货", 2},
	{"private-domain", "渠道", "私域流量", "把用户攒进自己池子", 2},
	{"ab-testing", "数据驱动", "A/B 测试", "用数据决定而非拍脑袋", 2},
	{"growth-hacking", "增长与转化", "增长黑客", "用小实验撬动增长", 2},
	{"product-market-fit", "营销基础", "产品市场契合", "做出人们真想要的", 2},
	{"brand-moat", "定位与品牌", "品牌护城河", "让对手抄不走的偏爱", 2},
}

// ============================ Seed ============================

// SeedConcepts 把三份课程表幂等写入 agent_concepts（按 agent_id+slug），并清掉已不在清单里的旧概念
// （让分母收敛到当前 30，去掉早期 100 版残留）。须在 Seed(写入 ToolAgent) 之后调用。
func SeedConcepts(db *gorm.DB) {
	if db == nil {
		return
	}
	for agentSlug, list := range curricula {
		var agent models.ToolAgent
		if db.Where("slug = ?", agentSlug).First(&agent).Error != nil {
			continue // Agent 尚未 seed（理论上 Seed 先跑）
		}
		content := curatedContent[agentSlug] // 可能为 nil：未精编课程返回零值 hookCheck
		slugs := make([]string, 0, len(list))
		for i := range list {
			sc := list[i]
			hc := content[sc.Slug]
			slugs = append(slugs, sc.Slug)
			var existing models.AgentConcept
			if db.Where("agent_id = ? AND slug = ?", agent.ID, sc.Slug).First(&existing).Error == gorm.ErrRecordNotFound {
				db.Create(&models.AgentConcept{
					ID: idgen.New(), AgentID: agent.ID, Slug: sc.Slug,
					Theme: sc.Theme, Tier: sc.Tier, Name: sc.Name, Blurb: sc.Blurb,
					Hook: hc.Hook, Check: hc.Check, Sort: i,
				})
				continue
			}
			db.Model(&existing).Updates(map[string]interface{}{
				"theme": sc.Theme, "tier": sc.Tier, "name": sc.Name, "blurb": sc.Blurb,
				"hook": hc.Hook, "check": hc.Check, "sort": i,
			})
		}
		// 清除不在当前清单的旧概念（user_concepts 里的孤儿记录无害，进度视图只遍历现有概念）。
		db.Where("agent_id = ? AND slug NOT IN ?", agent.ID, slugs).Delete(&models.AgentConcept{})
	}
}

// buildConceptPrompt 由人格/教学法 head + 概念清单，拼出该 Agent 的 system prompt（把核心概念地图嵌进去，按主题）。
// 概念名只在 curriculum 里维护一份，prompt 自动跟着更新。
func buildConceptPrompt(head string, list []seedConcept) string {
	var b strings.Builder
	b.WriteString(head)
	b.WriteString("\n\n== 你掌管的核心概念地图（共 ")
	b.WriteString(strconv.Itoa(len(list)))
	b.WriteString(" 个，按主题）==")
	curTheme := ""
	for i := range list {
		c := list[i]
		if c.Theme != curTheme {
			curTheme = c.Theme
			b.WriteString("\n【" + curTheme + "】")
		} else {
			b.WriteString("、")
		}
		b.WriteString(c.Name)
	}
	b.WriteString("\n== 概念地图结束 ==\n你只在这张地图的范围内教学；可主动提示「我们已经聊到 X，要不要顺着去 Y」，陪用户一步步把整张地图点亮。")
	return b.String()
}

// ============================ 复习引擎（检索练习） ============================
// 点亮≠记住：隔期抽查（retrieval practice）才是把概念钉进长期记忆的机制。
// 到期窗口：已点亮(1) 3 天、已掌握(2) 7 天没再碰过 → 到期。
// touches 每次命中都刷新 updated_at，所以「对话里又聊到」天然等于「复习过」。

const (
	reviewDueLitDays      = 3
	reviewDueMasteredDays = 7
)

// reviewPick 挑该用户在该 Agent 下待复习的概念，最久未碰的在前。
// onlyDue=true 只要过了到期窗口的；false 则不设窗口（用户主动要复习时，抽最生疏的也行）。
// limit<=0 表示不限量（用于数徽章）。
func reviewPick(db *gorm.DB, userID, agentID string, limit int, onlyDue bool) []models.AgentConcept {
	var ucs []models.UserConcept
	db.Where("user_id = ? AND agent_id = ? AND level >= 1", userID, agentID).
		Order("updated_at asc").Find(&ucs)
	if len(ucs) == 0 {
		return nil
	}
	now := time.Now()
	var ids []string
	for i := range ucs {
		if onlyDue {
			days := reviewDueLitDays
			if ucs[i].Level >= 2 {
				days = reviewDueMasteredDays
			}
			if now.Sub(ucs[i].UpdatedAt) < time.Duration(days)*24*time.Hour {
				continue
			}
		}
		ids = append(ids, ucs[i].ConceptID)
		if limit > 0 && len(ids) >= limit {
			break
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var cs []models.AgentConcept
	db.Where("id IN ?", ids).Find(&cs)
	byID := make(map[string]models.AgentConcept, len(cs))
	for i := range cs {
		byID[cs[i].ID] = cs[i]
	}
	out := make([]models.AgentConcept, 0, len(ids))
	for _, id := range ids {
		if c, ok := byID[id]; ok {
			out = append(out, c)
		}
	}
	return out
}

// dueCount 到期待复习的概念数（首页催课条 / 对话页复习徽章共用）。
func dueCount(db *gorm.DB, userID, agentID string) int {
	return len(reviewPick(db, userID, agentID, 0, true))
}

// reviewDirective 复习挑战的编排指令（chat mode=review 时追加为 system 段）。
// 优先抽到期的；没有到期就抽最生疏的已点亮概念——学员主动要复习不该被拒之门外。
// 一个可复习的概念都没有（还没点亮过任何东西）时返回 ""。
func (h *Handler) reviewDirective(userID, agentID string) string {
	picked := reviewPick(h.db, userID, agentID, 3, true)
	if len(picked) == 0 {
		picked = reviewPick(h.db, userID, agentID, 3, false)
	}
	if len(picked) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("== 复习挑战（快问快答，仅你可见；本指令优先于上面的开场编排）==\n")
	b.WriteString("学员点了「复习挑战」。本轮起只做快问快答复习，不开新概念：一次只出一题，等学员回答后先判定——答对就一句确认加一个小延伸，答错或含糊就一句话纠正讲透——再出下一题。全部问完后用一两句总结战果（哪些保住了、哪个建议重学），语气轻快，别拖长。\n待复习概念（按生疏程度排）：\n")
	for i := range picked {
		c := picked[i]
		b.WriteString(strconv.Itoa(i+1) + ". " + c.Name + "（" + c.Blurb + "）")
		if c.Check != "" {
			b.WriteString(" 检验题：" + c.Check)
		} else {
			b.WriteString(" 检验题：你自拟一道应用/迁移型的题（换个生活场景让学员自己用，不考背定义）。")
		}
		b.WriteString("\n")
	}
	b.WriteString("== 复习挑战结束 ==")
	return b.String()
}

// conceptDirective 「指定关卡」编排指令（学员从闯关地图点选某关进来时追加为 system 段）。
// slug 不在课程表里返回 ""（容错：地图数据过期/参数被篡改都不该打断对话）。
func (h *Handler) conceptDirective(agentID, slug string) string {
	var c models.AgentConcept
	if h.db.First(&c, "agent_id = ? AND slug = ?", agentID, slug).Error != nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("== 指定关卡（仅你可见；本指令优先于上面的开场编排）==\n")
	b.WriteString("学员从闯关地图点选了『" + c.Name + "』（" + c.Blurb + "）。本轮直接以它开课：\n")
	if c.Hook != "" {
		b.WriteString("开场用这个精编钩子（可贴学员语境微调措辞）：" + c.Hook + "\n")
	} else {
		b.WriteString("开场你自拟一个具体到画面的场景任务（把学员直接丢进情境，不要问「想学什么」）。\n")
	}
	if c.Check != "" {
		b.WriteString("讲透后用精编检验题检验：" + c.Check + "\n")
	} else {
		b.WriteString("讲透后你自拟一道应用/迁移型检验（换个场景让学员自己用），学员过了才算真点亮。\n")
	}
	b.WriteString("通关后问一句：回地图挑下一关，还是顺路继续？\n== 指定关卡结束 ==")
	return b.String()
}

// ============================ 点亮引擎 ============================

// conceptList 取该 Agent 的课程表（按 sort）。
func (h *Handler) conceptList(agentID string) []models.AgentConcept {
	var cs []models.AgentConcept
	h.db.Where("agent_id = ?", agentID).Order("sort asc").Find(&cs)
	return cs
}

// userConceptLevels 取用户在该 Agent 各概念上的档位 map[conceptID]level。
func (h *Handler) userConceptLevels(userID, agentID string) map[string]int {
	var ucs []models.UserConcept
	h.db.Where("user_id = ? AND agent_id = ?", userID, agentID).Find(&ucs)
	m := make(map[string]int, len(ucs))
	for i := range ucs {
		m[ucs[i].ConceptID] = ucs[i].Level
	}
	return m
}

// conceptProgressView 序列化进度给前端：总 total/lit/mastered + 按档（入门/进阶）分组的概念与各档进度。
func conceptProgressView(concepts []models.AgentConcept, levels map[string]int) gin.H {
	lit, mastered := 0, 0
	type agg struct {
		total, lit, mastered int
		items                []gin.H
	}
	byTier := make(map[int]*agg, len(tierOrder))
	for i := range concepts {
		c := concepts[i]
		lv := levels[c.ID]
		if lv >= 1 {
			lit++
		}
		if lv >= 2 {
			mastered++
		}
		a := byTier[c.Tier]
		if a == nil {
			a = &agg{}
			byTier[c.Tier] = a
		}
		a.total++
		if lv >= 1 {
			a.lit++
		}
		if lv >= 2 {
			a.mastered++
		}
		a.items = append(a.items, gin.H{"slug": c.Slug, "name": c.Name, "blurb": c.Blurb, "level": lv, "theme": c.Theme, "hook": c.Hook})
	}
	tiers := make([]gin.H, 0, len(tierOrder))
	for _, t := range tierOrder {
		a := byTier[t]
		if a == nil {
			continue
		}
		tiers = append(tiers, gin.H{
			"tier": tierLabels[t], "total": a.total, "lit": a.lit, "mastered": a.mastered,
			"concepts": a.items,
		})
	}
	return gin.H{"total": len(concepts), "lit": lit, "mastered": mastered, "tiers": tiers}
}

// loadConceptProgress 给 GET /agents/concepts/:id 用：当前进度视图。
func (h *Handler) loadConceptProgress(userID, agentID string) gin.H {
	return conceptProgressView(h.conceptList(agentID), h.userConceptLevels(userID, agentID))
}

// tierClearedSet 返回「已点满（lit==total）」的档位集合，用于检测本轮新打通了哪档。
func tierClearedSet(concepts []models.AgentConcept, levels map[string]int) map[int]bool {
	tot := map[int]int{}
	lit := map[int]int{}
	for i := range concepts {
		c := concepts[i]
		tot[c.Tier]++
		if levels[c.ID] >= 1 {
			lit[c.Tier]++
		}
	}
	res := map[int]bool{}
	for t, n := range tot {
		if n > 0 && lit[t] == n {
			res[t] = true
		}
	}
	return res
}

type conceptAssessResult struct {
	Touched  []string `json:"touched"`
	Mastered []string `json:"mastered"`
	Note     string   `json:"note"`
}

const conceptAssessPrompt = `你是「概念掌握判定器」。下面给你一份某学习领域的核心概念清单（每行格式 slug|概念名），以及用户与导师的一轮对话。判断这一轮里用户对清单中哪些概念有了**实质参与**，以及是否**展现出真正的理解**。
规则：
- touched（=点亮）：用户**实质参与过**的概念——用户自己尝试解释过、举过例、回答了导师围绕它的提问、或把它对应到了自己的事上。导师单方面讲到而用户没有任何回应的，**不算** touched。只放清单里确实存在的 slug，没有就给空数组。
- mastered（=掌握）：用户答对了检验并能自己迁移应用（独立解释/举出贴切新例）的概念 slug（必须也在 touched 里）。把握不准就别放。
- 宁缺毋滥：完全没对上就都空。绝不臆造清单外的 slug。
只输出 JSON：{"touched":["slug"],"mastered":[],"note":"<中文一句、20字内、可空>"}`

// englishAssessPrompt：英语陪练的判定语义——「概念」是场景关卡，点亮的标准是真开口。
const englishAssessPrompt = `你是「口语场景通关判定器」。下面给你一份英语口语场景关卡清单（每行格式 slug|场景名），以及学员与教练的一轮对话。判断这一轮里学员在清单中哪些场景上**真的用英语开口完成了核心任务**，以及是否**达到掌握水平**。
规则：
- touched（=点亮）：学员本轮**用英语**（至少 2 句有意义的英文，不是蹦单词）实质推进了该场景的核心任务——点了单、答了海关、做了自我介绍等，哪怕有语法错误或磕巴。全程说中文、只跟读教练给的句子、或只回答了「好/OK」的，**不算**。只放清单里确实存在的 slug，没有就给空数组。
- mastered（=掌握）：学员不仅完成任务，还用上了该场景的目标句式，且在教练抛出变体或突发状况（换个说法、单被做错、被追问）时仍能用英语接住（必须也在 touched 里）。把握不准就别放。
- 宁缺毋滥：完全没对上就都空。绝不臆造清单外的 slug。
只输出 JSON：{"touched":["slug"],"mastered":[],"note":"<中文一句、20字内、可空>"}`

// conceptAssessPrompts：按 agent slug 覆盖判定 prompt；未列出的走默认 conceptAssessPrompt。
var conceptAssessPrompts = map[string]string{
	"spoken-english": englishAssessPrompt,
}

// assessConcepts 对本轮一问一答判定点亮/掌握，按「只升不降」更新 user_concepts。
// 判定 prompt 按 agent slug 分派（英语=真开口语义，其余=概念理解语义）。
// 返回（进度视图, 新点亮概念名, 新掌握概念名, 本轮打通的档位名）。失败时返回当前进度、无新增（不拖累主对话）。
func (h *Handler) assessConcepts(a *models.ToolAgent, userID, userMsg, assistantMsg string) (gin.H, []string, []string, []string) {
	agentID := a.ID
	concepts := h.conceptList(agentID)
	if len(concepts) == 0 {
		return nil, nil, nil, nil
	}
	bySlug := make(map[string]*models.AgentConcept, len(concepts))
	var lines strings.Builder
	for i := range concepts {
		c := &concepts[i]
		bySlug[c.Slug] = c
		lines.WriteString(c.Slug)
		lines.WriteString("|")
		lines.WriteString(c.Name)
		lines.WriteString("\n")
	}
	levels := h.userConceptLevels(userID, agentID)

	assessPrompt := conceptAssessPrompt
	if p, ok := conceptAssessPrompts[a.Slug]; ok {
		assessPrompt = p
	}
	userContent := "概念清单：\n" + lines.String() + "\n本轮对话：\n用户：" + userMsg + "\n导师：" + assistantMsg
	raw, err := h.ds.Chat(
		[]deepseek.Message{
			{Role: "system", Content: assessPrompt},
			{Role: "user", Content: userContent},
		},
		deepseek.ChatOptions{Temperature: 0, MaxTokens: 200, ResponseFormat: "json_object"},
	)
	if err != nil {
		return conceptProgressView(concepts, levels), nil, nil, nil
	}
	var r conceptAssessResult
	if jerr := json.Unmarshal([]byte(strings.TrimSpace(raw)), &r); jerr != nil {
		return conceptProgressView(concepts, levels), nil, nil, nil
	}

	preCleared := tierClearedSet(concepts, levels) // 更新前已打通的档

	mastery := make(map[string]bool, len(r.Mastered))
	for _, s := range r.Mastered {
		mastery[s] = true
	}
	touched := make(map[string]bool, len(r.Touched)+len(r.Mastered))
	for _, s := range r.Touched {
		touched[s] = true
	}
	for s := range mastery {
		touched[s] = true // mastered 必含于 touched
	}

	var newlyLit, newlyMastered []string
	for slug := range touched {
		c := bySlug[slug]
		if c == nil {
			continue // 模型臆造的清单外 slug，丢弃
		}
		target := 1
		if mastery[slug] {
			target = 2
		}
		old := levels[c.ID]
		if target >= 2 && old < 2 {
			newlyMastered = append(newlyMastered, c.Name)
		} else if target >= 1 && old < 1 {
			newlyLit = append(newlyLit, c.Name)
		}
		h.bumpConcept(userID, agentID, c.ID, target) // 命中即 +touches；档位只升不降
		if target > old {
			levels[c.ID] = target
		}
	}

	// 本轮新打通的档（入门/进阶）。
	postCleared := tierClearedSet(concepts, levels)
	var tierCleared []string
	for _, t := range tierOrder {
		if postCleared[t] && !preCleared[t] {
			tierCleared = append(tierCleared, tierLabels[t])
		}
	}

	view := conceptProgressView(concepts, levels)
	if note := strings.TrimSpace(r.Note); note != "" {
		view["note"] = clipText(note, 80)
	}
	return view, newlyLit, newlyMastered, tierCleared
}

// conceptStateBrief 拼「学员进度」system 注入段（L1 主动教学的燃料）：分档进度、最近点亮、
// 建议下一个概念（按课程表顺序取第一个未点亮，天然入门优先）+ 本轮编排指令。
// fresh=true 表示新会话（要做开场编排：接续 + 场景钩子开课）。只给模型看，绝不直接展示给用户。
func (h *Handler) conceptStateBrief(userID, agentID string, fresh bool) string {
	concepts := h.conceptList(agentID)
	if len(concepts) == 0 {
		return ""
	}
	levels := h.userConceptLevels(userID, agentID)

	litByTier, totByTier := map[int]int{}, map[int]int{}
	lit, mastered := 0, 0
	byID := make(map[string]*models.AgentConcept, len(concepts))
	var next *models.AgentConcept
	for i := range concepts {
		c := &concepts[i]
		byID[c.ID] = c
		totByTier[c.Tier]++
		if levels[c.ID] >= 1 {
			lit++
			litByTier[c.Tier]++
		} else if next == nil {
			next = c
		}
		if levels[c.ID] >= 2 {
			mastered++
		}
	}

	// 最近点亮的 2 个（按更新时间），用于「接续」
	var recent []models.UserConcept
	h.db.Where("user_id = ? AND agent_id = ? AND level >= 1", userID, agentID).
		Order("updated_at desc").Limit(2).Find(&recent)
	var recentNames []string
	for i := range recent {
		if c := byID[recent[i].ConceptID]; c != nil {
			recentNames = append(recentNames, c.Name)
		}
	}

	var b strings.Builder
	b.WriteString("== 学员进度（仅你可见，据此编排本轮；绝不把这段原样复述给学员）==\n进度：")
	for _, t := range tierOrder {
		if totByTier[t] == 0 {
			continue
		}
		b.WriteString(tierLabels[t] + " " + strconv.Itoa(litByTier[t]) + "/" + strconv.Itoa(totByTier[t]) + "　")
	}
	b.WriteString("已点亮；已掌握 " + strconv.Itoa(mastered) + " 个。\n")
	if len(recentNames) > 0 {
		b.WriteString("最近点亮：" + strings.Join(recentNames, "、") + "。\n")
	}
	if next != nil {
		b.WriteString("建议下一个概念：" + next.Name + "（" + next.Theme + "｜" + next.Blurb + "）。\n")
		if next.Hook != "" {
			b.WriteString("　开课钩子（精编，用它开场，可贴学员语境微调措辞）：" + next.Hook + "\n")
		}
		if next.Check != "" {
			b.WriteString("　检验题（精编，讲透后用它检验，学员答上来才算真点亮）：" + next.Check + "\n")
		}
	} else {
		b.WriteString("整张地图已点亮：转入复习深化，挑「已点亮未掌握」的概念出检验，往掌握推。\n")
	}
	if n := dueCount(h.db, userID, agentID); n > 0 {
		b.WriteString("待复习：有 " + strconv.Itoa(n) + " 个已点亮的概念好几天没碰了。若学员没带自己的问题来，可先提议花一分钟快问快答热个身，再开新课。\n")
	}
	if fresh {
		if lit == 0 {
			b.WriteString("编排：这是学员的第一课。欢迎一句后，直接用「建议下一个概念」的生活场景钩子问题开课，不要问「想学什么」这类开放题。")
		} else {
			b.WriteString("编排：新的一节课。先用一句话接续进度（如「上次我们点亮了 X」），再用「建议下一个概念」的场景钩子问题开课。")
		}
		b.WriteString("若学员带着自己的问题来，先跟着 TA 的问题走，把相关概念顺势教透。\n")
	}
	b.WriteString("== 进度结束 ==")
	return b.String()
}

// bumpConcept 命中某概念：touches+1，档位 level 只升不降（upsert）。
func (h *Handler) bumpConcept(userID, agentID, conceptID string, level int) {
	var uc models.UserConcept
	if err := h.db.First(&uc, "user_id = ? AND concept_id = ?", userID, conceptID).Error; err == gorm.ErrRecordNotFound {
		uc = models.UserConcept{
			ID: idgen.New(), UserID: userID, AgentID: agentID, ConceptID: conceptID,
			Level: level, Touches: 1,
		}
		if cerr := h.db.Create(&uc).Error; cerr == nil {
			return
		}
		h.db.First(&uc, "user_id = ? AND concept_id = ?", userID, conceptID) // 并发：他人先建 → 重查走更新
	}
	updates := map[string]interface{}{"touches": gorm.Expr("touches + 1")}
	if level > uc.Level {
		updates["level"] = level
	}
	h.db.Model(&uc).Updates(updates)
}

// ============================ 首页催课条（L2 再入口主动） ============================

// NudgeLine 给「钉在首页」的学习 Agent 一句动态状态，替代静态 tagline——
// 用户还没点进来，主动性已经发生。返回 ("", false) 表示无状态可催（保持 tagline）。
// 措辞纪律：不做「今天」类日期判断，句子本身永远可行动；没开始学的人不催（tagline 即邀请）。
func NudgeLine(db *gorm.DB, a *models.ToolAgent, userID string) (string, bool) {
	if db == nil || a == nil || userID == "" {
		return "", false
	}
	// streak 火焰前缀：昨天学了、今天还没学 → 「别断」是最强的一句催课
	prefix := ""
	if days, risk := streakAtRisk(db, userID); risk {
		prefix = "🔥 连学 " + strconv.Itoa(days) + " 天，别断 · "
	}
	if a.Concept {
		var concepts []models.AgentConcept
		db.Where("agent_id = ?", a.ID).Order("sort asc").Find(&concepts)
		if len(concepts) == 0 {
			return "", false
		}
		var ucs []models.UserConcept
		db.Where("user_id = ? AND agent_id = ?", userID, a.ID).Find(&ucs)
		levels := make(map[string]int, len(ucs))
		for i := range ucs {
			levels[ucs[i].ConceptID] = ucs[i].Level
		}
		lit, mastered := 0, 0
		litByTier, totByTier := map[int]int{}, map[int]int{}
		var next *models.AgentConcept
		for i := range concepts {
			c := &concepts[i]
			totByTier[c.Tier]++
			if levels[c.ID] >= 1 {
				lit++
				litByTier[c.Tier]++
			} else if next == nil {
				next = c
			}
			if levels[c.ID] >= 2 {
				mastered++
			}
		}
		if lit == 0 {
			return "", false // 还没开始学：tagline 本身就是邀请，不催
		}
		if n := dueCount(db, userID, a.ID); n > 0 {
			return "🔁 " + strconv.Itoa(n) + " 个概念好几天没碰了，快问快答保住它们", true
		}
		if next != nil {
			return prefix + "下一个待点亮：『" + next.Name + "』 · " + tierLabels[next.Tier] + " " +
				strconv.Itoa(litByTier[next.Tier]) + "/" + strconv.Itoa(totByTier[next.Tier]), true
		}
		if mastered < len(concepts) {
			return prefix + "整张地图已点亮 · 还剩 " + strconv.Itoa(len(concepts)-mastered) + " 个概念待「掌握」", true
		}
		return "整张地图已通关 🎉 随时来聊聊新的困惑", true
	}
	if a.Assess {
		var sk models.AgentSkill
		if db.First(&sk, "user_id = ? AND agent_id = ?", userID, a.ID).Error != nil || sk.Assessed == 0 {
			return "", false // 未定级：tagline 已是邀请
		}
		level := levelFromDims(sk.Fluency, sk.Accuracy, sk.Expression)
		weak, min := "流利", sk.Fluency
		if sk.Accuracy < min {
			weak, min = "准确", sk.Accuracy
		}
		if sk.Expression < min {
			weak = "表达"
		}
		return prefix + "Lv." + strconv.Itoa(level) + " " + tierName(level) + " · 弱项「" + weak + "」，今天练两句？", true
	}
	return "", false
}
