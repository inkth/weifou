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
var curricula = map[string][]seedConcept{
	"learn-psychology": psychologyConcepts,
	"learn-economics":  economicsConcepts,
	"learn-philosophy": philosophyConcepts,
	"learn-ideas":      ideasConcepts,
	"learn-logic":      logicConcepts,
	"learn-science":    scienceConcepts,
	"learn-aesthetics": aestheticsConcepts,
	"learn-marketing":  marketingConcepts,
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
		slugs := make([]string, 0, len(list))
		for i := range list {
			sc := list[i]
			slugs = append(slugs, sc.Slug)
			var existing models.AgentConcept
			if db.Where("agent_id = ? AND slug = ?", agent.ID, sc.Slug).First(&existing).Error == gorm.ErrRecordNotFound {
				db.Create(&models.AgentConcept{
					ID: idgen.New(), AgentID: agent.ID, Slug: sc.Slug,
					Theme: sc.Theme, Tier: sc.Tier, Name: sc.Name, Blurb: sc.Blurb, Sort: i,
				})
				continue
			}
			db.Model(&existing).Updates(map[string]interface{}{
				"theme": sc.Theme, "tier": sc.Tier, "name": sc.Name, "blurb": sc.Blurb, "sort": i,
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
		a.items = append(a.items, gin.H{"name": c.Name, "blurb": c.Blurb, "level": lv, "theme": c.Theme})
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

const conceptAssessPrompt = `你是「概念掌握判定器」。下面给你一份某学习领域的核心概念清单（每行格式 slug|概念名），以及用户与导师的一轮对话。判断这一轮**实际涉及**了清单中的哪些概念，以及用户是否对该概念**展现出真正的理解**（能自己解释、举例或应用，而非仅被导师提及）。
规则：
- touched：本轮真实涉及/讲解到的概念 slug 列表（=被点亮）。只放清单里确实存在的 slug，没有就给空数组。
- mastered：用户已展现真正理解的概念 slug（mastered 里的必须也在 touched 里）。把握不准就别放。
- 宁缺毋滥：完全没对上就都空。绝不臆造清单外的 slug。
只输出 JSON：{"touched":["slug"],"mastered":[],"note":"<中文一句、20字内、可空>"}`

// assessConcepts 对本轮一问一答判定点亮/掌握，按「只升不降」更新 user_concepts。
// 返回（进度视图, 新点亮概念名, 新掌握概念名, 本轮打通的档位名）。失败时返回当前进度、无新增（不拖累主对话）。
func (h *Handler) assessConcepts(userID, agentID, userMsg, assistantMsg string) (gin.H, []string, []string, []string) {
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

	userContent := "概念清单：\n" + lines.String() + "\n本轮对话：\n用户：" + userMsg + "\n导师：" + assistantMsg
	raw, err := h.ds.Chat(
		[]deepseek.Message{
			{Role: "system", Content: conceptAssessPrompt},
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
