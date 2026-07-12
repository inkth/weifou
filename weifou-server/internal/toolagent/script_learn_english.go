// 英语反应力（spoken-english）32 关剧本：日常办事 / 旅行应急 / 职场协作 / 商务实战，
// 每幕末一场「全英模拟面」Boss（slug 前缀 boss-，小程序端据此挂 Boss 标记）。
// 常规关形态 = 两轮纯点选场景对话：第一轮裁决自然、得体、能办成事的表达；第二轮换信息或
// 加压迁移，仍能接住才点亮。模拟面形态 = 听力门（🎧 只听不看，见 listenMark）→ 两轮全英
// 裁决（混考本幕语块）→ 拼句两步（把本幕金句自己拼出来）。全程零 LLM、零强制录音。
// 品质纪律（课魂+测试守护）：
//   - 错误项的点破要具体到语言点（a/an、时态、语域），不是泛泛「不对」；
//   - 对方反应先用英文演一句，再中文点破——沉浸不断，教学不糊；
//   - 通关只代表完成场景迁移；延时复习答对才升「掌握」；
//   - CheckOpts 供复习挑战（中文判断题，Label ≤20 字）；Clear 带下一关悬念；Note ≤18 字。
package toolagent

// listenMark 听力门标记行：模拟面第一节点用它引出一句「只播不显」的英文——
// 小程序端（agent-chat）按同名约定把标记行的下一行隐藏为音频占位，自动朗读、🔊 可重听。
const listenMark = "🎧 只听不看："

// listenLine 组一段听力门：标记行 + 隐藏英文行。答错的点破里也用它，让学员当场重听。
func listenLine(en string) string { return listenMark + "\n" + en }

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
				Prompt: "端上来的却是杯美式，店员还挺开心：\"Here's your americano!\" ——礼貌指出问题并要求重做，三句挑一句：",
				Options: []nodeOption{
					{Label: "This is wrong. Change it.", Reply: "店员愣住，周围安静了一秒——\"...Oh. OK.\" 单子是能换，你也成了全店最凶的客人。指错的万能缓冲是先 Sorry / Excuse me 一句，火药味立刻归零。", Next: NodeRetry},
					{Label: "Sorry, I think this is an americano—I ordered an oat latte.", Reply: "店员一拍脑门：\"Oh no, my bad! Let me remake that for you.\" ——Sorry 开头缓冲、I think 留有余地、说清你点的是什么：指错三件套，一句全齐。记住这个结构。", Next: NodeClear},
					{Label: "Oh... never mind, it's OK.", Reply: "你端走了那杯不想喝的美式——最熟悉的忍。注意：英语里 it's OK 在这个场景就是「算了不用改」，店员真不会改。点单白点，钱白花，开口的机会也让出去了。", Next: NodeRetry},
				},
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
				Prompt: "牛排端上来，切开一看还渗着血水——太生了。服务生正好路过：\"How is everything?\" ——礼貌地请他拿回去再煎熟一点，三句挑一句：",
				Options: []nodeOption{
					{Label: "This steak is raw. Take it back.", Reply: "服务生僵在原地，邻桌都看了过来——\"...Right away.\" 盘子是收走了，气氛也冷了。raw 是全生才用的词，牛排偏生说 undercooked 或 too rare；更要命的是没有缓冲直接祈使——先来一句 Sorry 或 Excuse me，问题才好办。", Next: NodeRetry},
					{Label: "Sorry, this steak is too tender. Please burn it more.", Reply: "服务生一脸迷惑：\"Too... tender? And you want it burned?\" ——tender 是「嫩得恰到好处」，是夸厨师的词；burn 是烧焦。你想说的「太生、再煎一下」是 undercooked 和 cook it a bit more，两个假朋友一换，意思全反了。", Next: NodeRetry},
					{Label: "Sorry, my steak is undercooked—could you cook it a bit more?", Reply: "服务生连声道歉端起盘子：\"Oh, I'm so sorry—I'll get that fixed right away.\" ——Sorry 缓冲、undercooked 说清问题、could you 提出请求：一句话三步走，事办成了，体面也在。记住这个结构。", Next: NodeClear},
				},
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
					{Label: "Hello, my name is Lin. I am 28 and my job is designer.", Reply: "对方的笑容礼貌地凝固了半秒：\"Oh... nice.\" ——两个问题：my job is designer 缺冠词，该说 I'm a designer；更大的问题是寒暄一上来自报年龄职业，像在念简历。聚会破冰只要名字，加一个抛回去的话头。", Next: NodeRetry},
					{Label: "Oh, sorry... my English is not very good.", Reply: "对方赶紧安慰：\"No no, you're doing great!\" ——话题瞬间从「认识你」变成「安慰你」。自贬开场是中文式谦虚的假朋友，英语寒暄里它只会让对方不知道接什么，破冰变成了救场。", Next: NodeRetry},
					{Label: "Hi, I'm Lin. Nice to meet you—how do you know the host?", Reply: "对方眼睛一亮：\"Oh, we went to college together!\" ——名字、一句客套、一个问题，破冰三件套齐了。把话题抛回去，是让对话活下来的第一原则。记住这个结构。", Next: 1},
				},
			},
			{
				Prompt: "对方握完手自报家门：\"I'm Sam, by the way. I'm from Melbourne.\" ——别让话掉在地上，用一个跟进问题让对话继续：",
				Options: []nodeOption{
					{Label: "Oh, cool! What brought you here from Melbourne?", Reply: "Sam 打开了话匣子：\"Work, actually—but I stayed for the food!\" ——What brought you here 是万能跟进问题：接住对方给的信息，再把它变成下一个话题。记住这个结构。", Next: NodeClear},
					{Label: "Oh. Melbourne. Good.", Reply: "Sam 等了两秒，没等到下文：\"...Yeah.\" ——三个句号，三次终结。信息接住了却没回球，对话在你手里安静地死掉了。跟进问题哪怕只有一个 What's it like，也比 Good 强。", Next: NodeRetry},
					{Label: "Melbourne? I have been to there last year.", Reply: "Sam 听懂了，但你自己卡了一下——been to there 里 to 和 there 撞车，只能说 been there；而且 last year 是明确的过去时间，要用过去式 I went there last year，现在完成时不跟具体时间点。", Next: NodeRetry},
				},
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
					{Label: "Do you have this jacket in a medium? I'd like to try it on.", Reply: "店员利落转身：\"Sure! The fitting room is right over there.\" ——Do you have this in a medium 是买衣服的万能句型，颜色尺码全能套；I'd like to 提请求，礼貌又干脆。记住这个结构。", Next: 1},
					{Label: "This clothes is nice. Have M size? I can try?", Reply: "店员听懂了，但句句硌耳朵——clothes 永远是复数，单件外套说 this jacket；Have M size 缺了主语和冠词，完整问法是 Do you have this in a medium；I can try? 也得倒装成 Can I try it on。", Next: NodeRetry},
					{Label: "Would you be so kind as to provide this garment in medium?", Reply: "店员愣了一下才反应过来：\"...Of course.\" ——garment 是吊牌和合同上的词，provide 像在下采购单；买衣服的日常语域就是 Do you have this in a medium。客气抬到书面语，反而像隔着柜台念公文。", Next: NodeRetry},
				},
			},
			{
				Prompt: "试衣间出来，袖子长出一截——大了一号。店员迎上来：\"How does it fit?\" ——一次问清两件事：有没有小一码、不合适能不能退换：",
				Options: []nodeOption{
					{Label: "It's too big. Go get me a smaller one.", Reply: "店员脸上的笑淡了：\"...I'll check.\" ——Go get me 是使唤人的句式，配上 too big 的抱怨腔，你从客人变成了甲方。指使和请求之间隔着一个 Could you，这个距离决定服务的温度。", Next: NodeRetry},
					{Label: "It's a bit big—do you have a smaller size? Can I return it?", Reply: "店员答得干脆：\"We have a small—and yes, fourteen-day returns with the receipt.\" ——两个问题一口气问全，信息一次拿齐，这才是高效的开口。记住这个结构。", Next: NodeClear},
					{Label: "A little big. Can you change a small one for me?", Reply: "店员会意了，但 change 用岔了——英语里 change 是找零钱、换衣服（change clothes），换尺码换货要用 exchange，或者直接问 do you have a smaller size。中文的「换」一词多义，英语拆成了两个词。", Next: NodeRetry},
				},
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
					{Label: "Excuse me, how do I get to the subway? How long is the walk?", Reply: "路人停下脚步：\"Oh sure—it's about ten minutes that way.\" ——Excuse me 是搭话的入场券，how do I get to 是问路的万能句型，末尾再确认路程，一句不多一句不少。记住这个结构。", Next: 1},
					{Label: "Hey, tell me where the subway is.", Reply: "路人皱了下眉，脚步没停：\"Uh... that way?\" ——Hey 加祈使句 tell me，对陌生人像开口盘问。向陌生人求助，开场白只有一个标准答案：Excuse me。少了它，对方给个大概方向就想走。", Next: NodeRetry},
				},
			},
			{
				Prompt: "路人热心开讲，语速却快得像贯口：\"Go down two blocks, turn left, and then it's—\" 后半句全糊了，你只抓住一个 left。请 TA 说慢一点，并把听到的路线复述确认：",
				Options: []nodeOption{
					{Label: "Sorry, could you say it more slowly? Two blocks, then left?", Reply: "路人放慢重来：\"Right—two blocks, turn left, and it's on your right.\" ——请对方减速，加上复述确认，听力缺口当场补上。听不清就问，本来就是听懂的一部分。记住这个结构。", Next: NodeClear},
					{Label: "Sorry, I can't catch you. Say again, please.", Reply: "路人愣了一下——catch you 听起来像「抓不住你这个人」，没听清要说 I didn't catch that；Say again 缺了宾语也少了缓冲，完整版是 Could you say that again。差两个小词，礼貌和意思才都齐。", Next: NodeRetry},
					{Label: "OK, OK, I got it. Thanks!", Reply: "路人满意地走了，你站在原地，手里还是只有那个 left——装懂是听力最大的敌人：这一句省下的三十秒，等会儿要在岔路口加倍还。没听清就开口确认，没人嫌你慢。", Next: NodeRetry},
				},
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
					{Label: "Hi, I'd like to book a table for four at seven tonight.", Reply: "接线员马上去查：\"For four at seven—let me check that for you.\" ——I'd like to book a table for four at seven，人数时间一句打包，电话订位的标准开场就是它。记住这个结构。", Next: 1},
				},
			},
			{
				Prompt: "接线员查完面露难色：\"I'm sorry, seven is fully booked tonight. We have six or eight thirty.\" ——选一个时间，并把名字和电话留下：",
				Options: []nodeOption{
					{Label: "OK, eight thirty then. My name is called Lin.", Reply: "接线员记下了，但那句自我介绍拧着——is called 用在物件和绰号上，自报姓名就是 My name is Lin 或 I'm Lin。「我叫」直译成 is called，是中式英语的老熟人了。", Next: NodeRetry},
					{Label: "Eight thirty works. The name is Lin—I'll leave my number.", Reply: "接线员一路确认：\"Eight thirty, party of four, under Lin—and your number, please?\" ——选定时间、报上名字、主动留电话，订位三要素闭环，这一单跑不了。记住这个结构。", Next: NodeClear},
					{Label: "Oh, either is fine... you decide for me, it's OK.", Reply: "电话那头等着：\"So... which one shall I put down?\" ——客气过了头就是没答复：时间没定、名字电话没留，这通电话挂了等于没打。订位要给确定的答案，礼貌不等于把决定推回去。", Next: NodeRetry},
				},
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
					{Label: "I've had a headache for two days, and I have a slight fever.", Reply: "医生边听边记：\"Two days, with a fever—OK, let's take a look.\" ——症状加持续时间，问诊最想要的两样你一句给全；I've had... for two days 这个现在完成时，就是「已经疼了两天」的标准说法。记住这个结构。", Next: 1},
					{Label: "My head is very pain since two days. I have a little hot.", Reply: "医生听懂了七成，但三个词都拧着——pain 是名词，「头疼」说 my head hurts；since 后面跟时间点，「两天了」是 for two days；hot 是烫不是发烧，发烧要说 fever。诊室里，词不准，信息就打折。", Next: NodeRetry},
					{Label: "Um, I don't feel very well... it's nothing serious, I guess.", Reply: "医生的笔停在半空：\"OK... can you be more specific?\" ——诊室不是客气的地方，it's nothing 这种谦虚会把问诊拖成猜谜。医生要的就两样：哪里不舒服、持续多久。把它们直接递过去。", Next: NodeRetry},
				},
			},
			{
				Prompt: "医生边写处方边交代：\"Take one twice a day after meals.\" ——先复述你听懂了什么，再追问一句：有什么要忌口的吗：",
				Options: []nodeOption{
					{Label: "OK, so I eat two pills one time every day, right?", Reply: "医生赶紧摆手：\"No no—one pill, twice a day.\" ——twice a day 是一天两次、每次一片，不是一次两片。复述恰好把听岔的地方当场暴露了——这正是复述的价值；另外吃药不用 eat，英语说 take medicine。", Next: NodeRetry},
					{Label: "Yes, yes, I understand. Thank you, doctor. Bye!", Reply: "医生想叮嘱的话被你的 Bye 关在了门里——医嘱是全场最不能装懂的一段：没复述、没确认，出了诊室剂量就开始模糊。听完医嘱的标准动作是复述一遍，再把疑问当场问完。", Next: NodeRetry},
					{Label: "Twice a day after meals, right? Any food I should avoid?", Reply: "医生点头，又补了叮嘱：\"Correct. And yes—I'll write it all down for you.\" ——复述确认加主动追问忌口，医嘱这道听力题你拿了满分。剩下的，按医生说的做就行。记住这个结构。", Next: NodeClear},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "复述成一次吃两片", Reply: "twice a day 是一天两次、每次一片——复述错了剂量就危险了。再选一个。"},
			{Label: "复述一遍再问忌口", Reply: "对——复述把听岔当场暴露，追问把疑问当场清零，医嘱不带回家猜。"},
			{Label: "连声yes道谢就走", Reply: "医嘱是最不能装懂的一段，出门剂量就开始模糊。再想想。"},
		},
		Correct: 1,
		Clear:   "症状说清、医嘱确认、忌口问到——日常七关全部点亮。先别松劲：第一幕还剩一场「全英模拟面」，没有中文旁白，全靠你自己。",
		Note:    "开口把症状说清了",
	},
	// ============ 旅行 ============
	"flight-checkin": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句回地勤）",
				Options: []nodeOption{
					{Label: "Yes, just one bag. Could I get a window seat, please?", Reply: "地勤接过箱子贴上行李条：\"One bag—and window seat, let me see... done!\" ——一句话办成两件事：先直接回答问题（one bag），再用 Could I get 提请求，值机口语的标准节奏。现在记住这个结构。", Next: 1},
					{Label: "I have one baggage to check, and I want to sit window.", Reply: "地勤顿了一下：\"...One bag, you mean?\" 两个坑：baggage 是不可数名词，一件行李是 one bag 或 one piece of baggage；「坐窗边」是 a window seat，sit window 是中式直译，座位不是拿来坐窗户的。", Next: NodeRetry},
					{Label: "Check this. And window seat. Quick, OK?", Reply: "地勤挑了下眉，动作反而慢了半拍：\"...Sure.\" 每个词都听得懂，但全程省略句再加一个 Quick，像在催下属干活。柜台提请求，一句 Could I get 开头，礼貌到位，效率反而更高。", Next: NodeRetry},
				},
			},
			{
				Prompt: "刚要转身，地勤叫住你：\"Excuse me—this flight is overbooked. Would you take the next flight for a 200-dollar voucher?\" ——改签可以谈，但别白改，三句挑一句：",
				Options: []nodeOption{
					{Label: "Oh, OK... whatever you think is best. I don't mind.", Reply: "地勤立刻打出了新登机牌：\"Great, thank you!\" ——你被改签了，代金券一分没多，连下一班几点都没问。whatever you think 在谈判场景等于弃权：先问信息、再提条件，主动权才在你手里。", Next: NodeRetry},
					{Label: "If I change, can you give me more 100 dollars?", Reply: "地勤听懂了，忍着笑确认：\"You mean... 100 more?\" ——数量词的语序是 100 more dollars，more 跟在数字后面；more 100 是中式排法。还价的方向对了，句子得先立住。", Next: NodeRetry},
					{Label: "Maybe. What time is the next flight? Could you make it 300?", Reply: "地勤跟主管低声确认后回来：\"We can do 300.\" ——先问下一班几点（拿信息），再用 Could you make it... 还价（提条件），不冲也不软。这句话值一百美元，记住这个结构。", Next: NodeClear},
				},
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
					{Label: "Tourism. I'm staying ten days at my friend's place.", Reply: "官员点点头，在系统里敲了几下：\"Enjoy your stay.\" ——目的、天数、住处，三连问一句收齐：Tourism 直答，staying ten days 报时长，friend's place 所有格不丢。记住这三个信息位。", Next: 1},
				},
			},
			{
				Prompt: "官员盯着屏幕问：\"Are you carrying any food or plants?\" ——你包里有两盒送人的茶叶，三句挑一句：",
				Options: []nodeOption{
					{Label: "Yes, I have two boxes of tea leaves. Is that allowed?", Reply: "官员扫了一眼申报单：\"Tea is fine. Thanks for declaring.\" ——如实申报，再补一句 Is that allowed?，把判断交给官员，自己零风险。诚实在海关不只是美德，是最优策略。记住这个结构。", Next: NodeClear},
					{Label: "Yes, I bring two box of tea, can bring or not?", Reply: "官员放慢语速跟你确认：\"Two... boxes?\" ——两盒要加复数 boxes；can bring or not 是「能不能带」的中式直译，英语问许可说 Is that allowed? 或 Can I bring them in?，主语不能丢。", Next: NodeRetry},
					{Label: "No, nothing. Just clothes.", Reply: "官员指了指扫描仪：\"Then let's take a look.\" X 光下两盒茶叶清清楚楚——瞒报被查到，轻则没收，重则罚款留记录。茶叶本来能带，一句谎话把小事变成了大事。", Next: NodeRetry},
				},
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
					{Label: "I am Lin, I ordered a room in your hotel yesterday.", Reply: "前台笑着查了查：\"You... ordered?\" ——order 是点餐下单，订房要说 booked 或 I have a reservation；两个完整句用逗号硬连也是粘连句。一个动词用错，住客听着就成了外卖客。", Next: NodeRetry},
					{Label: "Yes, under Lin. Could I get a quiet room on a high floor?", Reply: "前台敲键盘的手没停：\"Room 1208—high floor, away from the elevator.\" ——under + 姓名是报预订的自然说法，a room on a high floor 表达楼层要求。两个愿望一句话，全中。记住这个结构。", Next: 1},
					{Label: "I hereby request accommodation under the name of Lin.", Reply: "前台愣了一下，差点站直了敬礼：\"...Certainly, sir?\" 语法满分，语域跑偏——hereby、request accommodation 是律师函措辞，前台不是法庭。口语订房，一句 under Lin 就够了。", Next: NodeRetry},
				},
			},
			{
				Prompt: "进房十分钟，空调怎么调都是热风。你拨通前台：\"Front desk, how can I help you?\" ——三句挑一句：",
				Options: []nodeOption{
					{Label: "This is unacceptable! I want a refund right now!", Reply: "前台连声道歉，但明显被将住了：\"I'm... sorry?\" ——空调坏了直接跳到退款，把小问题谈成了僵局。诉求要和问题同级：先说清故障，再给对方可选的动作，事情才动得起来。", Next: NodeRetry},
					{Label: "The air condition is bad, you quickly send people.", Reply: "前台猜了几秒：\"The... air conditioning?\" ——空调是 air conditioning 或 AC，air condition 少了 -ing 就不是那台机器了；quickly send people 是中式「快派人」，英语说 could you send someone up。", Next: NodeRetry},
					{Label: "Hi, the AC isn't working—could you fix it or move me?", Reply: "前台立刻应声：\"So sorry! Let me check what's available.\" ——先说清故障（isn't working），再给两个选项（fix it or move me），前台不用猜你要什么。有选项的投诉，才是好投诉。记住这个结构。", Next: NodeClear},
				},
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
					{Label: "The city museum, please—I'm in a hurry, so the fastest way.", Reply: "司机点头打了转向灯：\"Fastest way—got it.\" ——目的地加 please 报站，I'm in a hurry 给情况，so the fastest way 给指令。信息给足，司机不用猜。记住这个结构。", Next: 1},
				},
			},
			{
				Prompt: "导航明明指直行，司机却拐进了小路，计价表跳得飞快。三句挑一句：",
				Options: []nodeOption{
					{Label: "Sorry, why are we going this way? Could we follow the GPS?", Reply: "司机指了指前方：\"Construction on the main road—but sure, GPS it is.\" ——先问原因（也许真有路况），再提要求（follow the GPS），不指控也不吃闷亏。质疑的体面版，就长这样。记住这个结构。", Next: NodeClear},
					{Label: "Stop the car! You're cheating me. I'll call the police.", Reply: "司机猛地靠边急刹：\"Whoa—relax!\" ——还没问原因就定罪，万一前面真在修路，这一嗓子就下不来台了。先问 why，再提要求，指控永远放在最后一步。", Next: NodeRetry},
					{Label: "Driver, why you go this small road? Use GPS please.", Reply: "司机听懂了意思，但你自己听听——why you go 少了助动词，特殊疑问句要 why are we going；开头直呼 Driver 也生硬，换一句 Sorry 或 Excuse me，同样的话顺耳一倍。", Next: NodeRetry},
				},
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
					{Label: "Two adult tickets, please. Is there a student discount? What time do you close?", Reply: "售票员利落地敲着屏幕：\"Two adults—and yes, students get 20% off with ID.\" ——先报票数，再用两个自然问句把折扣和闭馆时间一次问齐，信息完整也不含糊。", Next: 1},
					{Label: "Two adult ticket. Student have discount? You close when?", Reply: "售票员逐个跟你确认：\"Two... tickets?\" ——三连坑：两张票 ticket 要加 s；主谓一致是 students get 或 Do students get；You close when 疑问词要提前加助动词：When do you close。", Next: NodeRetry},
					{Label: "I would like to inquire whether discounts are offered.", Reply: "售票员眨眨眼，后面队伍开始探头——\"...A discount? Yes.\" 这句放进邮件里满分，放在售票窗口太重：inquire whether 是书面语，窗口三秒一单，短句直问才是对的语域。", Next: NodeRetry},
				},
			},
			{
				Prompt: "刚拿到票，暴雨倾盆而下，你想改天再来。回到窗口，三句挑一句：",
				Options: []nodeOption{
					{Label: "Give me a refund. The rain is not my problem.", Reply: "售票员指了指窗口贴的告示：\"All sales are final.\" ——一开口就要退款还甩责任，对方立刻搬出规则挡你。规则内好商量的其实是改期：先问 change，别先谈钱。", Next: NodeRetry},
					{Label: "It's pouring—could I change my tickets to another day?", Reply: "售票员看了眼外面的雨，语气软了：\"Sure—any day this month works.\" ——退款难，改期易：could I change... to another day 给了对方一个不违反规则也能帮你的台阶。提请求，先挑对方答应得了的提。记住这个结构。", Next: NodeClear},
					{Label: "I want change my ticket to other day, rain too big.", Reply: "售票员大致听懂了：\"Change it, you mean?\" ——want 后面要接 to change；「改到别的日子」是 another day；「雨太大」英语说 it's pouring 或 heavy rain，rain too big 是把中文的尺寸搬进了英语。", Next: NodeRetry},
				},
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
					{Label: "It's a large blue suitcase with a red ribbon on the handle.", Reply: "工作人员边听边录入系统：\"Large, blue, red ribbon—got it.\" ——尺寸、颜色、特征一句排齐；with a red ribbon on the handle 这种细节，正是从几百个蓝箱子里认出你那只的钥匙。记住这个结构。", Next: 1},
					{Label: "It is a big size blue color box, very big.", Reply: "工作人员的笔尖停了停：\"A... box?\" ——big size、blue color 都是中式冗余，big 和 blue 自己就够了；行李箱是 suitcase，box 是纸箱——照这么登记，找回来的可能真是个纸箱。", Next: NodeRetry},
				},
			},
			{
				Prompt: "工作人员抬头：\"For the claim—what was inside?\" ——为了理赔，说出箱里三件重要物品和大致价值，三句挑一句：",
				Options: []nodeOption{
					{Label: "A laptop, clothes, and a camera—about 1,500 dollars total.", Reply: "工作人员在理赔栏里逐项登记：\"Fifteen hundred—noted.\" ——三件物品点名，about 给估值留了余地又不含糊。理赔表上写得清楚的人，赔付到账也快。记住这个结构。", Next: NodeClear},
					{Label: "Inside have computer, clothes and camera, very expensive.", Reply: "工作人员等你重说：\"Sorry—inside... what?\" ——中文「里面有」直译成 Inside have 是最经典的中式存在句，英语要说 There's a laptop inside，或干脆列清单；very expensive 也不如报个数字有用。", Next: NodeRetry},
					{Label: "Oh, just some old things... nothing important, I guess.", Reply: "工作人员如实录入：\"No significant contents.\" ——客气用错了地方：理赔单上写 nothing important，等于亲手把赔偿金归零。该谦虚的场合很多，报损失清单不在其中。", Next: NodeRetry},
				},
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
					{Label: "I lost my passport on the subway this morning. Can you help?", Reply: "警察立刻抽出一张登记表：\"Subway, this morning—let's file a report.\" ——lost 过去式定性，地点时间各就各位，Can you help 直接请求。越慌的时候，越短的句子越可靠。记住这个结构。", Next: 1},
				},
			},
			{
				Prompt: "警察递来表格：\"Fill this out. Do you want to contact your embassy?\" ——你确实得去趟大使馆，三句挑一句：",
				Options: []nodeOption{
					{Label: "Where is Chinese embassy? What time it opens?", Reply: "警察听懂了，但两处得修：国名加机构前要有 the——the Chinese embassy；疑问句语序是 What time does it open，助动词提到主语前，What time it opens 是陈述句的排法。", Next: NodeRetry},
					{Label: "How do I get to the Chinese embassy? And when does it open?", Reply: "警察在便签上画了张小地图：\"Two stops on Line 3—opens at nine.\" ——How do I get to... 是问路的万能句型，when does it open 把开门时间一并问到。一张便签，就是你明天的路线图。记住这个结构。", Next: NodeClear},
					{Label: "Take me to the embassy right now. It's your job.", Reply: "警察挑了挑眉，把表格往你面前又推了推：\"My job is this form.\" ——警察帮忙是情分，命令句加一句 It's your job，把情分聊成了对峙。你需要的是路线，不是输赢。", Next: NodeRetry},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "问路用 How do I get to", Reply: "对——问路万能句型，再补一句 when does it open，一次问全。"},
			{Label: "time it opens 是疑问语序", Reply: "那是陈述句排法——疑问句要 What time does it open。再想想。"},
			{Label: "让警察现在就送你去", Reply: "帮忙是情分不是义务——命令句把求助聊成对峙。再选。"},
		},
		Correct: 0,
		Clear:   "护照丢了也没慌，求助、问路、问时间全用英语办妥。旅行的硬仗还剩一场——「全英模拟面·旅途」：航班取消的那一夜，没有旁白替你翻译。",
		Note:    "丢了护照没丢开口",
	},
	// ============ 职场 ============
	"work-self-intro": {
		Nodes: []scriptNode{
			{
				Prompt: "（三句英文，挑一句向全组开口）",
				Options: []nodeOption{
					{Label: "It is a great honor to join this esteemed organization.", Reply: "同事们礼貌地鼓了掌，但没人记住你——\"Welcome... aboard?\" 这句像年会致辞不像打招呼：esteemed organization 是留给演讲稿的词。团队自我介绍要让人接得上话——名字、做什么、一点个人色彩，说人话就赢了。", Next: NodeRetry},
					{Label: "Hi, I'm Lin. I'll be working on design—and I hike a lot.", Reply: "话音刚落就有人接茬：\"Oh nice, we have a hiking group!\" ——名字、负责什么、一点个人色彩，三件套一句装下；最后那点「个人色彩」就是钩子，别人想认识你，总得有个话头。记住这个结构。", Next: 1},
					{Label: "My name is Lin. I am work on design. Nice to meet you.", Reply: "几位同事点头微笑，接着小声互相确认：\"...He does what?\" I am work 把 be 动词和实义动词撞在了一起——要么 I work on design，要么 I'll be working on design。开口用 I'm Lin 也比 My name is 更像口语。", Next: NodeRetry},
				},
			},
			{
				Prompt: "散会后一位同事凑过来：\"So how are you finding it so far?\" ——接住话头，再顺势约杯咖啡，三句挑一句：",
				Options: []nodeOption{
					{Label: "I'm fine, thank you. And you?", Reply: "同事笑容僵了半秒，摆摆手走了——\"...Good, good.\" 教科书三件套答非所问：How are you finding it 问的是「感觉如何」，不是「你好吗」。答感受才接得住：Really good，或者 A bit overwhelming, but exciting。", Next: NodeRetry},
					{Label: "All is good. I will invite you drink coffee.", Reply: "同事听懂了，但眉毛动了一下——invite you drink 少了个 to（invite sb to do sth）；而且 I will invite you 像在发正式请柬，对方还得琢磨要不要回礼。英语约咖啡轻得多：Want to grab a coffee?", Next: NodeRetry},
					{Label: "Really good! Want to grab a coffee and chat about the team?", Reply: "同事眼睛一亮：\"Sure! There's a great place downstairs.\" ——先答感受，再用 Want to... 发出零压力邀请；grab a coffee 是职场社交的硬通货，比 have coffee 随性，比 invite 轻巧。记住这个结构。", Next: NodeClear},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "答感受，再轻邀请喝咖啡", Reply: "对——How are you finding it 要答感受，接一句 grab a coffee，关系就此破冰。"},
			{Label: "答 I'm fine 就完事", Reply: "教科书三件套答非所问——对方问的是感受，不是问候。再想想。"},
			{Label: "郑重宣布要请喝咖啡", Reply: "I will invite you 太重像请柬，还少了个 to——轻轻一句 grab a coffee 才对味。再选。"},
		},
		Correct: 0,
		Clear:   "名字、职责、个人色彩，一句话让全组记住你。下一关「茶水间闲聊」——没有议程的对话，才最考验接话。",
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
				Prompt: "另一位同事抛出的新方案你完全不同意。老板转头看你：\"Thoughts?\" ——既要反对，又要给出路，三句挑一句：",
				Options: []nodeOption{
					{Label: "I see your point, but could we try a phased rollout instead?", Reply: "同事没恼，反而追问：\"Interesting—how would that work?\" ——I see your point 先接住对方，could we try... instead 把反对包装成提议：你否的是方案，捧的是讨论，这就是不伤和气的全部秘密。记住这个结构。", Next: NodeClear},
					{Label: "That won't work. My idea is better.", Reply: "同事的脸沉了下来，接下来十分钟他都在防御而不是讨论——反对最忌把「方案不行」说成「你不行」，My idea is better 更是火上浇油。先给一句 I see your point，火力立刻减半。", Next: NodeRetry},
					{Label: "I have a different opinion. We can do like this.", Reply: "\"...Like what?\" 同事一脸困惑。do like this 是「这样做」的直译，英文得说 do it this way，或者干脆把方案说出来；I have a different opinion 也硬得像开辩论。一句 but 带出替代方案就够了。", Next: NodeRetry},
				},
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
				Prompt: "同事忽然想起什么：\"Sure, but what if the client emails me directly?\" ——哪些 TA 自己定、哪些必须找你，三句挑一句划清楚：",
				Options: []nodeOption{
					{Label: "If have problem, you can call me every time.", Reply: "\"...Every time?\" 同事有点懵。两个坑：if have problem 丢了主语，得说 if there's a problem；every time 是「每一次」，你想说的「随时」是 anytime。更要命的是事事找你——那这趟交接等于没交。", Next: NodeRetry},
					{Label: "Use your judgment—but if it's about pricing, call me.", Reply: "同事竖起大拇指：\"Crystal clear—judgment calls on me, pricing goes to you.\" ——use your judgment 是授权，if it's about pricing 是红线：一句话切清「哪些你定、哪些找我」，这才叫交接完毕。记住这个结构。", Next: NodeClear},
					{Label: "Just do whatever you think is right, I trust you.", Reply: "听着大方，其实是把锅整个递了过去——万一客户要改报价，TA 也「自己看着办」吗？授权不等于放羊：不划红线的信任，回来要用加班来还。", Next: NodeRetry},
				},
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
				Prompt: "讲到第二部分，有人举手打断：\"Sorry, can you go back to the numbers?\" ——接住提问、澄清数据、再拉回主线，三句挑一句：",
				Options: []nodeOption{
					{Label: "Sure—this is Q3 revenue, up 8%. Now, back to the roadmap.", Reply: "提问的人边点头边记：\"Got it, thanks.\" 全场节奏没散——Sure 一秒接住，一句话澄清数字，Now, back to... 把方向盘稳稳拿回来。被打断不可怕，回不到主线才可怕。记住这个结构。", Next: NodeClear},
					{Label: "Please let me finish first. Questions at the end.", Reply: "对方缩回了手，脸上有点挂不住——把提问者噎回去，后面就再没人敢互动了，而没人互动的汇报和群发邮件没区别。先花十秒回应，再用 back to 拉回，两头都不丢。", Next: NodeRetry},
					{Label: "Ah sorry, wait me a moment, I find that page.", Reply: "\"...Wait you?\" wait me 是「等我」的直译，英语得说 give me a second 或 bear with me；后半句也该是 let me find that page。与其手忙脚乱翻页，不如一句稳稳的 Sure 先把场接住。", Next: NodeRetry},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "让提问的人等到最后再问", Reply: "问题会凉，人也会凉——十秒能答的当场答，互动是汇报的呼吸。再选。"},
			{Label: "十秒澄清，再拉回主线", Reply: "对——Sure 接住、一句澄清、Now, back to 拉回：方向盘始终在你手里。"},
			{Label: "停下来慢慢翻页找那张表", Reply: "全场看你翻三十秒页，气就泄光了——记不清就先接住再补。再想想。"},
		},
		Correct: 1,
		Clear:   "开场给地图、打断能拉回，整场汇报的方向盘都在你手里。下一关「延期与风险」——坏消息怎么说才不只是甩出问题？",
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
				Prompt: "对方摊开手：\"That's really the best we can do.\" ——价格谈死了，换个筹码再谈，三句挑一句：",
				Options: []nodeOption{
					{Label: "If you not give discount, we find another company.", Reply: "对方脸色冷了下来——if you not give 丢了助动词，该是 if you don't give；更伤的是当面亮「找别家」，谈判瞬间变最后通牒：就算对方让了价，心里也记你一笔，后面的合作全是刺。", Next: NodeRetry},
					{Label: "Understood. What if we doubled the order—any room then?", Reply: "对方眼睛一亮：\"Now we're talking.\" ——价格谈死就换筹码，what if 是谈判的万能钥匙：不逼对方让步，而是递给对方一个让步的台阶。量、账期、附赠服务，都是价格之外的牌。记住这个结构。", Next: NodeClear},
					{Label: "OK, I understand. 12,000 is fine then.", Reply: "对方内心已经开香槟了——\"best we can do\" 是谈判话术不是终点，一句话就缴械，预算照样超，对方还会觉得你刚才的压价没诚意。价格锁死了，手里还有量、账期、服务三张牌呢。", Next: NodeRetry},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "价格谈死就拿订量换空间", Reply: "对——what if we doubled the order：递台阶比逼让步高一档。"},
			{Label: "威胁转头去找别家", Reply: "最后通牒一出口，让价也成了结仇——筹码要递不要砸。再选。"},
			{Label: "对方说是底价就接受", Reply: "best we can do 是话术不是天花板——一句话缴械最亏。再想想。"},
		},
		Correct: 0,
		Clear:   "先给意向再报数字，谈死了就换筹码——价格没有裸降，条件也谈清了。最后一关把共识推进成行动。",
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
				Prompt: "对方聊起周末：\"We went hiking up in the hills—amazing views!\" ——别让话掉地上，追问一个细节再分享你自己的，三句挑一句：",
				Options: []nodeOption{
					{Label: "Oh, hiking. That's nice.", Reply: "对方\"Yeah...\"了一声，话题当场躺平——That's nice 是闲聊的句点符，不是逗号。救活它只需要一个具体的追问：Where did you go? How long was the trail? 细节才是话题的氧气。", Next: NodeRetry},
					{Label: "Climb mountain is very tired. I sleep in home weekend.", Reply: "\"...Climb mountain?\" 两个直译坑：口语的爬山是 go hiking，climb mountain 听着像要登珠峰；「累」的分工也错了——活动是 tiring，人才是 tired；in home 该是 at home。先接对方的球，再说自己。", Next: NodeRetry},
					{Label: "Oh nice, where did you go? I binged a show all weekend.", Reply: "对方兴奋地掏出手机给你看照片，还反问你看的什么剧——一个追问加一句自我暴露，话题就有了两条腿。顺带说一句：binge a show（刷剧）这个词一出口，你的口语立刻年轻五岁。记住这个结构。", Next: NodeClear},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "回一句 That's nice 就够", Reply: "That's nice 是句点不是逗号——话题当场躺平。再选。"},
			{Label: "跳过对方直接聊自己", Reply: "不接球就先发球，对方的分享全落了空——闲聊是接力不是抢跑。再想想。"},
			{Label: "追问一个细节再分享自己", Reply: "对——细节是话题的氧气，追问加自我分享，对话就有了两条腿。"},
		},
		Correct: 2,
		Clear:   "把球抛回去、用细节续命，一场没冷场的闲聊。下一关「跨国视频会」——信号卡成机器人声，听不清还硬撑才是大事故。",
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
				Prompt: "对方讲完了，但那个截止日期你还是没把握——15 号还是 50 号？用一句话既确认又兜底，三句挑一句：",
				Options: []nodeOption{
					{Label: "Just to confirm, did you say the 15th? I'll email a recap.", Reply: "对方确认：\"Yes, the fifteenth.\" 还补了句 \"Good idea on the recap.\" ——Just to confirm 把「我没听清」翻译成「我在做确认」，专业感瞬间反转；再提议邮件纪要，日期从此丢不了。记住这个结构。", Next: NodeClear},
					{Label: "OK, got it, no problem.", Reply: "你其实根本没 got it——fifteen 和 fifty 在破音质里就是一对孪生陷阱，装懂混过会议，错过截止日才是真事故。听不清就确认，这不丢人；丢人的是两周后交错日期。", Next: NodeRetry},
					{Label: "You say the date again, I write it down.", Reply: "\"...OK?\" 对方愣了一下。You say the date again 是「你再说一遍」的中文语序，光板祈使句在英语里像点名训话——加个 could 立刻回暖：Could you say the date again? 不过确认句式 did you say...? 更省一步。", Next: NodeRetry},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "没听清就先装懂混过去", Reply: "fifteen 还是 fifty？装懂的代价是两周后的事故。再选。"},
			{Label: "确认日期，再提议补纪要", Reply: "对——did you say 确认，邮件纪要兜底：听不清的人反而最准。"},
			{Label: "让对方把整段从头再讲", Reply: "全员陪你重听一遍，会议时间不答应——精确指定要哪段就够。再想想。"},
		},
		Correct: 1,
		Clear:   "确认加纪要，听不清的人反而信息最准。下一关进会议：有不同意见，怎样反对而不把人推远？",
		Note:    "破音里确认了截止日",
	},
	"delay-risk": {
		Nodes: []scriptNode{
			{
				Prompt: "老板问：\"Are we still on track?\" 三句里哪句既透明，又能推动决策？",
				Options: []nodeOption{
					{Label: "Yes, everything is fine.", Reply: "你把风险藏到了周五——那时它只会更贵。on track 不是安慰题，已知延期还报 fine，会让团队失去调整窗口。", Next: NodeRetry},
					{Label: "We're two days behind because the tech team is slow.", Reply: "事实说了，但责任全甩给技术团队，也没有下一步。风险沟通要交付影响和补救动作，不是寻找被告。", Next: NodeRetry},
					{Label: "We're two days behind; I can cut scope or move delivery to Tuesday.", Reply: "对——先量化差距，再给两个可决策选项。坏消息没有消失，但老板现在能选择怎么处理。", Next: 1},
				},
			},
			{
				Prompt: "迁移挑战：客户的报告要晚一天。哪句承担责任并锁定新时间？",
				Options: []nodeOption{
					{Label: "There may be a slight delay. We'll keep you posted.", Reply: "slight 和 keep you posted 都在回避关键问题：到底晚多久、什么时候交？没有新承诺就没有可执行信息。", Next: NodeRetry},
					{Label: "We found an issue, and I'll send the corrected report by 3 p.m. tomorrow.", Reply: "对——说明问题但不甩锅，并给出具体到小时的新承诺。客户可以据此安排后续工作。", Next: NodeClear},
					{Label: "Sorry, but these things happen sometimes.", Reply: "道歉后马上淡化问题，会让客户觉得你不重视影响。承担责任之后还必须给补救动作和时间。", Next: NodeRetry},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "只报坏消息等老板决定", Reply: "没有方案的坏消息只是把问题转交出去。再选。"},
			{Label: "说明影响并给明确选项", Reply: "对——透明、量化、带方案，风险才能变成可管理的决定。"},
			{Label: "先说一切正常稳住情绪", Reply: "短暂稳住了情绪，却失去了调整窗口。再想想。"},
		},
		Correct: 1,
		Clear:   "延期没有藏，也没有甩锅——你把风险变成了可选择的方案。出公司之前还有一场「全英模拟面」：紧急视频会已经拉起，没有字幕。",
		Note:    "把延期说成可选方案",
	},
	"business-networking": {
		Nodes: []scriptNode{
			{
				Prompt: "潜在客户问：\"What brings you here?\" 哪句介绍清楚又把球递回去？",
				Options: []nodeOption{
					{Label: "I'm Lin from Northstar. We help retailers forecast demand. What about you?", Reply: "对——姓名、公司、价值一句到位，再用 What about you 把对话交还给对方。短，但每个信息都有用。", Next: 1},
					{Label: "Our company was founded in 2016 and has over 200 employees.", Reply: "这像公司简介第一页，却没有回答你为什么来，也没给对方接话口。初见先说你解决什么问题，不要背工商档案。", Next: NodeRetry},
					{Label: "I'm just looking around. Nothing special.", Reply: "姿态很轻松，机会也一起轻掉了。行业活动的寒暄需要给对方一个能继续追问的业务钩子。", Next: NodeRetry},
				},
			},
			{
				Prompt: "聊了两轮，你想自然进入正题。哪句最好？",
				Options: []nodeOption{
					{Label: "Anyway, let's talk business now.", Reply: "意思清楚，但像突然敲桌子开会。商务寒暄的转场最好接住对方刚说过的内容。", Next: NodeRetry},
					{Label: "You mentioned inventory issues—would it be useful to compare notes?", Reply: "对——用 You mentioned 接回对方的话，再以邀请而非推销进入业务，转场自然且有理由。", Next: NodeClear},
					{Label: "Can I give you our full sales presentation?", Reply: "刚认识就要完整路演，负担太重。先确认一个共同问题，再决定是否深入。", Next: NodeRetry},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "先背完公司历史", Reply: "信息很多，却没有业务钩子和回球。再选。"},
			{Label: "寒暄越久越显得有礼貌", Reply: "没有目的的空聊会消耗双方时间。再想想。"},
			{Label: "接住对方的话再进正题", Reply: "对——从对方刚说的内容转场，既自然又有业务理由。"},
		},
		Correct: 2,
		Clear:   "没有硬推销，也没有聊到散场——你从一句寒暄走到了一个值得继续谈的问题。",
		Note:    "把寒暄带进了正题",
	},
	"company-product-intro": {
		Nodes: []scriptNode{
			{
				Prompt: "客户问产品做什么。哪句先说客户价值，而不是堆功能？",
				Options: []nodeOption{
					{Label: "It's an AI platform with dashboards, APIs, alerts, and many advanced features.", Reply: "功能很多，客户仍不知道为什么要在意。产品介绍先回答『帮谁解决什么』，功能是后面的证据。", Next: NodeRetry},
					{Label: "We help retail teams predict demand and reduce excess inventory.", Reply: "对——客户是谁、解决什么问题、结果是什么，一句全齐。客户若感兴趣，自然会追问怎么做到。", Next: 1},
					{Label: "It's the best forecasting product on the market.", Reply: "最高级没有证据，只会邀请客户质疑。具体结果比 best 更有说服力。", Next: NodeRetry},
				},
			},
			{
				Prompt: "客户说最关心效率。哪句把功能翻译成结果？",
				Options: []nodeOption{
					{Label: "Our dashboard has eight configurable modules.", Reply: "仍然在报功能。客户关心效率，就要说明这个功能替他省掉什么动作和时间。", Next: NodeRetry},
					{Label: "The interface is modern and very easy to use.", Reply: "方向接近，但 modern 和 easy 都太泛。用可观察的工作变化证明效率。", Next: NodeRetry},
					{Label: "Your team can review all stores in one place instead of merging spreadsheets.", Reply: "对——把功能翻译成客户每天会发生的变化：少合表、集中查看。价值变得具体可见。", Next: NodeClear},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "功能越多介绍越有说服力", Reply: "没有连接客户问题的功能只是清单。再选。"},
			{Label: "先说帮谁解决什么问题", Reply: "对——价值先行，功能随后作为证据。"},
			{Label: "直接说自己是市场第一", Reply: "没有证据的最高级只会引来质疑。再想想。"},
		},
		Correct: 1,
		Clear:   "你没有背产品说明书，而是让客户看见了工作会怎样变轻。下一关先别急着卖，学会问。",
		Note:    "把功能翻成了价值",
	},
	"needs-discovery": {
		Nodes: []scriptNode{
			{
				Prompt: "客户只说想提高效率。哪句先问出现状和问题？",
				Options: []nodeOption{
					{Label: "Great, let me show you all our features.", Reply: "目标还没弄清就开始演示，后面每个功能都可能打偏。先理解客户今天怎么做、哪里最痛。", Next: NodeRetry},
					{Label: "How are you handling this today, and where does it slow you down?", Reply: "对——先问现状，再定位阻力。客户的答案会决定后面该讲什么，而不是把整套产品都倒出来。", Next: 1},
					{Label: "How much budget do you have for this?", Reply: "预算重要，但第一问就谈钱会让对方进入防守。先建立问题价值，再讨论投入。", Next: NodeRetry},
				},
			},
			{
				Prompt: "客户讲完需求。哪句复述确认最稳？",
				Options: []nodeOption{
					{Label: "I understand everything. Our standard package will work.", Reply: "你宣布懂了，却没有给客户验证的机会。标准包是否适合也还是假设。", Next: NodeRetry},
					{Label: "So you need faster reporting, especially across regional teams—is that right?", Reply: "对——抓住目标和重点场景，再用 is that right 把解释权还给客户。确认过再提方案，返工会少很多。", Next: NodeClear},
					{Label: "Your current process sounds inefficient.", Reply: "也许事实如此，但直接评价客户现状会制造防御。复述需求，不给客户打分。", Next: NodeRetry},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "客户一开口就开始演示", Reply: "需求未明，演示越完整越容易跑偏。再选。"},
			{Label: "先问现状，再复述确认", Reply: "对——问出问题、确认理解，方案才有靶心。"},
			{Label: "第一问先锁预算", Reply: "过早谈钱容易让对方进入防守。再想想。"},
		},
		Correct: 1,
		Clear:   "你没有急着展示，而是让客户亲口说出了问题。下一关，方案终于有了靶心。",
		Note:    "先问再确认了需求",
	},
	"solution-proposal": {
		Nodes: []scriptNode{
			{
				Prompt: "轮到提方案。哪句把建议连接到客户刚说的需求？",
				Options: []nodeOption{
					{Label: "Let me walk you through our standard presentation.", Reply: "标准演示忽略了前面的需求对话。客户会怀疑刚才说的内容有没有被听见。", Next: NodeRetry},
					{Label: "Based on your reporting bottleneck, we'd start with automated regional reports.", Reply: "对——Based on your...证明你听见了问题，后半句给出聚焦建议，而不是把所有功能全端上来。", Next: 1},
					{Label: "Our solution is comprehensive and suitable for every company.", Reply: "适合所有公司通常意味着没有为这家公司思考。越具体，方案越可信。", Next: NodeRetry},
				},
			},
			{
				Prompt: "客户担心切换系统影响业务。哪句真正回应顾虑？",
				Options: []nodeOption{
					{Label: "Don't worry. Migration is usually easy.", Reply: "Don't worry 没有消除风险，只是否定了客户的担心。要给具体的降风险安排。", Next: NodeRetry},
					{Label: "The product has won several industry awards.", Reply: "奖项与切换风险不是同一个问题。回答必须紧贴客户刚提出的顾虑。", Next: NodeRetry},
					{Label: "We can pilot one region first, so daily operations continue during the switch.", Reply: "对——小范围试点直接降低切换风险，也解释了业务为何不会中断。方案与顾虑严丝合缝。", Next: NodeClear},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "完整展示所有标准功能", Reply: "内容完整不等于贴合需求。再选。"},
			{Label: "按客户问题给聚焦建议", Reply: "对——需求是前提，建议是动作，结果是价值。"},
			{Label: "用别担心安抚风险", Reply: "没有具体安排的安抚无法降低风险。再想想。"},
		},
		Correct: 1,
		Clear:   "方案没有悬在产品上，而是落在客户的问题里。下一关，客户会把最难听的质疑摆上桌。",
		Note:    "方案对准了客户问题",
	},
	"handle-objection": {
		Nodes: []scriptNode{
			{
				Prompt: "客户说你的报价高20%。哪句先弄清比较范围？",
				Options: []nodeOption{
					{Label: "That's our final price.", Reply: "你守住了价格，却关掉了了解异议的门。对方比较的范围可能根本不同。", Next: NodeRetry},
					{Label: "I understand. Which parts of the two proposals are you comparing?", Reply: "对——先承认顾虑，再查比较口径。若服务范围不同，价格异议可能不需要靠降价解决。", Next: 1},
					{Label: "No problem, we can reduce it by 20%.", Reply: "还没弄清原因就全额降价，既伤利润，也暗示原报价虚高。", Next: NodeRetry},
				},
			},
			{
				Prompt: "客户说真正担心的是上线周期。哪句接住并给选项？",
				Options: []nodeOption{
					{Label: "Our timeline is already very fast.", Reply: "你在评价自己，没有回应客户需要的时间。先确认具体期限，再给能调整的方案。", Next: NodeRetry},
					{Label: "When exactly do you need to launch? We could phase the rollout.", Reply: "对——先把模糊的『太慢』变成明确日期，再提出分阶段上线这个可谈选项。", Next: NodeClear},
					{Label: "The timeline shouldn't be a problem.", Reply: "客户已经说它是问题，shouldn't 只会让对方觉得没被听见。", Next: NodeRetry},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "听到贵就立刻同比例降价", Reply: "原因未明就让步，既伤利润也失去信息。再选。"},
			{Label: "先澄清异议再给选项", Reply: "对——异议不是拒绝，而是需要被拆清的问题。"},
			{Label: "强调自己已经做得很好", Reply: "自我证明没有回应客户真正担心的事。再想想。"},
		},
		Correct: 1,
		Clear:   "你没有被『太贵、太慢』牵着走，而是把模糊异议拆成了可以处理的问题。下一关进入报价谈判。",
		Note:    "把异议问成了真问题",
	},
	"close-next-step": {
		Nodes: []scriptNode{
			{
				Prompt: "客户说方案不错。哪句把兴趣推进成下一步？",
				Options: []nodeOption{
					{Label: "Great, we'll wait for your decision.", Reply: "礼貌，但把所有主动权交了出去。会议可能从此停在 promising。", Next: NodeRetry},
					{Label: "Shall we schedule a technical review with both teams next week?", Reply: "对——把模糊兴趣变成具体动作，参与人和时间范围也都有了。", Next: 1},
					{Label: "Can you sign the contract today?", Reply: "客户只说 promising，还没到签约承诺。下一步应与当前成熟度匹配。", Next: NodeRetry},
				},
			},
			{
				Prompt: "终局：会议结束前，哪句把责任人、动作和时间都锁定？",
				Options: []nodeOption{
					{Label: "Let's stay in touch and move quickly.", Reply: "听起来积极，却没有任何人知道下一步具体做什么。", Next: NodeRetry},
					{Label: "We'll follow up soon with more information.", Reply: "soon 和 more information 都不可执行。谁发什么、何时发，必须落到句子里。", Next: NodeRetry},
					{Label: "I'll send the revised proposal by Thursday, and Maya will confirm the review date.", Reply: "对——两个动作、两个责任人、一个明确时间。会议结束，项目仍然在向前走。", Next: NodeClear},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "客户说不错就等待决定", Reply: "兴趣没有下一步，很容易自然冷却。再选。"},
			{Label: "直接要求当天签合同", Reply: "动作超过当前成熟度，会制造压力。再想想。"},
			{Label: "锁定动作责任人与时间", Reply: "对——商务沟通的终点不是聊得好，而是下一步明确。"},
		},
		Correct: 2,
		Clear:   "从初见、需求、方案、异议到下一步，整场商务沟通被你推进到了可执行的结果。终局还剩一场——「全英模拟面·成交」：三十分钟，真刀真枪。",
		Note:    "把兴趣推进成了行动",
	},
	// ============ 全英模拟面（每幕 Boss）============
	// 五节点单链：听力门 → 两轮全英裁决（混考本幕语块）→ 拼句两步。对方台词零中文旁白，
	// 点破仍用中文（教学不糊）。init 的三轮改造不碰它们（len!=2），Variants 手写。
	"boss-daily-life": {
		Nodes: []scriptNode{
			{
				Prompt: "咖啡店里你刚报完取货单号，店员忽然说——\n\n" + listenLine("Sorry, we're out of oat milk today—would soy milk be OK instead?") + "\n\nTA 想告诉你什么？",
				Options: []nodeOption{
					{Label: "燕麦奶没了，问豆奶行不行", Reply: "你点点头：\"Sure, soy is fine.\" 店员比了个 OK——TA 说的是：\"Sorry, we're out of oat milk today—would soy milk be OK instead?\" out of（卖完了）和 instead（换成）两个关键词你都抓住了。往下走。", Next: 1},
					{Label: "今天的拿铁全部售罄了", Reply: "你转身要走，店员赶紧叫住你：\"No no, we have lattes!\" ——卖完的只是燕麦奶。out of 后面跟的那个词才是缺的东西，别把范围听大了。\n\n" + listenLine("Sorry, we're out of oat milk today—would soy milk be OK instead?"), Next: NodeRetry},
					{Label: "换豆奶要加两美元", Reply: "你掏出钱包，店员摆摆手：\"No extra charge!\" ——整句里没有出现任何数字和钱，would ... be OK instead 是在征求你同意，不是报价。再听一遍。\n\n" + listenLine("Sorry, we're out of oat milk today—would soy milk be OK instead?"), Next: NodeRetry},
				},
			},
			{
				Prompt: "蛋糕盒打开一看，名字挤成了 \"Happy Birthday Lee\"——寿星明明叫 Leo。店员问：\"All good?\" 三句挑一句：",
				Options: []nodeOption{
					{Label: "The name is wrong. This is a big problem.", Reply: "店员僵住：\"...I see.\" 改是能改，气氛先冷了——没有缓冲直接定罪，big problem 还把小事说大。指错的开场白永远先给一句 Sorry，咖啡馆那关就是这么赢的。", Next: NodeRetry},
					{Label: "Em... it's OK, Lee is also a nice name.", Reply: "店员笑着把盒子递了回来——今晚寿星 Leo 要对着 Lee 吹蜡烛。it's OK 在这个场景就是「不用改」，你又把开口的机会让出去了。这单是给朋友的，忍不得。", Next: NodeRetry},
					{Label: "Sorry, I think the name should be Leo, not Lee—could you fix it?", Reply: "店员凑近一看：\"Oh no, my bad—give me two minutes!\" ——Sorry 缓冲、I think 留余地、说清对错点、could you 提请求：第一关学的指错三件套，换家店照样好使。", Next: 2},
				},
			},
			{
				Prompt: "路上餐厅来电：\"Hi, this is Bella's Kitchen. About your table for eight tonight—we're overbooked at seven. Could you do six thirty or nine?\" 三句挑一句：",
				Options: []nodeOption{
					{Label: "Nine is too late, six thirty is too early... you decide.", Reply: "电话那头等了三秒：\"So... which one, sir?\" ——两个都嫌、再把决定推回去，等于没接这通电话。订位要给确定答案，电话预约那关就栽过这个坑。", Next: NodeRetry},
					{Label: "Six thirty works. It's under Leo—and could we get a table by the window?", Reply: "\"Six thirty, party of eight, under Leo, window table—all set!\" ——确定时间、报上订位名、顺手提要求，三要素一口气闭环，电话语域稳稳的。", Next: 3},
					{Label: "We are eight persons, we want six thirty, no problem?", Reply: "接线员听懂了，但记录卡了一下——口语说 eight people 或 a table for eight，persons 是法律文书里的词；no problem? 也不是确认句，确认要说 six thirty works 或 is that OK?。", Next: NodeRetry},
				},
			},
			{
				Prompt: "晚上聚会，一位没见过的客人朝你举了举杯。该你开口了——把破冰这句拼出来，先选上半句：",
				Options: []nodeOption{
					{Label: "Hi, I'm Lin—Leo and I work together.", Reply: "对方笑着碰了下杯：\"Oh nice!\" ——名字加一句「你和寿星的关系」，破冰上半句信息刚刚好。接着把话头抛回去。", Next: 4},
					{Label: "Hello, my name is called Lin, 28 years old.", Reply: "对方礼貌地眨了眨眼——is called 是给物件和绰号用的，自报姓名就是 I'm Lin；寒暄一开口就报年龄，也像在念简历。轻一点，短一点。", Next: NodeRetry},
					{Label: "Sorry, my English is not good, but hello.", Reply: "对方赶紧安慰你：\"You're doing great!\" ——话题瞬间从「认识你」变成「救你」。自贬开场是老毛病了，初次寒暄那关就点过它：破冰不需要道歉。", Next: NodeRetry},
				},
			},
			{
				Prompt: "下半句——把话头抛回去：",
				Options: []nodeOption{
					{Label: "Do you know me?", Reply: "对方愣了一下：\"...Should I?\" ——这句像在质问对方认不认识你。把话头抛回去，问的应该是 TA 和这场聚会的联系。", Next: NodeRetry},
					{Label: "I am a designer, my job is very busy.", Reply: "对方点头听完，接不上话——你把话头留在了自己身上，对话的球没有过网。破冰下半句的任务只有一个：递一个 TA 能接的问题。", Next: NodeRetry},
					{Label: "How do you know Leo?", Reply: "\"College roommates—he never told you about the guitar story?\" 对方打开了话匣子——How do you know... 一出手，话题自己往前跑。全英一整天，你一个人扛下来了。", Next: NodeClear},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "没听清就先点头应付过去", Reply: "装懂是听力最大的敌人——out of 没抓住，连豆奶都换不成。再想想。"},
			{Label: "抓关键词，错了就体面指出", Reply: "对——听抓 out of/instead，指错用 Sorry 三件套：一整天的场面全靠这两手。"},
			{Label: "名字写错了将就一下算了", Reply: "寿星对着别人的名字吹蜡烛——该开口时的忍，代价都在后头。再选。"},
		},
		Correct: 1,
		Variants: []checkVariant{
			{
				Ask: "延时迁移：超市自助结账机好像多扣了钱，店员走过来：\"What seems to be the problem?\" 哪句接得住？",
				Opts: []tapOption{
					{Label: "Sorry, I think the machine double-charged me—could you check?", Reply: "对——Sorry 缓冲、I think 留余地、说清问题、could you 提请求：指错三件套在哪儿都好使。"},
					{Label: "Your machine is broken. Give my money back now.", Reply: "没缓冲、直接定罪加命令——钱能退回来，你也成了全店最凶的客人。"},
					{Label: "Nothing... maybe it's my fault, never mind.", Reply: "多扣的钱就这么认了？maybe my fault 加 never mind，求助被你自己撤回了。"},
				},
				Correct: 0,
			},
		},
		Clear: "听力门、指错、订位、破冰——没有一句中文旁白，你全接住了。第一幕全英通关！第二幕「旅行应急」的机场广播，语速可比店员快多了。",
		Note:  "全英扛完了一整天",
	},
	"boss-travel-storm": {
		Nodes: []scriptNode{
			{
				Prompt: "广播只播一遍——\n\n" + listenLine("Flight 782 to Boston has been cancelled due to weather. Please proceed to the service desk for rebooking and a hotel voucher.") + "\n\n广播说了什么？",
				Options: []nodeOption{
					{Label: "航班延误两小时，原地等通知", Reply: "你在座位上坐了十分钟，抬头一看，队伍已经排到了门口——cancelled 是取消，不是 delayed 延误；proceed to the service desk 是让你去柜台，不是等。\n\n" + listenLine("Flight 782 to Boston has been cancelled due to weather. Please proceed to the service desk for rebooking and a hotel voucher."), Next: NodeRetry},
					{Label: "航班取消，去柜台改签、领酒店券", Reply: "你抓起背包直奔柜台，排进了前十——TA 说的是：\"Flight 782 has been cancelled... rebooking and a hotel voucher.\" cancelled（取消）、rebooking（改签）、voucher（酒店券）三个关键词全中。信息就是先机。", Next: 1},
					{Label: "登机口换了，要去新柜台登机", Reply: "你冲到新登机口，屏幕上一片红色的 CANCELLED——广播里根本没有 gate（登机口）这个词。抓词别脑补，再听一遍。\n\n" + listenLine("Flight 782 to Boston has been cancelled due to weather. Please proceed to the service desk for rebooking and a hotel voucher."), Next: NodeRetry},
				},
			},
			{
				Prompt: "柜台前地勤飞快敲着键盘：\"I can put you on the 6 a.m. flight, that's the earliest.\" 三句挑一句：",
				Options: []nodeOption{
					{Label: "6 a.m. works. Does the voucher cover dinner too? And could I get an aisle seat?", Reply: "\"Dinner's included—and aisle seat, done.\" ——接受方案的同时把该问的问清、该要的要到：值机那关学的「先拿信息再提条件」，深夜也不忘。", Next: 2},
					{Label: "OK... whatever you can do. Anything is fine.", Reply: "地勤十秒钟打完票把你打发走了——座位中间、晚饭没提。whatever 在柜台等于弃权，这是超售谈判那关就交过的学费。", Next: NodeRetry},
					{Label: "6 a.m.?! You must give me business class for tonight!", Reply: "地勤面无表情：\"I'm afraid that's not possible, sir.\" ——must 加感叹号是命令不是谈判；开口要的东西超出对方权限，只会把本来能拿的也谈丢。", Next: NodeRetry},
				},
			},
			{
				Prompt: "凌晨的酒店前台翻了半天电脑：\"Sorry, I don't see any reservation under your name.\" 三句挑一句：",
				Options: []nodeOption{
					{Label: "Impossible! The airline already pay you!", Reply: "前台的笑容降了温：\"Sir, I checked twice.\" ——Impossible 开头是指控，pay 也该是过去式 paid。冲前台发火最亏：弄丢记录的不是 TA，能捞你的正是 TA。", Next: NodeRetry},
					{Label: "Oh... OK, sorry, I go find another hotel.", Reply: "你拖着箱子走向凌晨的大街——酒店券还在手里攥着呢。查无记录不等于没有房，一句确认都没问就撤，这一夜输给了自己的「算了」。", Next: NodeRetry},
					{Label: "Hmm, the airline booked it—here's the voucher. Could you check under 'Flight 782'?", Reply: "前台接过券一敲回车：\"Ah, there it is—under the airline's block. My apologies!\" ——不指控、给凭证、递新的查询线索：把问题当谜题一起解，房卡就到手了。", Next: 3},
				},
			},
			{
				Prompt: "早上八点，行李果然没跟上航班。柜台问：\"Can you describe your bag?\" 把这句拼出来，先选上半句：",
				Options: []nodeOption{
					{Label: "You lost my box in the airplane—", Reply: "工作人员笔尖一顿：\"A... box?\" ——行李箱是 suitcase，box 是纸箱；You lost 开头也先定了罪。上半句要说清「什么东西、出了什么事」。", Next: NodeRetry},
					{Label: "My suitcase didn't make the connection—", Reply: "工作人员开始录入：\"Go on.\" ——didn't make the connection（没赶上转机）一词说清事故，suitcase 也用对了。接着给特征。", Next: 4},
					{Label: "I lose my baggage yesterday night—", Reply: "丢是已经发生的事，要用过去式 lost；「昨晚」是 last night，yesterday night 是中式拼法。两个小坑，护照那关都踩过。", Next: NodeRetry},
				},
			},
			{
				Prompt: "下半句——给出让人认得出的特征：",
				Options: []nodeOption{
					{Label: "it is big size, blue color, very expensive.", Reply: "big size、blue color 都是中式冗余——big 和 blue 自己就够了；very expensive 放在描述里也不如一个具体特征有用。行李丢失那关讲过的。", Next: NodeRetry},
					{Label: "it's a large blue one with a red ribbon on the handle.", Reply: "\"Large, blue, red ribbon—we'll call you the moment it lands.\" ——尺寸、颜色、独有特征一句排齐，几百个蓝箱子里就能认出你这只。", Next: NodeClear},
					{Label: "it's my bag, you will know when you see it.", Reply: "工作人员苦笑：\"They all look alike, sir.\" ——「见到就认识」帮不了任何人，寻回和理赔全靠你嘴里的特征清单。", Next: NodeRetry},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "广播没听清，跟着人流走", Reply: "人流去的是出口，你的改签柜台在另一头——关键词才是路标。再想想。"},
			{Label: "查无记录就自认倒霉换酒店", Reply: "券在手里就有凭证——给线索请对方再查，比深夜满街找房强得多。再选。"},
			{Label: "抓关键词行动，拿凭证沟通", Reply: "对——cancelled/voucher 定方向，凭证加线索解僵局：应急夜全靠这两板斧。"},
		},
		Correct: 2,
		Variants: []checkVariant{
			{
				Ask: "延时迁移：地铁广播 \"This train terminates at the next stop due to a signal failure. Please change to Line 4.\" 你该？",
				Opts: []tapOption{
					{Label: "下一站下车，换乘四号线", Reply: "对——terminates（到此为止）加 change to Line 4（换乘），两个关键词就是行动指南。"},
					{Label: "列车快到站了，坐着别动", Reply: "terminate 是「终止运行」不是「快到了」——坐过站的都是没抓住这个词的。"},
					{Label: "信号故障，全线停运回家吧", Reply: "广播给了明确出路：change to Line 4。别把一句坏消息听成世界末日。"},
				},
				Correct: 0,
			},
		},
		Clear: "取消的航班、消失的预订、迟到的箱子——一夜三连击全用英语拆完，第二幕全英通关。第三幕进办公室：老板的语速，不会等你。",
		Note:  "航班取消夜全英自救",
	},
	"boss-work-sprint": {
		Nodes: []scriptNode{
			{
				Prompt: "会议刚接通，老板开门见山——\n\n" + listenLine("Quick update, everyone—marketing wants to move the launch up a week, so let's see what that means for each team.") + "\n\n老板宣布了什么？",
				Options: []nodeOption{
					{Label: "发布要推迟一周，各组重排期", Reply: "你刚想松口气，屏幕里同事们的表情可不像多了一周——move up 是「提前」，推迟是 push back。方向听反，后面全错。\n\n" + listenLine("Quick update, everyone—marketing wants to move the launch up a week, so let's see what that means for each team."), Next: NodeRetry},
					{Label: "发布要提前一周，评估各组影响", Reply: "你在本子上写下「-7 天」——TA 说的是 move the launch up a week：move up = 提前，push back = 推迟。这对反义词是无数跨国会议事故的源头，你没踩。", Next: 1},
					{Label: "市场部要增加一周的营销预算", Reply: "整句没出现 budget——a week 修饰的是 move up 的幅度，不是预算。听力别抓到一个名词就编故事，再来。\n\n" + listenLine("Quick update, everyone—marketing wants to move the launch up a week, so let's see what that means for each team."), Next: NodeRetry},
				},
			},
			{
				Prompt: "老板点名：\"Lin, can your side make it?\" 提前一周，测试时间就不够了。三句挑一句：",
				Options: []nodeOption{
					{Label: "No. It's impossible. The schedule is already crazy.", Reply: "会议安静了两秒，老板皱起眉——你说的可能是事实，但 impossible 把门焊死了，crazy 还带着情绪。表异议的黄金句式忘了？先接住，再递顾虑。", Next: NodeRetry},
					{Label: "We can try—but I'm worried about QA. Could we cut one minor feature to make room?", Reply: "老板身体前倾：\"Interesting—which one?\" ——先接住目标，用 I'm worried about 递出真实风险，还带了个可行方案。会议表态那关的「糖在前药在后」，火线上照样灵。", Next: 2},
					{Label: "OK boss, no problem, we will do our best!", Reply: "老板满意地跳到下一组——三周后测试爆雷时，今天这句 no problem 就是追责的第一条证据。会上不说的风险，返工时要加倍还。", Next: NodeRetry},
				},
			},
			{
				Prompt: "会议结束前老板补了一句：\"Sarah's on leave this week—Lin, can you cover her client calls?\" 三句挑一句：",
				Options: []nodeOption{
					{Label: "Sure, I will handle everything, don't worry!", Reply: "接得爽快，锅也接得整齐——Sarah 的客户要改合同价你也「handle」吗？不划边界的承诺，出了岔子全算你的。任务交代那关的红线公式呢？", Next: NodeRetry},
					{Label: "Sure. I'll use my judgment on routine calls—but pricing goes to you first, right?", Reply: "\"Exactly. Thanks, Lin.\" ——接下任务的同时把红线划清：日常你定、报价找老板。授权公式反过来用，保护的是接活的人。", Next: 3},
					{Label: "Em... I am very busy... maybe you find other people?", Reply: "老板环视一圈没人接话，气氛僵住——推掉可以，但 maybe you find other people 是把问题扔回给全组。要推，也该给一个具体困难加一个替代人选。", Next: NodeRetry},
				},
			},
			{
				Prompt: "散会前轮到你收尾。把这句「靠谱的承诺」拼出来，先选上半句：",
				Options: []nodeOption{
					{Label: "I will try my best to send the timeline soon,", Reply: "try my best 和 soon 都测不了、追不上——听着努力，其实什么都没承诺。承诺要能被日历验证。", Next: NodeRetry},
					{Label: "You will get the timeline when it is ready,", Reply: "when it's ready 把主动权攥在自己手里，像在打发人。承诺的主语是 I，期限是具体的星期几。", Next: NodeRetry},
					{Label: "I'll send the updated timeline by Thursday,", Reply: "确定的动作、确定的期限——I'll do X by Y 是职场信用的基本句型。接着补上另一半。", Next: 4},
				},
			},
			{
				Prompt: "下半句——把风险也兜住：",
				Options: []nodeOption{
					{Label: "and I hope everything will be fine.", Reply: "hope 不是计划——风险不会因为祈祷改道。这半句该说的是你会「做什么」来兜住它。", Next: NodeRetry},
					{Label: "and I'll flag any risks early.", Reply: "\"Perfect—that's how I like it run.\" 老板合上电脑——动作、期限、风险预警，一句话三个承诺全部可验证。这场会你不只听懂了，还把事扛住了。", Next: NodeClear},
					{Label: "and please don't change the plan again.", Reply: "当众要求老板「别再改计划」——方向可以争，但这句的语气是抱怨不是收尾。把控制不了的放下，把控制得了的说清。", Next: NodeRetry},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "听清方向，表态先糖后药", Reply: "对——move up 定方向，先接住再 I'm worried about：跨国会议的两根救命稻草。"},
			{Label: "没听懂也先答 no problem", Reply: "方向都没听清就应承，三周后爆雷时没人记得你今天的爽快。再想想。"},
			{Label: "接活就该大包大揽显担当", Reply: "不划红线的担当是接锅——报价类的事永远先过老板。再选。"},
		},
		Correct: 0,
		Variants: []checkVariant{
			{
				Ask: "延时迁移：客户邮件说 \"Can we push the demo back to Friday?\" ——TA 想把演示怎么样？",
				Opts: []tapOption{
					{Label: "推迟到周五", Reply: "对——push back = 推迟，和 move up（提前）正好一对。这对词，搞反一次就够记一辈子。"},
					{Label: "提前到周五", Reply: "提前是 move up 或 bring forward——push back 是往后推。方向反了，会议室就空了。"},
					{Label: "取消这次演示", Reply: "取消是 cancel 或 call off——push back 只是挪时间，单子还在。"},
				},
				Correct: 0,
			},
		},
		Clear: "听懂方向、递出顾虑、划清红线、给出承诺——全英会议四件套齐活，第三幕通关。最后一幕是谈判桌：那里每个词都带着价签。",
		Note:  "全英例会稳稳扛住",
	},
	"boss-business-deal": {
		Nodes: []scriptNode{
			{
				Prompt: "客户落座就抛出一句——\n\n" + listenLine("Honestly, we got burned by a long rollout last time, so speed matters more to us than price.") + "\n\nTA 真正在意的是什么？",
				Options: []nodeOption{
					{Label: "TA 嫌你们的报价太贵了", Reply: "你刚要解释价格，客户摆了摆手——TA 明明说了 speed matters more than price：比起价格更在意速度。销售听力第一课：别用自己的心虚补对方的台词。\n\n" + listenLine("Honestly, we got burned by a long rollout last time, so speed matters more to us than price."), Next: NodeRetry},
					{Label: "上次上线周期太长吃了亏，最在意速度", Reply: "你在心里画了重点——TA 说的是 got burned by a long rollout（被漫长上线坑过）、speed over price（速度重于价格）。客户把命门主动递过来了，就看你接不接得住。", Next: 1},
					{Label: "上次合作烧坏了设备，要求赔偿", Reply: "got burned 是「吃过亏、栽过跟头」的习语，不是真着火——习语听字面，谈判就跑偏了。再听一遍。\n\n" + listenLine("Honestly, we got burned by a long rollout last time, so speed matters more to us than price."), Next: NodeRetry},
				},
			},
			{
				Prompt: "命门在「速度」。哪句能把 TA 的需求挖到可交付的深度？",
				Options: []nodeOption{
					{Label: "Don't worry! We are the fastest in this industry, trust me.", Reply: "客户礼貌地笑了笑，身体向后靠——上一家也是这么说的。空口的「最快」治不了吃过亏的人，能治的是具体的问题和数字。", Next: NodeRetry},
					{Label: "Our product has 47 functions, let me show you one by one.", Reply: "客户看了一眼手表——TA 在意的是速度，你掏出的是功能清单。介绍产品那关教过：客户买的不是功能，是自己问题的答案。", Next: NodeRetry},
					{Label: "What does 'fast' look like for you—and what slowed things down last time?", Reply: "客户掰着手指讲了十分钟——「fast 对你们意味着什么」把形容词逼成数字，「上次卡在哪」把伤疤变成需求清单。挖掘需求那关的两把铲子，一次用全。", Next: 2},
				},
			},
			{
				Prompt: "聊透了，客户抛出最后一压：\"Your quote is still 20% higher than the others. Meet them, and we'll sign today.\" 三句挑一句：",
				Options: []nodeOption{
					{Label: "OK, OK, 20% off. We really want to work with you.", Reply: "客户心里已经开了香槟——sign today 一晃，你就把两成利润裸送了。「今天就签」是谈判话术里最经典的钩子，咬钩之前先想想：急的到底是谁？", Next: NodeRetry},
					{Label: "Cheaper tools will burn you again, just like last time.", Reply: "客户脸色沉了下来——拿人家的伤疤当谈判筹码，句句戳心。贬低对手加恐吓客户，嘴上赢了，单子输了。", Next: NodeRetry},
					{Label: "What if we put a 30-day rollout guarantee in the contract—would the price work then?", Reply: "客户和同事对视了一眼：\"...That's interesting.\" ——不降价，把 TA 最在意的「速度」写进合同当筹码：报价谈判那关的 what if 换筹码，打在了听力关挖出的命门上。", Next: 3},
				},
			},
			{
				Prompt: "火候到了。把锁定下一步的这句拼出来，先选上半句：",
				Options: []nodeOption{
					{Label: "Please think about it and call us anytime.", Reply: "客户点头微笑收起名片——anytime 的意思通常是 never。方向盘整个交给了对方，两周的犹豫又要续费了。", Next: NodeRetry},
					{Label: "Shall we schedule a technical review with both teams next week?", Reply: "\"Tuesday works for us.\" ——把「有兴趣」推进成「有日程」：具体动作、双方团队、明确时间范围，推进成交那关的标准起手。", Next: 4},
					{Label: "So, can you sign the contract right now?", Reply: "客户刚从「有点兴趣」走到「可以细聊」，你直接把笔递了过去——动作超过火候，压力会把人压跑。下一步要和成熟度匹配。", Next: NodeRetry},
				},
			},
			{
				Prompt: "下半句——把球留在自己手里：",
				Options: []nodeOption{
					{Label: "We will wait for your good news.", Reply: "「等好消息」——没有动作、没有责任人、没有时间，三无收尾，单子在等待中冷掉。", Next: NodeRetry},
					{Label: "You should discuss inside and give us the answer quickly.", Reply: "should 加 quickly 是在给客户下指令——收尾的球要留在自己手里：我发什么、什么时候发，让对方只需要点头。", Next: NodeRetry},
					{Label: "I'll send the revised proposal with the 30-day guarantee by Thursday.", Reply: "\"Looking forward to it.\" 握手，成局——动作（发修订方案）、内容（30 天保证）、期限（周四），收尾一句话三个钉子。三十分钟，全英文，你把一单谈到了下一步。", Next: NodeClear},
				},
			},
		},
		CheckOpts: []tapOption{
			{Label: "客户压价就赶紧降两成", Reply: "sign today 是钩子——裸降两成，利润没了，尊重也没了。再想想。"},
			{Label: "多介绍功能显得更专业", Reply: "TA 在意的是速度，功能清单只会消耗耐心——先挖需求。再选。"},
			{Label: "听出命门，拿保证换价格", Reply: "对——听力挖出 speed 这个命门，30 天保证写进合同当筹码：一环扣一环。"},
		},
		Correct: 2,
		Variants: []checkVariant{
			{
				Ask: "延时迁移：客户说 \"We're a bit hesitant—the last vendor overpromised.\" 哪句接得最稳？",
				Opts: []tapOption{
					{Label: "That's fair. What did they promise that didn't happen?", Reply: "对——先接住情绪，再把「被放过的鸽子」挖成需求清单：吃过亏的客户最吃这一套。"},
					{Label: "We are different, we never overpromise, believe us!", Reply: "每个 overpromise 的供应商都说过这句——空保证治不了被空保证伤过的人。"},
					{Label: "Their price is low because their quality is low.", Reply: "贬低前任供应商，等于说客户当初眼光差——这句赢不了任何东西。"},
				},
				Correct: 0,
			},
		},
		Clear: "听懂弦外之音、问出真需求、用保证换价格、把兴趣钉成日程——四幕三十二关全部通关！从点一杯拿铁到谈下一单生意，这条路你是全英文走完的。",
		Note:  "全英谈成了一单",
	},
}

// 每关在两轮场景之间插入一次「规则辨析」，形成 裁决→辨析→迁移 三步；再把最后一轮
// 挂成延时复习题。主流程通关只升到点亮，到期后再次在英文语料里选对才升到掌握。
func init() {
	for slug, lv := range learnEnglishScript {
		if len(lv.Nodes) != 2 {
			continue
		}
		ruleOpts := make([]nodeOption, 0, len(lv.CheckOpts))
		for i, o := range lv.CheckOpts {
			next := NodeRetry
			if i == lv.Correct {
				next = 2
			}
			ruleOpts = append(ruleOpts, nodeOption{Label: o.Label, Reply: o.Reply, Next: next})
		}
		lv.Nodes = []scriptNode{
			lv.Nodes[0],
			{Prompt: "规则辨析：刚才真正该带走的是哪条？", Options: ruleOpts},
			lv.Nodes[1],
		}
		nd := lv.Nodes[len(lv.Nodes)-1]
		if len(nd.Options) == 0 {
			continue
		}
		opts := make([]tapOption, 0, len(nd.Options))
		correct := -1
		for i, o := range nd.Options {
			opts = append(opts, tapOption{Label: o.Label, Reply: o.Reply})
			if o.Next == NodeClear {
				correct = i
			}
		}
		if correct < 0 {
			continue
		}
		lv.Variants = append(lv.Variants, checkVariant{
			Ask: "延时迁移：" + nd.Prompt, Opts: opts, Correct: correct,
		})
		learnEnglishScript[slug] = lv
	}
}
