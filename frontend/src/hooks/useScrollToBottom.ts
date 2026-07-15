import { useEffect, useRef } from "react";

export function useScrollToBottom(deps: unknown[]) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = bottomRef.current?.parentElement;
    if (el) {
      const dist = el.scrollHeight - el.scrollTop - el.clientHeight;
      if (dist < 100) {
        bottomRef.current?.scrollIntoView({ behavior: "smooth" });
      }
    }
  }, deps);

  return bottomRef;
}
