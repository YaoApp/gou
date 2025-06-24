package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/graphrag/extraction/openai"
	"github.com/yaoapp/gou/graphrag/utils"
)

const (
	testConnectorName = "openai_test"
	testModel         = "gpt-4o"
)

// setupConnector creates connectors using the same pattern as openai_test.go
func setupConnector() error {
	// Create OpenAI connector using environment variables
	openaiKey := os.Getenv("OPENAI_TEST_KEY")
	if openaiKey == "" {
		openaiKey = "mock-key"
	}

	openaiDSL := fmt.Sprintf(`{
		"LANG": "1.0.0",
		"VERSION": "1.0.0", 
		"label": "OpenAI Test",
		"type": "openai",
		"options": {
			"proxy": "https://api.openai.com/v1",
			"model": "%s",
			"key": "%s"
		}
	}`, testModel, openaiKey)

	_, err := connector.New("openai", testConnectorName, []byte(openaiDSL))
	if err != nil {
		return fmt.Errorf("failed to create OpenAI connector: %v", err)
	}

	return nil
}

func main() {
	// 设置连接器
	err := setupConnector()
	if err != nil {
		log.Fatalf("Failed to setup connector: %v", err)
	}

	// 测试文本
	testText := "Yao is a tool that focuses on generative programming, helping to build applications faster and offering a great development experience."

	// 创建OpenAI提取器（使用toolcall模式）
	extractor, err := openai.NewOpenaiWithDefaults(testConnectorName)
	if err != nil {
		log.Fatalf("Failed to create extractor: %v", err)
	}

	fmt.Printf("=== 测试配置 ===\n")
	fmt.Printf("模型: %s\n", extractor.GetModel())
	fmt.Printf("工具调用模式: %t\n", extractor.GetToolcall())
	fmt.Printf("测试文本: %s\n", testText)
	fmt.Printf("\n")

	// 创建自定义解析器用于调试
	parser := utils.NewExtractionParser()
	parser.SetToolcall(true)

	// 测试工具调用定义
	fmt.Printf("=== 工具调用定义 ===\n")
	toolcall := utils.GetExtractionToolcall()
	toolcallJSON, _ := json.MarshalIndent(toolcall[0], "", "  ")
	fmt.Printf("%s\n\n", string(toolcallJSON))

	// 创建回调函数来捕获原始响应并传递给解析器
	var rawResponses []string
	callback := func(data []byte) error {
		if len(data) > 0 {
			rawResponses = append(rawResponses, string(data))
			fmt.Printf("原始流式响应块: %s\n", string(data))

			// 让解析器处理这个数据块
			_, _, err := parser.ParseExtractionEntities(data)
			if err != nil {
				fmt.Printf("解析器处理错误: %v\n", err)
			}
		}
		return nil
	}

	// 准备提取请求
	fmt.Printf("=== 开始提取 ===\n")

	// 手动构建请求来调试
	systemPrompt := utils.ExtractionPrompt("")
	fmt.Printf("系统提示长度: %d 字符\n", len(systemPrompt))

	messages := []map[string]interface{}{
		{
			"role":    "system",
			"content": systemPrompt,
		},
		{
			"role":    "user",
			"content": fmt.Sprintf("Please extract entities and relationships from the following text:\n\n%s", testText),
		},
	}

	payload := map[string]interface{}{
		"model":       extractor.GetModel(),
		"messages":    messages,
		"temperature": extractor.GetTemperature(),
		"max_tokens":  extractor.GetMaxTokens(),
		"tools":       utils.ExtractionToolcall,
		"tool_choice": "required",
	}

	fmt.Printf("请求payload: %+v\n\n", payload)

	// 使用连接器直接调用
	ctx := context.Background()
	err = utils.StreamLLM(ctx, extractor.Connector, "chat/completions", payload, callback)
	if err != nil {
		log.Fatalf("流式调用失败: %v", err)
	}

	fmt.Printf("\n=== 原始响应分析 ===\n")
	fmt.Printf("总响应块数: %d\n", len(rawResponses))

	// 分析累积的参数
	fmt.Printf("累积的工具调用参数: %s\n", parser.Arguments)

	if parser.Arguments == "" {
		fmt.Printf("❌ 未收到任何工具调用参数!\n")
		return
	}

	// 尝试解析参数
	fmt.Printf("\n=== 解析测试 ===\n")

	// 1. 直接JSON解析测试
	var argsMap map[string]interface{}
	err = json.Unmarshal([]byte(parser.Arguments), &argsMap)
	if err != nil {
		fmt.Printf("❌ JSON解析失败: %v\n", err)
		fmt.Printf("原始参数: %s\n", parser.Arguments)
	} else {
		fmt.Printf("✅ JSON解析成功\n")

		// 检查实体
		if entities, ok := argsMap["entities"].([]interface{}); ok {
			fmt.Printf("实体数量: %d\n", len(entities))
			for i, entity := range entities {
				if entityMap, ok := entity.(map[string]interface{}); ok {
					fmt.Printf("实体 %d:\n", i+1)
					for key, value := range entityMap {
						fmt.Printf("  %s: %v\n", key, value)
					}

					// 检查关键字段
					if _, hasLabels := entityMap["labels"]; hasLabels {
						fmt.Printf("  ✅ 有labels字段\n")
					} else {
						fmt.Printf("  ❌ 缺少labels字段\n")
					}

					if _, hasProps := entityMap["properties"]; hasProps {
						fmt.Printf("  ✅ 有properties字段\n")
					} else {
						fmt.Printf("  ❌ 缺少properties字段\n")
					}
				}
			}
		}

		// 检查关系
		if relationships, ok := argsMap["relationships"].([]interface{}); ok {
			fmt.Printf("关系数量: %d\n", len(relationships))
			for i, rel := range relationships {
				if relMap, ok := rel.(map[string]interface{}); ok {
					fmt.Printf("关系 %d:\n", i+1)
					for key, value := range relMap {
						fmt.Printf("  %s: %v\n", key, value)
					}

					// 检查关键字段
					if _, hasProps := relMap["properties"]; hasProps {
						fmt.Printf("  ✅ 有properties字段\n")
					} else {
						fmt.Printf("  ❌ 缺少properties字段\n")
					}

					if _, hasWeight := relMap["weight"]; hasWeight {
						fmt.Printf("  ✅ 有weight字段\n")
					} else {
						fmt.Printf("  ❌ 缺少weight字段\n")
					}
				}
			}
		}
	}

	// 2. 使用解析器解析
	fmt.Printf("\n=== 使用解析器解析 ===\n")
	nodes, relationships, err := parser.ParseExtractionToolcall(parser.Arguments)
	if err != nil {
		fmt.Printf("❌ 解析器解析失败: %v\n", err)
	} else {
		fmt.Printf("✅ 解析器解析成功\n")
		fmt.Printf("解析得到实体数: %d\n", len(nodes))
		fmt.Printf("解析得到关系数: %d\n", len(relationships))

		// 检查解析后的实体
		for i, node := range nodes {
			fmt.Printf("解析后实体 %d: ID=%s, Labels=%v, Properties=%v\n",
				i+1, node.ID, node.Labels, node.Properties)
		}

		// 检查解析后的关系
		for i, rel := range relationships {
			fmt.Printf("解析后关系 %d: Type=%s, Properties=%v, Weight=%f\n",
				i+1, rel.Type, rel.Properties, rel.Weight)
		}
	}
}
