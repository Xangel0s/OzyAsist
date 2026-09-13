---
name: ozy-agent-loop
description: >-
  Use this skill for any task involving the OzyAssist AI agent loop: modifying the ReAct
  loop, adding/removing tools, changing prompts, adjusting streaming behavior, working with
  providers, or understanding how messages flow from WebSocket to LLM and back.
  Critical for voice mode, tool execution, and response streaming.
---

# OzyAssist Agent Loop — Skill

## Request Flow (End-to-End)

```
Browser (React) 
  → WebSocket send: {type:"message", content:"...", voice_mode: bool, chat_id: "..."}
  
backend/internal/api/ws/chat.go (HandleWebSocket)
  → parse clientMessage
  → create AgentLoopParams{VoiceMode: msg.VoiceMode, ...}
  → go runReActLoopSession(params)  ← goroutine
  
backend/internal/agent/loop.go (runReActLoopSession)
  → select system prompt: voice (~50 tokens) or normal (~2000 tokens)
  → select tools: VoiceAgentTools (8) or AgentTools (18+)
  → maxTurns = 25
  → for turn := 0; turn < maxTurns; turn++:
      → StreamCompletion(messages, tools)  → LLM call
      → if response has tool_calls → executeToolCall() → append result → continue
      → if response is final text → emit("message:delta") per token
                                  → emit("agent:completed")
                                  → break
  
Browser receives events:
  → "agent:thinking" — show thinking indicator
  → "message:delta" — append streaming text
  → "agent:completed" / "done" — handleAssistantResponse() → speakText()
```

## AgentLoopParams Structure

```go
type AgentLoopParams struct {
    ChatID       string
    UserMessage  string
    VoiceMode    bool        // true = compact prompt + 8 tools (voice path)
    UserID       string
    Emit         func(AgentEvent)  // sends WS events to client
    Ctx          context.Context
}
```

## System Prompts

### Normal Mode (~2000 tokens)
Full assistant persona: date/time, OS context, user profile, tool descriptions, task guidelines.

### Voice Mode (~50 tokens)
```
You are Ozy, a fast voice AI assistant. Keep responses concise (1-3 sentences max).
Current time: {time}. User: {username}.
```

## Tool Sets

### VoiceAgentTools (8 tools — voice only)
1. `os_execute_command` — run shell commands
2. `os_open_application` — open apps
3. `os_get_system_info` — CPU/RAM/disk
4. `os_manage_windows` — window focus/minimize
5. `os_screenshot` — take screenshot
6. `web_search` — internet search
7. `os_read_file` — read file content
8. `os_write_file` — write file content

### AgentTools (18+ tools — full chat mode)
All VoiceAgentTools + project indexing, git operations, memory search, code analysis, etc.

## Adding a New Tool

### 1. Define schema in `agent/tools.go`
```go
{
    Type: "function",
    Function: &FunctionDef{
        Name:        "my_new_tool",
        Description: "What this tool does — be specific for the LLM",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "param1": map[string]interface{}{
                    "type": "string", "description": "...",
                },
            },
            "required": []string{"param1"},
        },
    },
}
```

### 2. Add execution in `agent/executor.go`
```go
case "my_new_tool":
    // Extract params from toolCall.Function.Arguments (JSON)
    var args struct { Param1 string `json:"param1"` }
    json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
    result = myNewToolImpl(args.Param1)
```

### 3. Also add to VoiceAgentTools if OS-related

## Modifying the System Prompt

In `agent/loop.go`, find the prompt construction:

```go
if params.VoiceMode {
    systemPrompt = fmt.Sprintf("You are Ozy, a fast voice AI...")
} else {
    systemPrompt = buildFullSystemPrompt(params)  // normal mode
}
```

## Streaming Response Events (WS)

The `Emit` function sends these events to the frontend:

```go
params.Emit(AgentEvent{Type: "agent:thinking"})        // start
params.Emit(AgentEvent{Type: "message:delta", Content: token}) // each token
params.Emit(AgentEvent{Type: "tool:start", ToolName: "..."})   // tool call
params.Emit(AgentEvent{Type: "tool:result", Content: "..."})   // tool result
params.Emit(AgentEvent{Type: "agent:completed"})               // done
params.Emit(AgentEvent{Type: "error", Error: "..."})           // error
```

Frontend subscribes via: `wsService.subscribe("agent:completed", handler)`

## Critical Rules
- Never block the goroutine in `runReActLoopSession` — use `ctx.Done()` for cancellation
- `VoiceMode=true` must ALWAYS use the compact prompt and VoiceAgentTools — never full tools
- `maxTurns=25` prevents infinite loops; if reached, emit a user-friendly completion message
- Tool errors should NOT terminate the loop — append as tool result and continue reasoning
- The `Emit` function is goroutine-safe but must check `ctx.Err()` before sending
