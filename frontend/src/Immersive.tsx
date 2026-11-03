import { useEffect, useRef, useState } from "react";
import {
  ChevronDown,
  Heart,
  LoaderCircle,
  Music2,
  Pause,
  Play,
  Repeat,
  Shuffle,
  SkipBack,
  SkipForward,
} from "lucide-react";
import { api, coverURL, duration } from "./api";
import { ArtistLink, Cover, IconButton } from "./components";
import type { Player, Lyric, Track } from "./types";

export function Immersive({
  player,
  onClose,
  command,
  onFavorite,
  onArtist,
}: {
  player: Player;
  onClose: () => void;
  command: (action: string, value?: number) => void;
  onFavorite: () => void;
  onArtist: (track: Track) => void;
}) {
  const [lyrics, setLyrics] = useState<Lyric[]>([]);
  const [loading, setLoading] = useState(false);
  const activeRef = useRef<HTMLParagraphElement>(null);
  const manualScroll = useRef(0);
  useEffect(() => {
    const controller = new AbortController();
    if (!player.track) return;
    setLyrics([]);
    setLoading(true);
    api<Lyric[]>("lyrics", player.track, controller.signal)
      .then(setLyrics)
      .catch(() => {})
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [player.track?.id, player.track?.cid]);
  const synced = lyrics.some((line) => line.time > 0);
  let active = -1;
  if (synced) {
    for (let i = 0; i < lyrics.length; i++) {
      if (lyrics[i].time <= player.position) active = i;
      else break;
    }
  }
  useEffect(() => {
    if (Date.now() - manualScroll.current > 5000)
      activeRef.current?.scrollIntoView({
        block: "center",
        behavior: matchMedia("(prefers-reduced-motion: reduce)").matches
          ? "instant"
          : "smooth",
      });
  }, [active]);
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    addEventListener("keydown", handler);
    return () => removeEventListener("keydown", handler);
  }, [onClose]);
  return (
    <div className="immersive">
      <div
        className="immersive-wash"
        style={
          player.track?.coverUrl
            ? { backgroundImage: `url("${coverURL(player.track.coverUrl)}")` }
            : undefined
        }
      />
      <header>
        <IconButton title="退出沉浸播放" onClick={onClose}>
          <ChevronDown size={25} />
        </IconButton>
        <span>正在播放 · BiliTune</span>
        <span className="quality">{player.quality}</span>
      </header>
      <div className="immersive-content">
        <section className="immersive-album">
          <Cover track={player.track} size={340} />
          <div className="immersive-title">
            <h1>{player.track?.title || "选择一首音乐"}</h1>
            <ArtistLink track={player.track} onArtist={onArtist} />
          </div>
          <div className="immersive-timeline">
            <input
              type="range"
              aria-label="沉浸播放进度"
              min={0}
              max={Math.max(player.duration, 1)}
              step={1}
              value={Math.min(player.position, Math.max(player.duration, 1))}
              onChange={(e) => command("seek", Number(e.target.value))}
            />
            <div>
              <time>{duration(player.position)}</time>
              <time>{duration(player.duration)}</time>
            </div>
          </div>
          <div className="immersive-controls">
            <IconButton
              title="随机播放"
              active={player.shuffle}
              onClick={() => command("shuffle")}
            >
              <Shuffle size={19} />
            </IconButton>
            <IconButton title="上一曲" onClick={() => command("previousTrack")}>
              <SkipBack size={26} fill="currentColor" />
            </IconButton>
            <button
              className="play-main"
              aria-label={player.playing ? "暂停" : "播放"}
              onClick={() => command("playPause")}
            >
              {player.buffering ? (
                <LoaderCircle className="spin" />
              ) : player.playing ? (
                <Pause size={25} fill="currentColor" />
              ) : (
                <Play size={25} fill="currentColor" />
              )}
            </button>
            <IconButton title="下一曲" onClick={() => command("nextTrack")}>
              <SkipForward size={26} fill="currentColor" />
            </IconButton>
            <IconButton title="添加到收藏夹" onClick={onFavorite}>
              <Heart size={20} />
            </IconButton>
          </div>
        </section>
        <section
          className="lyrics"
          onWheel={() => {
            manualScroll.current = Date.now();
          }}
          aria-label="歌词与字幕"
        >
          {loading ? (
            <div className="lyrics-empty">
              <LoaderCircle className="spin" />
              正在寻找歌词…
            </div>
          ) : lyrics.length ? (
            lyrics.map((line, index) => (
              <p
                key={index}
                ref={active === index ? activeRef : undefined}
                className={`${synced ? "synced" : "plain"} ${active === index ? "active" : ""}`}
                onDoubleClick={() => {
                  if (synced) command("seek", line.time);
                }}
              >
                {line.text}
              </p>
            ))
          ) : (
            <div className="lyrics-empty">
              <Music2 size={35} />
              <strong>让旋律自己说话</strong>
              <span>暂时没有找到这首歌的歌词或字幕</span>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
