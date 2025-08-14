目标与产物

产物类型：ARCHITECTURE.md、MODULES.md、FLOWS.md、API.md、DEPLOY.md、per-folder README、CHANGELOG 摘要、依赖/外部系统清单。
交互能力：RAG 驱动的问答与导航（支持路径/符号/模块过滤、引用行号与证据）。
可增量：监听变更，局部重建索引与文档。
总体流水线

项目发现与画像
使用 project 扫描构建树，统计节点数、文件类型分布、语言生态（go.mod、package.json、requirements.txt 等）。
用 Node.CountNodes 作为规模阈值分级 Small/Medium/Large。
结合 blame 和 git 统计，识别热点文件与高 churn 区域，作为优先分析目标。
对文件计算内容哈希（你已有 CalculateHash），产出快照指纹，后续用于增量。
计划制定（Planner）
Small（< ~500 files）：全量解析+索引+总结。
Medium（< ~5k files）：按模块/包分层，优先热点+入口/导出符号，剩余延迟加载。
Large（>= ~5k files/monorepo）：分仓/分包分区构建，先骨架（目录/依赖/入口）后按需下钻。
明确预算（token/时间/并发）与产出范围（哪些文档先生成）。
索引与检索
语义索引：用 rag（Qdrant 已在）写入向量库，Chunk 策略按语言定制（函数/类型/类级别，保留路径、包、符号名、导出/私有、行号范围等元数据）。
关键词/结构索引：结合 search 建倒排索引、符号表。
元数据设计：{path, pkg/module, lang, symbol, kind, hash, lines, deps, lastCommit, churn}。
静态分析
语言特定分析优先（Go：go list -deps、包图、接口实现、入口 main、构建标记），其他语言通过 MCP 工具/LSP 适配。
依赖/调用图、配置/环境变量、外部系统（DB/queue/http）的使用面，落到统一的 “项目画像” 模型。
总结与文档生成（Chains）
Map-Reduce 风格链路：文件摘要链 -> 包/目录聚合链 -> 全局架构链。
使用 coroutine 控制并发与速率限制；pack 组合上下文窗口。
产出文档组件化：概览、模块、关键流程（请求->处理->存储）、数据模型、关键算法、运维部署、边界与风险。
全程插入“证据片段”（路径+行号），降低幻觉。
Agents 编排
Analyzer Agent：按计划驱动索引/分析/总结，负责重试与降级。
Doc Composer Agent：装配文档（选择模板，引用证据）。
QA Agent：RAG 问答，支持 filter（path/module/symbol）和 cite。
Refiner Agent：对草稿做一致性/术语校对。
增量与同步
基于哈希/文件时间与 git 变更集，仅重建受影响索引与文档段落。
变更感知策略：文件内容变、依赖图变、入口/导出符号变 -> 触发关联再生成。
质量控制
验证链：抽样比对文档声明与源代码证据；链接可点击到行号。
健康度报告：覆盖率（被索引/被总结文件比例）、热点覆盖、未识别语言/大型文件清单。
CLI/命令规划（映射到 cmd）

tong project profile — 扫描并输出项目画像与规模分级
tong index [--refresh-changed] — 构建/更新语义与关键词索引
tong analyze [--scope pkgA,pkgB|--hotspot] — 运行静态分析与总结链
tong doc [--sections arch,modules,flows] [--out ./docs] — 生成文档
tong ask "How does X work?" [--path ./pkg/foo] — RAG 问答，带引用
tong sync — 基于 git 变更做增量更新
tong serve — 本地文档与问答服务（可选）
配置（tong.json 建议扩展）

sources: include/exclude、最大文件大小、二进制忽略
chunking: per-language 策略与最大 token
rag: provider、维度、相似度阈值、命名空间
llm: 模型、并发、速率限制、重试
docs: sections、模板、输出目录、语言
plan: small/medium/large 阈值与预算
mcp/tools: 启用的外部分析器列表
与你现有代码的映射

project.Node/Tree：扫描、计数、路径、哈希、增量判定与并发遍历
project/search 与 pack：候选集筛选与上下文打包
rag/*：向量库写入/检索，Qdrant 管理
helper/coroutine：并发管控/池化
mcp/*：调用语言工具链/LSP/构建系统
prompt/manager：模板化各级链路提示
helper/git + blame：热点/变更驱动的优先级输入
规模分级建议

Small < 500 文件：全量索引+全量文档
Medium < 5k 文件：按包分层，热点优先，其余延迟
Large >= 5k/monorepo：骨架优先（架构/依赖/入口），其余按需与交互驱动
里程碑

M1：profile+index+基础 doc（概览/模块）+ask
M2：静态分析（依赖/调用图）+flows 文档+增量更新
M3：跨语言 MCP 适配+质量验证链+服务化
M4：大仓优化（分区/命名空间/缓存）+团队协作产物（PR 注释/可视化）
关键原则

证据优先（路径+行号），分层总结，先骨架后细节，增量优先，成本可控，可观测且可回滚。
需要时我可以帮你先落个最小版：profile/index/analyze/doc 四个命令与基础链路模板。