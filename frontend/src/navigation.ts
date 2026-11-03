import { useCallback, useState } from "react";

export type Page =
  | "discover"
  | "charts"
  | "search"
  | "library"
  | "downloads"
  | "settings";

export function usePageNavigation() {
  const [history, setHistory] = useState<{ entries: Page[]; index: number }>({
    entries: ["discover"],
    index: 0,
  });
  const navigate = useCallback((page: Page) => {
    setHistory((current) =>
      current.entries[current.index] === page
        ? current
        : {
            entries: [...current.entries.slice(0, current.index + 1), page],
            index: current.index + 1,
          },
    );
  }, []);
  const back = useCallback(
    () =>
      setHistory((current) => ({
        ...current,
        index: Math.max(0, current.index - 1),
      })),
    [],
  );
  const forward = useCallback(
    () =>
      setHistory((current) => ({
        ...current,
        index: Math.min(current.entries.length - 1, current.index + 1),
      })),
    [],
  );
  return {
    page: history.entries[history.index],
    navigate,
    back,
    forward,
    canBack: history.index > 0,
    canForward: history.index < history.entries.length - 1,
  };
}
