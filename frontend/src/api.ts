export async function api<T = unknown>(
  path: string,
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const response = await fetch("/api/" + path, {
    method: body === undefined ? "GET" : "POST",
    headers:
      body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  });
  const data = await response.json();
  if (!response.ok)
    throw new Error(data.error || `请求失败 (${response.status})`);
  return data as T;
}
export const coverURL = (url?: string) =>
  url ? "/api/image?url=" + encodeURIComponent(url) : undefined;
export function duration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  const total = Math.floor(seconds);
  if (total >= 3600)
    return `${Math.floor(total / 3600)}:${String(Math.floor(total / 60) % 60).padStart(2, "0")}:${String(total % 60).padStart(2, "0")}`;
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, "0")}`;
}
export function bytes(size: number): string {
  if (size < 0) return "未知";
  if (size < 1024) return `${size} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let n = size / 1024,
    i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(1)} ${units[i]}`;
}
export function filterTracks<T extends { title: string; artist: string }>(
  tracks: T[],
  query: string,
): T[] {
  const q = query.trim().toLocaleLowerCase();
  return q
    ? tracks.filter((t) =>
        (t.title + " " + t.artist).toLocaleLowerCase().includes(q),
      )
    : tracks;
}
export function keyToken(event: KeyboardEvent): string | null {
  if (["Control", "Alt", "Shift", "Meta"].includes(event.key)) return null;
  const media: Record<string, string> = {
    MediaPlayPause: "MediaPlayPause",
    MediaTrackNext: "MediaNextTrack",
    MediaTrackPrevious: "MediaPreviousTrack",
    MediaStop: "MediaStop",
  };
  if (media[event.key]) return media[event.key];
  if (/^Key[A-Z]$/.test(event.code)) return event.code.slice(3);
  if (/^Digit[0-9]$/.test(event.code)) return event.code.slice(5);
  if (/^Numpad[0-9]$/.test(event.code)) return event.code;
  if (/^F([1-9]|1[0-9]|2[0-4])$/.test(event.key)) return event.key;
  const named: Record<string, string> = {
    Space: "Space",
    Enter: "Enter",
    NumpadEnter: "Enter",
    Escape: "Escape",
    Backspace: "Backspace",
    Tab: "Tab",
    ArrowLeft: "ArrowLeft",
    ArrowRight: "ArrowRight",
    ArrowUp: "ArrowUp",
    ArrowDown: "ArrowDown",
    Home: "Home",
    End: "End",
    PageUp: "PageUp",
    PageDown: "PageDown",
    Insert: "Insert",
    Delete: "Delete",
    Period: "Period",
    Comma: "Comma",
    Minus: "Minus",
    Equal: "Equal",
    Slash: "Slash",
    Backquote: "Backquote",
    BracketLeft: "BracketLeft",
    BracketRight: "BracketRight",
    Backslash: "Backslash",
    Semicolon: "Semicolon",
    Quote: "Quote",
    NumpadAdd: "NumpadAdd",
    NumpadSubtract: "NumpadSubtract",
    NumpadMultiply: "NumpadMultiply",
    NumpadDivide: "NumpadDivide",
    NumpadDecimal: "NumpadDecimal",
  };
  return named[event.code] ?? null;
}
