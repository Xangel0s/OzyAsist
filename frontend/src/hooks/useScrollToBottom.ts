import { useEffect, useRef } from "react";

export function useScrollToBottom(deps: unknown[]) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, deps);

  return bottomRef;
}
