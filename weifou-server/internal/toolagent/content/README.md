# 课程内容目录（2026-07-16 内容外置）

`courses/<agent-slug>.json` 一门课一个文件，装下该课的全部作者内容。这里是**内容的唯一真源**——改课程内容改这里，不再改 Go 代码。

## 文件结构

```jsonc
{
  "slug": "learn-negotiation",        // 必须与文件名一致
  "tierLabels": { "1": "第一幕·…" },  // 幕名（键=幕号）
  "tierOrder": [1, 2, 3],             // 幕展示顺序
  "freeJudgeRubric": "…",             // 产出节点(free)的 LLM 判分口径；无产出节点可省略
  "concepts": [                        // 课程表：数组顺序=关卡顺序(Sort)
    {
      "slug": "batna",                // 关卡稳定 id：⚠️ 改了会丢用户进度，只增不改
      "theme": "换一副谈判脑",         // 章名
      "name": "你的底牌",              // 关名
      "blurb": "一句话点题",
      "tier": 1,                       // 所属幕
      "hook": "开课钩子（场景问题）",
      "check": "检验题（应用/迁移型）",
      "takeaway": "章末知识卡结论",     // 仅 Boss 关
      "source": "引用来源"             // 仅 Boss 关
    }
  ],
  "levels": {                          // 剧本：concept slug → 关卡剧本
    "batna": {
      "taps": [{ "label": "点选文字", "reply": "选中后的回应" }],
      "checkOpts": [{ "label": "…", "reply": "答错时的纠偏语" }],
      "correct": 0,                    // checkOpts 中正确项下标
      "variants": [ … ],               // 复习变式题库（可选）
      "nodes": [ … ],                  // 对手戏节点图（产出课；有 nodes 时 taps 不用）
      "clear": "点亮语+下一关悬念",
      "note": "战报（≤20字）"
    }
  }
}
```

## 生效方式

- **默认**：文件经 `go:embed` 编进二进制——改动后重新编译部署即生效，测试守护
  （`TestCuratedContentComplete` 等）在 CI 里保证内容完整性。
- **线上热改（不重编译）**：把改过的 JSON 放到服务器某目录，服务进程带
  `COURSE_CONTENT_DIR=<目录>` 启动/重启即生效。目录里没有的课回落内嵌版；
  文件解析失败只告警并回落，不会拖垮启动。**热改是应急通道，改完记得把同样的改动
  提回仓库**，否则下次发版会被内嵌版覆盖。

## 校验

改完先跑：`go test ./internal/toolagent/`——课程结构、Hook/Check 完整性、Boss 卡
一一对应、剧本选项正确下标等都有测试守护，红了别上线。

## 纪律

- `concepts[].slug` 是用户进度的锚：**只增不改不删**（删除=该关用户进度成孤儿，改名=进度丢失）。
- 课程上下架不在这里：在 `toolagent.go` 的 `Seed` presets（下架=软退役，进度保留）。
- 新增一门课 = 新建 `<slug>.json` + 在 `Seed` presets 里加一条元信息。
