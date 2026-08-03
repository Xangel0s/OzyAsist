import { useEffect } from "react";

export function useKeyboard(
  key: string,
  handler: () => void,
  modifiers?: { meta?: boolean; ctrl?: boolean },
) {
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      const requiresMod = modifiers?.meta || modifiers?.ctrl;
      const hasMod = e.metaKey || e.ctrlKey;
      const modMatch = requiresMod ? hasMod : true;
      if (modMatch && e.key.toLowerCase() === key.toLowerCase()) {
        e.preventDefault();
        handler();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [key, handler, modifiers]);
}
