// 开口（spoken-english）28 关剧本：四主题（生活/旅行/职场/面试）。与 englishContent
// （Hook/Check 题面）配套。形态 = 剧本对话 + ASR 跟读（多邻国式，零 LLM 保住「真开口」）：
// 每关四节点——[0] 对方开口（Hook 已含英文触发语），学员从三句英文里挑一句接住
// （1 得体 / 1 中式英语或语法错 / 1 语域错——太冲、太软或太生硬）；[1] 跟读节点：
// 按住麦克风把选对的那句读出来（小程序 WechatSI en_US 转写，matchSay 模糊匹配）；
// [2] 场景加压（精编 Check 的突发状况）再选一轮；[3] 再跟读 → 点亮。
// 品质纪律（课魂+测试守护）：
//   - 错误项的点破要具体到语言点（a/an、时态、语域），不是泛泛「不对」；
//   - 对方反应先用英文演一句，再中文点破——沉浸不断，教学不糊；
//   - SayFail 永远鼓励（慢一点、咬清楚），跟读不罚、读到中为止；
//   - 通关 = 点亮概念 + 三维段位确定性爬升（bumpSkillScripted）；
//   - CheckOpts 供复习挑战（中文判断题，Label ≤20 字）；Clear 带下一关悬念；Note ≤18 字。
package toolagent

var learnEnglishScript = map[string]levelScript{
	// ============ 生活 ============
	"cafe-order": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回店员）",
				Options: []nodeOption{
					{Label: "Give me an oat latte, less sugar.", Reply: "店员照做了，但笑容淡了半秒——\"...Sure.\" 这句语法没错，错在语气：祈使句开头像下命令。英语点单的礼貌不靠堆 please，靠句式——Could I get... 一开口就顺耳。", Next: NodeRetry},
					{Label: "I want a oat latte with little sugar.", Reply: "店员歪了下头：\"A... what?\" 两个坑：oat 元音开头要用 an；little sugar 是「几乎没糖」，你要的「少糖」是 less sugar。中式直译最容易栽在这种小词上。", Next: NodeRetry},
					{Label: "Hi! Could I get an oat latte with less sugar, please?", Reply: "店员笑了：\"Of course!\" ——Could I get 是点单的万能开头：礼貌、自然、不卑不亢；an oat、less sugar 两个语言点也都踩对了。现在，把它变成你自己的。", Next: 1},
				},
			},
			{
				Say:     "Hi! Could I get an oat latte with less sugar, please?",
				SayOK:   "店员边做边应：\"Oat latte, less sugar—coming right up!\" ——这句从你嘴里出来了，不是从屏幕上滑过去的。别急着走，店里马上出状况。",
				SayFail: "没对上——别慌，放慢，一个词一个词咬清楚：Could I get... an oat latte... with less sugar。机器听得懂，你就说得清。",
				SayNext: 2,
			},
			{
				Prompt: "端上来的却是杯美式，店员还挺开心：\"Here's your americano!\" ——礼貌指出问题并要求重做，三句挑一句：",
				Options: []nodeOption{
					{Label: "This is wrong. Change it.", Reply: "店员愣住，周围安静了一秒——\"...Oh. OK.\" 单子是能换，你也成了全店最凶的客人。指错的万能缓冲是先 Sorry / Excuse me 一句，火药味立刻归零。", Next: NodeRetry},
					{Label: "Sorry, I think this is an americano—I ordered an oat latte.", Reply: "店员一拍脑门：\"Oh no, my bad! Let me remake that for you.\" ——Sorry 开头缓冲、I think 留有余地、说清你点的是什么：指错三件套，一句全齐。读出来。", Next: 3},
					{Label: "Oh... never mind, it's OK.", Reply: "你端走了那杯不想喝的美式——最熟悉的忍。注意：英语里 it's OK 在这个场景就是「算了不用改」，店员真不会改。点单白点，钱白花，开口的机会也让出去了。", Next: NodeRetry},
				},
			},
			{
				Say:     "Sorry, I think this is an americano—I ordered an oat latte.",
				SayOK:   "店员重做了一杯，还多给你贴了张会员贴纸：\"Sorry again!\" ——指出问题、拿回自己那杯，全程体面。这才是点单这关的完整版。",
				SayFail: "再来一次——重音放在 americano 和 oat latte 两个词上，对比一出来，意思就清楚了。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "冷脸说 This is wrong", Reply: "语法对，语气冲——全店最凶客人就是这么诞生的。再选一个。"},
			{Label: "Sorry开头，说清点的是拿铁", Reply: "对——Sorry 缓冲、I think 留余地、说清原单：指错三件套，事办成人也体面。"},
			{Label: "It's OK，将就喝了吧", Reply: "英语场景里 it's OK 就是「不用改」——单白点了。再想想。"},
		},
		Correct: 1,
		Clear:   "第一句英文出口了，还顺手要回了自己那杯拿铁。下一关「餐厅点餐」——菜单看不懂？让服务生推荐，忌口也得说清。",
		Note:    "开口点成了一杯拿铁",
	},
	// ============ 面试 ============
	"interview-self-intro": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句开场作答）",
				Options: []nodeOption{
					{Label: "I was born in 1998 and studied hard since primary school.", Reply: "面试官笔尖停了：\"...OK, and more recently?\" 从出生年讲起是中式简历式开场——面试官要的不是编年史，是你和这份工作的关系。记住结构：现在的角色→过去的积累→未来想做什么，60 秒讲完。", Next: NodeRetry},
					{Label: "I'm a product designer with three years in e-commerce.", Reply: "面试官点头记下：\"Great, go on.\" ——一句话立住「现在」：职位＋年限＋领域，三个信息点全带到。这就是『现在-过去-未来』结构的第一步：先让对方知道你是谁，再展开。读出来，这是你的开场白。", Next: 1},
					{Label: "Well... I'm nobody special, just an ordinary employee.", Reply: "面试官礼貌地笑：\"Oh, don't say that.\" 中文语境的谦虚，在英语面试里会被当成字面意思——你说自己 nobody，对方就按 nobody 打分。面试语域里自信陈述事实不算自夸，第一句先立住自己。", Next: NodeRetry},
				},
			},
			{
				Say:     "I'm a product designer with three years in e-commerce.",
				SayOK:   "面试官在纸上写了什么：\"E-commerce, nice—we do a lot of that here.\" ——开场这句从你嘴里出来了，稳。但别松劲，下一个问题往往更扎心：为什么离开现在的公司？",
				SayFail: "没对上——放慢，按意群断开：I'm a product designer... with three years... in e-commerce。信息点咬清楚，机器和面试官都听得懂。",
				SayNext: 2,
			},
			{
				Prompt: "追问来了：\"Why are you leaving your current job?\" ——不抱怨前司，三句挑一句：",
				Options: []nodeOption{
					{Label: "My boss is terrible and the company is a total mess.", Reply: "面试官面无表情记了一笔：\"I see.\" 抱怨前司是离职题的头号雷：对方听到的不是前司多糟，而是你将来也会这样说他们。离职原因的黄金策略是向前看——谈你要去哪，不谈你在逃什么。", Next: NodeRetry},
					{Label: "Honestly, I just couldn't handle the pressure there.", Reply: "面试官眉头动了一下：\"Hmm, this role can be intense too.\" 把离职归因于「扛不住」，等于当场给自己贴上抗压差的标签——真诚不等于自我拆台。谈成长空间，不谈承受极限。", Next: NodeRetry},
					{Label: "I've grown a lot there, and I'm ready for a new challenge.", Reply: "面试官追问的神情松了：\"That makes sense.\" ——先肯定前司（I've grown a lot），再向前看（ready for a new challenge），一句话既真诚又体面，谁都不用踩。这是离职题的标准解法，读出来。", Next: 3},
				},
			},
			{
				Say:     "I've grown a lot there, and I'm ready for a new challenge.",
				SayOK:   "面试官合上你的简历笑了：\"Good answer.\" ——最难的离职题你答得向前看、不带刺。开场立住了，但真正的深水区还在后面：优点和缺点，怎么讲才不掉进陷阱？",
				SayFail: "别急——分两段读：I've grown a lot there... and I'm ready for a new challenge。grown 和 challenge 两个词咬准。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "自我介绍从出生年月讲起", Reply: "编年史式开场是中式简历腔——面试官要的是你和岗位的关系。再想想。"},
			{Label: "离职原因如实吐槽前司", Reply: "对方听到的不是前司多糟，而是你将来也会这样说他们。再选。"},
			{Label: "按现在-过去-未来讲60秒", Reply: "对——先立住现在的角色，再讲过去积累和未来方向，60 秒结构清晰不跑题。"},
		},
		Correct: 2,
		Clear:   "开场60秒立住了，离职题也答得体面。但下一题九成面试都会问：「你最大的缺点是什么」——说「我太追求完美」的人，已经输了。",
		Note:    "60秒立住了自己",
	},
	"strengths-weaknesses": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句答缺点题）",
				Options: []nodeOption{
					{Label: "I'm often late and I can't really focus on boring tasks.", Reply: "面试官挑了下眉：\"...Noted.\" 真诚过头成了自爆——迟到、无法专注都是岗位核心能力的硬伤。缺点题的选材原则：挑一个真实但不致命、且你正在改进的点，真诚和策略不冲突。", Next: NodeRetry},
					{Label: "My biggest weakness is that I'm a perfectionist.", Reply: "面试官在心里叹了口气：\"Right, everyone says that.\" 「我太追求完美」是面试圈最烂大街的假缺点——对方一天能听八遍，只会觉得你在背模板。缺点题考的不是缺点本身，是你的自我认知。", Next: NodeRetry},
					{Label: "I used to say yes to everything, so now I set priorities.", Reply: "面试官身体前倾了：\"Interesting—tell me more.\" ——这句的结构是缺点题的满分模板：used to（真实的过去弱点）＋ so now（正在做的改进）。弱点一旦配上行动，就成了成长故事。读出来。", Next: 1},
				},
			},
			{
				Say:     "I used to say yes to everything, so now I set priorities.",
				SayOK:   "面试官点头：\"That's a very self-aware answer.\" ——弱点从你嘴里说出来，反而加了分，这就是结构的力量。别松口气，镜像问题马上到：你最大的优点呢？",
				SayFail: "再来——句子有个转折轴：I used to say yes to everything... so now I set priorities。前后对比读出来，故事感就有了。",
				SayNext: 2,
			},
			{
				Prompt: "紧接着：\"And your greatest strength?\" ——空喊口号不算数，三句挑一句：",
				Options: []nodeOption{
					{Label: "I'd say adaptability—I picked up our new tools in a week.", Reply: "面试官记下了：\"A week? Impressive.\" ——优点题的公式：一个词点题（adaptability）＋一个 30 秒能讲完的小例子。口号谁都会喊，例子才是证据。这句读出来。", Next: 3},
					{Label: "I'm hardworking, responsible, and a fast learner.", Reply: "面试官礼貌点头，眼神已经飘了：\"Mm-hm.\" 三个形容词连报是典型的口号式回答——没有例子支撑的优点，一个都不算数。挑一个优点，配一个具体的小故事，比堆十个词有力。", Next: NodeRetry},
					{Label: "Everything, honestly. I'm the best hire you'll ever meet.", Reply: "面试官笑了，但不是欣赏的那种：\"Confident, aren't we.\" 英语面试欢迎自信、警惕自大——best ever 这种最高级一出口，对方立刻想找反例。陈述事实＋证据，比宣言更有说服力。", Next: NodeRetry},
				},
			},
			{
				Say:     "I'd say adaptability—I picked up our new tools in a week.",
				SayOK:   "面试官在评估表上勾了一格：\"Good example.\" ——缺点讲成成长，优点配上证据，这对镜像题你都接住了。但光说「我能行」还不够，面试官马上要听一个完整的故事。",
				SayFail: "慢一点——I'd say adaptability 先停半拍，再讲例子：I picked up our new tools in a week。两段分开，各自清楚。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "缺点配上正在做的改进", Reply: "对——used to + so now 结构：真实弱点＋改进行动，弱点就成了成长故事。"},
			{Label: "缺点答我太追求完美", Reply: "面试圈最烂大街的假缺点，对方一天听八遍，只会觉得你在背模板。再选。"},
			{Label: "优点连报三个形容词", Reply: "没有例子支撑的优点一个都不算数——一个词＋一个小例子才是证据。再想想。"},
		},
		Correct: 0,
		Clear:   "缺点讲成了成长故事，优点配上了证据。下一关面试官会说「讲一个你解决难题的经历」——没有结构的故事会散成流水账，STAR 四步撑得住。",
		Note:    "弱点讲成了成长故事",
	},
	"star-story": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句开讲你的故事）",
				Options: []nodeOption{
					{Label: "Last year our top client almost left, so I stepped in.", Reply: "面试官放下笔听你讲：\"Go on.\" ——好故事的第一句是 Situation：时间＋场景＋冲突，一句话把人拉进现场。这正是 STAR 四步（情境-任务-行动-结果）的开头，后面每一步都有地方放。读出来。", Next: 1},
					{Label: "I have solved many difficult problems in my last company.", Reply: "面试官等了两秒，发现你讲完了：\"...For example?\" 题目问的是 a time——一次具体经历，不是业绩总结。「解决过很多问题」在面试官耳朵里约等于一个都讲不出来。挑一件事，讲进细节。", Next: NodeRetry},
					{Label: "That time our project meet a big trouble, I very worry.", Reply: "面试官努力跟上：\"Sorry—when was this?\" 三个硬伤：讲过去的事动词要过去式 met；trouble 不可数，要说 a big problem；I very worry 是中式直译，英语说 I was really worried。故事再好，语法先绊倒了。", Next: NodeRetry},
				},
			},
			{
				Say:     "Last year our top client almost left, so I stepped in.",
				SayOK:   "面试官身体前倾：\"Almost left? What happened?\" ——一句 Situation 就钓住了面试官，这是好故事的证明。接着讲行动和结果，但小心，讲完还有一个反杀问题等着你。",
				SayFail: "重来一次——重音放在 almost left 和 stepped in 上：危机和你的登场，故事的两个支点。",
				SayNext: 2,
			},
			{
				Prompt: "你讲完了结果，面试官追问：\"What would you do differently next time?\" ——三句挑一句：",
				Options: []nodeOption{
					{Label: "Nothing, really. I think I handled it perfectly.", Reply: "面试官在表格上写了个词，八成不是好词：\"Perfectly. OK.\" 拒绝复盘等于告诉对方你的天花板就在这——这题考的不是当年对错，是你有没有反思的习惯。成长型的人永远答得出 differently。", Next: NodeRetry},
					{Label: "I'd loop in the team earlier instead of fixing it alone.", Reply: "面试官点头认可：\"That's a mature take.\" ——承认一个可改进点（loop in the team earlier），且不推翻已有成果：反思题的标准姿势。故事加分而不露怯，就靠这一句。读出来。", Next: 3},
					{Label: "Honestly, I made a mess. I'd probably fail again.", Reply: "面试官同情地笑了笑：\"Oh, I'm sure it wasn't that bad.\" 反思过头成了自我否定——made a mess、fail again 把刚讲完的成就整个拆掉了。复盘的分寸：改进一个环节，保住整个故事。", Next: NodeRetry},
				},
			},
			{
				Say:     "I'd loop in the team earlier instead of fixing it alone.",
				SayOK:   "面试官合上笔记本：\"Great story, thanks for walking me through it.\" ——STAR 讲成就、反思不露怯，这个故事你从头立到了尾。下一题换方向了：他们要问你为什么想来。",
				SayFail: "分三段读：I'd loop in the team earlier... instead of... fixing it alone。loop in 是整个句子的钥匙，咬清楚。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "答我解决过很多问题", Reply: "a time 要的是一次具体经历——「很多」在面试官耳朵里等于一个都讲不出。再选。"},
			{Label: "按STAR四步讲一次经历", Reply: "对——情境、任务、行动、结果，四步一走，流水账就成了有支点的故事。"},
			{Label: "被问改进时答毫无遗憾", Reply: "拒绝复盘等于告诉对方你的天花板就在这。反思一个环节，保住整个故事。再想想。"},
		},
		Correct: 1,
		Clear:   "STAR 四步讲住了一个完整故事，追问也接得漂亮。下一题看似最软其实最险：「为什么选我们」——吹捧和空话都会被一句追问戳穿。",
		Note:    "一个故事讲出四个支点",
	},
	"why-us": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句答动机题）",
				Options: []nodeOption{
					{Label: "Because your company is famous and the salary is good.", Reply: "面试官笑容不变，兴趣归零：\"Thanks for being honest.\" famous 和 salary 是放在哪家公司都成立的理由——等于没有理由。动机题的解法：一个只属于这家公司的点，加一个只属于你的点。", Next: NodeRetry},
					{Label: "I've used your app for years, and this role fits my skills.", Reply: "面试官眼睛亮了：\"Oh, you're a user?\" ——两个理由各占一半：关于公司的（真用过产品）＋关于自己的（岗位和能力匹配），具体、真诚、不谄媚。动机题的满分结构就长这样。读出来。", Next: 1},
					{Label: "I'll take any job here. Please just give me a chance.", Reply: "面试官的笔停住了：\"...Any job?\" 恳求式回答是动机题的反面教材——any job 说明你没想清楚要什么，please give me a chance 把姿态压到了地板上。面试是双向选择，你也在挑他们。", Next: NodeRetry},
				},
			},
			{
				Say:     "I've used your app for years, and this role fits my skills.",
				SayOK:   "面试官来了兴致：\"Which feature do you use most?\" ——真用户的理由一开口就有后续话题，这就是具体的力量。不过对方还有一手：他们会质疑这个理由不够独特。",
				SayFail: "再读一次——两个理由中间有个 and，前后各自完整：I've used your app for years... and this role fits my skills。",
				SayNext: 2,
			},
			{
				Prompt: "追问来了：\"You could get that at other companies too, no?\" ——三句挑一句接住：",
				Options: []nodeOption{
					{Label: "Few companies let designers talk to users weekly—you do.", Reply: "面试官笑着摊手认了：\"Fair point.\" ——接住质疑的关键是给出排他性理由：Few companies... you do，一句话把「哪都一样」变成「只有你家」。做过功课的人才答得出这种细节。读出来。", Next: 3},
					{Label: "Yeah, you're right. I'm applying everywhere, actually.", Reply: "面试官笑了，你的动机分归了零：\"At least you're honest.\" 诚实值得尊重，但把底牌全掀等于告诉对方：你对他们没有特别的兴趣。真诚和交底是两回事，守住你研究过这家公司的证据。", Next: NodeRetry},
					{Label: "No! Your company is the best company in the world!", Reply: "面试官被逗笑了：\"The whole world? That's generous.\" 空洞的最高级撑不住任何追问——best in the world 后面只要再问一句 why，就会当场垮掉。吹捧不是热情，细节才是。", Next: NodeRetry},
				},
			},
			{
				Say:     "Few companies let designers talk to users weekly—you do.",
				SayOK:   "面试官在你名字旁边画了颗星：\"You've done your homework.\" ——动机题连追问都接住了，真诚又有备而来。接下来是所有人最紧张的一题：谈钱。",
				SayFail: "这句有个对比结构：Few companies... you do。前半句压低，最后 you do 抬起来，反差读出来就成了。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "夸对方是世界最好公司", Reply: "空洞最高级撑不住一句追问——吹捧不是热情，细节才是。再选。"},
			{Label: "坦白说我到处都在投", Reply: "诚实值得尊重，但把底牌全掀等于宣布你对他们没有特别兴趣。再想想。"},
			{Label: "一条公司理由一条自己的", Reply: "对——只属于这家公司的点＋只属于你的点，具体真诚不谄媚，追问也接得住。"},
		},
		Correct: 2,
		Clear:   "动机题答得真诚不谄媚，追问也被你一句排他性理由接住。下一关是最多人吃亏的一题：谈薪资——先报数的人，往往先出局。",
		Note:    "动机答得真诚不谄媚",
	},
	"salary-talk": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句接住薪资题）",
				Options: []nodeOption{
					{Label: "May I ask the budgeted range for this role?", Reply: "HR 顿了一下，笑了：\"Sure, let me pull that up.\" ——谈薪第一定律：先报数的人先暴露底牌。May I ask the budgeted range for this role? 一句礼貌的反问，把信息差扳回你这边。读出来。", Next: 1},
					{Label: "My expected salary is depend on your company.", Reply: "HR 愣了半秒：\"...Depends on what exactly?\" 两个问题：is depend 双动词撞车，要么 depends on 要么 is dependent on；内容上「看公司给」等于把定价权全交出去。语法和策略，这句一个都没站住。", Next: NodeRetry},
					{Label: "Anything is fine. I'm not really in it for the money.", Reply: "HR 眼睛一亮——那是替公司省钱的亮：\"Good to know!\" 「多少都行」在谈判桌上只有一个效果：拿到区间的最低值。不谈钱显得清高，会谈钱才是职业。你的报价没人会替你争取。", Next: NodeRetry},
				},
			},
			{
				Say:     "May I ask the budgeted range for this role?",
				SayOK:   "HR 报出了区间：\"We're looking at 15 to 20k.\" ——一句反问，你还没出价就拿到了对方的底牌。但别高兴太早，对方的第一个数字通常贴着区间下限来。",
				SayFail: "放慢——May I ask... the budgeted range... for this role? 疑问句尾音轻轻上扬，礼貌就到位了。",
				SayNext: 2,
			},
			{
				Prompt: "果然，HR 压着下限报价：\"We were thinking around 15k.\" ——低于你的预期，三句挑一句往上谈：",
				Options: []nodeOption{
					{Label: "OK, 15k is fine. I don't want to cause any trouble.", Reply: "HR 愉快地合上了电脑：\"Great, deal!\" 一句 fine，几万块一年就没了——谈判桌上退让不会换来好感，只会换来成交。要求合理的数字不是 trouble，是职业素养。", Next: NodeRetry},
					{Label: "15k? That's insulting. I make more than that now.", Reply: "HR 的笑容冷了下来：\"That's our range, I'm afraid.\" insulting 一出口，谈判就成了对抗——数字可以争，情绪不能上桌。不满意报价时，用理由抬价，不用形容词砸人。", Next: NodeRetry},
					{Label: "Based on my experience, I was hoping for closer to 18k.", Reply: "HR 翻了翻你的简历点头：\"Let me check with the team.\" ——Based on my experience 把抬价锚在了理由上，closer to 18k 给出明确数字又留了弹性：不吃亏也不失礼，一句全齐。读出来。", Next: 3},
				},
			},
			{
				Say:     "Based on my experience, I was hoping for closer to 18k.",
				SayOK:   "HR 记下了 18k：\"I'll see what we can do.\" ——你没掀桌也没躺平，把数字体面地顶了上去。钱谈完，面试只剩最后一分钟——很多人就败在那一分钟。",
				SayFail: "两段读：Based on my experience 先立理由，停半拍，再报数：I was hoping for closer to 18k。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "先礼貌反问岗位预算区间", Reply: "对——先报数的人先暴露底牌，一句 May I ask the range 把信息差扳回来。"},
			{Label: "答多少都行显得好相处", Reply: "「多少都行」在谈判桌上只有一个结果：拿到区间最低值。再选。"},
			{Label: "嫌报价低就直说太侮辱人", Reply: "数字可以争，情绪不能上桌——用理由抬价，不用形容词砸人。再想想。"},
		},
		Correct: 0,
		Clear:   "该问的问了，该顶的顶了，钱谈得不吃亏也不失礼。下一关是面试最后一分钟：「你有什么问题问我吗」——答 No 的人，前面全白答了。",
		Note:    "谈钱没吃亏也没失礼",
	},
	"ask-interviewer": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句反问）",
				Options: []nodeOption{
					{Label: "No, I think you've covered everything. Thanks.", Reply: "面试官合上本子，面试提前结束了：\"Alright then.\" No 是这道题唯一的错误答案——反问环节是你最后一次展示思考质量的机会，放弃提问等于告诉对方：我对这份工作没那么好奇。", Next: NodeRetry},
					{Label: "What does success look like in this role after six months?", Reply: "面试官来了精神：\"Great question.\" ——这一问显了三层水平：你在想入职后的事、你关心标准而不只是待遇、你给了对方发挥的空间。What does success look like 是反问环节的黄金句式。读出来。", Next: 1},
					{Label: "How many vacation days do I get, and can I leave early?", Reply: "面试官笑容淡了：\"We can... cover that later.\" 假期和早退本身没错，错在时机——offer 到手前，问题该指向工作本身；福利细节留给 HR 谈 offer 时再问，顺序反了印象分就没了。", Next: NodeRetry},
				},
			},
			{
				Say:     "What does success look like in this role after six months?",
				SayOK:   "面试官认真答了起来：\"Good question—success here means owning projects end to end.\" ——一个问题把面试官变成了回答者，攻守易位。但高手不止问一层。",
				SayFail: "长句分三段：What does success look like... in this role... after six months? 尾音上扬，问题就活了。",
				SayNext: 2,
			},
			{
				Prompt: "面试官答完看着你。就此打住还是再追一层？三句挑一句收尾：",
				Options: []nodeOption{
					{Label: "That's helpful. How does the team support new hires there?", Reply: "面试官答得更具体了，气氛完全打开：\"We pair every new hire with a mentor.\" ——That's helpful 先接住对方的回答，再顺着往下追一层：听了才问，问了有回应，这是对话不是问答题。读出来。", Next: 3},
					{Label: "OK, got it. So... when do I get the result?", Reply: "面试官公式化地笑了笑：\"We'll be in touch.\" 催结果是收尾最常见的失分动作——显急、显没底气。想表达期待，用 I look forward to hearing from you，体面又不给对方压力。", Next: NodeRetry},
					{Label: "Thank you, I very hope I can working with you.", Reply: "面试官听懂了意思，也听见了破绽。两个硬伤：very 不能直接修饰动词，要说 I really hope；can 后面接动词原形 work，不是 working。收尾句是对方记住的最后一句，语法值得磨到无懈可击。", Next: NodeRetry},
				},
			},
			{
				Say:     "That's helpful. How does the team support new hires there?",
				SayOK:   "面试官起身和你握手：\"It's been a real pleasure talking to you.\" ——从开场到反问，你把最后一分钟也变成了加分项。只剩一件事了：把这六关串成一整场，真刀真枪走一遍。",
				SayFail: "先短后长：That's helpful 落定，停一拍，再问：How does the team support new hires there?",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "答No表示都听明白了", Reply: "No 是这题唯一的错误答案——放弃提问等于放弃最后一次展示思考的机会。再选。"},
			{Label: "问这岗位半年后的成功标准", Reply: "对——问成功标准显思考质量，比问福利更能加分，还给对方留了发挥空间。"},
			{Label: "先问清年假和加班安排", Reply: "福利细节留给拿 offer 后再谈——现在问，顺序反了印象分就没了。再想想。"},
		},
		Correct: 1,
		Clear:   "三个问题问出了水平，最后一分钟也在加分。下一关是终点：全英模拟面——开场、压力题、反问收尾一场连走，前六关的句子全都用得上。",
		Note:    "反问问出了水平",
	},
	"mock-full-interview": {
		Nodes: []scriptNode{
			{
				Prompt: "深呼吸。面试官推门坐下：\"Thanks for coming in. Let's start—tell me about yourself.\" ——三句挑一句开场：",
				Options: []nodeOption{
					{Label: "I'm a product designer with three years in e-commerce.", Reply: "面试官点头示意你继续：\"Go on.\" ——第一关练的开场白在真枪实弹的场子里照样立得住：现在的角色一句话说清，稳稳的开局。读出来，这次是整场的第一句。", Next: 1},
					{Label: "I was born in a small town and I like play basketball.", Reply: "面试官礼貌地打断了你：\"Let's focus on your work experience.\" 两个问题：like 后面要接 playing 或 to play；内容上出生地和篮球跟岗位无关——60 秒很贵，每句话都要花在你和工作的关系上。", Next: NodeRetry},
					{Label: "Um, sorry, my English is poor, I will try my best.", Reply: "面试官温和地摆手：\"Take your time, you're doing fine.\" 用道歉开场等于让对方从第一秒开始扣分——你前六关练出来的口语根本不 poor，别替自己降价。深呼吸，直接开讲。", Next: NodeRetry},
				},
			},
			{
				Say:     "I'm a product designer with three years in e-commerce.",
				SayOK:   "面试官记下第一笔：\"Three years, good.\" ——整场面试的第一句稳稳落地。热身结束，压力题上桌了。",
				SayFail: "开场白你早就会了——放慢，找回第一关的节奏：I'm a product designer... with three years... in e-commerce。",
				SayNext: 2,
			},
			{
				Prompt: "面试官放下简历盯着你：\"Your resume looks fine, but why should we hire you over the others?\" ——压力题，三句挑一句：",
				Options: []nodeOption{
					{Label: "Because other candidates are probably worse than me.", Reply: "面试官眉毛抬了起来：\"That's quite a claim.\" 踩别人抬自己是压力题的经典陷阱——你根本不了解其他候选人，这句只暴露傲慢。比较题的解法：不比人，摆自己的证据。", Next: NodeRetry},
					{Label: "I'm not sure, to be honest. Maybe you shouldn't.", Reply: "面试官等着你的下文，但你没有下文：\"...OK.\" 压力题的目的就是看你会不会自乱阵脚——这句等于当场缴械。谦虚在这里不是美德，你练过 STAR，你有证据，把它说出来。", Next: NodeRetry},
					{Label: "Last year I kept our top client—I can do that here too.", Reply: "面试官身体前倾了：\"Tell me more about that.\" ——一句话两个支点：过去的硬结果（kept our top client）＋对未来的迁移（here too）。这就是压力题的正解：不比别人，用事实说自己。读出来。", Next: 3},
				},
			},
			{
				Say:     "Last year I kept our top client—I can do that here too.",
				SayOK:   "面试官在评估表上重重画了一笔：\"Impressive.\" ——压力最大的一题被你用事实压了回去。面试进入尾声，最后一分钟到了。",
				SayFail: "破折号是转折点：Last year I kept our top client... 停一拍... I can do that here too。前讲战绩，后接承诺。",
				SayNext: 4,
			},
			{
				Prompt: "面试官看了看表，微笑：\"Well, that's all from me. Any questions before we wrap up?\" ——最后一分钟，三句挑一句：",
				Options: []nodeOption{
					{Label: "No questions. Can I go now?", Reply: "面试官愣了一下才笑出来：\"...Sure, I suppose.\" 都走到最后一分钟了——你在第六关刚学过：No 是这题唯一的错误答案，Can I go now 更是把体面直接丢在了门口。再选。", Next: NodeRetry},
					{Label: "Just one—what would my first month here look like?", Reply: "面试官认真想了想才回答：\"Week one is onboarding, then you'd own a small project.\" ——问入职后的第一个月，说明你已经在想象把工作干起来的样子。最后一分钟还在加分。读出来。", Next: 5},
					{Label: "Yes. How much money you will give me every month?", Reply: "面试官笑容僵了半秒。两个问题：疑问句语序要倒装，How much will you give me，不是 how much you will give；而且收尾一分钟把话题拽回钱上，显得整场只关心待遇——你在谈薪关练过体面得多的问法。", Next: NodeRetry},
				},
			},
			{
				Say:     "Just one—what would my first month here look like?",
				SayOK:   "面试官起身伸出手，握得很实：\"It was a real pleasure. You'll hear from us soon.\" ——开场、压力题、反问收尾，一整场全英面试从你嘴里完整走了下来。走出这扇门，你已经不是28关前的那个你了。",
				SayFail: "收尾句放松读：Just one 先落定，再问：what would my first month here look like? 尾音上扬。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "开场先为英语不好道歉", Reply: "道歉开场等于让对方从第一秒开始扣分——你的口语不 poor，别替自己降价。再选。"},
			{Label: "压力题答我比别人都强", Reply: "踩别人抬自己只暴露傲慢——不比人，摆自己的证据。再想想。"},
			{Label: "压力题用真实战绩接住", Reply: "对——过去的硬结果＋对未来的迁移，一句事实胜过十句宣言。整场面试的脊梁就是它。"},
		},
		Correct: 2,
		Clear:   "一整场全英面试走完了——28 个场景的句子，每一句都从你嘴里真实地出来过：不是看会的，是说会的。真场景来的那天，你张得开口。常回复习挑战里走走，把「点亮」磨成「掌握」。开口这门课，毕业快乐。",
		Note:    "全英面试完整走了一场",
	},
	// ============ 生活（续） ============
	"restaurant-order": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回服务生）",
				Options: []nodeOption{
					{Label: "Might I trouble you to recommend your most exquisite dishes?", Reply: "服务生眉毛一挑，用同样的腔调回敬：\"Certainly, sir. Might I suggest our chef's tasting menu?\" ——他给你推了全店最贵的一套。Might I trouble you、exquisite 是宫廷剧台词，日常点餐说 What would you recommend 就好；语域抬太高，价格也跟着抬。", Next: NodeRetry},
					{Label: "A lemonade, please. What do you recommend? Nothing spicy.", Reply: "服务生笑着记下：\"Good choice—our grilled fish is great, and it's not spicy at all.\" ——饮品、请推荐、忌口，三件事一口气说清；What do you recommend 是菜单看不懂时的救命句。现在把它变成你自己的。", Next: 1},
					{Label: "I want a lemonade. What is delicious here? I can't eat hot.", Reply: "服务生迟疑了一下：\"Hot? You mean... temperature?\" ——hot 在英语里既是烫又是辣，说忌口得用 spicy；「什么好吃」直译成 what is delicious 也很生硬，地道问法是 What's good here 或 What do you recommend。", Next: NodeRetry},
				},
			},
			{
				Say:     "A lemonade, please. What do you recommend? Nothing spicy.",
				SayOK:   "服务生朝厨房报了单：\"One lemonade, nothing spicy—you got it!\" ——三个要点一句带走，这句已经是你的了。先别放松，牛排端上来的时候有点不对劲。",
				SayFail: "没对上——放慢，按意群断开：A lemonade, please... What do you recommend... Nothing spicy。一段一段来，机器和人都听得清。",
				SayNext: 2,
			},
			{
				Prompt: "牛排端上来，切开一看还渗着血水——太生了。服务生正好路过：\"How is everything?\" ——礼貌地请他拿回去再煎熟一点，三句挑一句：",
				Options: []nodeOption{
					{Label: "This steak is raw. Take it back.", Reply: "服务生僵在原地，邻桌都看了过来——\"...Right away.\" 盘子是收走了，气氛也冷了。raw 是全生才用的词，牛排偏生说 undercooked 或 too rare；更要命的是没有缓冲直接祈使——先来一句 Sorry 或 Excuse me，问题才好办。", Next: NodeRetry},
					{Label: "Sorry, this steak is too tender. Please burn it more.", Reply: "服务生一脸迷惑：\"Too... tender? And you want it burned?\" ——tender 是「嫩得恰到好处」，是夸厨师的词；burn 是烧焦。你想说的「太生、再煎一下」是 undercooked 和 cook it a bit more，两个假朋友一换，意思全反了。", Next: NodeRetry},
					{Label: "Sorry, my steak is undercooked—could you cook it a bit more?", Reply: "服务生连声道歉端起盘子：\"Oh, I'm so sorry—I'll get that fixed right away.\" ——Sorry 缓冲、undercooked 说清问题、could you 提出请求：一句话三步走，事办成了，体面也在。读出来。", Next: 3},
				},
			},
			{
				Say:     "Sorry, my steak is undercooked—could you cook it a bit more?",
				SayOK:   "牛排重新端上来，火候正好，服务生还补了句：\"Enjoy!\" ——从点单到把问题送回后厨，这一餐你全程自己扛下来了。",
				SayFail: "再来一遍——重音放在 undercooked 上，后半句 could you cook it a bit more 连起来说顺，别一个词一个词蹦。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "直接说 Take it back", Reply: "raw 用错还没缓冲，全场看你——牛排是回去了，饭是吃不香了。再选一个。"},
			{Label: "说牛排 too tender 请回锅", Reply: "tender 是嫩得好吃，这句等于先夸厨师再让他改——他只会更迷惑。再想想。"},
			{Label: "Sorry 开头说没煎熟", Reply: "对——Sorry 缓冲、undercooked 点明问题、could you 提请求，三步一句话，体面又管用。"},
		},
		Correct: 2,
		Clear:   "点了菜、护住忌口，太生的牛排也体面地送回了后厨——一顿饭全程自己扛。下一关「初次寒暄」：聚会上有人伸手说没见过你，三句话内让对话活起来。",
		Note:    "开口点了菜还守住忌口",
	},
	"first-smalltalk": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句接住这只伸过来的手）",
				Options: []nodeOption{
					{Label: "Hello, my name is Lin. I am 28 and my job is engineer.", Reply: "对方的笑容礼貌地凝固了半秒：\"Oh... nice.\" ——两个问题：my job is engineer 缺冠词，该说 I'm an engineer；更大的问题是寒暄一上来自报年龄职业，像在念简历。聚会破冰只要名字，加一个抛回去的话头。", Next: NodeRetry},
					{Label: "Oh, sorry... my English is not very good.", Reply: "对方赶紧安慰：\"No no, you're doing great!\" ——话题瞬间从「认识你」变成「安慰你」。自贬开场是中文式谦虚的假朋友，英语寒暄里它只会让对方不知道接什么，破冰变成了救场。", Next: NodeRetry},
					{Label: "Hi, I'm Lin. Nice to meet you—how do you know the host?", Reply: "对方眼睛一亮：\"Oh, we went to college together!\" ——名字、一句客套、一个问题，破冰三件套齐了。把话题抛回去，是让对话活下来的第一原则。读出来。", Next: 1},
				},
			},
			{
				Say:     "Hi, I'm Lin. Nice to meet you—how do you know the host?",
				SayOK:   "对方接得飞快：\"We go way back! And you?\" ——你一句话就把球传了出去，对话转起来了。别走神，对方马上要报家门。",
				SayFail: "再试一次——Hi, I'm Lin 说完停半拍，后半句 how do you know the host 整句连出去，节奏就对了。",
				SayNext: 2,
			},
			{
				Prompt: "对方握完手自报家门：\"I'm Sam, by the way. I'm from Melbourne.\" ——别让话掉在地上，用一个跟进问题让对话继续：",
				Options: []nodeOption{
					{Label: "Oh, cool! What brought you here from Melbourne?", Reply: "Sam 打开了话匣子：\"Work, actually—but I stayed for the food!\" ——What brought you here 是万能跟进问题：接住对方给的信息，再把它变成下一个话题。读出来。", Next: 3},
					{Label: "Oh. Melbourne. Good.", Reply: "Sam 等了两秒，没等到下文：\"...Yeah.\" ——三个句号，三次终结。信息接住了却没回球，对话在你手里安静地死掉了。跟进问题哪怕只有一个 What's it like，也比 Good 强。", Next: NodeRetry},
					{Label: "Melbourne? I have been to there last year.", Reply: "Sam 听懂了，但你自己卡了一下——been to there 里 to 和 there 撞车，只能说 been there；而且 last year 是明确的过去时间，要用过去式 I went there last year，现在完成时不跟具体时间点。", Next: NodeRetry},
				},
			},
			{
				Say:     "Oh, cool! What brought you here from Melbourne?",
				SayOK:   "你和 Sam 从墨尔本聊到了咖啡，中途没有一次冷场——寒暄这关，你是靠问题赢下来的。",
				SayFail: "放慢——Oh, cool 先出口带上语气，后面 What brought you here from Melbourne 按 brought、here、Melbourne 三个重音走。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "问TA怎么来这儿的", Reply: "对——What brought you here 把对方的信息变成下一个话题，对话永远有球可接。"},
			{Label: "Oh, good. 然后点头", Reply: "句点式回应是聊天死刑：信息接住了不回球，冷场三秒就到。再选一个。"},
			{Label: "说 been to there", Reply: "to 和 there 撞车，last year 也该配过去式 went——语法先绊住了自己。再想想。"},
		},
		Correct: 0,
		Clear:   "自我介绍出了手、话题抛了回去，整场没有一次冷场。下一关「买衣服」：看中的外套码数不对，开口之前先分清 change 和 exchange。",
		Note:    "接住了递过来的话头",
	},
	"shopping-clothes": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回店员）",
				Options: []nodeOption{
					{Label: "Do you have this jacket in a medium? I'd like to try it on.", Reply: "店员利落转身：\"Sure! The fitting room is right over there.\" ——Do you have this in a medium 是买衣服的万能句型，颜色尺码全能套；I'd like to 提请求，礼貌又干脆。读出来。", Next: 1},
					{Label: "This clothes is nice. Have M size? I can try?", Reply: "店员听懂了，但句句硌耳朵——clothes 永远是复数，单件外套说 this jacket；Have M size 缺了主语和冠词，完整问法是 Do you have this in a medium；I can try? 也得倒装成 Can I try it on。", Next: NodeRetry},
					{Label: "Would you be so kind as to provide this garment in medium?", Reply: "店员愣了一下才反应过来：\"...Of course.\" ——garment 是吊牌和合同上的词，provide 像在下采购单；买衣服的日常语域就是 Do you have this in a medium。客气抬到书面语，反而像隔着柜台念公文。", Next: NodeRetry},
				},
			},
			{
				Say:     "Do you have this jacket in a medium? I'd like to try it on.",
				SayOK:   "店员把中码递过来：\"Here you go—let me know how it fits!\" ——句型到手，衣服也到手了。先去试衣间，尺码的事还没完。",
				SayFail: "别急——jacket 和 medium 两个词咬清楚，两句中间换口气：Do you have this jacket in a medium... I'd like to try it on。",
				SayNext: 2,
			},
			{
				Prompt: "试衣间出来，袖子长出一截——大了一号。店员迎上来：\"How does it fit?\" ——一次问清两件事：有没有小一码、不合适能不能退换：",
				Options: []nodeOption{
					{Label: "It's too big. Go get me a smaller one.", Reply: "店员脸上的笑淡了：\"...I'll check.\" ——Go get me 是使唤人的句式，配上 too big 的抱怨腔，你从客人变成了甲方。指使和请求之间隔着一个 Could you，这个距离决定服务的温度。", Next: NodeRetry},
					{Label: "It's a bit big—do you have a smaller size? Can I return it?", Reply: "店员答得干脆：\"We have a small—and yes, fourteen-day returns with the receipt.\" ——两个问题一口气问全，信息一次拿齐，这才是高效的开口。读出来。", Next: 3},
					{Label: "A little big. Can you change a small one for me?", Reply: "店员会意了，但 change 用岔了——英语里 change 是找零钱、换衣服（change clothes），换尺码换货要用 exchange，或者直接问 do you have a smaller size。中文的「换」一词多义，英语拆成了两个词。", Next: NodeRetry},
				},
			},
			{
				Say:     "It's a bit big—do you have a smaller size? Can I return it?",
				SayOK:   "小码合身，小票攥在手里，退换政策也门儿清——这件外套买得明明白白，一分冤枉钱没花。",
				SayFail: "再来——两个问题中间停一拍：do you have a smaller size，停，Can I return it。问句尾音上扬，机器一听就懂。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "Go get me 拿小码", Reply: "命令式开路，店员心里已经翻了页——请求要走 Could you。再选一个。"},
			{Label: "问小码顺带问退换", Reply: "对——两件事并成一次开口，尺码和退路一起拿到手，买得不慌。"},
			{Label: "说 change a small one", Reply: "change 是找零和换装，换码要用 exchange——一词之差意思全跑。再想想。"},
		},
		Correct: 1,
		Clear:   "尺码问到了、退换问清了，这件外套买得明明白白。下一关「问路指路」：陌生街头拦下一位语速飞快的路人，你能接住几成？",
		Note:    "开口换到了合身尺码",
	},
	"ask-directions": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句开口拦人）",
				Options: []nodeOption{
					{Label: "Excuse me, where is subway? How long I will walk?", Reply: "路人听懂了大概，但两处硌了一下——subway 前面缺了 the；How long I will walk 疑问句忘了倒装，该是 How long is the walk 或 How long does it take。小词和语序，是问路句最容易掉的两颗螺丝。", Next: NodeRetry},
					{Label: "Excuse me, how do I get to the subway? How long is the walk?", Reply: "路人停下脚步：\"Oh sure—it's about ten minutes that way.\" ——Excuse me 是搭话的入场券，how do I get to 是问路的万能句型，末尾再确认路程，一句不多一句不少。读出来。", Next: 1},
					{Label: "Hey, tell me where the subway is.", Reply: "路人皱了下眉，脚步没停：\"Uh... that way?\" ——Hey 加祈使句 tell me，对陌生人像开口盘问。向陌生人求助，开场白只有一个标准答案：Excuse me。少了它，对方给个大概方向就想走。", Next: NodeRetry},
				},
			},
			{
				Say:     "Excuse me, how do I get to the subway? How long is the walk?",
				SayOK:   "路人认真起来，转身开始指路：\"OK, so you go down this street—\" ——问对了，人就愿意帮你。但接下来这段指路，语速有点超纲。",
				SayFail: "慢慢来——Excuse me 先出口停半拍，再问 how do I get to the subway，最后 How long is the walk。三段式，不赶。",
				SayNext: 2,
			},
			{
				Prompt: "路人热心开讲，语速却快得像贯口：\"Go down two blocks, turn left, and then it's—\" 后半句全糊了，你只抓住一个 left。请 TA 说慢一点，并把听到的路线复述确认：",
				Options: []nodeOption{
					{Label: "Sorry, could you say it more slowly? Two blocks, then left?", Reply: "路人放慢重来：\"Right—two blocks, turn left, and it's on your right.\" ——请对方减速，加上复述确认，听力缺口当场补上。听不清就问，本来就是听懂的一部分。读出来。", Next: 3},
					{Label: "Sorry, I can't catch you. Say again, please.", Reply: "路人愣了一下——catch you 听起来像「抓不住你这个人」，没听清要说 I didn't catch that；Say again 缺了宾语也少了缓冲，完整版是 Could you say that again。差两个小词，礼貌和意思才都齐。", Next: NodeRetry},
					{Label: "OK, OK, I got it. Thanks!", Reply: "路人满意地走了，你站在原地，手里还是只有那个 left——装懂是听力最大的敌人：这一句省下的三十秒，等会儿要在岔路口加倍还。没听清就开口确认，没人嫌你慢。", Next: NodeRetry},
				},
			},
			{
				Say:     "Sorry, could you say it more slowly? Two blocks, then left?",
				SayOK:   "路人竖了个大拇指：\"Exactly! You got it.\" ——复述确认一遍，路线钉进了脑子。到地铁站的这十分钟，你不用再问第二个人。",
				SayFail: "断句来——Sorry, could you say it more slowly，停，Two blocks，停，then left。复述的部分一个词一个词咬实。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "笑着说OK其实没听懂", Reply: "装懂省了三十秒，岔路口加倍还——听不清就该开口。再选一个。"},
			{Label: "说 I can't catch you", Reply: "像在说抓不住这个人——没听清是 I didn't catch that。再想想。"},
			{Label: "请TA说慢并复述路线", Reply: "对——减速加复述，两步把听力缺口当场补上，路才真正到手。"},
		},
		Correct: 2,
		Clear:   "路问清了，还复述确认了一遍——不装懂的人不迷路。下一关「电话预约」：一通没有手势可借的电话，时间人数得一次说全。",
		Note:    "接住了飞快的指路",
	},
	"phone-booking": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句开口订位）",
				Options: []nodeOption{
					{Label: "Hello, I want to order a table tonight. We are four persons.", Reply: "接线员反应了半秒：\"You'd like to... book a table?\" ——order 是点菜，订位要用 book 或 reserve；persons 是法律文书里的词，口语说 four people，或者更顺的 a table for four。两个词换掉，这句就地道了。", Next: NodeRetry},
					{Label: "Yeah so, um, tonight, four of us, you got space or what?", Reply: "电话那头顿了顿：\"...For what time exactly, sir?\" ——电话里没有表情和手势兜底，um 和碎片信息只会让对方反复追问；or what 的尾音还带着质问味。电话语域要的是要素齐全、一次说清。", Next: NodeRetry},
					{Label: "Hi, I'd like to book a table for four at seven tonight.", Reply: "接线员马上去查：\"For four at seven—let me check that for you.\" ——I'd like to book a table for four at seven，人数时间一句打包，电话订位的标准开场就是它。读出来。", Next: 1},
				},
			},
			{
				Say:     "Hi, I'd like to book a table for four at seven tonight.",
				SayOK:   "键盘声响了几秒：\"One moment, please...\" ——要素一次说全，对方直接去办事。先别挂，七点的位子可能有变数。",
				SayFail: "电话里更要清楚——book a table 连读，三个信息点放慢：for four... at seven... tonight。慢半拍，字字清。",
				SayNext: 2,
			},
			{
				Prompt: "接线员查完面露难色：\"I'm sorry, seven is fully booked tonight. We have six or eight thirty.\" ——选一个时间，并把名字和电话留下：",
				Options: []nodeOption{
					{Label: "OK, eight thirty then. My name is called Lin.", Reply: "接线员记下了，但那句自我介绍拧着——is called 用在物件和绰号上，自报姓名就是 My name is Lin 或 I'm Lin。「我叫」直译成 is called，是中式英语的老熟人了。", Next: NodeRetry},
					{Label: "Eight thirty works. The name is Lin—I'll leave my number.", Reply: "接线员一路确认：\"Eight thirty, party of four, under Lin—and your number, please?\" ——选定时间、报上名字、主动留电话，订位三要素闭环，这一单跑不了。读出来。", Next: 3},
					{Label: "Oh, either is fine... you decide for me, it's OK.", Reply: "电话那头等着：\"So... which one shall I put down?\" ——客气过了头就是没答复：时间没定、名字电话没留，这通电话挂了等于没打。订位要给确定的答案，礼貌不等于把决定推回去。", Next: NodeRetry},
				},
			},
			{
				Say:     "Eight thirty works. The name is Lin—I'll leave my number.",
				SayOK:   "\"All set—see you at eight thirty, Lin!\" 电话挂了，今晚的位子写上了你的名字——全程没靠一个手势。",
				SayFail: "再来——Eight thirty works 说定，停一拍，The name is Lin，再把 I'll leave my number 连出去。电话腔就是慢而清。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "选8点半并留名电话", Reply: "对——时间、名字、电话三要素闭环，电话订位这才算落袋。"},
			{Label: "说 name is called Lin", Reply: "is called 用于物件绰号，自报姓名是 My name is Lin。再想想。"},
			{Label: "说哪个都行你定吧", Reply: "没答案没联系方式，电话挂了位子还是悬着。再选一个。"},
		},
		Correct: 0,
		Clear:   "时间敲定、名字电话留全，今晚的位子稳了——没有手势可借，你也把话说清了。下一关「看医生」：诊室里，症状得用英语自己讲。",
		Note:    "开口订下了今晚的位子",
	},
	"see-doctor": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回医生）",
				Options: []nodeOption{
					{Label: "I've had a headache for two days, and I have a slight fever.", Reply: "医生边听边记：\"Two days, with a fever—OK, let's take a look.\" ——症状加持续时间，问诊最想要的两样你一句给全；I've had... for two days 这个现在完成时，就是「已经疼了两天」的标准说法。读出来。", Next: 1},
					{Label: "My head is very pain since two days. I have a little hot.", Reply: "医生听懂了七成，但三个词都拧着——pain 是名词，「头疼」说 my head hurts；since 后面跟时间点，「两天了」是 for two days；hot 是烫不是发烧，发烧要说 fever。诊室里，词不准，信息就打折。", Next: NodeRetry},
					{Label: "Um, I don't feel very well... it's nothing serious, I guess.", Reply: "医生的笔停在半空：\"OK... can you be more specific?\" ——诊室不是客气的地方，it's nothing 这种谦虚会把问诊拖成猜谜。医生要的就两样：哪里不舒服、持续多久。把它们直接递过去。", Next: NodeRetry},
				},
			},
			{
				Say:     "I've had a headache for two days, and I have a slight fever.",
				SayOK:   "医生点点头开始检查：\"Alright, let's check your temperature first.\" ——症状说清，看病就成功了一半。等着，医嘱那关才是听力考试。",
				SayFail: "慢一点——I've had a headache 连读，for two days 停一拍，再说 and I have a slight fever。slight 的 s 咬出来。",
				SayNext: 2,
			},
			{
				Prompt: "医生边写处方边交代：\"Take one twice a day after meals.\" ——先复述你听懂了什么，再追问一句：有什么要忌口的吗：",
				Options: []nodeOption{
					{Label: "OK, so I eat two pills one time every day, right?", Reply: "医生赶紧摆手：\"No no—one pill, twice a day.\" ——twice a day 是一天两次、每次一片，不是一次两片。复述恰好把听岔的地方当场暴露了——这正是复述的价值；另外吃药不用 eat，英语说 take medicine。", Next: NodeRetry},
					{Label: "Yes, yes, I understand. Thank you, doctor. Bye!", Reply: "医生想叮嘱的话被你的 Bye 关在了门里——医嘱是全场最不能装懂的一段：没复述、没确认，出了诊室剂量就开始模糊。听完医嘱的标准动作是复述一遍，再把疑问当场问完。", Next: NodeRetry},
					{Label: "Twice a day after meals, right? Any food I should avoid?", Reply: "医生点头，又补了叮嘱：\"Correct. And yes—I'll write it all down for you.\" ——复述确认加主动追问忌口，医嘱这道听力题你拿了满分。剩下的，按医生说的做就行。读出来。", Next: 3},
				},
			},
			{
				Say:     "Twice a day after meals, right? Any food I should avoid?",
				SayOK:   "医生把写好的单子递给你：\"Take care!\" ——症状说清、医嘱确认、忌口问到，剩下的照医生说的做。生活主题六关，到这儿全亮了。",
				SayFail: "再读——Twice a day after meals 一口气，停，right 尾音上扬，最后 Any food I should avoid 把 avoid 咬清楚。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "复述成一次吃两片", Reply: "twice a day 是一天两次、每次一片——复述错了剂量就危险了。再选一个。"},
			{Label: "复述一遍再问忌口", Reply: "对——复述把听岔当场暴露，追问把疑问当场清零，医嘱不带回家猜。"},
			{Label: "连声yes道谢就走", Reply: "医嘱是最不能装懂的一段，出门剂量就开始模糊。再想想。"},
		},
		Correct: 1,
		Clear:   "症状说清、医嘱确认、忌口问到——生活主题六关全部点亮。下一关「机场值机」，旅行主题开幕：护照递出去之前，先想好靠窗还是过道。",
		Note:    "开口把症状说清了",
	},
	// ============ 旅行 ============
	"flight-checkin": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回地勤）",
				Options: []nodeOption{
					{Label: "Yes, just one bag. Could I get a window seat, please?", Reply: "地勤接过箱子贴上行李条：\"One bag—and window seat, let me see... done!\" ——一句话办成两件事：先直接回答问题（one bag），再用 Could I get 提请求，值机口语的标准节奏。现在，把它读出来。", Next: 1},
					{Label: "I have one baggage to check, and I want to sit window.", Reply: "地勤顿了一下：\"...One bag, you mean?\" 两个坑：baggage 是不可数名词，一件行李是 one bag 或 one piece of baggage；「坐窗边」是 a window seat，sit window 是中式直译，座位不是拿来坐窗户的。", Next: NodeRetry},
					{Label: "Check this. And window seat. Quick, OK?", Reply: "地勤挑了下眉，动作反而慢了半拍：\"...Sure.\" 每个词都听得懂，但全程省略句再加一个 Quick，像在催下属干活。柜台提请求，一句 Could I get 开头，礼貌到位，效率反而更高。", Next: NodeRetry},
				},
			},
			{
				Say:     "Yes, just one bag. Could I get a window seat, please?",
				SayOK:   "地勤敲了几下键盘：\"12A, window seat. Here's your boarding pass!\" ——登机牌到手，两件事全办成。别急着走，柜台马上有你的坏消息。",
				SayFail: "没对上——放慢，分两段读：Yes, just one bag... Could I get a window seat, please。请求句的重音落在 window 上。",
				SayNext: 2,
			},
			{
				Prompt: "刚要转身，地勤叫住你：\"Excuse me—this flight is overbooked. Would you take the next flight for a 200-dollar voucher?\" ——改签可以谈，但别白改，三句挑一句：",
				Options: []nodeOption{
					{Label: "Oh, OK... whatever you think is best. I don't mind.", Reply: "地勤立刻打出了新登机牌：\"Great, thank you!\" ——你被改签了，代金券一分没多，连下一班几点都没问。whatever you think 在谈判场景等于弃权：先问信息、再提条件，主动权才在你手里。", Next: NodeRetry},
					{Label: "If I change, can you give me more 100 dollars?", Reply: "地勤听懂了，忍着笑确认：\"You mean... 100 more?\" ——数量词的语序是 100 more dollars，more 跟在数字后面；more 100 是中式排法。还价的方向对了，句子得先立住。", Next: NodeRetry},
					{Label: "Maybe. What time is the next flight? Could you make it 300?", Reply: "地勤跟主管低声确认后回来：\"We can do 300.\" ——先问下一班几点（拿信息），再用 Could you make it... 还价（提条件），不冲也不软。这句话值一百美元，读出来。", Next: 3},
				},
			},
			{
				Say:     "Maybe. What time is the next flight? Could you make it 300?",
				SayOK:   "地勤把新登机牌和 300 美元代金券一起递来：\"And lounge access—on us.\" ——多出的一百美元，是你开口谈来的。值机这关，办得漂亮。",
				SayFail: "别急——分三段读：Maybe... What time is the next flight... Could you make it 300。数字 300 咬清楚，那是重点。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "改签前先问下一班几点", Reply: "对——先拿信息再谈条件，Could you make it 300 才有底气。"},
			{Label: "whatever 一句显得随和", Reply: "谈判场景里这句等于弃权，代金券一分都多不了。再想想。"},
			{Label: "more 100 dollars 语序对", Reply: "语序反了——是 100 more dollars，more 跟在数字后面。再选。"},
		},
		Correct: 0,
		Clear:   "托运、选座、还价，三件事一口气办漂亮。下一关「过关问答」——海关官员面无表情的三连问，答得稳才放行。",
		Note:    "值机还多谈了一百美元",
	},
	"customs-qa": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回海关官员）",
				Options: []nodeOption{
					{Label: "I come here for travel, stay ten days, live in friend home.", Reply: "官员抬眼又看了看你的护照：\"...Pardon?\" ——三处硬伤：人已经到了要说 I'm here（不是 I come）；短住是 stay，live 是长住；「朋友家」是 my friend's place，friend home 把所有格丢了。", Next: NodeRetry},
					{Label: "Why do you ask? That's my personal business.", Reply: "官员放下护照，朝旁边的同事抬了抬下巴——\"Step aside, please.\" 海关问话是法定程序，不是闲聊；把例行问题当冒犯，只会换来更久的盘问。如实、简短、直接，才是最快通道。", Next: NodeRetry},
					{Label: "Tourism. I'm staying ten days at my friend's place.", Reply: "官员点点头，在系统里敲了几下：\"Enjoy your stay.\" ——目的、天数、住处，三连问一句收齐：Tourism 直答，staying ten days 报时长，friend's place 所有格不丢。读出来，稳住这一句。", Next: 1},
				},
			},
			{
				Say:     "Tourism. I'm staying ten days at my friend's place.",
				SayOK:   "官员在系统里记下：\"Ten days, tourism—OK.\" ——三连问稳稳接住。但他没盖章，又抬起了头：追问来了。",
				SayFail: "别紧张，海关最不缺的就是耐心——断开读：Tourism... I'm staying ten days... at my friend's place。friend's 的 s 要带出来。",
				SayNext: 2,
			},
			{
				Prompt: "官员盯着屏幕问：\"Are you carrying any food or plants?\" ——你包里有两盒送人的茶叶，三句挑一句：",
				Options: []nodeOption{
					{Label: "Yes, I have two boxes of tea leaves. Is that allowed?", Reply: "官员扫了一眼申报单：\"Tea is fine. Thanks for declaring.\" ——如实申报，再补一句 Is that allowed?，把判断交给官员，自己零风险。诚实在海关不只是美德，是最优策略。读出来。", Next: 3},
					{Label: "Yes, I bring two box of tea, can bring or not?", Reply: "官员放慢语速跟你确认：\"Two... boxes?\" ——两盒要加复数 boxes；can bring or not 是「能不能带」的中式直译，英语问许可说 Is that allowed? 或 Can I bring them in?，主语不能丢。", Next: NodeRetry},
					{Label: "No, nothing. Just clothes.", Reply: "官员指了指扫描仪：\"Then let's take a look.\" X 光下两盒茶叶清清楚楚——瞒报被查到，轻则没收，重则罚款留记录。茶叶本来能带，一句谎话把小事变成了大事。", Next: NodeRetry},
				},
			},
			{
				Say:     "Yes, I have two boxes of tea leaves. Is that allowed?",
				SayOK:   "官员在申报单上勾了一笔，把护照连同茶叶一起递回：\"Welcome, and enjoy the tea.\" ——如实申报，十秒放行。这就是过关的全部秘密。",
				SayFail: "再来——重音落在 two boxes 和 tea leaves 上；最后的 Is that allowed 尾音上扬，是在问，不是在认。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "说 No, nothing 省事", Reply: "X 光一扫就穿帮——茶叶没事，谎话有事。再想想。"},
			{Label: "带茶叶如实申报再问许可", Reply: "对——申报加一句 Is that allowed?，判断交给官员，自己零风险。"},
			{Label: "friend home 不用 's", Reply: "「朋友家」是 friend's place，所有格丢了意思就变了。再选。"},
		},
		Correct: 1,
		Clear:   "三连问加申报追问全接住，诚实反而走了最快通道。下一关「酒店入住」——报姓名之外，你还想要间高楼层的安静房。",
		Note:    "如实申报十秒过关",
	},
	"hotel-checkin": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回前台）",
				Options: []nodeOption{
					{Label: "I am Li Ming, I ordered a room in your hotel yesterday.", Reply: "前台笑着查了查：\"You... ordered?\" ——order 是点餐下单，订房要说 booked 或 I have a reservation；两个完整句用逗号硬连也是粘连句。一个动词用错，住客听着就成了外卖客。", Next: NodeRetry},
					{Label: "Yes, under Li Ming. Could I get a quiet high-floor room?", Reply: "前台敲键盘的手没停：\"Room 1208—high floor, away from the elevator.\" ——under + 姓名是报预订的最地道说法，Could I get 提要求不硬不软。两个愿望一句话，全中。读出来。", Next: 1},
					{Label: "I hereby request accommodation under the name of Li Ming.", Reply: "前台愣了一下，差点站直了敬礼：\"...Certainly, sir?\" 语法满分，语域跑偏——hereby、request accommodation 是律师函措辞，前台不是法庭。口语订房，一句 under Li Ming 就够了。", Next: NodeRetry},
				},
			},
			{
				Say:     "Yes, under Li Ming. Could I get a quiet high-floor room?",
				SayOK:   "前台递来房卡：\"1208—twelfth floor, quiet side. Enjoy!\" ——要求提得体面，房间就给得痛快。不过这间房，马上要给你出道难题。",
				SayFail: "不急——断成两段：Yes, under Li Ming... Could I get a quiet high-floor room。quiet 读两拍：qui-et，别吞成一拍。",
				SayNext: 2,
			},
			{
				Prompt: "进房十分钟，空调怎么调都是热风。你拨通前台：\"Front desk, how can I help you?\" ——三句挑一句：",
				Options: []nodeOption{
					{Label: "This is unacceptable! I want a refund right now!", Reply: "前台连声道歉，但明显被将住了：\"I'm... sorry?\" ——空调坏了直接跳到退款，把小问题谈成了僵局。诉求要和问题同级：先说清故障，再给对方可选的动作，事情才动得起来。", Next: NodeRetry},
					{Label: "The air condition is bad, you quickly send people.", Reply: "前台猜了几秒：\"The... air conditioning?\" ——空调是 air conditioning 或 AC，air condition 少了 -ing 就不是那台机器了；quickly send people 是中式「快派人」，英语说 could you send someone up。", Next: NodeRetry},
					{Label: "Hi, the AC isn't working—could you fix it or move me?", Reply: "前台立刻应声：\"So sorry! Let me check what's available.\" ——先说清故障（isn't working），再给两个选项（fix it or move me），前台不用猜你要什么。有选项的投诉，才是好投诉。读出来。", Next: 3},
				},
			},
			{
				Say:     "Hi, the AC isn't working—could you fix it or move me?",
				SayOK:   "五分钟后前台来电：\"We've moved you to 1506—same quiet side, two floors up.\" ——空调坏了，反而升了两层楼。会开口的人，运气都不会太差。",
				SayFail: "再来——isn't working 别连糊了，断开：isn't... working。fix 和 move 两个动词都咬重一点，那是你的两个选项。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "说清故障并给修或换选项", Reply: "对——fix it or move me，前台不用猜，事情立刻动起来。"},
			{Label: "订房动词用 ordered", Reply: "order 是点餐下单——订房是 booked 或 have a reservation。再想想。"},
			{Label: "先喊 unacceptable 施压", Reply: "小事谈成僵局——先说故障再给选项，比施压快得多。再选。"},
		},
		Correct: 0,
		Clear:   "入住办妥，空调坏了还体面地换到了更高一层。下一关「打车出行」——司机一句 \"Where to?\"，目的地和路线你得一口说清。",
		Note:    "空调坏了换来好房",
	},
	"taxi-ride": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回司机）",
				Options: []nodeOption{
					{Label: "I want go to the city museum, I very hurry.", Reply: "司机从后视镜里看你：\"You want... what?\" ——want 后面接动词要带 to：want to go；「我很急」是 I'm in a hurry，very hurry 既丢了 be 动词又搭错了词。地名说对了，句子得先站稳。", Next: NodeRetry},
					{Label: "Anywhere near the museum is fine... no rush at all, really.", Reply: "司机悠哉地汇入了慢车道：\"Sure, no rush.\" ——你明明赶时间，嘴上却说 no rush，司机只会照字面执行。英语不靠察言观色：需求不说出口，就等于不存在。", Next: NodeRetry},
					{Label: "The city museum, please—I'm in a hurry, so the fastest way.", Reply: "司机点头打了转向灯：\"Fastest way—got it.\" ——目的地加 please 报站，I'm in a hurry 给情况，so the fastest way 给指令。信息给足，司机不用猜。读出来。", Next: 1},
				},
			},
			{
				Say:     "The city museum, please—I'm in a hurry, so the fastest way.",
				SayOK:   "车子利落地上了主路：\"Twenty minutes if the highway's clear.\" ——需求说清了，路也顺了。不过开着开着，你觉得有点不对劲。",
				SayFail: "再来——I'm in a hurry 四个词连成一口气；fastest 重音在前：FAST-est。",
				SayNext: 2,
			},
			{
				Prompt: "导航明明指直行，司机却拐进了小路，计价表跳得飞快。三句挑一句：",
				Options: []nodeOption{
					{Label: "Sorry, why are we going this way? Could we follow the GPS?", Reply: "司机指了指前方：\"Construction on the main road—but sure, GPS it is.\" ——先问原因（也许真有路况），再提要求（follow the GPS），不指控也不吃闷亏。质疑的体面版，就长这样。读出来。", Next: 3},
					{Label: "Stop the car! You're cheating me. I'll call the police.", Reply: "司机猛地靠边急刹：\"Whoa—relax!\" ——还没问原因就定罪，万一前面真在修路，这一嗓子就下不来台了。先问 why，再提要求，指控永远放在最后一步。", Next: NodeRetry},
					{Label: "Driver, why you go this small road? Use GPS please.", Reply: "司机听懂了意思，但你自己听听——why you go 少了助动词，特殊疑问句要 why are we going；开头直呼 Driver 也生硬，换一句 Sorry 或 Excuse me，同样的话顺耳一倍。", Next: NodeRetry},
				},
			},
			{
				Say:     "Sorry, why are we going this way? Could we follow the GPS?",
				SayOK:   "司机切回主路，导航的声音重新响起：\"Ten minutes to go.\" ——问得体面，改得痛快，车费一分没多跳。这一路，是你自己盯下来的。",
				SayFail: "别急——前半句 why are we going this way 尾音上扬是疑问；GPS 三个字母逐个读清：G-P-S。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "起步就喊 Stop the car", Reply: "还没问原因就定罪，真遇上修路就尴尬了。再想想。"},
			{Label: "先问原因再要求走导航", Reply: "对——why are we going this way 加 follow the GPS，不冤枉人也不吃亏。"},
			{Label: "very hurry 表达很急", Reply: "丢了 be 动词——是 I'm in a hurry。再选。"},
		},
		Correct: 1,
		Clear:   "路线自己盯住了，车费一分没多花。下一关「景点购票」——两张票之外，学生折扣和闭馆时间也要一并问到。",
		Note:    "绕路被你问回正道",
	},
	"attraction-tickets": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回售票员）",
				Options: []nodeOption{
					{Label: "Two adults, please. Student discount? And your closing time?", Reply: "售票员利落地敲着屏幕：\"Two adults—and yes, students get 20% off with ID.\" ——窗口英语就该这么省：Two adults, please 报需求，两个短问句把折扣和闭馆时间一次问齐，身后排队的人都感谢你。读出来。", Next: 1},
					{Label: "Two adult ticket. Student have discount? You close when?", Reply: "售票员逐个跟你确认：\"Two... tickets?\" ——三连坑：两张票 ticket 要加 s；主谓一致是 students get 或 Do students get；You close when 疑问词要提前加助动词：When do you close。", Next: NodeRetry},
					{Label: "I would like to inquire whether discounts are offered.", Reply: "售票员眨眨眼，后面队伍开始探头——\"...A discount? Yes.\" 这句放进邮件里满分，放在售票窗口太重：inquire whether 是书面语，窗口三秒一单，短句直问才是对的语域。", Next: NodeRetry},
				},
			},
			{
				Say:     "Two adults, please. Student discount? And your closing time?",
				SayOK:   "售票员递出两张票：\"We close at six—last entry at five.\" ——票、折扣、时间，三件事一分钟办完。可惜天公不作美，头顶响了声闷雷。",
				SayFail: "再来——三个短句之间停半拍：Two adults, please... Student discount... And your closing time。问句尾音抬起来。",
				SayNext: 2,
			},
			{
				Prompt: "刚拿到票，暴雨倾盆而下，你想改天再来。回到窗口，三句挑一句：",
				Options: []nodeOption{
					{Label: "Give me a refund. The rain is not my problem.", Reply: "售票员指了指窗口贴的告示：\"All sales are final.\" ——一开口就要退款还甩责任，对方立刻搬出规则挡你。规则内好商量的其实是改期：先问 change，别先谈钱。", Next: NodeRetry},
					{Label: "It's pouring—could I change my tickets to another day?", Reply: "售票员看了眼外面的雨，语气软了：\"Sure—any day this month works.\" ——退款难，改期易：could I change... to another day 给了对方一个不违反规则也能帮你的台阶。提请求，先挑对方答应得了的提。读出来。", Next: 3},
					{Label: "I want change my ticket to other day, rain too big.", Reply: "售票员大致听懂了：\"Change it, you mean?\" ——want 后面要接 to change；「改到别的日子」是 another day；「雨太大」英语说 it's pouring 或 heavy rain，rain too big 是把中文的尺寸搬进了英语。", Next: NodeRetry},
				},
			},
			{
				Say:     "It's pouring—could I change my tickets to another day?",
				SayOK:   "售票员在票背面盖了个改期章：\"Just show this at the gate—stay dry!\" ——一场暴雨没浇掉你的门票。改天晴了再来，这次是带着口语来的。",
				SayFail: "再来——pouring 重音在前：POUR-ing。后半句连贯读：could I change my tickets to another day，尾音抬起来。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "雨太大说 rain too big", Reply: "中文尺寸搬进英语了——是 it's pouring 或 heavy rain。再想想。"},
			{Label: "开口先要退款最直接", Reply: "对方一句 All sales are final 就挡回来了——先问改期。再选。"},
			{Label: "先提对方能答应的改期", Reply: "对——退款难改期易，could I change 给足了对方台阶。"},
		},
		Correct: 2,
		Clear:   "暴雨里保住了两张门票，折扣和闭馆时间也问了个清楚。下一关「行李丢失」——转盘转空了，你的箱子没出来。",
		Note:    "暴雨天改签了门票",
	},
	"lost-luggage": {
		Nodes: []scriptNode{
			{
				Prompt: "柜台工作人员抬起头：\"I'm sorry to hear that. Can you describe your bag?\" ——三句挑一句：",
				Options: []nodeOption{
					{Label: "You lost my bag! Find it now, or I'll never fly again.", Reply: "工作人员公事公办地推来一张表：\"Please fill this out.\" ——冲柜台发火最亏：弄丢行李的不是眼前这个人，能帮你找回来的却是。此刻最值钱的不是气势，是箱子的颜色和尺寸。", Next: NodeRetry},
					{Label: "It's a large blue suitcase with a red ribbon on the handle.", Reply: "工作人员边听边录入系统：\"Large, blue, red ribbon—got it.\" ——尺寸、颜色、特征一句排齐；with a red ribbon on the handle 这种细节，正是从几百个蓝箱子里认出你那只的钥匙。读出来。", Next: 1},
					{Label: "It is a big size blue color box, very big.", Reply: "工作人员的笔尖停了停：\"A... box?\" ——big size、blue color 都是中式冗余，big 和 blue 自己就够了；行李箱是 suitcase，box 是纸箱——照这么登记，找回来的可能真是个纸箱。", Next: NodeRetry},
				},
			},
			{
				Say:     "It's a large blue suitcase with a red ribbon on the handle.",
				SayOK:   "工作人员打出行李单，记下你的酒店地址：\"We'll deliver it as soon as it lands.\" ——描述清楚，地址留好。但为了理赔，她还有一个问题。",
				SayFail: "再来——形容词排队读顺：a large blue suitcase。后半句断开：with a red ribbon... on the handle。",
				SayNext: 2,
			},
			{
				Prompt: "工作人员抬头：\"For the claim—what was inside?\" ——为了理赔，说出箱里三件重要物品和大致价值，三句挑一句：",
				Options: []nodeOption{
					{Label: "A laptop, clothes, and a camera—about 1,500 dollars total.", Reply: "工作人员在理赔栏里逐项登记：\"Fifteen hundred—noted.\" ——三件物品点名，about 给估值留了余地又不含糊。理赔表上写得清楚的人，赔付到账也快。读出来。", Next: 3},
					{Label: "Inside have computer, clothes and camera, very expensive.", Reply: "工作人员等你重说：\"Sorry—inside... what?\" ——中文「里面有」直译成 Inside have 是最经典的中式存在句，英语要说 There's a laptop inside，或干脆列清单；very expensive 也不如报个数字有用。", Next: NodeRetry},
					{Label: "Oh, just some old things... nothing important, I guess.", Reply: "工作人员如实录入：\"No significant contents.\" ——客气用错了地方：理赔单上写 nothing important，等于亲手把赔偿金归零。该谦虚的场合很多，报损失清单不在其中。", Next: NodeRetry},
				},
			},
			{
				Say:     "A laptop, clothes, and a camera—about 1,500 dollars total.",
				SayOK:   "工作人员递来理赔回执：\"If it's not found in 24 hours, compensation starts from this list.\" ——一句英文，给箱子上了保险。剩下的交给系统。",
				SayFail: "再来——列清单要有节奏：A laptop... clothes... and a camera。数字整段读顺：fifteen hundred dollars。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "「里面有」说 Inside have", Reply: "中式存在句——英语用 There is/are，或直接列清单。再想想。"},
			{Label: "报清单和估值利于理赔", Reply: "对——三件物品加 about 1,500 dollars，理赔才有依据。"},
			{Label: "理赔时越谦虚越好", Reply: "一句 nothing important，等于亲手把赔偿金归零。再选。"},
		},
		Correct: 1,
		Clear:   "箱子挂了失，理赔也备了案，剩下交给系统。下一关「旅途求助」——比丢箱子更糟的来了：护照不见了，去哪儿开口？",
		Note:    "挂失理赔一次办妥",
	},
	"travel-help": {
		Nodes: []scriptNode{
			{
				Prompt: "值班警察放下笔看向你：\"What happened?\" ——三句挑一句：",
				Options: []nodeOption{
					{Label: "I lose my passport at this morning in the subway maybe.", Reply: "警察停下笔跟你确认：\"You mean you lost it?\" ——丢护照是已经发生的事，要用过去式 lost；「今天早上」直接说 this morning，前面不加 at。时间地点说不准，笔录都没法起头。", Next: NodeRetry},
					{Label: "Sorry, sorry... my English is bad... nothing, never mind.", Reply: "警察疑惑地看着你往门口退：\"Wait—do you need help or not?\" ——最要命的是 never mind：护照还丢着，求助先被你自己撤回了。英语不好从来不是问题，把事说出口才是全部。", Next: NodeRetry},
					{Label: "I lost my passport on the subway this morning. Can you help?", Reply: "警察立刻抽出一张登记表：\"Subway, this morning—let's file a report.\" ——lost 过去式定性，地点时间各就各位，Can you help 直接请求。越慌的时候，越短的句子越可靠。读出来。", Next: 1},
				},
			},
			{
				Say:     "I lost my passport on the subway this morning. Can you help?",
				SayOK:   "警察边写边点头：\"We'll notify the subway's lost and found right away.\" ——什么时候、在哪儿、要什么，三十秒说清。他接着提了个建议。",
				SayFail: "深呼吸，慢慢来——lost 结尾的 t 收住，别读成 loss。断句：I lost my passport... on the subway... this morning。",
				SayNext: 2,
			},
			{
				Prompt: "警察递来表格：\"Fill this out. Do you want to contact your embassy?\" ——你确实得去趟大使馆，三句挑一句：",
				Options: []nodeOption{
					{Label: "Where is Chinese embassy? What time it opens?", Reply: "警察听懂了，但两处得修：国名加机构前要有 the——the Chinese embassy；疑问句语序是 What time does it open，助动词提到主语前，What time it opens 是陈述句的排法。", Next: NodeRetry},
					{Label: "How do I get to the Chinese embassy? And when does it open?", Reply: "警察在便签上画了张小地图：\"Two stops on Line 3—opens at nine.\" ——How do I get to... 是问路的万能句型，when does it open 把开门时间一并问到。一张便签，就是你明天的路线图。读出来。", Next: 3},
					{Label: "Take me to the embassy right now. It's your job.", Reply: "警察挑了挑眉，把表格往你面前又推了推：\"My job is this form.\" ——警察帮忙是情分，命令句加一句 It's your job，把情分聊成了对峙。你需要的是路线，不是输赢。", Next: NodeRetry},
				},
			},
			{
				Say:     "How do I get to the Chinese embassy? And when does it open?",
				SayOK:   "你捏着那张手绘小地图走出警局，雨停了。\"Good luck!\" 警察在身后说。——护照会补回来的，而你已经证明：出了任何事，你都开得了口。",
				SayFail: "再来——How do I get to 五个词连成一口气；embassy 重音在前：EM-bassy。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "问路用 How do I get to", Reply: "对——问路万能句型，再补一句 when does it open，一次问全。"},
			{Label: "time it opens 是疑问语序", Reply: "那是陈述句排法——疑问句要 What time does it open。再想想。"},
			{Label: "让警察现在就送你去", Reply: "帮忙是情分不是义务——命令句把求助聊成对峙。再选。"},
		},
		Correct: 0,
		Clear:   "护照丢了也没慌，求助、问路、问时间全用英语办妥——旅行主题通关。下一关进职场：「同事初见」，新同事正朝你走来。",
		Note:    "丢了护照没丢开口",
	},
	// ============ 职场 ============
	"work-self-intro": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句向全组开口）",
				Options: []nodeOption{
					{Label: "It is a great honor to join this esteemed organization.", Reply: "同事们礼貌地鼓了掌，但没人记住你——\"Welcome... aboard?\" 这句像年会致辞不像打招呼：esteemed organization 是留给演讲稿的词。团队自我介绍要让人接得上话——名字、做什么、一点个人色彩，说人话就赢了。", Next: NodeRetry},
					{Label: "Hi, I'm Lin. I'll be working on data—and I hike a lot.", Reply: "话音刚落就有人接茬：\"Oh nice, we have a hiking group!\" ——名字、负责什么、一点个人色彩，三件套一句装下；最后那点「个人色彩」就是钩子，别人想认识你，总得有个话头。把它读出来。", Next: 1},
					{Label: "My name is Lin. I am work on data. Nice to meet you.", Reply: "几位同事点头微笑，接着小声互相确认：\"...He does what?\" I am work 把 be 动词和实义动词撞在了一起——要么 I work on data，要么 I'll be working on data。开口用 I'm Lin 也比 My name is 更像口语。", Next: NodeRetry},
				},
			},
			{
				Say:     "Hi, I'm Lin. I'll be working on data—and I hike a lot.",
				SayOK:   "经理带头鼓掌：\"Welcome aboard, Lin!\" 已经有人约你周五爬山了——被记住的从来不是职位，是那点个人色彩。不过散会后，真正的一对一考验才来。",
				SayFail: "别急，三个信息点分开咬：Hi, I'm Lin... I'll be working on data... and I hike a lot。一段一口气。",
				SayNext: 2,
			},
			{
				Prompt: "散会后一位同事凑过来：\"So how are you finding it so far?\" ——接住话头，再顺势约杯咖啡，三句挑一句：",
				Options: []nodeOption{
					{Label: "I'm fine, thank you. And you?", Reply: "同事笑容僵了半秒，摆摆手走了——\"...Good, good.\" 教科书三件套答非所问：How are you finding it 问的是「感觉如何」，不是「你好吗」。答感受才接得住：Really good，或者 A bit overwhelming, but exciting。", Next: NodeRetry},
					{Label: "All is good. I will invite you drink coffee.", Reply: "同事听懂了，但眉毛动了一下——invite you drink 少了个 to（invite sb to do sth）；而且 I will invite you 像在发正式请柬，对方还得琢磨要不要回礼。英语约咖啡轻得多：Want to grab a coffee?", Next: NodeRetry},
					{Label: "Really good! Want to grab a coffee and chat about the team?", Reply: "同事眼睛一亮：\"Sure! There's a great place downstairs.\" ——先答感受，再用 Want to... 发出零压力邀请；grab a coffee 是职场社交的硬通货，比 have coffee 随性，比 invite 轻巧。读出来。", Next: 3},
				},
			},
			{
				Say:     "Really good! Want to grab a coffee and chat about the team?",
				SayOK:   "你们约好明天下午三点——入职第一天，全组记住了你的名字，还多了一场咖啡局。开口这事，一回生二回熟。",
				SayFail: "语气松一点——Want to grab a coffee 是邀请不是审问，尾音轻轻上扬。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "答感受，再轻邀请喝咖啡", Reply: "对——How are you finding it 要答感受，接一句 grab a coffee，关系就此破冰。"},
			{Label: "答 I'm fine 就完事", Reply: "教科书三件套答非所问——对方问的是感受，不是问候。再想想。"},
			{Label: "郑重宣布要请喝咖啡", Reply: "I will invite you 太重像请柬，还少了个 to——轻轻一句 grab a coffee 才对味。再选。"},
		},
		Correct: 0,
		Clear:   "名字、职责、个人色彩，一句话让全组记住你。下一关「会议表态」——老板当众问你对新方案怎么看，先肯定还是先泼冷水？",
		Note:    "一句话让全组记住了你",
	},
	"meeting-opinion": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回老板）",
				Options: []nodeOption{
					{Label: "I don't agree. The timeline is impossible.", Reply: "会议室温度骤降，老板挑了挑眉：\"...OK. Noted.\" 你明明是部分同意，这句却全盘否了，还用 impossible 把门焊死。英语职场表异议讲究缓冲：先说喜欢什么，再用 but I'm worried about 递顾虑，话才有人接。", Next: NodeRetry},
					{Label: "I think the plan is good, but the time is not enough.", Reply: "老板点了点头，但明显没走心——time is not enough 是「时间不够」的直译，地道说法是 the timeline is tight 或 we're short on time；前半句 the plan is good 也太平了，肯定要具体：I like the direction。", Next: NodeRetry},
					{Label: "I like the direction, but I'm worried about the timeline.", Reply: "老板身体前倾：\"Go on—what's your concern?\" ——先肯定方向，再用 but I'm worried about 把顾虑递出去。这是职场表态的黄金句式：糖在前药在后，反对意见也能被当成建设性意见听。", Next: 1},
				},
			},
			{
				Say:     "I like the direction, but I'm worried about the timeline.",
				SayOK:   "老板把 timeline 写上了白板：\"Fair point. Let's look at it.\" ——你的顾虑进了会议纪要，而不是消散在空气里。不过更难的还在后面：完全不同意的时候呢？",
				SayFail: "在 but 前停半拍：I like the direction... but I'm worried about the timeline。转折要让人听见。",
				SayNext: 2,
			},
			{
				Prompt: "另一位同事抛出的新方案你完全不同意。老板转头看你：\"Thoughts?\" ——既要反对，又要给出路，三句挑一句：",
				Options: []nodeOption{
					{Label: "I see your point, but could we try a phased rollout instead?", Reply: "同事没恼，反而追问：\"Interesting—how would that work?\" ——I see your point 先接住对方，could we try... instead 把反对包装成提议：你否的是方案，捧的是讨论，这就是不伤和气的全部秘密。读出来。", Next: 3},
					{Label: "That won't work. My idea is better.", Reply: "同事的脸沉了下来，接下来十分钟他都在防御而不是讨论——反对最忌把「方案不行」说成「你不行」，My idea is better 更是火上浇油。先给一句 I see your point，火力立刻减半。", Next: NodeRetry},
					{Label: "I have a different opinion. We can do like this.", Reply: "\"...Like what?\" 同事一脸困惑。do like this 是「这样做」的直译，英文得说 do it this way，或者干脆把方案说出来；I have a different opinion 也硬得像开辩论。一句 but 带出替代方案就够了。", Next: NodeRetry},
				},
			},
			{
				Say:     "I see your point, but could we try a phased rollout instead?",
				SayOK:   "散会时老板拍了拍你肩膀：\"Good discussion today.\" ——反对了两次，没得罪一个人，方案还朝你担心的方向修了。接下来，换你布置任务了。",
				SayFail: "I see your point 四个词要连得顺——缓冲句读得越自然，火药味越少。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "直接说 That won't work", Reply: "语法没错，人得罪全了——方案被否成这样，谁还愿意讨论？再选。"},
			{Label: "先接住对方观点再提议", Reply: "对——I see your point 缓冲，could we try 递替代案：反对变共创。"},
			{Label: "不同意就先憋着不说", Reply: "憋着不是情商是失职——会上不说，返工时说的就是脏话了。再想想。"},
		},
		Correct: 1,
		Clear:   "先给糖再给药，完全反对也能变成共同探讨。下一关「任务交代」——出差三天，报表任务要口头交给同事，说漏一句都得远程救火。",
		Note:    "会上体面递出了反对票",
	},
	"task-handover": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句开口交接）",
				Options: []nodeOption{
					{Label: "Could you cover the report while I'm away? It's due Friday.", Reply: "同事转过椅子爽快点头：\"Sure, no problem.\" ——Could you cover 是请托的标准开头：cover 一词自带「临时顶上」的分寸感，任务和时限一句说清。但交接才完成一半，注意事项还没落地。", Next: 1},
					{Label: "Please help me do the report. You must finish it Friday.", Reply: "同事接了活，心里却咯噔一下——对同级同事说 you must，命令感比中文的「必须」重得多；please help me do 也偏中式。把压力给事不给人：It needs to be done by Friday，同一件事，听感天差地别。", Next: NodeRetry},
					{Label: "I hereby entrust the weekly report to you in my absence.", Reply: "同事笑出了声：\"Hereby? Am I signing a contract?\" ——hereby、entrust 全是合同用语，口头交接要用 cover、take over 这种轻词。正式感一上来，帮忙的人情味就下去了。", Next: NodeRetry},
				},
			},
			{
				Say:     "Could you cover the report while I'm away? It's due Friday.",
				SayOK:   "同事掏出手机记了条备忘：\"Got it—report, due Friday.\" ——任务和时限都进了对方脑子。但 TA 紧接着想到一个你还没交代的情况。",
				SayFail: "两个信息点之间停一拍：cover the report while I'm away... It's due Friday。时限要读得清清楚楚。",
				SayNext: 2,
			},
			{
				Prompt: "同事忽然想起什么：\"Sure, but what if the client emails me directly?\" ——哪些 TA 自己定、哪些必须找你，三句挑一句划清楚：",
				Options: []nodeOption{
					{Label: "If have problem, you can call me every time.", Reply: "\"...Every time?\" 同事有点懵。两个坑：if have problem 丢了主语，得说 if there's a problem；every time 是「每一次」，你想说的「随时」是 anytime。更要命的是事事找你——那这趟交接等于没交。", Next: NodeRetry},
					{Label: "Use your judgment—but if it's about pricing, call me.", Reply: "同事竖起大拇指：\"Crystal clear—judgment calls on me, pricing goes to you.\" ——use your judgment 是授权，if it's about pricing 是红线：一句话切清「哪些你定、哪些找我」，这才叫交接完毕。读出来。", Next: 3},
					{Label: "Just do whatever you think is right, I trust you.", Reply: "听着大方，其实是把锅整个递了过去——万一客户要改报价，TA 也「自己看着办」吗？授权不等于放羊：不划红线的信任，回来要用加班来还。", Next: NodeRetry},
				},
			},
			{
				Say:     "Use your judgment—but if it's about pricing, call me.",
				SayOK:   "出差三天，你只收到一条短信：「客户问报价，等你回」——该 TA 定的都定了，该等你的都等着。交接的最高境界就是无事发生。接下来，轮到你上台了。",
				SayFail: "破折号处换口气：Use your judgment... but if it's about pricing, call me。红线那半句要一字一顿。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "让同事事事打电话问你", Reply: "那交接就白交了，你出差等于没出。授权和红线一个都不能少。再选。"},
			{Label: "全权放手让 TA 看着办", Reply: "不划红线的放手是甩锅——报价出了岔子，加班的还是你。再想想。"},
			{Label: "授权日常，划清找你的红线", Reply: "对——use your judgment 授权，if it's about pricing 划线，交接闭环。"},
		},
		Correct: 2,
		Clear:   "任务、时限、授权边界，三样交齐才敢关机登机。下一关「项目汇报」——会议室安静下来，所有人看着你，第一句说什么？",
		Note:    "交接清爽，出差零救火",
	},
	"project-present": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句做开场白）",
				Options: []nodeOption{
					{Label: "Good morning. Today I will introduce about our project.", Reply: "台下有人低头看了眼手机——introduce about 多了个 about，introduce 直接带宾语；而且它偏「介绍某人」，汇报开场的地道动词是 walk you through：自带「我带你们走一遍」的引导感。第一句磕绊，气场先漏。", Next: NodeRetry},
					{Label: "Morning! I'll walk you through Q3 in three parts.", Reply: "有人合上手机坐直了——walk you through 是汇报开场的黄金动词，比 introduce 自然、比 show 有引导感；in three parts 一出口，听众脑子里立刻有了地图。开场十秒，定一整场的基调。", Next: 1},
					{Label: "OK so, um, let me just show you some stuff we did.", Reply: "台下两个人交换了一下眼神——um、some stuff 让正式汇报听起来像没备课。轻松可以，含糊不行：开场必须给听众确定感，讲什么、分几块，一句话立住。", Next: NodeRetry},
				},
			},
			{
				Say:     "Morning! I'll walk you through Q3 in three parts.",
				SayOK:   "全场安静下来，老板放下了笔：\"Go ahead.\" ——框架给出去了，接下来每一页都有人跟着你走。不过讲到一半，总会有人举手。",
				SayFail: "结构词要咬重：Q3... three parts。听众全靠这两个词搭框架，含糊不得。",
				SayNext: 2,
			},
			{
				Prompt: "讲到第二部分，有人举手打断：\"Sorry, can you go back to the numbers?\" ——接住提问、澄清数据、再拉回主线，三句挑一句：",
				Options: []nodeOption{
					{Label: "Sure—this is Q3 revenue, up 8%. Now, back to the roadmap.", Reply: "提问的人边点头边记：\"Got it, thanks.\" 全场节奏没散——Sure 一秒接住，一句话澄清数字，Now, back to... 把方向盘稳稳拿回来。被打断不可怕，回不到主线才可怕。读出来。", Next: 3},
					{Label: "Please let me finish first. Questions at the end.", Reply: "对方缩回了手，脸上有点挂不住——把提问者噎回去，后面就再没人敢互动了，而没人互动的汇报和群发邮件没区别。先花十秒回应，再用 back to 拉回，两头都不丢。", Next: NodeRetry},
					{Label: "Ah sorry, wait me a moment, I find that page.", Reply: "\"...Wait you?\" wait me 是「等我」的直译，英语得说 give me a second 或 bear with me；后半句也该是 let me find that page。与其手忙脚乱翻页，不如一句稳稳的 Sure 先把场接住。", Next: NodeRetry},
				},
			},
			{
				Say:     "Sure—this is Q3 revenue, up 8%. Now, back to the roadmap.",
				SayOK:   "收尾时，刚才提问的同事带头点头：\"Nice presentation.\" ——接得住打断，才算真的会汇报。下一场硬仗在会议室外：供应商的报价单已经发来了。",
				SayFail: "三段分开读稳：Sure—this is Q3 revenue, up 8%... Now, back to the roadmap。转场词 Now 要提气。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "让提问的人等到最后再问", Reply: "问题会凉，人也会凉——十秒能答的当场答，互动是汇报的呼吸。再选。"},
			{Label: "十秒澄清，再拉回主线", Reply: "对——Sure 接住、一句澄清、Now, back to 拉回：方向盘始终在你手里。"},
			{Label: "停下来慢慢翻页找那张表", Reply: "全场看你翻三十秒页，气就泄光了——记不清就先接住再补。再想想。"},
		},
		Correct: 1,
		Clear:   "开场给地图、打断能拉回，整场汇报的方向盘都在你手里。下一关「谈判议价」——供应商报价 12000 美元，超预算两千，怎么开口砍？",
		Note:    "汇报被打断也拉得回来",
	},
	"price-negotiation": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回供应商）",
				Options: []nodeOption{
					{Label: "Twelve thousand? No way. Give me your real price.", Reply: "对方收起了笑容：\"That IS the real price.\" ——No way 掀桌，real price 更是暗指对方报假价、在质疑诚信，谈判还没开始就先结了仇。体面压价的开法：先表兴趣稳住关系，再把预算数字亮出来。", Next: NodeRetry},
					{Label: "The price is too expensive. Can you make it cheap?", Reply: "对方听懂了但皱了皱眉——两个坑：price 只有 high/low，「贵」的是东西不是价格，得说 the price is too high；而 cheap 在商务语境里带「廉价劣质」味，压价要说 come down a bit，或者直接报你的目标价。", Next: NodeRetry},
					{Label: "We're interested, but our budget is closer to 10,000.", Reply: "对方沉吟了一下：\"Hmm. Let me see what I can do.\" ——interested 先稳住关系，our budget is closer to 把目标价亮出来还不显硬：你不是在砍价，你是在「对齐预算」。措辞一换，姿态全变。", Next: 1},
				},
			},
			{
				Say:     "We're interested, but our budget is closer to 10,000.",
				SayOK:   "对方真的开始按计算器了——数字进了谈判桌，关系还没伤。但老练的销售不会就这么让步，硬话马上就到。",
				SayFail: "数字必须读清楚：closer to ten thousand。谈判桌上数字一含糊，亏的是自己。",
				SayNext: 2,
			},
			{
				Prompt: "对方摊开手：\"That's really the best we can do.\" ——价格谈死了，换个筹码再谈，三句挑一句：",
				Options: []nodeOption{
					{Label: "If you not give discount, we find another company.", Reply: "对方脸色冷了下来——if you not give 丢了助动词，该是 if you don't give；更伤的是当面亮「找别家」，谈判瞬间变最后通牒：就算对方让了价，心里也记你一笔，后面的合作全是刺。", Next: NodeRetry},
					{Label: "Understood. What if we doubled the order—any room then?", Reply: "对方眼睛一亮：\"Now we're talking.\" ——价格谈死就换筹码，what if 是谈判的万能钥匙：不逼对方让步，而是递给对方一个让步的台阶。量、账期、附赠服务，都是价格之外的牌。读出来。", Next: 3},
					{Label: "OK, I understand. 12,000 is fine then.", Reply: "对方内心已经开香槟了——\"best we can do\" 是谈判话术不是终点，一句话就缴械，预算照样超，对方还会觉得你刚才的压价没诚意。价格锁死了，手里还有量、账期、服务三张牌呢。", Next: NodeRetry},
				},
			},
			{
				Say:     "Understood. What if we doubled the order—any room then?",
				SayOK:   "第二天对方回了邮件：订量翻倍，单价降到 9,800——比你的目标价还低两百。体面砍价的复利到账。不过职场最难的开口，其实在没有议程的地方。",
				SayFail: "what if 要读出「提议」的味道，尾音在 any room then 轻轻上扬——你在递台阶，不是在逼宫。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "价格谈死就拿订量换空间", Reply: "对——what if we doubled the order：递台阶比逼让步高一档。"},
			{Label: "威胁转头去找别家", Reply: "最后通牒一出口，让价也成了结仇——筹码要递不要砸。再选。"},
			{Label: "对方说是底价就接受", Reply: "best we can do 是话术不是天花板——一句话缴械最亏。再想想。"},
		},
		Correct: 0,
		Clear:   "先给意向再报数字，谈死了就换筹码——两千美元砍得体体面面。下一关「茶水间闲聊」——没有议程的对话，才最考验开口的功夫。",
		Note:    "体面砍下了两千美元",
	},
	"pantry-smalltalk": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句接住）",
				Options: []nodeOption{
					{Label: "Pretty busy, but good! How about you—how's your week?", Reply: "对方立刻靠在台边接话：\"Same here! We just shipped something big.\" ——闲聊的秘密不在答得多精彩，而在把球抛回去：How about you 四个词，对话就死不了。半截话 busy but good 反而最像真人。", Next: 1},
					{Label: "Very busy. My this week have many works.", Reply: "\"...Works?\" 对方礼貌地眨了眨眼。三连坑：my this week 是中文语序，直接说 this week；work 不可数，many works 成了「许多工厂」；最可惜的是句号收尾没回球——天就是这么聊死的。", Next: NodeRetry},
					{Label: "My week has been highly productive, thank you for asking.", Reply: "对方端咖啡的手顿了半秒——highly productive、thank you for asking 是述职报告的词，茶水间里听着像 AI 客服上身。闲聊要松弛：耸耸肩来一句 busy but good，比满分作文可爱多了。", Next: NodeRetry},
				},
			},
			{
				Say:     "Pretty busy, but good! How about you—how's your week?",
				SayOK:   "对方顺势打开了话匣子，从项目吐槽到咖啡机——你没说什么金句，只是把球回了过去，闲聊就成了。不过话题马上会拐进你不熟的领域。",
				SayFail: "这句要读得松：Pretty busy, but good 像耸肩，How about you 像把咖啡递过去。",
				SayNext: 2,
			},
			{
				Prompt: "对方聊起周末：\"We went hiking up in the hills—amazing views!\" ——别让话掉地上，追问一个细节再分享你自己的，三句挑一句：",
				Options: []nodeOption{
					{Label: "Oh, hiking. That's nice.", Reply: "对方\"Yeah...\"了一声，话题当场躺平——That's nice 是闲聊的句点符，不是逗号。救活它只需要一个具体的追问：Where did you go? How long was the trail? 细节才是话题的氧气。", Next: NodeRetry},
					{Label: "Climb mountain is very tired. I sleep in home weekend.", Reply: "\"...Climb mountain?\" 两个直译坑：口语的爬山是 go hiking，climb mountain 听着像要登珠峰；「累」的分工也错了——活动是 tiring，人才是 tired；in home 该是 at home。先接对方的球，再说自己。", Next: NodeRetry},
					{Label: "Oh nice, where did you go? I binged a show all weekend.", Reply: "对方兴奋地掏出手机给你看照片，还反问你看的什么剧——一个追问加一句自我暴露，话题就有了两条腿。顺带说一句：binge a show（刷剧）这个词一出口，你的口语立刻年轻五岁。读出来。", Next: 3},
				},
			},
			{
				Say:     "Oh nice, where did you go? I binged a show all weekend.",
				SayOK:   "五分钟后你俩已经互推剧单了——隔壁组这位，从此就是你在公司里的「自己人」。人脉这东西，一半是在茶水间聊出来的。最后一关，跨国信号在等你。",
				SayFail: "where did you go 语调上扬，要有真好奇的味道；binged 读成一个音节，别拆开。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "回一句 That's nice 就够", Reply: "That's nice 是句点不是逗号——话题当场躺平。再选。"},
			{Label: "跳过对方直接聊自己", Reply: "不接球就先发球，对方的分享全落了空——闲聊是接力不是抢跑。再想想。"},
			{Label: "追问一个细节再分享自己", Reply: "对——细节是话题的氧气，追问加自我分享，对话就有了两条腿。"},
		},
		Correct: 2,
		Clear:   "把球抛回去、用细节续命，一场没冷场的闲聊。最后一关「跨国视频会」——信号卡成机器人声，听不清还硬撑才是大事故。",
		Note:    "茶水间聊出个自己人",
	},
	"video-call-clarify": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句打断对方）",
				Options: []nodeOption{
					{Label: "What? I can't hear you. Say it again.", Reply: "整个会议室都听见了这声\"What?\"——单独一个 What 在英语里相当于「啥?!」，自带火气。听不清是信号的锅，措辞却是你的：一句 you're breaking up 把责任交给信号，谁都体面。", Next: NodeRetry},
					{Label: "Sorry, you're breaking up—could you repeat the last part?", Reply: "对方立刻放慢重讲：\"Sorry about that—I said the deadline is...\" ——you're breaking up 是线上会议的救场神句：把问题归给信号而不是任何人；repeat the last part 精确指定要哪一段，不必让对方从头再来。", Next: 1},
					{Label: "Sorry, my listening is not good, please say slowly.", Reply: "一片好心，坑了自己——my listening is not good 把信号问题揽成了英语水平问题，对方从此跟你说话都像哄小孩。顺带一个语法点：say slowly 缺宾语，该是 say it slowly。不过这句压根不用说，报信号就行。", Next: NodeRetry},
				},
			},
			{
				Say:     "Sorry, you're breaking up—could you repeat the last part?",
				SayOK:   "对方一字一句重讲了一遍，还特意开了摄像头确认你跟上了——打断得体面，配合就痛快。可关键的那个日期，破音里你还是没抓牢。",
				SayFail: "breaking up 两个词连读要顺，could you repeat 语调放软——打断别人时，语气就是礼貌本身。",
				SayNext: 2,
			},
			{
				Prompt: "对方讲完了，但那个截止日期你还是没把握——15 号还是 50 号？用一句话既确认又兜底，三句挑一句：",
				Options: []nodeOption{
					{Label: "Just to confirm, did you say the 15th? I'll email a recap.", Reply: "对方确认：\"Yes, the fifteenth.\" 还补了句 \"Good idea on the recap.\" ——Just to confirm 把「我没听清」翻译成「我在做确认」，专业感瞬间反转；再提议邮件纪要，日期从此丢不了。读出来。", Next: 3},
					{Label: "OK, got it, no problem.", Reply: "你其实根本没 got it——fifteen 和 fifty 在破音质里就是一对孪生陷阱，装懂混过会议，错过截止日才是真事故。听不清就确认，这不丢人；丢人的是两周后交错日期。", Next: NodeRetry},
					{Label: "You say the date again, I write it down.", Reply: "\"...OK?\" 对方愣了一下。You say the date again 是「你再说一遍」的中文语序，光板祈使句在英语里像点名训话——加个 could 立刻回暖：Could you say the date again? 不过确认句式 did you say...? 更省一步。", Next: NodeRetry},
				},
			},
			{
				Say:     "Just to confirm, did you say the 15th? I'll email a recap.",
				SayOK:   "会后你的纪要邮件第一个发出，对方秒回：\"Thanks—super clear.\" ——全场听得最费劲的人，成了信息最准的人。职场七关，到此通关。",
				SayFail: "fifteenth 的 th 要咬住舌尖——这个音一含糊，15 和 50 就真的分不清了。",
				SayNext: NodeClear,
			},
		},
		CheckOpts: []tapOption{
			{Label: "没听清就先装懂混过去", Reply: "fifteen 还是 fifty？装懂的代价是两周后的事故。再选。"},
			{Label: "确认日期，再提议补纪要", Reply: "对——did you say 确认，邮件纪要兜底：听不清的人反而最准。"},
			{Label: "让对方把整段从头再讲", Reply: "全员陪你重听一遍，会议时间不答应——精确指定要哪段就够。再想想。"},
		},
		Correct: 1,
		Clear:   "确认加纪要，听不清的人反而信息最准——职场七关通关！下一主题「面试」开幕：Tell me about yourself，60 秒定第一印象。",
		Note:    "破音里确认了截止日",
	},
}
