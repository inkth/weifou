# 微否小程序 · 命令行部署（miniprogram-ci）

用官方 [`miniprogram-ci`](https://developers.weixin.qq.com/miniprogram/dev/devtools/ci.html) 直接从命令行上传 / 预览小程序，不用再开微信开发者工具点「上传」。

编译设置与 appid 全部从 `weifou-miniapp/project.config.json` 读取并对齐——单一真源，不会和 DevTools 跑偏。

## 一次性准备（只做一次）

1. **下载上传密钥**：[mp.weixin.qq.com](https://mp.weixin.qq.com) → 开发管理 → 开发设置 →「小程序代码上传」→ 生成并下载**上传密钥**。
2. 把密钥重命名为 `private.wx62c5c38678568751.key`，放到本目录 `tools/mp-ci/` 下。
   （已被 `.gitignore` 忽略，**绝不会进仓库**；根 `.gitignore` 也已忽略 `*.key`。）
3. **IP 白名单**：同一页面，把你的公网出口 IP 加进「IP 白名单」，否则上传会被微信拒绝。
   - 查公网 IP：`curl ifconfig.me`
   - 换网络（如从公司到家里）要记得补白名单。

## 安装

```bash
cd tools/mp-ci
npm install
```

## 用法

```bash
npm run check                       # 预检：工程 / appid / 密钥是否就位（不上传）
npm run upload                      # 上传开发版（版本号默认 日期-commit，描述默认最近一条 commit）
npm run upload -- --ver 1.2.0 --desc "碧绿配色上线"   # 自定义版本号 / 描述
npm run preview                     # 生成预览二维码（终端打印 + preview-qr.jpg），微信扫码真机预览
```

## 上线完整流程

```
改代码 → git push（main）→ cd tools/mp-ci && npm run upload
        → mp.weixin.qq.com → 版本管理 → 开发版本（找到刚传的版本）
        → 设为「体验版」自测，或「提交审核」→ 审核通过后「发布」
```

> `upload` 只负责把代码传成「开发版」。**「提交审核 / 发布」仍需在微信公众平台点**——这是微信的硬性流程（涉及人工审核），任何 CI 都绕不过。

## 备注

- `robot` 机器人编号默认 `1`（1–30，用于在版本管理里区分上传来源），可加 `--robot 2`。
- 本目录独立于 `weifou-miniapp/` 工程，`node_modules` 不会被打进小程序包。
