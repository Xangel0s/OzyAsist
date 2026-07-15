import { useChatStore } from "../../store/chatStore";
import ChatMessage from "./ChatMessage";
import ChatInput from "../Home/ChatInput";
import { useScrollToBottom } from "../../hooks";

export default function ChatPage() {
  const activeChatId = useChatStore((s) => s.activeChatId);
  const chats = useChatStore((s) => s.chats);
  const isResponding = useChatStore((s) => s.isResponding);
  const sendMessage = useChatStore((s) => s.sendMessage);
  const createChat = useChatStore((s) => s.createChat);
  const updateChatProvider = useChatStore((s) => s.updateChatProvider);

  const activeChat = chats.find((c) => c.id === activeChatId);
  const bottomRef = useScrollToBottom([
    activeChat?.messages.length,
    isResponding,
  ]);

  const handleSend = async (message: string) => {
    let chatId = activeChatId;
    if (!chatId) {
      const newId = await createChat("chat");
      if (!newId) return;
      chatId = newId;
    }
    sendMessage(chatId, message);
  };

  const handleModelChange = (modelId: string, provider: string) => {
    if (activeChatId) {
      updateChatProvider(activeChatId, provider, modelId);
    } else {
      useChatStore.setState({ defaultModel: modelId, defaultProvider: provider });
    }
  };

  return (
    <div className="flex flex-1 flex-col h-full bg-[#1a1a1a] min-h-0">
      <div className="flex-1 overflow-y-auto px-4 py-6 min-h-0">
        <div className="max-w-[800px] mx-auto flex flex-col gap-6">
          {(!activeChat || activeChat.messages.length === 0) && (
            <div className="flex-1 flex flex-col items-center justify-center min-h-[400px] gap-6">
              <div className="w-14 h-14 rounded-xl flex items-center justify-center overflow-hidden">
                <img src="/ozybaselogo.png" alt="OzyBase" className="w-full h-full object-contain" />
              </div>
              <h2 className="text-[20px] font-semibold text-white text-center">
                ¿En qué puedo ayudarte?
              </h2>
              <p className="text-white/40 text-[14px] text-center max-w-md">
                Conversación general con selección de modelo y proveedor.
              </p>
            </div>
          )}
          {activeChat?.messages.map((msg) => (
            <ChatMessage key={msg.id} message={msg} chatId={activeChat.id} />
          ))}
          {isResponding && (
            <div className="flex justify-start">
              <div className="flex gap-4 max-w-[90%]">
                <div className="w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 overflow-hidden">
                  <img src="/ozybaselogo.png" alt="OzyBase" className="w-full h-full object-contain" />
                </div>
                <div className="flex gap-1.5 py-3">
                  <span className="w-2 h-2 rounded-full bg-white/30 animate-bounce" style={{ animationDelay: "0ms" }} />
                  <span className="w-2 h-2 rounded-full bg-white/30 animate-bounce" style={{ animationDelay: "150ms" }} />
                  <span className="w-2 h-2 rounded-full bg-white/30 animate-bounce" style={{ animationDelay: "300ms" }} />
                </div>
              </div>
            </div>
          )}
          <div ref={bottomRef} />
        </div>
      </div>

      <div className="px-4 pb-6 pt-2 bg-gradient-to-t from-[#1a1a1a] via-[#1a1a1a] to-transparent">
        <div className="max-w-[800px] mx-auto">
          <ChatInput
            onSend={handleSend}
            placeholder="Escribe tu mensaje..."
            modelOverride={activeChat?.model}
            onModelChange={handleModelChange}
            compact
          />
          <div className="text-center mt-3 text-[11px] text-white/30">
            Ozy puede cometer errores. Verifica la información importante.
          </div>
        </div>
      </div>
    </div>
  );
}
