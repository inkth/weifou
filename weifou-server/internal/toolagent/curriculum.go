// Package toolagent — curriculum.go：概念型学习 Agent 的「核心概念」课程表与点亮引擎。
//
// 四门完备课（英语/心理/营销/逻辑）各装载一份 curated 的 ~30 个核心概念/关卡，
// 且全部人工精编 Hook+Check（人工策展骨架，不让模型现编 = 这条线的护城河，由 TestCuratedContentComplete 守护）。
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
// 2026-07-04 课程线收缩为四门完备课（经济/哲学/思想/科学/审美已退役，见 Seed 的 retired）。
var curricula = map[string][]seedConcept{
	"learn-psychology": psychologyConcepts,
	"learn-logic":      logicConcepts,
	"learn-marketing":  marketingConcepts,
	"spoken-english":   englishScenarios,
}

// hookCheck 人工精编内容（开课钩子 + 检验题）。与 seedConcept 分开存，SeedConcepts 时按 slug 合并写入。
type hookCheck struct{ Hook, Check string }

// curatedContent：agent slug → concept slug → 精编内容。四门课全部精编。
// 这是这条产品线的护城河：钩子制造好奇缺口、检验题考迁移应用——不是模型现编能稳定给出的品质。
var curatedContent = map[string]map[string]hookCheck{
	"learn-psychology": psychologyContent,
	"spoken-english":   englishContent,
	"learn-logic":      logicContent,
	"learn-marketing":  marketingContent,
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

// englishContent：英语陪练精编 Hook/Check，28 关全精编。
// Hook＝把学员直接丢进情境的开场任务（带一句英文触发语）；Check＝变体/突发状况，接得住才算掌握。
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
	// —— 生活 ——
	"restaurant-order": {
		Hook:  "服务生递上菜单：\"Can I get you started with something to drink?\"——先点杯喝的，再请对方推荐招牌菜，并说明你不吃辣。开口试试。",
		Check: "上菜后发现牛排太生了。用英语礼貌地请服务生拿回去再煎熟一点，别忘了先给一句缓冲。",
	},
	"shopping-clothes": {
		Hook:  "店员迎上来：\"Hi, are you looking for anything special?\"——你看中一件外套，用英语问有没有中码、能不能试穿。",
		Check: "试完发现大了一号。用英语一次问清两件事：有没有小一码，以及不合适能不能退换。",
	},
	"ask-directions": {
		Hook:  "你在陌生街头找地铁站，拦住一位路人：\"Excuse me…\"——把路问清楚，记得确认走路要多久。",
		Check: "对方语速太快你没听清。用英语请 TA 说慢一点，并把路线复述一遍向 TA 确认。",
	},
	"phone-booking": {
		Hook:  "你打电话订今晚 7 点、4 个人的位子。电话接通：\"Hello, Bella Italia, how can I help?\"——电话里没有手势可借，全靠开口。",
		Check: "餐厅说 7 点满了，只有 6 点或 8 点半。用英语选一个时间，并留下你的名字和电话。",
	},
	"see-doctor": {
		Hook:  "医生抬头问：\"So, what brings you in today?\"——你头疼两天了，还有点发烧。用英语把症状说清楚。",
		Check: "医生开药时说 \"Take one twice a day after meals.\"——先说说你听懂了什么，再用英语追问一句：有什么要忌口的吗？",
	},
	// —— 旅行 ——
	"customs-qa": {
		Hook:  "海关官员面无表情：\"What's the purpose of your visit?\"——旅游、待 10 天、住朋友家，用英语稳稳答上这三连问。",
		Check: "官员追问：\"Are you carrying any food or plants?\"——你包里有两盒茶叶。用英语如实申报，并问能不能带入境。",
	},
	"hotel-checkin": {
		Hook:  "前台微笑：\"Good evening, checking in?\"——报姓名办入住，顺便提一个要求：高楼层、安静的房间。",
		Check: "进房发现空调坏了。打给前台，用英语说明问题，并要求换房或马上派人来修。",
	},
	"taxi-ride": {
		Hook:  "上车后司机回头：\"Where to?\"——用英语说清目的地，再加一句：你赶时间，请走最快的路。",
		Check: "你发现司机好像在绕路。用英语礼貌地问为什么走这条路，并说你想按导航的路线走。",
	},
	"attraction-tickets": {
		Hook:  "售票窗口前：\"Next, please!\"——两张成人票，再用英语问两件事：学生证有没有折扣、几点闭馆。",
		Check: "买完票突然下起大雨，你想改天再来。用英语问这张票能不能改期或退款。",
	},
	"lost-luggage": {
		Hook:  "行李转盘转空了，你的箱子没出来。走到柜台开口：\"Hi, my luggage didn't arrive.\"——描述箱子的颜色和大小，并留下酒店地址。",
		Check: "工作人员问：\"What was inside?\"——为了理赔，用英语说出箱子里三件重要物品和大致价值。",
	},
	"travel-help": {
		Hook:  "糟糕，护照不见了。你走进警察局：\"Excuse me, I need help.\"——用英语说清什么时候、在哪里可能丢的，请对方帮忙。",
		Check: "警察请你先填表，问你要不要联系使馆。用英语问中国大使馆怎么走、几点开门。",
	},
	// —— 职场 ——
	"work-self-intro": {
		Hook:  "入职第一天，经理向大家介绍：\"This is our new colleague.\"——用三四句英语让团队记住你：名字、负责什么、加一点个人色彩。",
		Check: "会后一位同事问：\"So how are you finding it so far?\"——用英语得体回应，并顺势约 TA 喝杯咖啡聊聊团队。",
	},
	"task-handover": {
		Hook:  "你要出差三天，得把报表任务交给同事：\"Hey, do you have a minute?\"——用英语说清任务内容、截止时间和注意事项。",
		Check: "同事问：\"Sure, but what if the client emails me directly?\"——用英语交代清楚：什么情况 TA 自己定，什么情况必须找你。",
	},
	"project-present": {
		Hook:  "会议室安静下来，轮到你了。用英语完成开场三件套：问候、一句话点明主题、预告接下来要讲的三个部分。",
		Check: "讲到一半有人举手：\"Sorry, can you go back to the numbers?\"——用英语从容接住：回应、澄清数据、再把节奏拉回你的主线。",
	},
	"price-negotiation": {
		Hook:  "供应商报价 12000 美元，超了你的预算。对方问：\"So, what do you think?\"——用英语表达兴趣但明确压价，报出你的目标价。",
		Check: "对方说：\"That's really the best we can do.\"——别掀桌子，用英语换个筹码再谈：量大折扣、延长账期或附赠服务，任选其一。",
	},
	"pantry-smalltalk": {
		Hook:  "茶水间遇到隔壁组同事，对方笑着说：\"Hey! How's your week going?\"——用英语自然接住，再把话题递回去。",
		Check: "对方聊起周末去爬山了。用英语追问一个细节，再顺势分享一句你自己的周末。",
	},
	"video-call-clarify": {
		Hook:  "跨国例会上对方声音断断续续，你只听到一半。用英语礼貌打断，请对方重复关键部分。",
		Check: "你还是没听清那个截止日期。用 \"Just to confirm, did you say…?\" 向对方确认，并提议会后邮件补一份纪要。",
	},
	// —— 面试 ——
	"strengths-weaknesses": {
		Hook:  "面试官问：\"What would you say is your biggest weakness?\"——别说「我太追求完美」。用英语讲一个真实的弱点，加上你正在怎么改进。",
		Check: "紧接着：\"And your greatest strength?\"——用英语讲一个优势，并配一个 30 秒的小例子，不空喊口号。",
	},
	"star-story": {
		Hook:  "面试官说：\"Tell me about a time you solved a difficult problem.\"——用 STAR 四步（情境-任务-行动-结果）讲一段你的真实经历。",
		Check: "面试官追问：\"What would you do differently next time?\"——用英语补上你的反思，让这个故事加分而不是露怯。",
	},
	"why-us": {
		Hook:  "面试官问：\"So, why do you want to join us?\"——不吹捧、不空泛，用英语给出两个具体理由：一个关于这家公司，一个关于你自己。",
		Check: "追问来了：\"You could get that at other companies too, no?\"——用英语接住，把理由落到只有这家才有的点上。",
	},
	"ask-interviewer": {
		Hook:  "面试尾声：\"Do you have any questions for me?\"——千万别说 No。用英语问一个显水平的问题：关于团队、挑战或成长空间。",
		Check: "面试官答完后，用英语再追问一层，然后自然收尾、表达期待——让最后一分钟也在加分。",
	},
	"mock-full-interview": {
		Hook:  "终极关：一场 10 分钟的全英模拟面试。我来演面试官，从 \"Tell me about yourself\" 一路问到反问环节——准备好了就说 \"I'm ready.\"",
		Check: "复盘时刻：用英语说出这场面试里你最满意的一句回答、和最想重来的一句——用英语复盘自己，也是面试力。",
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

// ============================ 学逻辑 · 精编钩子与检验题 ============================
// 钩子＝拿一个「听起来很有道理」的说法当靶子（制造找茬的快感）；
// 检验题＝多为找茬/判断型（天然适合点选作答），或让学员当场完成一次小推理。

var logicContent = map[string]hookCheck{
	"argument-structure": {
		Hook:  "「他说得好有道理」——可你说得出 TA 的结论靠哪几根柱子撑着吗？撑不住的道理，再顺耳也是空中楼阁。",
		Check: "「游戏有害，因为我表弟玩游戏后成绩下降了。」这个论证的前提和结论各是什么？这根柱子撑得住吗？",
	},
	"premise-conclusion": {
		Hook:  "吵架最常见的错位：你在反驳 TA 的结论，TA 却觉得你没听懂理由。先学会拆——哪句是主张，哪句是理由。",
		Check: "「加班多的公司发展都快，所以我们该多加班」——这句话里藏了几个前提？哪个最可疑？",
	},
	"deductive-inductive": {
		Hook:  "「天鹅都是白的」和「单身汉都未婚」——两句话的可靠程度天差地别，差就差在推理方式上。",
		Check: "「过去十年房价年年涨，所以明年也会涨」——这是演绎还是归纳？它的结论有多硬？",
	},
	"validity": {
		Hook:  "一个论证可以前提全真、结论却不成立；也可以前提全是胡说、结构却无懈可击——先学会只看结构。",
		Check: "「如果下雨，地会湿。现在地湿了，所以刚下过雨。」——结构有效吗？哪里断了？",
	},
	"soundness": {
		Hook:  "结构对≠结论真。「所有猫都会飞，加菲是猫，所以加菲会飞」——推理完美，结论荒谬，问题出在哪？",
		Check: "拿到一个结构有效的论证，你还要检查什么才敢信它的结论？举个例子说明。",
	},
	"correlation-causation": {
		Hook:  "冰淇淋销量涨，溺水人数也涨——所以冰淇淋导致溺水？几乎所有「研究表明」的坑都埋在这里。",
		Check: "「常喝红酒的人更长寿」——除了「红酒延寿」，你能给出两种别的解释吗？",
	},
	"ad-hominem": {
		Hook:  "「你一个单身狗也配谈婚姻？」——注意，这句话攻击的是人。可对方的观点被驳倒了吗？",
		Check: "「他自己都做不到，凭什么劝别人」——这算人身攻击谬误吗？什么时候「看这个人」反而是合理的？",
	},
	"straw-man": {
		Hook:  "你说「想少熬夜」，对方回「你是想躺平吗？」——你被立了个稻草人，打倒它可不算赢了你。",
		Check: "「我觉得该控制刷短视频的时间。」「你就是觉得娱乐有罪！」——稻草人在哪？原观点被歪成了什么？",
	},
	"false-dilemma": {
		Hook:  "「你不支持我，就是反对我。」——当世界被硬切成两半时，通常是有人不想让你看到第三条路。",
		Check: "「要么拼命加班，要么回家躺平」——说出至少两个被这句话藏起来的选项。",
	},
	"slippery-slope": {
		Hook:  "「今天迟到一次，明天就敢旷工，后天公司就完了」——从第一级台阶直接滑进深渊，中间的刹车全被拆了。",
		Check: "「放开游戏时间，孩子就会沉迷，然后辍学」——这个滑坡缺了什么？什么时候担心连锁反应又是合理的？",
	},
	"burden-of-proof": {
		Hook:  "「你倒是证明鬼不存在啊！」——等等，该举证的是谁？「谁主张谁举证」这条规则一翻转，骗子就赢了。",
		Check: "朋友说「这保健品有奇效，不信你证明它没用」——举证责任在谁？你该怎么回这句话？",
	},
	"occams-razor-logic": {
		Hook:  "钥匙不见了：是被平行宇宙的自己拿走了，还是忘在外套兜里？两个解释都「说得通」，选哪个？",
		Check: "网站打不开：是全球黑客攻击，还是你家 Wi-Fi 断了？用剃刀给排查顺序排个序，并说出理由。",
	},
	"circular-reasoning": {
		Hook:  "「这本书说的都是真的，因为书里写着『本书句句属实』。」——绕了一圈，证据就是结论本身。",
		Check: "「他值得信任，因为他是个可信的人」——问题在哪？再从广告语里找一个类似的例子。",
	},
	"appeal-to-authority": {
		Hook:  "「诺贝尔奖得主都说了……」——且慢，TA 得的是物理奖，谈的却是育儿。权威也有边界。",
		Check: "什么时候引用专家是合理的、什么时候是谬误？给出两个可操作的判断标准。",
	},
	"appeal-to-emotion": {
		Hook:  "「想想孩子们！」——眼眶一热，脑子就让位了。煽情不是论证，是绕过论证。",
		Check: "一则募捐广告全是催泪画面、没有任何数据——它在诉诸什么？怎么在被打动的同时保住判断力？",
	},
	"hasty-generalization": {
		Hook:  "「我遇到的两个那儿的人都很小气」——两个人，就给几千万人定了性。你的样本呢？",
		Check: "「我们店 5 个顾客 4 个好评」——这能说明什么、不能说明什么？多大的样本才配下结论？",
	},
	"survivorship-bias": {
		Hook:  "「读书无用，你看盖茨退学都成首富了」——你看到的是活下来的那一个，沉底的一万个没机会开口。",
		Check: "「这些百年老店都不做广告，所以广告没用」——用幸存者偏差拆一下：这个观察漏掉了谁？",
	},
	"base-rate-fallacy": {
		Hook:  "检测准确率 99%，你被测出阳性——你真的有 99% 的概率患病吗？大多数人都算错了这道题。",
		Check: "发病率万分之一的病，检测准确率 99%，测出阳性后真患病的概率是高是低？说说直觉为什么错。",
	},
	"equivocation": {
		Hook:  "「法律面前人人平等，所以票价对富人穷人一个价才平等」——同一个「平等」，中途被悄悄换了含义。",
		Check: "「自由就是想做什么就做什么，所以红绿灯限制了自由」——哪个词被偷换了？它的两个含义各是什么？",
	},
	"begging-the-question": {
		Hook:  "「为什么他是对的？因为他从不犯错。」——看起来在论证，其实把要证明的东西当成了起点。",
		Check: "「灵魂不朽，因为灵魂是不会消亡的」——预设藏在哪？它和循环论证是什么关系？",
	},
	"necessary-sufficient-logic": {
		Hook:  "「努力就能成功」和「成功都需要努力」——一字之差，逻辑上是两个世界。",
		Check: "「有驾照」对「合法开车」是必要条件还是充分条件？再举一个生活里两者被混淆的例子。",
	},
	"counterexample": {
		Hook:  "「成功人士都早起」——推翻这句话不需要写论文，只需要一个人。",
		Check: "「压力都是坏事」——给出一个反例。一个反例能推翻什么样的命题、推不翻什么样的？",
	},
	"reductio": {
		Hook:  "想驳倒一个观点，最优雅的方式有时是先假装同意它——然后陪它一路走到荒谬的终点。",
		Check: "「越贵的东西越好」——顺着这句话推出一个荒谬结论，完成一次归谬。",
	},
	"bayesian-thinking": {
		Hook:  "朋友迟到了，你的第一反应是「TA 不重视我」——但下结论前该先问一句：TA 平时迟到吗？",
		Check: "你本来觉得某产品不错（八成把握），现在刷到一条差评——信念该更新多少？由什么决定更新幅度？",
	},
	"falsifiability-logic": {
		Hook:  "「水逆导致你倒霉」——倒霉了算它对，顺利了就说「化解了」。永远不会错的理论，恰恰最有问题。",
		Check: "「这股票要么涨要么跌」和「明天收盘涨 2% 以上」——哪句可证伪？哪句更有信息量？",
	},
	"steelman": {
		Hook:  "想真正赢一场争论？先替对方把论证修到最强再反驳——打倒最强版本，才算真的赢。",
		Check: "挑一个你反对的观点，先用两句话把它讲到最有说服力，再指出它最要害的弱点。",
	},
	"analogy-argument": {
		Hook:  "「治国如烹小鲜」——类比让人秒懂，也最容易把人带偏：相似的地方成立，不相似的地方呢？",
		Check: "「公司就像球队，所以表现不行就该换人」——这个类比哪里贴切、哪里瘸腿？",
	},
	"gambler-fallacy": {
		Hook:  "连开五把大，下一把「该」开小了吧？——硬币没有记忆，可惜赌徒有。",
		Check: "「面试连挂三家，下一家肯定稳了」——哪里犯了赌徒谬误？什么情况下「连败后概率变高」又是真的？",
	},
	"moving-goalposts": {
		Hook:  "你拿出证据，对方说「这不算，你得有 XX」；你拿出 XX，对方说「那也不算」——球门一直在跑，球永远进不了。",
		Check: "怎么识别对话里的移动球门？给出一句能提前把标准钉死的话术。",
	},
	"cherry-picking": {
		Hook:  "广告说「93% 用户表示有效」——没说的是：问卷只发给了复购用户。被挑出来的樱桃，都很甜。",
		Check: "用数据说谎不需要造假，只需要筛选。举一个「只报喜不报忧」的常见套路，并给出你的反问。",
	},
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

// ============================ 学营销 · 精编钩子与检验题 ============================
// 钩子＝用一个熟悉的商业现象制造好奇缺口；
// 检验题＝把概念用到学员自己的生意上（这门课的迁移检验天然指向「你自己怎么卖」）。

var marketingContent = map[string]hookCheck{
	"positioning": {
		Hook:  "提到「怕上火喝什么」，你脑子里蹦出了哪三个字？定位不是你说你是谁，而是用户心智里那个格子归了谁。",
		Check: "你的生意想占用户心里哪个词？用「对谁 + 什么场景 + 第一选择」造一句你自己的定位。",
	},
	"target-audience": {
		Hook:  "「我的产品老少咸宜」——听着是优点，其实是营销里最贵的一句话：想讨好所有人，广告费就全打了水漂。",
		Check: "把「25-40 岁女性」这种假人群，细化成一个你真实客户的画像：TA 为什么非你不可？",
	},
	"value-proposition": {
		Hook:  "电梯里遇到理想客户，只有 15 秒——你能一句话说清「选我而不选别家」的理由吗？你说不清，用户就懒得懂。",
		Check: "套这个公式给你的生意写一句：帮【谁】在【什么场景】解决【什么问题】，和别家不同的是【什么】。",
	},
	"brand": {
		Hook:  "同样的白 T 恤，印上某个对勾就贵十倍——多出来的钱买的不是布料，是什么？",
		Check: "品牌＝预期的稳定兑现。客户第二次找你，是冲着什么预期来的？这个预期你每次都兑现了吗？",
	},
	"marketing-mix-4p": {
		Hook:  "生意不好，该降价、换渠道还是砸广告？先别急着动手——4P 是张体检表，得一格一格查。",
		Check: "拿你的生意过一遍 4P：产品、价格、渠道、推广——哪一格最弱？为什么该先补它？",
	},
	"customer-needs": {
		Hook:  "用户买电钻，要的从来不是钻头，是墙上那个洞——你在卖钻头，还是在卖洞？",
		Check: "你的客户掏钱那一刻，真正想买走的是什么？说出功能之下的那层需求。",
	},
	"funnel": {
		Hook:  "100 个人看到你，10 个点进来，1 个下单——生意的秘密全藏在这两道「漏」里：漏在哪，补哪。",
		Check: "画一下你生意的漏斗（看见→兴趣→咨询→成交），估个数：哪一层漏得最凶？",
	},
	"differentiation": {
		Hook:  "一条街五家奶茶店，凭什么排队的是那一家？「比别人好一点」没用，「和别人不一样」才有用。",
		Check: "说出你和同行最实在的一个不同点——如果你说的是「更用心」，重新想一个客户看得见摸得着的。",
	},
	"word-of-mouth": {
		Hook:  "最好的广告是客户那句「我推荐你去找 TA」——免费，但也最难买到。口碑是设计出来的，不是等出来的。",
		Check: "想让客户忍不住转介绍，你能在服务里加一个什么「可谈论的瞬间」？",
	},
	"call-to-action": {
		Hook:  "文案写得再动人，结尾没有「下一步」，用户点个赞就走了——每一次曝光都该有一个明确的动作出口。",
		Check: "翻出你最近发的一条内容：看完的人「接下来该做什么」清楚吗？改写一个更明确的行动号召。",
	},
	"segmentation": {
		Hook:  "同一门课，卖给宝妈讲「省时间」，卖给学生讲「性价比」——市场不是一块饼，是好几块口味不同的饼。",
		Check: "把你的客户切成两三类，每类挑一个最打动他们的卖点——哪一类你最该聚焦？",
	},
	"pricing-strategy": {
		Hook:  "定价 99 和 100 只差一块钱，成交差一截；有时涨价反而卖得更好——价格不是数字，是信号。",
		Check: "你现在的价格在向客户传递什么信号（便宜？专业？高端？）——和你想立的定位一致吗？",
	},
	"usp": {
		Hook:  "「怕上火」三个字让一罐凉茶卖了几百亿——你的生意里，那个让人一句话记住的卖点是什么？",
		Check: "给你的服务写一个独特卖点：一句话、有具体承诺、竞品抄不走——三个条件缺一不可。",
	},
	"customer-journey": {
		Hook:  "客户不是「看到广告就下单」的直线，而是刷到→观望→比价→犹豫→被朋友一句话推了一把——每一站你都在场吗？",
		Check: "复盘你最近一个成交客户：TA 从第一次听说你到付款走过哪几站？哪一站差点把 TA 丢了？",
	},
	"aida": {
		Hook:  "一条爆款文案的骨架从来是四步：让人停下、让人好奇、让人心痒、让人行动——少一步都白写。",
		Check: "用 AIDA 四步给你的产品写一条朋友圈：注意、兴趣、欲望、行动，每步一句话。",
	},
	"conversion-rate": {
		Hook:  "与其多拉 1000 个人来看，不如让 100 个看的人里多 5 个下单——转化率每提一个点，都是纯利润。",
		Check: "你的咨询转成交大概几成？挑一个最卡客户的点（价格、信任还是流程），给一个提转化的具体改法。",
	},
	"cac-ltv": {
		Hook:  "拉来一个客户花 300，TA 一辈子在你这儿消费 500——这生意能做；反过来，就是烧钱买热闹。",
		Check: "粗算你自己的账：获客一个人的成本 vs TA 长期带来的收入——这笔账健康吗？临界线在哪？",
	},
	"retention": {
		Hook:  "拉新的成本是留客的五倍——可大多数人把九成力气花在拉新上。老客户才是那口井。",
		Check: "你的生意里回头客占几成？设计一个让客户「下次还来」的最小动作——打折不算。",
	},
	"viral-loop": {
		Hook:  "最省钱的增长：让每个客户平均带来超过一个新客户——雪球就自己滚起来了。裂变不是运气，是机制。",
		Check: "给你的生意设计一个裂变钩子：老客推荐新客，双方各得什么？TA 为什么愿意开口？",
	},
	"content-marketing": {
		Hook:  "硬广没人看，但「干货」有人追着要——先给足有用的，再顺便卖。内容是先付出后收获的获客。",
		Check: "围绕你的专业列出客户最常问的三个问题——每个都能变成一条内容，你先写哪条？为什么？",
	},
	"storytelling": {
		Hook:  "「我们采用先进工艺」没人记得；「为了女儿的湿疹我试了 47 种配方」过目不忘——人记不住参数，记得住故事。",
		Check: "用「困境→尝试→转折→现在」四步，把你为什么做这一行讲成一个 100 字的故事。",
	},
	"social-proof-mkt": {
		Hook:  "你也曾因为「已售 10 万+」下过单吧？人在拿不准时，会抄别人的作业——所以要让别人看见有人在买。",
		Check: "你的主页现在有几种社会证明（评价、案例、销量、合作方）？最该补的是哪一种？",
	},
	"scarcity-urgency": {
		Hook:  "「仅剩 3 席」「今晚截止」——为什么明知是套路还是会心动？稀缺感一到，拖延症就好了。",
		Check: "给你的服务设计一个真实不造假的稀缺或紧迫（名额、时段、赠品）——为什么「真实」这条底线碰不得？",
	},
	"anchoring-price": {
		Hook:  "菜单第一页那道 888 的招牌菜，可能压根没打算卖——它站在那儿，是为了让 288 显得便宜。",
		Check: "给你的服务设计三档价格（锚定档、主推档、入门档），说说每一档各自的任务是什么。",
	},
	"kol-marketing": {
		Hook:  "用户不信广告，但信 TA 关注了三年的博主——达人带货的本质，是租用别人攒下的信任。",
		Check: "挑达人合作时，粉丝量和「粉丝像不像你的客户」哪个更重要？给你的生意选一类最配的达人。",
	},
	"private-domain": {
		Hook:  "平台流量是租的，随时涨租断供；加到你微信里的客户才是自己的地——私域就是把租客变成家人。",
		Check: "你现在有多少客户沉淀在自己手里？设计一个让客户愿意加你的理由——「方便联系」不算。",
	},
	"ab-testing": {
		Hook:  "两版文案吵不出谁好？别开会了——各发一半，让数据投票。增长高手都让市场说话。",
		Check: "挑一个你拿不准的决定（价格、标题、头像），设计一个最小 A/B 测试：测什么、怎么分组、看什么数。",
	},
	"growth-hacking": {
		Hook:  "没有大预算也能增长：找到用户「啊哈」的那个瞬间，用一周一个小实验去撬——增长是实验出来的，不是砸出来的。",
		Check: "你的客户在哪个瞬间真正觉得「值了」？设计一个让新客户更快到达那个瞬间的小实验。",
	},
	"product-market-fit": {
		Hook:  "营销救不了没人要的产品——广告投得越猛，死得越快。先问一句：停掉推广，还有人主动找来吗？",
		Check: "一个残忍的检验：如果明天起停更停投放，一个月后还有多少客户找上门？这个答案说明什么？",
	},
	"brand-moat": {
		Hook:  "你的打法今天有效，明天就会被同行抄走——抄不走的是什么？老客户的偏爱、你的名字、你攒下的信任。",
		Check: "盘点你生意里同行抄不走的资产，挑一个最值得加固的，说说打算怎么加固。",
	},
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

// SeedConcepts 把四份课程表幂等写入 agent_concepts（按 agent_id+slug），并清掉已不在清单里的旧概念
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
