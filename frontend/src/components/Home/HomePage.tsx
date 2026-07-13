import { useChatStore } from "../../store/chatStore";
import { useUIStore } from "../../store/uiStore";
import ActionChips from "./ActionChips";
import ChatInput from "./ChatInput";

export default function HomePage() {
  const createChat = useChatStore((s) => s.createChat);
  const setActiveView = useUIStore((s) => s.setActiveView);

  const handleSend = async (message: string) => {
    const chatId = await createChat("chat");
    if (chatId) {
      const { sendMessage } = useChatStore.getState();
      sendMessage(chatId, message);
    }
    setActiveView("chat");
  };

  return (
    <div className="flex-1 flex flex-col items-center justify-center relative bg-surface-deep overflow-y-auto">
      <div className="w-full max-w-content-max-width px-gutter flex flex-col items-center z-10 -mt-20">
        <div className="flex items-center justify-center gap-4 mb-12">
          <div className="w-12 h-12 rounded-lg flex items-center justify-center overflow-hidden">
            <img src="/ozybaselogo.png" alt="OzyBase" className="w-full h-full object-contain" />
          </div>
          <h1 className="text-display-hero-mobile md:text-display-hero font-display-hero text-on-surface text-center">
            ¿Qué vamos a construir hoy?
          </h1>
        </div>

        <ChatInput onSend={handleSend} />
        <ActionChips />
      </div>

      <div className="absolute inset-0 pointer-events-none overflow-hidden opacity-30">
        <div className="absolute top-1/4 left-1/4 w-[40vw] h-[40vw] bg-primary-fixed-dim/5 rounded-full blur-[100px] mix-blend-screen" />
        <div className="absolute bottom-1/4 right-1/4 w-[30vw] h-[30vw] bg-secondary-fixed-dim/5 rounded-full blur-[100px] mix-blend-screen" />
      </div>
    </div>
  );
}
