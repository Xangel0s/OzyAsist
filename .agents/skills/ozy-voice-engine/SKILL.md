---
name: ozy-voice-engine
description: >-
  Use this skill for any task involving the OzyAssist voice system: Hey Ozy wake word,
  OzyLive overlay, TTS (Text-to-Speech), STT (Speech-to-Text), VoiceMode backend path,
  watchdog timer, echo prevention, or audio hardware integration. Contains all known bugs,
  gotchas, and the current architecture of the voice pipeline.
---

# OzyAssist Voice Engine — Skill

## Voice Pipeline Overview

```
[Micrófono]
    │
    ▼ Web Speech API (SpeechRecognition)
[STT — OzyLiveOverlay.tsx]
    │  rec.continuous = true, lang = "es-ES"
    │  silenceTimer debounce: 1100ms
    │  SKIP if stateRef.current === "speaking" | "thinking"  ← echo prevention
    ▼
[handleDispatchMessage(text)]
    │  watchdog: 90s timeout → cancelResponse() + startListening()
    │  sendMessage(chatId, text, voiceMode=true)
    ▼
[WebSocket → backend ws/chat.go]
    │  clientMessage.VoiceMode = true
    │  runReActLoopSession(AgentLoopParams{VoiceMode: true})
    ▼
[agent/loop.go — VoiceMode path]
    │  System prompt: ~50 tokens (vs 2000 normal)
    │  Tools: VoiceAgentTools (8 tools, ~700 tokens vs 3000)
    │  Watchdog backend: stream timeout guard
    ▼
[WS streaming response → frontend]
    │  wsService events: "message:delta", "agent:completed", "done"
    ▼
[handleAssistantResponse()]
    │  Guard: isSpeakingResponseRef (prevents double-fire)
    │  speakText(content, onFinish → startListening)
    ▼
[TTS — SpeechSynthesis]
    │  synthRef.current.resume()  ← MANDATORY: Chrome suspends on idle
    │  synthRef.current.speak(utterance)
    │  voicesRef: loaded via onvoiceschanged (async)
    │  Preference: "Google" | "Natural" | "Sabina" | "Helena" | "Pablo" es-*
    ▼
[setState("speaking")]
    │  onend / onerror → setState("listening") + startListening()
    ▼
[Loop back to STT]
```

## Key Files

| File | Responsibility |
|------|---------------|
| `frontend/src/components/Voice/OzyLiveOverlay.tsx` | Main voice UI, STT, TTS, watchdog, state machine |
| `backend/internal/agent/loop.go` | `AgentLoopParams.VoiceMode` — compact prompt + tools |
| `backend/internal/agent/tools.go` | `VoiceAgentTools` — 8 OS tools for voice |
| `backend/internal/api/ws/chat.go` | Reads `voice_mode` from WS JSON, sets VoiceMode |
| `frontend/src/store/chatStore.ts` | `sendMessage(chatId, content, voiceMode)` |
| `frontend/src/services/ws.ts` | Sends `voice_mode: true` in WS payload |

## State Machine (OzyLiveOverlay)

```
idle → listening → thinking → speaking → listening → ...
                ↘ (error/busy) ↗
```

- **idle**: modal closed
- **listening**: mic active, waiting for speech
- **thinking**: sent to backend, waiting for response
- **speaking**: TTS playing response
- Transitions driven by `setState()` + `stateRef.current` (mirror ref for closures)

## Known Issues & Fixes Applied

| Bug | Fix |
|-----|-----|
| TTS silent after tab idle | `synth.resume()` before `speak()` |
| Voces españolas no disponibles en primer render | `onvoiceschanged` async loader → `voicesRef` |
| Watchdog loop infinito | Guard `isSpeakingResponseRef`, `cancelResponse()` en watchdog, `Error("busy")` en store |
| Eco de Ozy captado por mic | `onresult` ignora si `stateRef === "speaking"\|"thinking"` |
| Doble disparo speakText (store + WS event) | `isSpeakingResponseRef` set a `true` al inicio de `handleAssistantResponse` |
| Prefill 14s con modelos locales | VoiceMode: 50 token prompt + 8 tools (~700 tokens, era ~5000) |

## Watchdog Logic (90 seconds)

```typescript
watchdogTimerRef.current = setTimeout(() => {
  useChatStore.getState().cancelResponse?.();  // desbloquea isResponding
  setState("listening");
  setTranscript("");
  watchdogTimerRef.current = null;
  speakTextRef.current("El modelo tardó...", () => startListeningRef.current());
}, 90000);
```

## Adding New Wake Word / Hardware Integration
For native audio (WASAPI) wake word detection, see:
- `backend/internal/audio/` — Go audio capture code
- API endpoints: `GET /api/audio/devices`, `GET /api/audio/status`, `POST /api/audio/wakeword/test`
- Frontend: `api.audio.devices()`, `api.audio.wakewordTest()`

## VoiceMode Token Budget
| Component | Normal | VoiceMode |
|-----------|--------|-----------|
| System prompt | ~2000 tokens | ~50 tokens |
| Tool schemas | ~3000 tokens (18 tools) | ~700 tokens (8 tools) |
| **Total prefill** | **~5000 tokens / ~14s** | **~750 tokens / ~2s** |
