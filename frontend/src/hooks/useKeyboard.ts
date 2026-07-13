import { useEffect } from "react";

export function useKeyboard(
  key: string,
  handler: () => void,
  modifiers?: { meta?: boolean; ctrl?: boolean },
) {
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      const mod =
        (!modifiers?.meta || e.metaKey) && (!modifiers?.ctrl || e.ctrlKey);
      if (mod && e.key.toLowerCase() === key.toLowerCase()) {
        e.preventDefault();
        handler();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [key, handler, modifiers]);
}
