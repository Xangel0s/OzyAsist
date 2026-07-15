import { useChatStore } from "../../store/chatStore";
import ChatInput from "../Home/ChatInput";

interface CodeInputProps {
  onSend: (message: string) => void;
}

export default function CodeInput({ onSend }: CodeInputProps) {
  const activeChatId = useChatStore((s) => s.activeChatId);
  const updateChatProvider = useChatStore((s) => s.updateChatProvider);

  const handleModelChange = (modelId: string, provider: string) => {
    if (activeChatId) {
      updateChatProvider(activeChatId, provider, modelId);
    } else {
      useChatStore.setState({ defaultModel: modelId, defaultProvider: provider });
    }
  };

  return (
    <div className="px-4 pb-6 pt-2 bg-gradient-to-t from-[#1a1a1a] via-[#1a1a1a] to-transparent z-10">
      <div className="max-w-[800px] mx-auto">
        <ChatInput
          onSend={onSend}
          placeholder="Responder a Ozy..."
          onModelChange={handleModelChange}
          compact
        />
        <div className="text-center mt-3 text-[11px] text-white/30">
          Ozy puede cometer errores. Considera verificar el código crítico.
        </div>
      </div>
    </div>
  );
}
