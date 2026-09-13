---
name: ozy-frontend-react
description: >-
  Use this skill for any task involving the OzyAssist React/TypeScript frontend:
  modifying components, Zustand stores, Tailwind styles, WebSocket client, API calls,
  or any file under frontend/src/. Provides component map, state architecture, and dev workflow.
---

# OzyAssist Frontend React — Skill

## Tech Stack
- **React 18** + TypeScript + Vite
- **Zustand** — state management (no Redux, no Context for global state)
- **Tailwind CSS** — utility-first, brand color `#d1f107` (lime)
- **Dark surfaces**: `#131313` (dim), `#1e1e1e` / `#222222` (container)
- **lucide-react** — icons (never use other icon libraries)

## Component Map

```
frontend/src/
├── components/
│   ├── Sidebar.tsx             # Left nav: brand, + New Session, nav items, recent chats, user card
│   ├── ChatArea.tsx            # Main chat view: message list + input bar
│   ├── MessageBubble.tsx       # Individual message rendering (markdown, code, artifacts)
│   ├── InputBar.tsx            # Textarea + send button + voice toggle
│   ├── SettingsModal.tsx       # 8-tab settings dialog (Skills, Connectors, LLM Providers, etc.)
│   ├── Voice/
│   │   └── OzyLiveOverlay.tsx  # Fullscreen voice mode overlay (TTS + STT + watchdog)
│   ├── Artifacts/              # Rich artifact renderers (code, markdown, images)
│   └── OsControl/              # OS permission UI, confirm dialogs, audit trail
├── store/
│   ├── chatStore.ts            # chats, messages, activeChatId, isResponding, sendMessage
│   ├── uiStore.ts              # voiceLiveOpen, settingsOpen, theme, activeView
│   └── authStore.ts            # profile, PIN verification
├── services/
│   ├── ws.ts                   # WebSocket client: connect, send, subscribe(event, handler)
│   └── api.ts                  # REST client: api.chats, api.models, api.skills, api.settings
└── App.tsx                     # Root: routing between Inicio/Proyectos/Artefactos/Personalizar
```

## State Architecture

### chatStore (Zustand)
```typescript
{
  chats: Chat[]
  activeChatId: string | null
  isResponding: boolean          // true while WS streaming — guards double-send
  consentPending: boolean
  sendMessage(chatId, content, voiceMode?: boolean): Promise<void>
  createChat(title): Promise<string>
  cancelResponse(): void
}
```

### uiStore (Zustand)
```typescript
{
  voiceLiveOpen: boolean         // toggles OzyLiveOverlay
  settingsOpen: boolean
  openSettings(tab?: string): void
  toggleVoiceLive(): void
}
```

### wsService (singleton)
```typescript
wsService.connect()
wsService.send(payload)
wsService.subscribe("message:delta" | "agent:completed" | "done" | "error", handler)
wsService.unsubscribe(event, handler)
```

## Key Patterns

### Adding a New Zustand Action
```typescript
// In the store file — always use `set()` or `get()` from the create callback
addFeature: (data) => set(state => ({ ... state, newField: data }))
```

### Adding a New API Call
```typescript
// In api.ts — follow the existing pattern
export const api = {
  myFeature: {
    list: () => fetch('/api/my-feature').then(r => r.json()),
    create: (body) => fetch('/api/my-feature', { method: 'POST', body: JSON.stringify(body) })
  }
}
```

### Adding a New Settings Tab
1. Add tab entry in `SettingsModal.tsx` tabs array
2. Add `case 'tab-id': return <MyTabComponent />` in the render switch
3. Create `components/Settings/MyTabComponent.tsx`

## Brand Theme Rules
- Primary action buttons: `bg-[#d1f107] text-[#181e00] font-bold`
- Active nav items: `bg-[#d1f107]/10 text-[#d1f107]`
- Danger/error: `text-red-400` (NOT red-500 or red-600)
- Border accent: `border-[#d1f107]/30`
- Never use Tailwind `blue-*` or `green-*` for brand elements

## Build Commands
```powershell
# Dev server (hot reload) — NOT used for production
cd frontend; npm run dev

# Production build — output to frontend/dist/
cd frontend; npm run build

# Type check only
cd frontend; npx tsc --noEmit

# After build, copy dist to backend for serving
Copy-Item -Recurse -Force "frontend\dist" "backend\dist"
```

## Critical Rules
- **Never** bypass Zustand for global state — no `useState` for shared state
- **`isResponding` guard**: In `sendMessage`, if `isResponding=true` in voiceMode → throw `Error("busy")`, else silently return
- **WS events**: Always unsubscribe in `useEffect` cleanup to prevent memory leaks
- **TTS**: Always call `synth.resume()` before `synth.speak()` — Chrome suspends synthesis on inactive tabs
- **Tailwind classes**: Use full class names (no dynamic concatenation like `text-${color}-400`)
