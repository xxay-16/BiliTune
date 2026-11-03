import { useEffect, useMemo, useRef, useState } from "react";
import { ArrowUpRight, Disc3, Music2, Play, RefreshCw } from "lucide-react";
import { api, coverURL, filterTracks } from "./api";
import {
  ArtistLink,
  Cover,
  Empty,
  IconButton,
  SearchField,
  Spinner,
  TrackList,
} from "./components";
import type {
  Album,
  AlbumDetail,
  Artist,
  ArtistVideos,
  Player,
  Track,
} from "./types";

interface MusicActions {
  player: Player;
  onPlay: (track: Track, queue: Track[]) => void;
  onArtist: (track: Track) => void;
  onLike: (track: Track) => void;
  onContext: (track: Track, x: number, y: number, queue: Track[]) => void;
  notify: (text: string, error?: boolean) => void;
}

export function ArtistPage({
  artist,
  onAlbum,
  ...actions
}: MusicActions & { artist: Artist; onAlbum: (album: Album) => void }) {
  const [profile, setProfile] = useState(artist),
    [albums, setAlbums] = useState<Album[]>([]),
    [videos, setVideos] = useState<Track[]>([]);
  const [tab, setTab] = useState<"albums" | "videos">("albums"),
    [albumLoading, setAlbumLoading] = useState(true),
    [albumError, setAlbumError] = useState(""),
    [profileError, setProfileError] = useState("");
  const [videoLoading, setVideoLoading] = useState(false),
    [videoError, setVideoError] = useState(""),
    [page, setPage] = useState(0),
    [hasMore, setHasMore] = useState(false),
    [total, setTotal] = useState(0),
    [query, setQuery] = useState(""),
    [reload, setReload] = useState(0);
  const videoRequest = useRef<AbortController | null>(null);
  useEffect(() => {
    const controller = new AbortController();
    setProfile(artist);
    setAlbums([]);
    setVideos([]);
    setPage(0);
    setQuery("");
    setProfileError("");
    setAlbumError("");
    setAlbumLoading(true);
    setTab("albums");
    api<Artist>(`artist?id=${artist.mid}`, undefined, controller.signal)
      .then(setProfile)
      .catch((err) => {
        if (!controller.signal.aborted) setProfileError(err.message);
      });
    api<Album[]>(`artist/albums?id=${artist.mid}`, undefined, controller.signal)
      .then((data) => {
        setAlbums(data);
        if (!data.length) setTab("videos");
      })
      .catch((err) => {
        if (!controller.signal.aborted) setAlbumError(err.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setAlbumLoading(false);
      });
    return () => {
      controller.abort();
      videoRequest.current?.abort();
    };
  }, [artist.mid, reload]);
  async function loadVideos(next: number) {
    videoRequest.current?.abort();
    const controller = new AbortController();
    videoRequest.current = controller;
    setVideoLoading(true);
    setVideoError("");
    try {
      const data = await api<ArtistVideos>(
        `artist/videos?id=${artist.mid}&page=${next}`,
        undefined,
        controller.signal,
      );
      if (controller.signal.aborted) return;
      setVideos((old) =>
        next === 1
          ? data.tracks
          : [
              ...old,
              ...data.tracks.filter(
                (track) => !old.some((item) => item.id === track.id),
              ),
            ],
      );
      setPage(data.page);
      setHasMore(data.hasMore);
      setTotal(data.total);
    } catch (err) {
      if (!controller.signal.aborted) setVideoError((err as Error).message);
    } finally {
      if (!controller.signal.aborted) setVideoLoading(false);
    }
  }
  useEffect(() => {
    if (tab === "videos" && page === 0 && !videoLoading && !videoError)
      void loadVideos(1);
  }, [tab, page, artist.mid, videoLoading, videoError]);
  const visibleAlbums = albums.filter((album) =>
    album.title.toLocaleLowerCase().includes(query.toLocaleLowerCase()),
  );
  const shownTracks = useMemo(
    () =>
      filterTracks(videos, query).map((track) => ({
        ...track,
        artist: track.artist || profile.name,
        artistMid: track.artistMid || artist.mid,
      })),
    [videos, query, profile.name, artist.mid],
  );
  const external = () =>
    void api("open", { url: `https://space.bilibili.com/${artist.mid}` }).catch(
      (err) => actions.notify(err.message, true),
    );
  return (
    <div className="page artist-page">
      <header className="artist-profile">
        <div className="artist-avatar">
          {profile.avatar ? (
            <img src={coverURL(profile.avatar)} alt="" />
          ) : (
            <span>{profile.name.slice(0, 1) || <Music2 size={30} />}</span>
          )}
        </div>
        <div className="artist-profile-info">
          <span className="eyebrow">UP 主 · Bilibili 创作者</span>
          <h1>{profile.name || "UP 主主页"}</h1>
          <p>{profile.signature || "在这里听见 TA 的音乐与作品"}</p>
          <div className="artist-stats">
            <span>{albums.length} 张专辑</span>
            {!!profile.videoCount && <span>{profile.videoCount} 个作品</span>}
            {!!profile.followers && (
              <span>{profile.followers.toLocaleString()} 位粉丝</span>
            )}
          </div>
        </div>
        <div className="artist-profile-actions">
          <IconButton
            title="刷新 UP 主主页"
            onClick={() => setReload((value) => value + 1)}
          >
            <RefreshCw size={18} />
          </IconButton>
          <button className="button outline" onClick={external}>
            B 站主页 <ArrowUpRight size={15} />
          </button>
        </div>
      </header>
      {profileError && (
        <p className="profile-note">
          UP 主资料暂时不可用，仍可查看公开专辑。{profileError}
        </p>
      )}
      <div className="artist-toolbar">
        <div className="artist-tabs" role="group" aria-label="UP 主内容">
          <button
            className={tab === "albums" ? "selected" : ""}
            aria-pressed={tab === "albums"}
            onClick={() => {
              setTab("albums");
              setQuery("");
            }}
          >
            专辑 <span>{albums.length}</span>
          </button>
          <button
            className={tab === "videos" ? "selected" : ""}
            aria-pressed={tab === "videos"}
            onClick={() => {
              setTab("videos");
              setQuery("");
            }}
          >
            全部作品 {total > 0 && <span>{total}</span>}
          </button>
        </div>
        <SearchField
          value={query}
          onChange={setQuery}
          placeholder={
            tab === "albums" ? "搜索 UP 主的专辑" : "搜索已加载的作品"
          }
        />
      </div>
      {tab === "albums" ? (
        <div className="artist-albums-scroll">
          {albumLoading ? (
            <Spinner />
          ) : albumError ? (
            <Empty
              title="专辑暂时无法加载"
              detail={albumError}
              icon={<Disc3 size={30} />}
            >
              <button
                className="button outline"
                onClick={() => setReload((value) => value + 1)}
              >
                重新加载
              </button>
            </Empty>
          ) : visibleAlbums.length ? (
            <div className="artist-album-grid">
              {visibleAlbums.map((album) => (
                <button
                  className="album-card"
                  key={`${album.type}:${album.id}`}
                  onClick={() => onAlbum({ ...album, artist: profile.name })}
                >
                  <div className="album-card-art">
                    <div className="album-card-disc" />
                    <Cover folder={{ ...album, private: false }} size={220} />
                    <span className="album-open">
                      <Disc3 size={20} />
                    </span>
                  </div>
                  <strong title={album.title}>{album.title}</strong>
                  <span>
                    {album.count} 首 ·{" "}
                    {album.type === "season" ? "合集专辑" : "系列专辑"}
                  </span>
                </button>
              ))}
            </div>
          ) : (
            <Empty
              title={query ? "没有找到匹配的专辑" : "这位 UP 主还没有公开专辑"}
              detail={
                query ? "试试其他关键词" : "TA 的 B 站合集和系列会显示在这里"
              }
              icon={<Disc3 size={30} />}
            >
              <button
                className="button outline"
                onClick={() => setTab("videos")}
              >
                查看全部作品
              </button>
            </Empty>
          )}
        </div>
      ) : (
        <>
          <p className="artist-list-summary">
            已加载 {videos.length} 个作品
            {query && ` · 匹配 ${shownTracks.length} 个`}
          </p>
          {shownTracks.length ? (
            <TrackList
              tracks={shownTracks}
              player={actions.player}
              onPlay={(track) => actions.onPlay(track, shownTracks)}
              onArtist={actions.onArtist}
              onLike={actions.onLike}
              onContext={(track, x, y) =>
                actions.onContext(track, x, y, shownTracks)
              }
            />
          ) : videoLoading ? (
            <Spinner />
          ) : (
            !videoError && (
              <Empty title={query ? "没有匹配的作品" : "还没有公开作品"} />
            )
          )}{" "}
          {videoError && (
            <div className="artist-load-error">
              <span>{videoError}</span>
              <button
                className="text-button"
                onClick={() => void loadVideos(page + 1)}
              >
                重试
              </button>
            </div>
          )}
          {hasMore && (
            <button
              className="button outline load-more"
              disabled={videoLoading}
              onClick={() => void loadVideos(page + 1)}
            >
              {videoLoading ? "正在加载…" : "加载更多作品"}
            </button>
          )}
        </>
      )}
    </div>
  );
}

export function AlbumPage({
  album,
  ...actions
}: MusicActions & { album: Album }) {
  const [detail, setDetail] = useState<AlbumDetail | null>(null),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [query, setQuery] = useState(""),
    [reload, setReload] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setDetail(null);
    setQuery("");
    setError("");
    setLoading(true);
    api<AlbumDetail>(
      `album?mid=${album.mid}&id=${album.id}&type=${album.type}`,
      undefined,
      controller.signal,
    )
      .then(setDetail)
      .catch((err) => {
        if (!controller.signal.aborted) setError(err.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [album.mid, album.id, album.type, reload]);
  const current = {
    ...album,
    ...detail?.album,
    title: detail?.album.title || album.title,
    coverUrl: detail?.album.coverUrl || album.coverUrl,
  };
  const tracks = useMemo(
    () =>
      (detail?.tracks || []).map((track) => ({
        ...track,
        artist: track.artist || album.artist || "UP 主",
        artistMid: track.artistMid || album.mid,
      })),
    [detail?.tracks, album.artist, album.mid],
  );
  const visible = useMemo(() => filterTracks(tracks, query), [tracks, query]);
  const duration = tracks.reduce((sum, track) => sum + track.durationMs, 0);
  return (
    <div className="page album-page">
      <header className="album-hero">
        <div className="album-hero-cover">
          <Cover folder={{ ...current, private: false }} size={174} />
          <span>
            <Disc3 size={17} />
            专辑
          </span>
        </div>
        <div className="album-hero-info">
          <span className="eyebrow">
            {album.type === "season" ? "Bilibili 合集" : "Bilibili 系列"} · 专辑
          </span>
          <h1 title={current.title}>{current.title || "专辑"}</h1>
          <ArtistLink
            track={{
              id: `artist:${album.mid}`,
              title: "",
              artist: album.artist || "UP 主主页",
              artistMid: album.mid,
              durationMs: 0,
              type: "video",
            }}
            onArtist={actions.onArtist}
          />
          <p>
            {loading ? "正在加载完整专辑…" : `${tracks.length} 首`}
            {duration > 0 && ` · ${Math.round(duration / 60000)} 分钟`}
          </p>
          <div className="album-hero-buttons">
            <button
              className="button primary"
              disabled={!visible.length || loading}
              onClick={() => actions.onPlay(visible[0], visible)}
            >
              <Play size={16} fill="currentColor" />
              播放全部
            </button>
            <IconButton
              title="刷新专辑"
              onClick={() => setReload((value) => value + 1)}
            >
              <RefreshCw size={18} />
            </IconButton>
          </div>
        </div>
      </header>
      {current.intro && <p className="album-intro">{current.intro}</p>}
      <div className="album-song-heading">
        <h2>
          专辑歌曲 <span>{loading ? "" : tracks.length}</span>
        </h2>
        <SearchField
          value={query}
          onChange={setQuery}
          placeholder="搜索专辑内的歌曲"
        />
      </div>
      {loading ? (
        <Spinner />
      ) : error ? (
        <Empty title="专辑暂时无法加载" detail={error}>
          <button
            className="button outline"
            onClick={() => setReload((value) => value + 1)}
          >
            重试
          </button>
        </Empty>
      ) : visible.length ? (
        <TrackList
          tracks={visible}
          player={actions.player}
          onArtist={actions.onArtist}
          onPlay={(track) => actions.onPlay(track, visible)}
          onLike={actions.onLike}
          onContext={(track, x, y) => actions.onContext(track, x, y, visible)}
        />
      ) : (
        <Empty
          title={query ? "没有匹配的歌曲" : "这张专辑暂时没有公开歌曲"}
          detail={query ? "试试其他关键词" : undefined}
        />
      )}
    </div>
  );
}
