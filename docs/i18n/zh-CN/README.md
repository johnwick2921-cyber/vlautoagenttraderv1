names rewritten to vl on 2026-09-30 (VL rename)
<h1 align="center">VL Intelligent</h1>

<p align="center">
  <strong>你的个人 AI 交易助手。</strong><br/>

</p>

<p align="center">
</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go"></a>
  <a href="https://reactjs.org/"><img src="https://img.shields.io/badge/React-18+-61DAFB?style=flat&logo=react" alt="React"></a>
</p>

<p align="center">
  <a href="../../../README.md">English</a> ·
  <a href="README.md">中文</a> ·
  <a href="../ja/README.md">日本語</a> ·
  <a href="../ko/README.md">한국어</a> ·
  <a href="../ru/README.md">Русский</a> ·
  <a href="../uk/README.md">Українська</a> ·
  <a href="../vi/README.md">Tiếng Việt</a>
</p>

> **语言声明：** 本中文版本文档仅为方便海外华人社区阅读而提供，不代表本软件面向中国大陆、香港、澳门或台湾地区用户开放。如您位于上述地区，请勿使用本软件。

---

VL Intelligent 是一个开源的**自主式** AI 交易助手。与需要手动配置模型、管理 API Key、接入数据源的传统 AI 工具不同 —— VL Intelligent 的 AI **自主感知市场、自选模型、自动获取数据**。零人工干预。你只需设定策略，AI 负责一切。





打开 **http://127.0.0.1:3000**，完成。

---

## 快速演示

<p align="center">
  <a href="https://drive.google.com/file/d/1frzw-HDZ3viQvLOQKsAJGc9bT0dXs68D/view">
    <img src="../../../screenshots/demo-cover.png" alt="VL Intelligent 快速演示视频" width="900"/>
  </a>
</p>

<p align="center">
  点击封面图即可观看 Demo 视频。
</p>

---

## 功能概览

| 功能 | 描述 |
| **多 AI** | DeepSeek、Qwen、GPT、Claude、Gemini、Grok、Kimi、MiniMax — 随时切换 |

| **策略工作室** | 可视化构建器 — 币种来源、指标、风控 |
| **AI 竞赛** | AI 实时竞争，排行榜排名 |
| **Telegram Agent** | 与交易助手对话 — 流式输出、工具调用、记忆 |
| **回测实验室** | 历史模拟，权益曲线和性能指标 |
| **仪表板** | 实时持仓、盈亏、AI 决策日志与思维链 |

### 市场

加密货币 · 美股 · 外汇 · 贵金属

### AI 模型 (API Key 模式)

| AI 模型 | 状态 | 获取 API Key |
| <img src="../../../web/public/icons/deepseek.svg" width="20" height="20" style="vertical-align: middle;"/> **DeepSeek** | ✅ | [获取 API Key](https://platform.deepseek.com) |
| <img src="../../../web/public/icons/qwen.svg" width="20" height="20" style="vertical-align: middle;"/> **通义千问** | ✅ | [获取 API Key](https://dashscope.console.aliyun.com) |
| <img src="../../../web/public/icons/openai.svg" width="20" height="20" style="vertical-align: middle;"/> **OpenAI (GPT)** | ✅ | [获取 API Key](https://platform.openai.com) |
| <img src="../../../web/public/icons/claude.svg" width="20" height="20" style="vertical-align: middle;"/> **Claude** | ✅ | [获取 API Key](https://console.anthropic.com) |
| <img src="../../../web/public/icons/gemini.svg" width="20" height="20" style="vertical-align: middle;"/> **Gemini** | ✅ | [获取 API Key](https://aistudio.google.com) |
| <img src="../../../web/public/icons/grok.svg" width="20" height="20" style="vertical-align: middle;"/> **Grok** | ✅ | [获取 API Key](https://console.x.ai) |
| <img src="../../../web/public/icons/kimi.svg" width="20" height="20" style="vertical-align: middle;"/> **Kimi** | ✅ | [获取 API Key](https://platform.moonshot.cn) |
| <img src="../../../web/public/icons/minimax.svg" width="20" height="20" style="vertical-align: middle;"/> **MiniMax** | ✅ | [获取 API Key](https://platform.minimaxi.com) |

---

## 安装

安装 [Docker Desktop](https://www.docker.com/products/docker-desktop/)，然后：

### 从源码构建

```bash
# 前置条件: Go 1.21+, Node.js 18+, TA-Lib
# macOS: brew install ta-lib
# Ubuntu: sudo apt-get install libta-lib0-dev

go build -o vl-bin && ./vl-bin          # 后端
cd web && npm install && npm run dev  # 前端 (新终端)
```

### 更新

---

## 配置

**新手模式**：首次使用的用户可以在注册时选择新手模式，系统会引导你逐步完成 AI、交易所和策略的配置。

**进阶模式**：


2. **交易所** — 连接交易所 API 凭证
3. **策略** — 在策略工作室构建
4. **交易员** — 组合 AI + 交易所 + 策略
5. **交易** — 从仪表板启动

所有操作通过 Web 界面完成：**http://127.0.0.1:3000**

---

## 文档

| [架构概览](../../architecture/README.md) | 系统设计和模块索引 |
| [策略模块](../../architecture/STRATEGY_MODULE.md) | 币种选择、AI 提示词、执行 |
| [常见问题](../../faq/README.md) | FAQ |
| [快速开始](../../getting-started/README.md) | 部署指南 |

---

## 贡献

查看 [贡献指南](../../../CONTRIBUTING.md) · [行为准则](../../../CODE_OF_CONDUCT.md) · [安全政策](../../../SECURITY.md)

### 贡献者空投计划

所有贡献在 GitHub 上追踪。当 VL Intelligent 产生收入时，贡献者将获得空投。

| 贡献类型 | 权重 |
| 置顶 Issue PR | ★★★★★★ |
| 代码提交 (合并的 PR) | ★★★★★ |
| Bug 修复 | ★★★★ |
| 功能建议 | ★★★ |
| Bug 报告 | ★★ |
| 文档 | ★★ |

---

> **风险提示**: AI 自动交易存在重大风险。建议仅用于学习/研究或小额测试。

---

## License

[AGPL-3.0](../../../LICENSE)

