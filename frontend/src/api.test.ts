import { describe, it, expect } from "vitest";
import { duration, filterTracks, keyToken } from "./api";
describe("desktop music helpers", () => {
  it("keeps long durations on a single timestamp", () => {
    expect(duration(14 * 3600 + 30 * 60 + 58)).toBe("14:30:58");
    expect(duration(61)).toBe("1:01");
  });
  it("filters titles and creators without removing tracks when blank", () => {
    const tracks = [
      { title: "中文音乐", artist: "某歌手" },
      { title: "Live", artist: "UP" },
    ];
    expect(filterTracks(tracks, "歌手")).toHaveLength(1);
    expect(filterTracks(tracks, "live")).toHaveLength(1);
    expect(filterTracks(tracks, "")).toHaveLength(2);
  });
  it("records physical punctuation, numpad and media keys", () => {
    const key = (code: string, value: string) =>
      keyToken({ code, key: value } as KeyboardEvent);
    expect(key("KeyA", "a")).toBe("A");
    expect(key("Semicolon", ":")).toBe("Semicolon");
    expect(key("Numpad1", "1")).toBe("Numpad1");
    expect(key("", "MediaTrackPrevious")).toBe("MediaPreviousTrack");
    expect(key("ControlLeft", "Control")).toBeNull();
  });
});
