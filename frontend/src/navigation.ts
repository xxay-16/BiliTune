import { useCallback, useState } from "react";
import type { Artist, Album } from "./types";

export type Page =
  | "discover"
  | "charts"
  | "search"
  | "library"
  | "downloads"
  | "settings";
export type PageRoute =
  | { page: Page }
  | { page: "artist"; artist: Artist }
  | { page: "album"; album: Album };
const routeKey = (route: PageRoute) =>
  route.page === "artist"
    ? `artist:${route.artist.mid}`
    : route.page === "album"
      ? `album:${route.album.mid}:${route.album.type}:${route.album.id}`
      : route.page;

export function usePageNavigation() {
  const [history, setHistory] = useState<{
    entries: PageRoute[];
    index: number;
  }>({
    entries: [{ page: "discover" }],
    index: 0,
  });
  const navigate = useCallback((target: Page | PageRoute) => {
    const route = typeof target === "string" ? { page: target } : target;
    setHistory((current) =>
      routeKey(current.entries[current.index]) === routeKey(route)
        ? current
        : {
            entries: [...current.entries.slice(0, current.index + 1), route],
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
    page: history.entries[history.index].page,
    route: history.entries[history.index],
    navigate,
    back,
    forward,
    canBack: history.index > 0,
    canForward: history.index < history.entries.length - 1,
  };
}
