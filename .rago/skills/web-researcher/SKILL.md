---
name: web-researcher
description: "通过 URL 抓取网页并生成结构化的分析报告"
version: 1.1.0
category: research
tags: [web, scrape, research]
variables:
  - name: url
    type: string
    required: true
    description: "要分析的网页 URL"
---

# 网页总结助手

## 执行步骤
1. 使用 `mcp_fetch_get` 工具（或任何可用的网页抓取工具）获取输入的 URL 原始内容。
2. 提取网页的核心观点、关键数据和最终结论。
3. 为忙碌的开发者生成一份结构化的 Markdown 技术摘要。

```input:url
```
