import { useMemo } from "react";
import { useChatStore } from "../../store/chatStore";
import { useUIStore } from "../../store/uiStore";
import { useAuthStore } from "../../store/authStore";
import { OzyLogo } from "../Brand/OzyLogo";
import { OzyChatConsole } from "../Chat/OzyChatConsole";

function getGreeting(name?: string): string {
  const hour = new Date().getHours();
  const firstName = name ? name.trim().split(" ")[0] : "";

  let timeGreeting = "¡Hola!";
  if (hour >= 5 && hour < 12) {
    timeGreeting = "Buenos días";
  } else if (hour >= 12 && hour < 20) {
    timeGreeting = "Buenas tardes";
  } else {
    timeGreeting = "Buenas noches";
  }

  if (firstName) {
    return `${timeGreeting}, ${firstName}`;
  }
  return timeGreeting;
}

export default function HomePage() {
  const createChat = useChatStore((s) => s.createChat);
  const setActiveView = useUIStore((s) => s.setActiveView);
  const isResponding = useChatStore((s) => s.isResponding);
  const stopStreaming = useChatStore((s) => s.stopStreaming);
  const user = useAuthStore((s) => s.user);

  const greeting = useMemo(() => getGreeting(user?.name), [user?.name]);

  const handleSend = async (message: string, mode: "chat" | "cowork", model: string) => {
    const chatId = await createChat(mode === "cowork" ? "code" : "chat", undefined, undefined, model);
    if (chatId) {
      const { sendMessage } = useChatStore.getState();
      sendMessage(chatId, message);
    }
    setActiveView("chat");
  };

  return (
    <div className="flex-1 flex flex-col items-center justify-center relative bg-[#131313] overflow-y-auto px-4 py-8">
      <div className="w-full max-w-3xl flex flex-col items-center z-10 -mt-12">
        {/* Hero Minimalista Estilo Claude con Logo Original */}
        <div className="flex items-center justify-center gap-3.5 mb-8 select-none">
          <OzyLogo size={42} />
          <h1 className="font-serif text-4xl sm:text-5xl font-normal tracking-tight text-neutral-100">
            {greeting}
          </h1>
        </div>

        {/* Consola OzyChatConsole */}
        <OzyChatConsole
          onSendMessage={handleSend}
          onStopStreaming={stopStreaming}
          isStreaming={isResponding}
        />
      </div>

      <div className="absolute inset-0 pointer-events-none overflow-hidden opacity-25">
        <div className="absolute top-1/4 left-1/4 w-[40vw] h-[40vw] bg-[#d1f107]/5 rounded-full blur-[120px] mix-blend-screen" />
        <div className="absolute bottom-1/4 right-1/4 w-[30vw] h-[30vw] bg-[#d1f107]/3 rounded-full blur-[100px] mix-blend-screen" />
      </div>
    </div>
  );
}
