# 目标

实现按语义分割的方案

## 算法说明

**Step 1**

先调用 structured 对给定的 Stream 进行大切片分割。 切片参数为:

- structured 切片大小: 读取 SemanticOptions.ContextSize 参数， 默认为 ChunkingOptions.Size x ChunkingOptions.MaxDepth x 3。 SemanticOptions.ContextSize 不合法时使用默认参数。
- ChunkingOptions.Overlap 默认值为 50, 如果用户输入的 ChunkingOptions.Overlap 不合法或者 <=0 或 > Size ， 则设置为 默认值 50
- structured 切片层级为: 1

**Step 2**

并发调用 LLM ， 对 structured 切片按语义， LLM 回复标记分段 start_pos 和 end_pos

- SemanticOptions.Connector 为用于语义分割的 LLM 连接器，从中读取 URL 和 Model 等信息。
- SemanticOptions.Options 调用 LLM Request 时候的扩展参数。
- SemanticOptions.Prompt 自定义语义分割提示词。默认写一个标准的提示词。
- SemanticOptions.Toolcall 该 Model 是否支持 toolcall, 如果支持 Toolcall 优先使用 Toolcall 方式输出。
- ChunkingOptions.Size 为切片的最大长度，原则上不应该超过这个长度。

默认提示词和输出要求:

- LLM 输出应该为一个 JSON Array， 包含 start_pos 和 end_pos 数组，切记不要让 LLM 输出切片的 Text，以提升生成速度。
- 如果支持 Toolcall 调用，优先使用 Toolcall 的方式输出。 如果不支持，应该有返回标记，根据返回值验证结果。
- 应该有重试机制，如果 LLM 输出不符合要求，应该可以自动 Retry 携带错误信息，让 LLM 修复，直至成功处理或者到达 SemanticOptions.MaxRetry (默认 9 次)限制。
- 处理 Toolcall 返回数据时，应该使用 容错的 JSON 解析。
- 对于有些模型可能会有 Thinking 输出，注意处理。

并发调用要求:

- 使用 Stream 方式，实时输出结果。
- 最大并发数量为 SemanticOptions.MaxConcurrent, **注意不是 ChunkingOptions.MaxConcurrent**
- 处理过程中应该有个处理进度的回调，实时返回每个切片处理过程。 start_pos 和 end_pos，这个信息将被用于前台试试呈现。
- 进度回调应该在创建 semantic 对象时传入，可以为 nil ， 如果为 null 则不回调。

**Step 3**

根据标注的位置信息，分割切片。

- 注意 Overlap 的处理，按语义切片无需包含 Overlap 部分的内容。
- 需要实时通报进度，这个信息将被用于前台格式呈现。

**Step 4**

按 ChunkingOptions.MaxDepth 递归向上合并产生父级切片。

## 代码要求

- 过程中需要建立的通用的工具函数，写到 utils 包里。 比如 LLM 调用等。
- 一些通用的算法等。

## 单元测试要求

1. 需要覆盖实现的所有代码。
2. 需要有压力测试
3. 需要有内存检查测试。
4. 测试文本使用 tests/semantic-en.text, tests/semantic-zh.text
5. 测试的模型分别使用：
   支持 Toolcall 的连接器环境变量: Key OPENAI_TEST_KEY, 模型固定为 gpt-4o-mini, 自动构建一个测试用 Connector
   不支持 Toolcall 的连接器环境变量: URL RAG_LLM_TEST_URL, Key RAG_LLM_TEST_KEY, 模型 RAG_LLM_TEST_SMODEL, 自动构建一个测试用 Connector
