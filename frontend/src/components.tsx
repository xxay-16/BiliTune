import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode,
  type CSSProperties,
} from "react";
import { createPortal } from "react-dom";
import {
  Music2,
  Play,
  Pause,
  Volume2,
  Heart,
  Clock3,
  Download,
  SkipBack,
  SkipForward,
  Shuffle,
  Repeat,
  Repeat1,
  ListMusic,
  Maximize2,
  LoaderCircle,
  X,
  Search,
  Folder,
  Lock,
  ArrowRight,
  Plus,
} from "lucide-react";
import { coverURL, duration } from "./api";
import type { Track, Player, Folder as FolderType } from "./types";

export function IconButton({
  title,
  children,
  onClick,
  active = false,
  disabled = false,
  className = "",
}: {
  title: string;
  children: ReactNode;
  onClick?: () => void;
  active?: boolean;
  disabled?: boolean;
  className?: string;
}) {
  return (
    <button
      type="button"
      className={`icon-button ${active ? "active" : ""} ${className}`}
      title={title}
      aria-label={title}
      onClick={onClick}
      disabled={disabled}
    >
      {children}
    </button>
  );
}
export function Cover({
  track,
  folder,
  size = 48,
  className = "",
}: {
  track?: Track | null;
  folder?: FolderType;
  size?: number;
  className?: string;
}) {
  const [failed, setFailed] = useState(false);
  const raw = track?.coverUrl || folder?.coverUrl;
  useEffect(() => setFailed(false), [raw]);
  const seed = (track?.id || folder?.title || "music")
    .split("")
    .reduce((s, c) => s + c.charCodeAt(0), 0);
  return (
    <div
      className={"cover " + className}
      style={{
        width: size,
        height: size,
        background: `linear-gradient(140deg,hsl(${seed % 360} 40% 37%),hsl(${(seed + 65) % 360} 45% 18%))`,
      }}
    >
      {raw && !failed ? (
        <img
          src={coverURL(raw)}
          onError={() => setFailed(true)}
          alt=""
          loading="lazy"
          draggable={false}
        />
      ) : (
        <Music2 size={Math.max(19, size * 0.4)} />
      )}
    </div>
  );
}
export function Empty({
  title,
  detail,
  children,
  icon = <Music2 size={30} />,
}: {
  title: string;
  detail?: string;
  children?: ReactNode;
  icon?: ReactNode;
}) {
  return (
    <div className="empty">
      <div className="empty-icon">{icon}</div>
      <h3>{title}</h3>
      {detail && <p>{detail}</p>}
      {children}
    </div>
  );
}
export function SearchField({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (s: string) => void;
  placeholder: string;
}) {
  return (
    <div className="search-field">
      <Search size={18} />
      <input
        aria-label={placeholder}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
      {value && (
        <IconButton title="清空搜索" onClick={() => onChange("")}>
          <X size={15} />
        </IconButton>
      )}
    </div>
  );
}
export function TrackList({
  tracks,
  player,
  onPlay,
  onContext,
  onLike,
  liked = false,
}: {
  tracks: Track[];
  player: Player;
  onPlay: (track: Track) => void;
  onContext: (track: Track, x: number, y: number) => void;
  onLike?: (track: Track) => void;
  liked?: boolean;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [viewport, setViewport] = useState({ top: 0, height: 600 });
  useLayoutEffect(() => {
    const node = ref.current;
    if (!node) return;
    const observer = new ResizeObserver(() =>
      setViewport((old) => ({ ...old, height: node.clientHeight })),
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);
  useEffect(() => {
    if (ref.current) ref.current.scrollTop = 0;
    setViewport((old) => ({ ...old, top: 0 }));
  }, [tracks]);
  const virtual = tracks.length > 120;
  const start = virtual ? Math.max(0, Math.floor(viewport.top / 64) - 8) : 0;
  const end = virtual
    ? Math.min(
        tracks.length,
        Math.ceil((viewport.top + viewport.height) / 64) + 8,
      )
    : tracks.length;
  return (
    <div
      className="track-list"
      ref={ref}
      onScroll={(e) => {
        const top = e.currentTarget.scrollTop;
        if (virtual) setViewport((old) => ({ ...old, top }));
      }}
    >
      <div
        style={
          virtual
            ? { height: tracks.length * 64, position: "relative" }
            : undefined
        }
      >
        {tracks.slice(start, end).map((track, n) => {
          const i = n + start;
          return (
            <div
              key={`${track.id}-${i}`}
              className={`track-row ${player.track?.id === track.id ? "current" : ""}`}
              style={
                virtual
                  ? { position: "absolute", top: i * 64, left: 0, right: 0 }
                  : undefined
              }
              onDoubleClick={() => onPlay(track)}
              onContextMenu={(e) => {
                e.preventDefault();
                onContext(track, e.clientX, e.clientY);
              }}
            >
              <button
                className="track-index"
                title={"播放 " + track.title}
                aria-label={"播放 " + track.title}
                onClick={() => onPlay(track)}
              >
                {player.track?.id === track.id && player.playing ? (
                  <Volume2 size={17} />
                ) : (
                  <>
                    <span>{i + 1}</span>
                    <Play size={17} fill="currentColor" />
                  </>
                )}
              </button>
              <Cover track={track} size={42} />
              <button
                className="track-title"
                onClick={() => onPlay(track)}
                title={track.title}
              >
                <strong>{track.title || "失效的视频"}</strong>
                <span>{track.artist || "未知创作者"}</span>
              </button>
              <span
                className="track-kind"
                title={track.type === "audio" ? "音频" : "视频音轨"}
              >
                {track.type === "audio" ? (
                  <Music2 size={16} />
                ) : (
                  <Play size={14} />
                )}
              </span>
              <IconButton
                title={liked ? "从当前收藏夹删除" : "添加到收藏夹"}
                active={liked}
                onClick={() => onLike?.(track)}
              >
                <Heart size={18} fill={liked ? "currentColor" : "none"} />
              </IconButton>
              <span className="track-duration">
                {track.durationMs > 0 ? duration(track.durationMs / 1000) : "—"}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
export function FolderTile({
  folder,
  title,
  count,
  selected,
  onClick,
  onContext,
  icon,
}: {
  folder?: FolderType;
  title?: string;
  count?: number;
  selected?: boolean;
  onClick: () => void;
  onContext?: (x: number, y: number) => void;
  icon?: ReactNode;
}) {
  return (
    <button
      className={`folder-tile ${selected ? "selected" : ""}`}
      onClick={onClick}
      onContextMenu={(e) => {
        if (onContext) {
          e.preventDefault();
          onContext(e.clientX, e.clientY);
        }
      }}
    >
      <Cover folder={folder} size={48} />
      <span className="folder-label">
        <strong>{folder?.title || title}</strong>
        <small>{folder?.count ?? count ?? 0} 首</small>
      </span>
      <span className="folder-trailing">
        {icon ||
          (folder?.private ? <Lock size={15} /> : <ArrowRight size={16} />)}
      </span>
    </button>
  );
}
export function Modal({
  title,
  children,
  onClose,
  wide = false,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  wide?: boolean;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const close = useRef(onClose);
  close.current = onClose;
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    ref.current?.focus();
    const listener = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopPropagation();
        close.current();
      }
      if (e.key === "Tab" && ref.current) {
        const nodes = Array.from(
          ref.current.querySelectorAll<HTMLElement>(
            'button:not(:disabled),input,textarea,select,[tabindex="0"]',
          ),
        );
        if (!nodes.length) return;
        const first = nodes[0],
          last = nodes[nodes.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    document.addEventListener("keydown", listener);
    return () => {
      document.removeEventListener("keydown", listener);
      previous?.focus();
    };
  }, []);
  return createPortal(
    <div
      className="modal-backdrop"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className={`modal ${wide ? "wide" : ""}`}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        tabIndex={-1}
        ref={ref}
      >
        <header>
          <h2>{title}</h2>
          <IconButton title="关闭" onClick={onClose}>
            <X size={20} />
          </IconButton>
        </header>
        {children}
      </div>
    </div>,
    document.body,
  );
}
export function ContextMenu({
  x,
  y,
  children,
  onClose,
}: {
  x: number;
  y: number;
  children: ReactNode;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ x, y });
  const close = useRef(onClose);
  close.current = onClose;
  useEffect(() => {
    if (ref.current) {
      const box = ref.current.getBoundingClientRect();
      setPos({
        x: Math.max(8, Math.min(x, innerWidth - box.width - 8)),
        y: Math.max(8, Math.min(y, innerHeight - box.height - 8)),
      });
      ref.current.focus();
    }
    const down = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) close.current();
    };
    const key = (e: KeyboardEvent) => {
      if (e.key === "Escape") close.current();
      if ((e.key === "ArrowDown" || e.key === "ArrowUp") && ref.current) {
        e.preventDefault();
        const buttons = Array.from(
          ref.current.querySelectorAll<HTMLButtonElement>(
            "button:not(:disabled)",
          ),
        );
        let index = buttons.indexOf(
          document.activeElement as HTMLButtonElement,
        );
        index =
          (index + (e.key === "ArrowDown" ? 1 : -1) + buttons.length) %
          buttons.length;
        buttons[index]?.focus();
      }
    };
    document.addEventListener("mousedown", down);
    document.addEventListener("keydown", key);
    const blur = () => close.current();
    window.addEventListener("blur", blur);
    return () => {
      document.removeEventListener("mousedown", down);
      document.removeEventListener("keydown", key);
      window.removeEventListener("blur", blur);
    };
  }, [x, y]);
  return createPortal(
    <div
      className="context-menu"
      role="menu"
      tabIndex={-1}
      ref={ref}
      style={{ left: pos.x, top: pos.y }}
    >
      {children}
    </div>,
    document.body,
  );
}
export function MenuItem({
  icon,
  children,
  onClick,
  danger = false,
}: {
  icon: ReactNode;
  children: ReactNode;
  onClick: () => void;
  danger?: boolean;
}) {
  return (
    <button
      role="menuitem"
      className={danger ? "danger" : ""}
      onClick={onClick}
    >
      {icon}
      <span>{children}</span>
    </button>
  );
}
export function PlayerBar({
  player,
  command,
  onFavorite,
  onQueue,
  onImmersive,
}: {
  player: Player;
  command: (action: string, value?: number) => void;
  onFavorite: () => void;
  onQueue: () => void;
  onImmersive: () => void;
}) {
  const [seeking, setSeeking] = useState(false);
  const [position, setPosition] = useState(0);
  const [volume, setVolume] = useState(player.volume);
  const [volumeDrag, setVolumeDrag] = useState(false);
  useEffect(() => {
    if (!volumeDrag) setVolume(player.volume);
  }, [player.volume, volumeDrag]);
  const seek = () => {
    if (seeking) {
      command("seek", position);
      setSeeking(false);
    }
  };
  return (
    <footer
      className="player-bar"
      style={
        {
          "--player-art": player.track?.coverUrl
            ? `url("${coverURL(player.track.coverUrl)}")`
            : "none",
        } as CSSProperties
      }
    >
      <button
        className="now-playing"
        onClick={onImmersive}
        title="打开沉浸播放"
      >
        <Cover track={player.track} size={52} />
        <span>
          <strong>{player.track?.title || "还没有播放的音乐"}</strong>
          <small>{player.track?.artist || "从收藏夹或搜索开始聆听"}</small>
        </span>
      </button>
      <IconButton
        title="添加到收藏夹"
        disabled={!player.track}
        onClick={onFavorite}
      >
        <Heart size={19} />
      </IconButton>
      <div className="transport">
        <div className="transport-buttons">
          <IconButton
            title="随机播放"
            active={player.shuffle}
            onClick={() => command("shuffle")}
          >
            <Shuffle size={18} />
          </IconButton>
          <IconButton
            title="上一曲"
            disabled={!player.track}
            onClick={() => command("previousTrack")}
          >
            <SkipBack size={22} fill="currentColor" />
          </IconButton>
          <button
            className="play-main"
            aria-label={player.playing ? "暂停" : "播放"}
            disabled={!player.track}
            onClick={() => command("playPause")}
          >
            {player.buffering ? (
              <LoaderCircle className="spin" size={24} />
            ) : player.playing ? (
              <Pause size={22} fill="currentColor" />
            ) : (
              <Play size={23} fill="currentColor" />
            )}
          </button>
          <IconButton
            title="下一曲"
            disabled={!player.track}
            onClick={() => command("nextTrack")}
          >
            <SkipForward size={22} fill="currentColor" />
          </IconButton>
          <IconButton
            title={
              player.repeat === "one"
                ? "单曲循环"
                : player.repeat === "all"
                  ? "列表循环"
                  : "循环关闭"
            }
            active={player.repeat !== "off"}
            onClick={() => command("repeat")}
          >
            {player.repeat === "one" ? (
              <Repeat1 size={19} />
            ) : (
              <Repeat size={19} />
            )}
          </IconButton>
        </div>
        <div className="timeline">
          <time>{duration(seeking ? position : player.position)}</time>
          <input
            aria-label="播放进度"
            type="range"
            min={0}
            max={Math.max(player.duration, 1)}
            step={0.1}
            value={
              seeking
                ? position
                : Math.min(player.position, Math.max(player.duration, 1))
            }
            style={
              {
                "--progress": `${player.duration > 0 ? Math.min(100, Math.max(0, ((seeking ? position : player.position) / player.duration) * 100)) : 0}%`,
              } as CSSProperties
            }
            onChange={(e) => {
              setSeeking(true);
              setPosition(Number(e.target.value));
            }}
            onPointerUp={seek}
            onKeyUp={seek}
            onBlur={seek}
            disabled={!player.track}
          />
          <time>{duration(player.duration)}</time>
        </div>
      </div>
      <div className="player-options">
        <span className="quality">{player.quality}</span>
        <IconButton title="播放队列" onClick={onQueue}>
          <ListMusic size={20} />
        </IconButton>
        <IconButton title="沉浸播放" onClick={onImmersive}>
          <Maximize2 size={18} />
        </IconButton>
        <IconButton
          title={volume === 0 ? "取消静音" : "静音"}
          onClick={() => command("volume", volume === 0 ? 0.7 : 0)}
        >
          <Volume2 size={19} />
        </IconButton>
        <input
          aria-label="音量"
          className="volume-slider"
          type="range"
          min={0}
          max={1}
          step={0.01}
          value={volume}
          style={{ "--progress": `${volume * 100}%` } as CSSProperties}
          onPointerDown={() => setVolumeDrag(true)}
          onChange={(e) => setVolume(Number(e.target.value))}
          onPointerUp={() => {
            command("volume", volume);
            setVolumeDrag(false);
          }}
          onKeyUp={() => command("volume", volume)}
        />
      </div>
    </footer>
  );
}
export function Spinner() {
  return (
    <div className="loading">
      <LoaderCircle className="spin" size={23} />
      <span>正在加载…</span>
    </div>
  );
}
export const standardFolderIcons = {
  recent: <Clock3 size={17} />,
  downloads: <Download size={17} />,
  folder: <Folder size={17} />,
};
export function CreateButton({ onClick }: { onClick: () => void }) {
  return (
    <button className="button outline" onClick={onClick}>
      <Plus size={17} />
      新建歌单
    </button>
  );
}
