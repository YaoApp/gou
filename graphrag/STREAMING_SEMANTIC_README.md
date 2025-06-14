# Streaming Semantic Analysis

## Overview

This project implements LLM-based streaming semantic analysis functionality with real-time progress reporting and semantic position parsing. Key features include:

- **Real-time Streaming Processing**: Support for real-time parsing of LLM streaming responses
- **Semantic Position Extraction**: Accumulate and parse semantic segment positions from streaming data
- **Fault-tolerant JSON Parsing**: Use JSONRepair to handle incomplete JSON data
- **Progress Callbacks**: Provide detailed real-time progress reporting
- **Multiple Response Formats**: Support both regular response and tool call modes

## Core Components

### 1. StreamParser (Stream Parser)

Location: `gou/graphrag/utils/utils.go`

**Features**:

- Parse LLM streaming response data
- Accumulate content and extract semantic positions
- Handle incomplete JSON data
- Support SSE format (`data:` prefix)

**Usage Example**:

```go
// Create parser
parser := utils.NewStreamParser(false) // false = regular response, true = tool call

// Parse streaming data chunk
data, err := parser.ParseStreamChunk(chunkBytes)
if err != nil {
    log.Printf("Parse error: %v", err)
}

// Get accumulated semantic positions
positions := data.Positions
fmt.Printf("Parsed %d semantic positions", len(positions))
```

### 2. TolerantJSONUnmarshal (Fault-tolerant JSON Parsing)

**Features**:

- Use JSONRepair to automatically repair damaged JSON
- Provide stronger fault tolerance than standard json.Unmarshal

**Usage Example**:

```go
var result map[string]interface{}
err := utils.TolerantJSONUnmarshal(jsonBytes, &result)
```

### 3. SemanticChunker (Semantic Chunker)

Location: `gou/graphrag/chunking/semantic.go`

**Features**:

- Use streaming LLM for semantic analysis
- Real-time progress reporting
- Support concurrent processing
- Automatic retry mechanism

**Configuration Options**:

```go
options := &types.ChunkingOptions{
    Size:          200,
    Overlap:       20,
    MaxDepth:      2,
    MaxConcurrent: 2,
    SemanticOptions: &types.SemanticOptions{
        Connector:     "local-llm",
        MaxRetry:      3,
        MaxConcurrent: 4,
        ContextSize:   800,
        Toolcall:      false,
        Prompt:        "", // Empty string uses default prompt
    },
}
```

## Connector Configuration

### OpenAI Connector

```yaml
name: "openai"
type: "openai"
options:
  model: "gpt-4o-mini"
  proxy: "https://api.openai.com/v1" # Required field
  key: "your-api-key"
```

### Local LLM Connector

```yaml
name: "local-llm"
type: "openai"
options:
  model: "qwen3:8b"
  host: "127.0.0.1:11434" # Automatically adds /v1 suffix
```

## Progress Callback System

Progress callbacks provide detailed real-time status information:

```go
progressCallback := func(chunkID, progress, step string, data interface{}) error {
    switch step {
    case "llm_response":
        // Streaming LLM response progress
        if dataMap, ok := data.(map[string]interface{}); ok {
            posCount := dataMap["positions_count"]
            contentLen := dataMap["content_length"]
            finished := dataMap["finished"]
            fmt.Printf("Positions: %v, Content: %v, Finished: %v", posCount, contentLen, finished)
        }
    case "semantic_analysis":
        // Semantic analysis progress
    case "semantic_chunk":
        // Semantic chunk generation progress
    }
    return nil
}
```

## Streaming Data Formats

### Regular Response Format

```json
{
  "choices": [
    {
      "delta": {
        "content": "[{\"start_pos\": 0, \"end_pos\": 100}]"
      }
    }
  ]
}
```

### Tool Call Format

```json
{
  "choices": [
    {
      "delta": {
        "tool_calls": [
          {
            "function": {
              "arguments": "{\"segments\": [{\"start_pos\": 0, \"end_pos\": 100}]}"
            }
          }
        ]
      }
    }
  ]
}
```

### SSE Format

```
data: {"choices":[{"delta":{"content":"..."}}]}
data: [DONE]
```

## Semantic Position Data Structure

```go
type SemanticPosition struct {
    StartPos int `json:"start_pos"`
    EndPos   int `json:"end_pos"`
}
```

## Test Coverage

The project includes comprehensive test suites:

- **Streaming Parse Tests**: Verify parsing of various streaming data formats
- **JSON Completion Tests**: Test automatic repair of incomplete JSON
- **Semantic Analysis Integration Tests**: End-to-end semantic analysis process tests
- **Connector Tests**: Verify configuration of different LLM connectors
- **Concurrency Tests**: Verify thread safety
- **Error Handling Tests**: Test handling of various exception scenarios

Run tests:

```bash
# Run all tests
go test ./graphrag/utils ./graphrag/chunking -v

# Run specific tests
go test ./graphrag/utils -run "TestStreamParser" -v
```

## Performance Metrics

Benchmark results:

- **StreamParser**: ~61μs/operation
- **TolerantJSONUnmarshal**: ~0.9μs/operation
- **Memory Usage**: Optimized accumulative buffer management
- **Concurrency Performance**: Support configurable concurrency levels

## Demo Program

Run the demo program to see complete functionality:

```bash
go run graphrag/examples/streaming_semantic_demo.go
```

Demo program showcases:

- Real-time progress reporting
- Semantic position parsing
- Streaming data processing
- Result statistics analysis

## Major Improvements

1. **Fixed Connector 404 Errors**:

   - OpenAI connector adds required `proxy` field
   - Local LLM connector automatically adds `/v1` path suffix

2. **Implemented Streaming Processing**:

   - Replace `PostLLM` with `StreamLLM`
   - Real-time semantic position parsing
   - Accumulative content processing

3. **Enhanced Fault Tolerance**:

   - JSONRepair integration
   - Incomplete JSON auto-completion
   - Graceful error handling

4. **Optimized User Experience**:
   - Detailed progress callbacks
   - Real-time status updates
   - Rich debugging information

## Technical Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   User Request  │───▶│  SemanticChunker │───▶│ Progress Callback│
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   StreamLLM     │◀───│   LLM Connector  │───▶│ Streaming Response│
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                                               │
         ▼                                               ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  StreamParser   │───▶│ Position Parsing │───▶│ Semantic Chunks │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │
         ▼
┌─────────────────┐
│ TolerantJSON    │
│ Unmarshal       │
└─────────────────┘
```

This implementation provides a complete streaming semantic analysis solution with high performance, high reliability, and excellent user experience.
