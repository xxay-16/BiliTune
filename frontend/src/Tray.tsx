import { useEffect, useLayoutEffect, useRef, useState } from "react";
import {
  Pause,
  Play,
  SkipBack,
  SkipForward,
  Square,
  Monitor,
  EyeOff,
  LogOut,
} from "lucide-react";
import { api } from "./api";
import type { Player } from "./types";

export function Tray() {
  const [player, setPlayer] = useState<Player | null>(null);
  const title = player?.track?.title || "BiliTune";
  const outer = useRef<HTMLDivElement>(null);
  const inner = useRef<HTMLSpanElement>(null);
  const [overflow, setOverflow] = useState(false);
  const [width, setWidth] = useState(0);
  useEffect(() => {
    document.documentElement.dataset.theme = "dark";
    let stopped = false;
    const refresh = () =>
      api<Player>("player")
        .then((p) => {
          if (!stopped) setPlayer(p);
        })
        .catch(() => {});
    refresh();
    const timer = setInterval(refresh, 400);
    const key = (e: KeyboardEvent) => {
      if (e.key === "Escape") void api("window", { action: "tray-hide" });
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        const nodes = Array.from(
          document.querySelectorAll<HTMLButtonElement>(".tray-menu button"),
        );
        const next =
          (nodes.indexOf(document.activeElement as HTMLButtonElement) +
            (e.key === "ArrowDown" ? 1 : -1) +
            nodes.length) %
          nodes.length;
        nodes[next]?.focus();
      }
    };
    addEventListener("keydown", key);
    return () => {
      stopped = true;
      clearInterval(timer);
      removeEventListener("keydown", key);
    };
  }, []);
  useLayoutEffect(() => {
    if (outer.current && inner.current) {
      setWidth(inner.current.scrollWidth);
      setOverflow(inner.current.scrollWidth > outer.current.clientWidth);
    }
  }, [title]);
  async function act(action: string) {
    await api("window", { action: "tray-hide" });
    try {
      await api("command", { action });
    } catch {}
  }
  return (
    <div className="tray-menu" role="menu">
      <div
        className={"tray-title " + (overflow ? "overflow" : "")}
        ref={outer}
        title={title}
        style={
          {
            "--marquee-distance": `${width + 32}px`,
            "--marquee-duration": `${Math.max(8, (width + 32) / 30)}s`,
          } as React.CSSProperties
        }
      >
        <div className="marquee-track" key={title}>
          <span ref={inner}>{title}</span>
          {overflow && <span aria-hidden="true">{title}</span>}
        </div>
      </div>
      <hr />
      <button role="menuitem" onClick={() => act("previousTrack")}>
        <SkipBack size={15} />
        上一曲
      </button>
      <button role="menuitem" onClick={() => act("playPause")}>
        {player?.playing ? <Pause size={15} /> : <Play size={15} />}{" "}
        {player?.playing ? "暂停" : "播放"}
      </button>
      <button role="menuitem" onClick={() => act("nextTrack")}>
        <SkipForward size={15} />
        下一曲
      </button>
      <button role="menuitem" onClick={() => act("stop")}>
        <Square size={14} />
        停止
      </button>
      <hr />
      <button role="menuitem" onClick={() => act("show")}>
        <Monitor size={15} />
        显示主窗口
      </button>
      <button role="menuitem" onClick={() => act("hide")}>
        <EyeOff size={15} />
        隐藏到托盘
      </button>
      <hr />
      <button role="menuitem" onClick={() => act("exit")}>
        <LogOut size={15} />
        退出
      </button>
    </div>
  );
}
