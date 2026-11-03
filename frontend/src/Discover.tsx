import {
  CalendarDays,
  ChevronRight,
  Disc3,
  Headphones,
  Play,
  RefreshCw,
} from "lucide-react";
import { useMemo } from "react";
import { Cover, Empty, IconButton, Spinner, TrackList } from "./components";
import type { Player, Shelf, Track } from "./types";

interface Props {
  mode: "discover" | "charts";
  name?: string;
  shelves: Shelf[];
  error: string;
  history: Track[];
  player: Player;
  chartTitle: string;
  onChartTitle: (title: string) => void;
  onCharts: (title?: string) => void;
  onSearch: () => void;
  onRefresh: () => void;
  onPlay: (track: Track, queue: Track[]) => void;
  onLike: (track: Track) => void;
  onContext: (track: Track, x: number, y: number, queue: Track[]) => void;
}

function greeting() {
  const hour = new Date().getHours();
  return hour < 6
    ? "夜深了"
    : hour < 11
      ? "早上好"
      : hour < 14
        ? "中午好"
        : hour < 18
          ? "下午好"
          : "晚上好";
}
function plays(value: number) {
  return value >= 1e8
    ? `${(value / 1e8).toFixed(1).replace(/\.0$/, "")}亿`
    : value >= 1e4
      ? `${(value / 1e4).toFixed(1).replace(/\.0$/, "")}万`
      : String(value);
}

export function DiscoverPage(props: Props) {
  const { mode, shelves, error, player, onPlay, onLike, onContext } = props;
  const date = new Intl.DateTimeFormat("zh-CN", {
    month: "long",
    day: "numeric",
  }).format(new Date());
  const weekday = new Intl.DateTimeFormat("zh-CN", { weekday: "long" }).format(
    new Date(),
  );
  const picks = useMemo(() => {
    const seen = new Set<string>();
    return shelves
      .flatMap((shelf) => shelf.tracks)
      .filter((track) => {
        if (seen.has(track.id)) return false;
        seen.add(track.id);
        return true;
      });
  }, [shelves]);
  const selected =
    shelves.find((shelf) => shelf.title === props.chartTitle) || shelves[0];
  const card = (track: Track, queue: Track[]) => (
    <button
      className="recommendation-card"
      key={track.id}
      onClick={() => onPlay(track, queue)}
      onContextMenu={(event) => {
        event.preventDefault();
        onContext(track, event.clientX, event.clientY, queue);
      }}
    >
      <div className="recommendation-art">
        <Cover track={track} size={240} />
        {!!track.playCount && (
          <span className="listen-count" title={`播放量 ${track.playCount}`}>
            <Headphones size={12} />
            {plays(track.playCount)}
          </span>
        )}
        <span className="recommendation-play">
          <Play size={22} fill="currentColor" />
        </span>
      </div>
      <strong title={track.title}>{track.title}</strong>
      <small>{track.artist || "Bilibili 音乐"}</small>
    </button>
  );
  return (
    <div
      className={`page discover-page ${mode === "charts" ? "charts-page" : ""}`}
    >
      <div className="discover-scroll">
        <header className="discovery-heading">
          <div>
            <span className="discovery-date">
              {date} {weekday}
            </span>
            <h1>
              {mode === "charts"
                ? "音乐排行榜"
                : `${greeting()}，${props.name || "欢迎回来"}`}
            </h1>
          </div>
          <IconButton title="刷新推荐" onClick={props.onRefresh}>
            <RefreshCw size={19} />
          </IconButton>
        </header>
        {mode === "discover" ? (
          <>
            <section className="daily-hero" aria-label="每日音乐推荐">
              <div className="daily-hero-copy">
                <span className="daily-hero-date">
                  {date} {weekday} · 来自 Bilibili 的音乐精选
                </span>
                <h2>每日推荐</h2>
                <p>
                  {picks.length
                    ? `${picks
                        .slice(0, 3)
                        .map((track) => track.artist)
                        .filter(Boolean)
                        .join("、")} 等 ${picks.length} 首`
                    : "从原创到现场，找到属于今天的旋律"}
                </p>
                <button
                  className="hero-play"
                  onClick={() =>
                    picks.length ? onPlay(picks[0], picks) : props.onSearch()
                  }
                >
                  <Play size={17} fill="currentColor" />
                  {picks.length ? "播放" : "寻找音乐"}
                </button>
              </div>
              <div className="hero-records" aria-hidden="true">
                <div className="vinyl-disc" />
                <div className="hero-sleeve sleeve-back">
                  <Cover track={picks[2]} size={160} />
                </div>
                <div className="hero-sleeve sleeve-middle">
                  <Cover track={picks[1]} size={178} />
                </div>
                <div className="hero-sleeve sleeve-front">
                  <Cover track={picks[0]} size={198} />
                  {!picks.length && (
                    <span className="sleeve-label">
                      <Disc3 size={25} />
                      <strong>音乐，让此刻不同</strong>
                      <small>BILITUNE</small>
                    </span>
                  )}
                </div>
              </div>
            </section>
            {error && (
              <div className="discovery-error">
                <span>推荐暂时无法加载</span>
                <p>{error}</p>
                <button className="text-button" onClick={props.onRefresh}>
                  重新加载 <RefreshCw size={13} />
                </button>
              </div>
            )}
            {shelves.length ? (
              shelves.map((shelf, index) => (
                <section className="recommendation-section" key={shelf.title}>
                  <div className="recommendation-heading">
                    <h2>{index === 0 ? "今日为你推荐" : shelf.title}</h2>
                    <button onClick={() => props.onCharts(shelf.title)}>
                      查看全部 <ChevronRight size={15} />
                    </button>
                  </div>
                  <div className="recommendation-grid">
                    {shelf.tracks
                      .slice(0, 4)
                      .map((track) => card(track, shelf.tracks))}
                  </div>
                </section>
              ))
            ) : error ? (
              props.history.length ? (
                <section className="recommendation-section">
                  <div className="recommendation-heading">
                    <h2>继续聆听</h2>
                    <span>你的最近播放</span>
                  </div>
                  <div className="recommendation-grid">
                    {props.history
                      .slice(0, 4)
                      .map((track) => card(track, props.history))}
                  </div>
                </section>
              ) : (
                <Empty
                  title="从喜欢的音乐开始"
                  detail="搜索歌曲、歌手或 BV 号，登录后还能同步收藏夹"
                  icon={<Headphones size={29} />}
                >
                  <button className="button outline" onClick={props.onSearch}>
                    搜索音乐
                  </button>
                </Empty>
              )
            ) : (
              <section
                className="recommendation-section"
                aria-busy="true"
                aria-label="正在加载推荐"
              >
                <div className="recommendation-heading">
                  <h2>今日为你推荐</h2>
                </div>
                <div className="recommendation-grid">
                  {[0, 1, 2, 3].map((index) => (
                    <div className="recommendation-skeleton" key={index}>
                      <div />
                      <span />
                      <small />
                    </div>
                  ))}
                </div>
              </section>
            )}
            <div className="discovery-endnote">
              <Headphones size={14} />
              <span>好音乐，值得被听见。</span>
            </div>
          </>
        ) : (
          <>
            <p className="charts-description">
              看看大家都在听什么，让喜欢的旋律多一点。
            </p>
            <div className="chart-tabs" role="group" aria-label="音乐榜单">
              {shelves.map((shelf) => (
                <button
                  key={shelf.title}
                  className={selected?.title === shelf.title ? "selected" : ""}
                  onClick={() => props.onChartTitle(shelf.title)}
                >
                  {shelf.title}
                </button>
              ))}
            </div>
            {selected ? (
              <>
                <div className="chart-summary">
                  <Cover track={selected.tracks[0]} size={84} />
                  <div>
                    <span>
                      <CalendarDays size={13} />
                      Bilibili 分区榜单
                    </span>
                    <h2>{selected.title}</h2>
                    <p>{selected.tracks.length} 首歌曲</p>
                  </div>
                  <button
                    className="button primary"
                    disabled={!selected.tracks.length}
                    onClick={() => onPlay(selected.tracks[0], selected.tracks)}
                  >
                    <Play size={15} fill="currentColor" />
                    播放全部
                  </button>
                </div>
                <TrackList
                  tracks={selected.tracks}
                  player={player}
                  onPlay={(track) => onPlay(track, selected.tracks)}
                  onLike={onLike}
                  onContext={(track, x, y) =>
                    onContext(track, x, y, selected.tracks)
                  }
                />
              </>
            ) : error ? (
              <Empty title="榜单暂时不可用" detail={error}>
                <button className="button outline" onClick={props.onRefresh}>
                  重试
                </button>
              </Empty>
            ) : (
              <Spinner />
            )}
          </>
        )}
      </div>
    </div>
  );
}
