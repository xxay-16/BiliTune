import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Search,
  Download as DownloadIcon,
  Settings,
  Plus,
  RefreshCw,
  LogIn,
  Minus,
  Square,
  X,
  Sun,
  Moon,
  PanelLeftClose,
  PanelLeftOpen,
  Play,
  ListMusic,
  FolderInput,
  Heart,
  Trash2,
  ArrowUpRight,
  FolderOpen,
  Pause,
  FileAudio,
  Clock3,
  Check,
  AlertCircle,
  ChevronRight,
  Home,
  ChartNoAxesColumnIncreasing,
  ChevronLeft,
  ChevronsUpDown,
} from "lucide-react";
import { api, bytes, filterTracks, coverURL } from "./api";
import {
  ContextMenu,
  Cover,
  CreateButton,
  Empty,
  FolderTile,
  IconButton,
  MenuItem,
  PlayerBar,
  SearchField,
  Spinner,
  TrackList,
  standardFolderIcons,
} from "./components";
import { CreateFolderDialog, FolderPicker, LoginDialog } from "./dialogs";
import { SettingsPage } from "./Settings";
import { Immersive } from "./Immersive";
import { DiscoverPage } from "./Discover";
import { usePageNavigation } from "./navigation";
import type {
  Account,
  Bootstrap,
  Download,
  Folder,
  Player,
  Shelf,
  Track,
} from "./types";

type Selection =
  | { kind: "folder"; id: number }
  | { kind: "recent" | "downloads" };
type Menu = {
  track: Track;
  x: number;
  y: number;
  from?: number;
  queue?: Track[];
};

export function App() {
  const [boot, setBoot] = useState<Bootstrap | null>(null),
    [fatal, setFatal] = useState("");
  const {
    page,
    navigate: setPage,
    back,
    forward,
    canBack,
    canForward,
  } = usePageNavigation();
  const [chartTitle, setChartTitle] = useState("");
  const [collapsed, setCollapsed] = useState(false),
    [folders, setFolders] = useState<Folder[]>([]),
    [selection, setSelection] = useState<Selection>({ kind: "recent" });
  const [tracks, setTracks] = useState<Track[]>([]),
    [loading, setLoading] = useState(false),
    [loadError, setLoadError] = useState(""),
    [filter, setFilter] = useState(""),
    [reload, setReload] = useState(0);
  const [shelves, setShelves] = useState<Shelf[]>([]),
    [discoverError, setDiscoverError] = useState("");
  const [search, setSearch] = useState(""),
    [query, setQuery] = useState(""),
    [musicOnly, setMusicOnly] = useState(false),
    [searchTracks, setSearchTracks] = useState<Track[]>([]),
    [searchPage, setSearchPage] = useState(1),
    [searchBusy, setSearchBusy] = useState(false),
    [more, setMore] = useState(false),
    [searchError, setSearchError] = useState("");
  const [downloads, setDownloads] = useState<Download[]>([]),
    [queue, setQueue] = useState<Track[]>([]),
    [queueOpen, setQueueOpen] = useState(false),
    [immersive, setImmersive] = useState(false);
  const [login, setLogin] = useState(false),
    [create, setCreate] = useState(false),
    [picker, setPicker] = useState<{ track: Track; from?: number } | null>(
      null,
    ),
    [menu, setMenu] = useState<Menu | null>(null),
    [folderMenu, setFolderMenu] = useState<{
      folder: Folder;
      x: number;
      y: number;
    } | null>(null);
  const [notice, setNotice] = useState<{ text: string; error: boolean } | null>(
    null,
  );
  const noticeTimer = useRef<ReturnType<typeof setTimeout> | undefined>(
    undefined,
  );
  const notify = useCallback((text: string, error = false) => {
    setNotice({ text, error });
    clearTimeout(noticeTimer.current);
    noticeTimer.current = setTimeout(
      () => setNotice(null),
      error ? 7000 : 3500,
    );
  }, []);
  const applyBoot = useCallback((data: Bootstrap) => setBoot(data), []);
  const refresh = useCallback(async () => {
    const data = await api<Bootstrap>("bootstrap");
    setBoot(data);
    return data;
  }, []);
  const refreshFolders = useCallback(async () => {
    try {
      setFolders(await api<Folder[]>("folders"));
    } catch (err) {
      notify((err as Error).message, true);
    }
  }, [notify]);
  useEffect(() => {
    let canceled = false;
    api<Bootstrap>("bootstrap")
      .then((data) => {
        if (!canceled) setBoot(data);
      })
      .catch((err) => setFatal(err.message));
    return () => {
      canceled = true;
      clearTimeout(noticeTimer.current);
    };
  }, []);
  useEffect(() => {
    let canceled = false;
    api<{ account: Account | null }>("account")
      .then(async (data) => {
        if (canceled) return;
        setBoot((old) => (old ? { ...old, account: data.account } : old));
        if (data.account) await refreshFolders();
      })
      .catch((err) => notify("登录状态验证失败：" + err.message, true));
    return () => {
      canceled = true;
    };
  }, [refreshFolders, notify]);
  useEffect(() => {
    api<Download[]>("downloads")
      .then(setDownloads)
      .catch(() => {});
  }, []);
  useEffect(() => {
    if (!boot?.player.track) return;
    api<Track[]>("history")
      .then((history) => {
        setBoot((old) => (old ? { ...old, history } : old));
      })
      .catch(() => {});
  }, [boot?.player.track?.id]);
  const lastError = useRef("");
  useEffect(() => {
    let canceled = false,
      busy = false;
    const poll = async () => {
      if (busy) return;
      busy = true;
      try {
        const player = await api<Player>("player");
        if (canceled) return;
        setBoot((old) => (old ? { ...old, player } : old));
        if (player.error && player.error !== lastError.current)
          notify(player.error, true);
        lastError.current = player.error || "";
      } catch {
      } finally {
        busy = false;
      }
    };
    const timer = setInterval(poll, 400);
    return () => {
      canceled = true;
      clearInterval(timer);
    };
  }, [notify]);
  useEffect(() => {
    if (!boot) return;
    const media = matchMedia("(prefers-color-scheme: dark)");
    const theme = () => {
      document.documentElement.dataset.theme =
        boot.settings.theme === "system"
          ? media.matches
            ? "dark"
            : "light"
          : boot.settings.theme;
    };
    theme();
    media.addEventListener("change", theme);
    return () => media.removeEventListener("change", theme);
  }, [boot?.settings.theme]);
  useEffect(() => {
    if (page !== "library") return;
    const controller = new AbortController();
    setFilter("");
    setLoading(true);
    setLoadError("");
    const path =
      selection.kind === "folder"
        ? `folder/tracks?id=${selection.id}`
        : selection.kind === "recent"
          ? "history"
          : "downloads";
    api<Track[] | Download[]>(path, undefined, controller.signal)
      .then((data) =>
        setTracks(
          selection.kind === "downloads"
            ? (data as Download[])
                .filter((d) => d.status === "completed")
                .map((d) => ({ ...d.track, localPath: d.path }))
            : (data as Track[]),
        ),
      )
      .catch((err) => {
        if (!controller.signal.aborted) setLoadError(err.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [page, selection, reload]);
  useEffect(() => {
    if ((page !== "discover" && page !== "charts") || shelves.length) return;
    const controller = new AbortController();
    setDiscoverError("");
    api<Shelf[]>("discover", undefined, controller.signal)
      .then(setShelves)
      .catch((err) => {
        if (!controller.signal.aborted) setDiscoverError(err.message);
      });
    return () => controller.abort();
  }, [page, shelves.length, reload]);
  useEffect(() => {
    if (page !== "downloads") return;
    let done = false,
      busy = false;
    const poll = async () => {
      if (busy) return;
      busy = true;
      try {
        const data = await api<Download[]>("downloads");
        if (!done) setDownloads(data);
      } catch (err) {
        if (!done) notify((err as Error).message, true);
      } finally {
        busy = false;
      }
    };
    void poll();
    const timer = setInterval(poll, 1000);
    return () => {
      done = true;
      clearInterval(timer);
    };
  }, [page, notify]);
  useEffect(() => {
    if (!queueOpen) return;
    void api<Track[]>("queue")
      .then(setQueue)
      .catch((err) => notify(err.message, true));
  }, [queueOpen, boot?.player.queueVersion, notify]);
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (
        e.key === " " &&
        !e.repeat &&
        !["INPUT", "TEXTAREA", "SELECT", "BUTTON"].includes(
          (e.target as HTMLElement).tagName,
        ) &&
        !document.querySelector("[role=dialog]")
      ) {
        e.preventDefault();
        void api("command", { action: "playPause" }).catch((err) =>
          notify(err.message, true),
        );
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [notify]);
  const command = useCallback(
    (action: string, value = 0) => {
      void api("command", { action, value }).catch((err) =>
        notify(err.message, true),
      );
    },
    [notify],
  );
  const windowAction = (action: string) =>
    void api("window", { action }).catch((err) => notify(err.message, true));
  const play = useCallback(
    (track: Track, list: Track[]) => {
      void api("play", { track, queue: list }).catch((err) =>
        notify(err.message, true),
      );
    },
    [notify],
  );
  const onLogin = useCallback(
    (account: Account) => {
      setBoot((old) => (old ? { ...old, account } : old));
      setLogin(false);
      void refreshFolders();
      notify("已登录 " + account.name);
    },
    [refreshFolders, notify],
  );
  const chooseFavorite = (track: Track, from?: number) => {
    if (!boot?.account) {
      setLogin(true);
      return;
    }
    setPicker({ track, from });
  };
  const removeFavorite = async (track: Track, from: number) => {
    try {
      await api("favorite", { track, from, to: 0 });
      setTracks((old) => old.filter((t) => t.id !== track.id));
      void refreshFolders();
      notify("已从收藏夹移除");
    } catch (err) {
      notify((err as Error).message, true);
    }
  };
  const addDownload = async (track: Track) => {
    try {
      await api("download", track);
      notify("已加入下载任务");
    } catch (err) {
      notify((err as Error).message, true);
    }
  };
  const queueAction = async (action: string, track?: Track, id?: string) => {
    try {
      await api("queue", { action, track, id });
      setQueue(await api<Track[]>("queue"));
      if (action === "next" || action === "append") notify("已加入播放队列");
    } catch (err) {
      notify((err as Error).message, true);
    }
  };
  const visible = useMemo(() => filterTracks(tracks, filter), [tracks, filter]);
  const activeFolder =
    selection.kind === "folder"
      ? folders.find((f) => f.id === selection.id)
      : undefined;
  const collectionTitle =
    selection.kind === "recent"
      ? "最近播放"
      : selection.kind === "downloads"
        ? "我的下载"
        : activeFolder?.title || "收藏夹";
  async function searchFor(text: string, pageNumber = 1) {
    const q = text.trim();
    if (!q) return;
    setPage("search");
    setQuery(q);
    setSearch(q);
    setSearchBusy(true);
    setSearchError("");
    try {
      const data = await api<Track[]>(
        `search?q=${encodeURIComponent(q)}&page=${pageNumber}&music=${musicOnly}`,
      );
      setSearchTracks((old) =>
        pageNumber === 1
          ? data
          : [...old, ...data.filter((t) => !old.some((o) => o.id === t.id))],
      );
      setSearchPage(pageNumber);
      setMore(data.length === 20);
      void refresh();
    } catch (err) {
      setSearchError((err as Error).message);
    } finally {
      setSearchBusy(false);
    }
  }
  function selectFolder(value: Selection) {
    setSelection(value);
    setPage("library");
    setMenu(null);
  }
  async function localFiles() {
    try {
      const files = await api<Track[]>("dialog/files", {});
      if (files.length) {
        play(files[0], files);
        notify("已载入 " + files.length + " 首本地音频");
      }
    } catch (err) {
      notify((err as Error).message, true);
    }
  }
  if (!boot)
    return (
      <div className="startup">
        <img src="/logo.png" alt="" />
        <h1>BiliTune</h1>
        {fatal ? (
          <>
            <p className="error">{fatal}</p>
            <button
              className="button outline"
              onClick={() => location.reload()}
            >
              重试
            </button>
          </>
        ) : (
          <Spinner />
        )}
      </div>
    );
  return (
    <div className={"app " + (collapsed ? "sidebar-collapsed" : "")}>
      <header className="titlebar">
        <div className="page-history">
          <IconButton title="后退" disabled={!canBack} onClick={back}>
            <ChevronLeft size={22} />
          </IconButton>
          <IconButton title="前进" disabled={!canForward} onClick={forward}>
            <ChevronRight size={22} />
          </IconButton>
        </div>
        <div className="titlebar-drag" title="拖动窗口" />
        <IconButton
          title="切换主题"
          onClick={async () => {
            try {
              await api("settings", {
                ...boot.settings,
                theme:
                  document.documentElement.dataset.theme === "dark"
                    ? "light"
                    : "dark",
              });
              await refresh();
            } catch (err) {
              notify((err as Error).message, true);
            }
          }}
        >
          {document.documentElement.dataset.theme === "dark" ? (
            <Sun size={18} />
          ) : (
            <Moon size={18} />
          )}
        </IconButton>
        <div className="window-controls">
          <button
            title="最小化"
            aria-label="最小化"
            onClick={() => windowAction("minimize")}
          >
            <Minus size={17} />
          </button>
          <button
            title="最大化或还原"
            aria-label="最大化或还原"
            onClick={() => windowAction("maximize")}
          >
            <Square size={14} />
          </button>
          <button
            className="window-close"
            title="关闭窗口"
            aria-label="关闭窗口"
            onClick={() => windowAction("close")}
          >
            <X size={20} />
          </button>
        </div>
      </header>
      <aside className="sidebar">
        <div className="sidebar-search-row">
          {collapsed ? (
            <IconButton title="搜索音乐" onClick={() => setPage("search")}>
              <Search size={21} />
            </IconButton>
          ) : (
            <form
              className="sidebar-search"
              onSubmit={(event) => {
                event.preventDefault();
                void searchFor(search);
              }}
            >
              <Search size={16} />
              <input
                aria-label="搜索歌曲、歌手"
                placeholder="搜索歌曲、歌手、BV 号"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                onFocus={() => setPage("search")}
              />
            </form>
          )}
        </div>
        <nav aria-label="音乐导航">
          <button
            className={page === "discover" ? "selected" : ""}
            title="为你推荐"
            onClick={() => setPage("discover")}
          >
            <Home size={20} />
            {!collapsed && <span>为你推荐</span>}
          </button>
          <button
            className={page === "charts" ? "selected" : ""}
            title="音乐排行榜"
            onClick={() => setPage("charts")}
          >
            <ChartNoAxesColumnIncreasing size={20} />
            {!collapsed && <span>音乐排行榜</span>}
          </button>
          <button
            className={
              page === "library" && selection.kind !== "recent"
                ? "selected"
                : ""
            }
            title="我的音乐"
            onClick={() => {
              if (selection.kind === "recent" && folders.length)
                selectFolder({ kind: "folder", id: folders[0].id });
              else setPage("library");
            }}
          >
            <ListMusic size={20} />
            {!collapsed && <span>我的音乐</span>}
          </button>
          <button
            className={
              page === "library" && selection.kind === "recent"
                ? "selected"
                : ""
            }
            title="最近播放"
            onClick={() => selectFolder({ kind: "recent" })}
          >
            <Clock3 size={20} />
            {!collapsed && <span>最近播放</span>}
          </button>
          <button
            className={page === "downloads" ? "selected" : ""}
            title="下载管理"
            onClick={() => setPage("downloads")}
          >
            <DownloadIcon size={20} />
            {!collapsed && <span>下载管理</span>}
          </button>
          <button
            className={page === "search" ? "selected" : ""}
            title="搜索结果"
            onClick={() => setPage("search")}
          >
            <Search size={20} />
            {!collapsed && <span>搜索结果</span>}
          </button>
        </nav>
        <div className="sidebar-label">
          {!collapsed && (
            <button onClick={() => setPage("library")}>我的歌单</button>
          )}
          <IconButton title="我的歌单" onClick={() => setPage("library")}>
            <ListMusic size={15} />
          </IconButton>
          {!collapsed && (
            <IconButton
              title="新建歌单"
              onClick={() => (boot.account ? setCreate(true) : setLogin(true))}
            >
              <Plus size={16} />
            </IconButton>
          )}
        </div>
        <div className="sidebar-folders">
          {!collapsed && (
            <>
              {folders.map((folder) => (
                <button
                  key={folder.id}
                  className={
                    page === "library" &&
                    selection.kind === "folder" &&
                    selection.id === folder.id
                      ? "selected"
                      : ""
                  }
                  onClick={() =>
                    selectFolder({ kind: "folder", id: folder.id })
                  }
                  onContextMenu={(event) => {
                    event.preventDefault();
                    setFolderMenu({
                      folder,
                      x: event.clientX,
                      y: event.clientY,
                    });
                  }}
                >
                  <Cover folder={folder} size={29} />
                  <span>
                    <strong>{folder.title}</strong>
                    <small>{folder.count} 首</small>
                  </span>
                </button>
              ))}
              {!boot.account && (
                <button
                  className="sidebar-login"
                  onClick={() => setLogin(true)}
                >
                  <LogIn size={18} />
                  <span>登录同步收藏夹</span>
                </button>
              )}
              {!!boot.account && !folders.length && (
                <p className="sidebar-no-folders">
                  收藏你喜欢的音乐， 让好旋律留在身边。
                </p>
              )}
            </>
          )}
        </div>
        <div className="sidebar-tools">
          <button
            className="local-button"
            title="打开本地音频"
            onClick={localFiles}
          >
            <FileAudio size={18} />
            {!collapsed && <span>本地音乐</span>}
          </button>
          <IconButton
            title="设置"
            active={page === "settings"}
            onClick={() => setPage("settings")}
          >
            <Settings size={18} />
          </IconButton>
          <IconButton
            title={collapsed ? "展开侧栏" : "收起侧栏"}
            onClick={() => setCollapsed(!collapsed)}
          >
            {collapsed ? (
              <PanelLeftOpen size={17} />
            ) : (
              <PanelLeftClose size={17} />
            )}
          </IconButton>
        </div>
        {boot.player.track && !collapsed && (
          <button
            className="sidebar-now-playing"
            onClick={() => setImmersive(true)}
            title="打开正在播放"
          >
            <Cover track={boot.player.track} size={45} />
            <span>
              <strong>{boot.player.track.title}</strong>
              <small>{boot.player.track.artist}</small>
            </span>
            {boot.player.playing && (
              <span className="playing-indicator">
                <i />
                <i />
                <i />
              </span>
            )}
          </button>
        )}
        <button
          className="sidebar-account"
          onClick={() => (boot.account ? setPage("settings") : setLogin(true))}
          title={boot.account?.name || "登录 Bilibili"}
        >
          <span className="avatar">
            {boot.account?.avatar ? (
              <img
                src={coverURL(boot.account.avatar)}
                alt=""
                onError={(event) => {
                  event.currentTarget.style.display = "none";
                }}
              />
            ) : null}
            {boot.account?.name.slice(0, 1) || <img src="/logo.png" alt="" />}
          </span>
          {!collapsed && (
            <>
              <strong>{boot.account?.name || "登录 Bilibili"}</strong>
              <ChevronsUpDown size={15} />
            </>
          )}
        </button>
      </aside>
      <main className="main-content">
        {page === "library" && (
          <div className="page library-page">
            <header className="page-heading">
              <div>
                <span className="eyebrow">你的音乐，随时相伴</span>
                <h1>我的歌单</h1>
              </div>
              <CreateButton
                onClick={() =>
                  boot.account ? setCreate(true) : setLogin(true)
                }
              />
            </header>
            <div className="library-columns">
              <section className="collection-panel">
                <div className="section-heading">
                  <h2>歌单</h2>
                  <IconButton
                    title="刷新收藏夹"
                    onClick={() => {
                      void refreshFolders();
                      setReload((v) => v + 1);
                    }}
                  >
                    <RefreshCw size={17} />
                  </IconButton>
                </div>
                <div className="collection-scroll">
                  <FolderTile
                    title="最近播放"
                    count={boot.history.length}
                    selected={selection.kind === "recent"}
                    onClick={() => selectFolder({ kind: "recent" })}
                    icon={standardFolderIcons.recent}
                  />
                  <FolderTile
                    title="我的下载"
                    count={
                      downloads.filter((d) => d.status === "completed").length
                    }
                    selected={selection.kind === "downloads"}
                    onClick={() => selectFolder({ kind: "downloads" })}
                    icon={standardFolderIcons.downloads}
                  />
                  {folders.map((folder) => (
                    <FolderTile
                      key={folder.id}
                      folder={folder}
                      selected={
                        selection.kind === "folder" &&
                        selection.id === folder.id
                      }
                      onClick={() =>
                        selectFolder({ kind: "folder", id: folder.id })
                      }
                      onContext={(x, y) => setFolderMenu({ folder, x, y })}
                    />
                  ))}
                  {!boot.account && (
                    <Empty
                      title="收藏夹等你登录"
                      detail="登录 Bilibili，同步全部收藏音乐"
                    >
                      <button
                        className="button outline"
                        onClick={() => setLogin(true)}
                      >
                        登录账号
                      </button>
                    </Empty>
                  )}
                </div>
              </section>
              <section className="songs-panel">
                <div className="collection-hero">
                  <Cover folder={activeFolder} size={124} />
                  <div>
                    <span className="eyebrow">
                      {selection.kind === "folder"
                        ? "B 站收藏夹"
                        : "本地音乐库"}
                    </span>
                    <h2 title={collectionTitle}>{collectionTitle}</h2>
                    <p>
                      {loading ? (activeFolder?.count ?? "—") : tracks.length}{" "}
                      首{filter && " · 搜索到 " + visible.length + " 首"}
                    </p>
                    <button
                      className="button primary"
                      disabled={!visible.length || loading}
                      onClick={() => play(visible[0], visible)}
                    >
                      <Play size={16} fill="currentColor" />
                      播放
                    </button>
                  </div>
                </div>
                <SearchField
                  value={filter}
                  onChange={setFilter}
                  placeholder="搜索当前歌单的歌曲"
                />
                {loading ? (
                  <Spinner />
                ) : loadError ? (
                  <Empty title="加载失败" detail={loadError}>
                    <button
                      className="button outline"
                      onClick={() => setReload((v) => v + 1)}
                    >
                      重试
                    </button>
                  </Empty>
                ) : visible.length ? (
                  <TrackList
                    tracks={visible}
                    player={boot.player}
                    onPlay={(t) => play(t, visible)}
                    onContext={(track, x, y) =>
                      setMenu({ track, x, y, from: activeFolder?.id })
                    }
                    liked={!!activeFolder}
                    onLike={(t) =>
                      activeFolder
                        ? void removeFavorite(t, activeFolder.id)
                        : chooseFavorite(t)
                    }
                  />
                ) : (
                  <Empty
                    title={filter ? "没有匹配的歌曲" : "这里还没有歌曲"}
                    detail={
                      filter
                        ? "换个关键词试试"
                        : "从搜索或收藏夹开始你的音乐时光"
                    }
                  />
                )}
              </section>
            </div>
          </div>
        )}
        {(page === "discover" || page === "charts") && (
          <DiscoverPage
            mode={page}
            name={boot.account?.name}
            shelves={shelves}
            error={discoverError}
            history={boot.history}
            player={boot.player}
            chartTitle={chartTitle}
            onChartTitle={setChartTitle}
            onCharts={(title) => {
              setChartTitle(title || "");
              setPage("charts");
            }}
            onSearch={() => setPage("search")}
            onRefresh={() => {
              setShelves([]);
              setReload((value) => value + 1);
            }}
            onPlay={play}
            onLike={(track) => chooseFavorite(track)}
            onContext={(track, x, y, queue) => setMenu({ track, x, y, queue })}
          />
        )}
        {page === "search" && (
          <div className="page search-page">
            <header className="page-heading">
              <div>
                <span className="eyebrow">每一个关键词，都有旋律</span>
                <h1>搜索</h1>
              </div>
            </header>
            <form
              className="search-form"
              onSubmit={(e) => {
                e.preventDefault();
                void searchFor(search);
              }}
            >
              <SearchField
                value={search}
                onChange={setSearch}
                placeholder="搜索歌曲、歌手，或粘贴 BV / AV 号与链接"
              />
              <button
                className="button primary"
                disabled={searchBusy || !search.trim()}
                type="submit"
              >
                搜索
              </button>
            </form>
            <label className="checkbox search-option">
              <input
                type="checkbox"
                checked={musicOnly}
                onChange={(e) => setMusicOnly(e.target.checked)}
              />
              仅音乐分区
            </label>
            {!query ? (
              <div className="search-initial">
                <h2>最近搜索</h2>
                <div className="history-chips">
                  {boot.searchHistory.map((q) => (
                    <button key={q} onClick={() => void searchFor(q)}>
                      {q}
                      <ChevronRight size={14} />
                    </button>
                  ))}
                </div>
                <Empty
                  title="寻找你的下一首循环"
                  detail="支持哔哩哔哩视频、音频和收藏夹音轨"
                  icon={<Search size={30} />}
                />
              </div>
            ) : (
              <>
                <div className="search-result-heading">
                  <span>
                    “{query}” 的搜索结果 · {searchTracks.length} 首
                  </span>
                  {searchBusy && <RefreshCw size={16} className="spin" />}
                </div>
                {searchError && <p className="error">{searchError}</p>}
                {searchTracks.length ? (
                  <TrackList
                    tracks={searchTracks}
                    player={boot.player}
                    onPlay={(t) => play(t, searchTracks)}
                    onLike={(t) => chooseFavorite(t)}
                    onContext={(track, x, y) => setMenu({ track, x, y })}
                  />
                ) : searchBusy ? (
                  <Spinner />
                ) : (
                  <Empty
                    title="没有找到相关内容"
                    detail="试试歌曲名、UP 主或 BV 号"
                  />
                )}
                {more && (
                  <button
                    className="button outline load-more"
                    disabled={searchBusy}
                    onClick={() => void searchFor(query, searchPage + 1)}
                  >
                    加载更多
                  </button>
                )}
              </>
            )}
          </div>
        )}
        {page === "downloads" && (
          <div className="page downloads-page">
            <header className="page-heading">
              <div>
                <span className="eyebrow">让喜欢，留在身边</span>
                <h1>下载</h1>
              </div>
              <button className="button outline" onClick={localFiles}>
                <FileAudio size={17} />
                打开本地音频
              </button>
            </header>
            <div className="downloads-summary">
              <span>
                {downloads.filter((d) => d.status === "completed").length}{" "}
                首已完成
              </span>
              <span>
                {downloads.filter((d) => d.status === "downloading").length}{" "}
                个正在下载
              </span>
              <button
                className="text-button"
                onClick={() => setPage("settings")}
              >
                下载设置 <ChevronRight size={14} />
              </button>
            </div>
            <div className="page-scroll">
              {downloads.length ? (
                downloads.map((task) => (
                  <div className="download-row" key={task.id}>
                    <Cover track={task.track} size={52} />
                    <div className="download-info">
                      <strong>{task.track.title}</strong>
                      <small>{task.track.artist}</small>
                      <div className="download-progress">
                        <span
                          style={{
                            width: `${task.total > 0 ? Math.min(100, (task.received / task.total) * 100) : 0}%`,
                          }}
                        />
                      </div>
                      <span
                        className={
                          task.status === "failed" ? "error" : "download-status"
                        }
                      >
                        {task.error ||
                          {
                            queued: "等待下载",
                            downloading: "正在下载",
                            paused: "已暂停",
                            completed: "已完成",
                            failed: "下载失败",
                          }[task.status] ||
                          task.status}{" "}
                        · {bytes(task.received)}
                        {task.total > 0 && " / " + bytes(task.total)}
                      </span>
                    </div>
                    <div className="download-actions">
                      {task.status === "completed" ? (
                        <>
                          <IconButton
                            title="播放下载的音频"
                            onClick={() =>
                              play(
                                { ...task.track, localPath: task.path },
                                downloads
                                  .filter((d) => d.status === "completed")
                                  .map((d) => ({
                                    ...d.track,
                                    localPath: d.path,
                                  })),
                              )
                            }
                          >
                            <Play size={18} />
                          </IconButton>
                          <IconButton
                            title="打开文件位置"
                            onClick={() =>
                              void api("download/action", {
                                id: task.id,
                                action: "reveal",
                              }).catch((err) => notify(err.message, true))
                            }
                          >
                            <FolderOpen size={18} />
                          </IconButton>
                        </>
                      ) : (
                        <IconButton
                          title={
                            ["downloading", "queued"].includes(task.status)
                              ? "暂停下载"
                              : "继续下载"
                          }
                          onClick={() =>
                            void api("download/action", {
                              id: task.id,
                              action: ["downloading", "queued"].includes(
                                task.status,
                              )
                                ? "pause"
                                : "resume",
                            }).catch((err) => notify(err.message, true))
                          }
                        >
                          {["downloading", "queued"].includes(task.status) ? (
                            <Pause size={18} />
                          ) : (
                            <Play size={18} />
                          )}
                        </IconButton>
                      )}
                      <IconButton
                        title="移除下载记录，保留文件"
                        onClick={() =>
                          void api("download/action", {
                            id: task.id,
                            action: "remove",
                          }).catch((err) => notify(err.message, true))
                        }
                      >
                        <X size={18} />
                      </IconButton>
                    </div>
                  </div>
                ))
              ) : (
                <Empty
                  title="还没有下载任务"
                  detail="在歌曲上右键，选择下载音频即可保存到本机"
                  icon={<DownloadIcon size={30} />}
                />
              )}
            </div>
          </div>
        )}
        {page === "settings" && (
          <SettingsPage
            bootstrap={boot}
            onUpdate={applyBoot}
            notify={notify}
            onLogin={() => setLogin(true)}
            onLogout={async () => {
              try {
                await api("logout", {});
                setFolders([]);
                setSelection({ kind: "recent" });
                await refresh();
                notify("已退出登录");
              } catch (err) {
                notify((err as Error).message, true);
              }
            }}
          />
        )}
        {queueOpen && (
          <aside className="queue-panel">
            <header>
              <div>
                <h2>播放队列</h2>
                <small>{queue.length} 首</small>
              </div>
              <IconButton
                title="清空播放队列"
                onClick={() => void queueAction("clear")}
              >
                <Trash2 size={17} />
              </IconButton>
              <IconButton
                title="关闭播放队列"
                onClick={() => setQueueOpen(false)}
              >
                <X size={20} />
              </IconButton>
            </header>
            <div className="queue-scroll">
              {queue.map((track, i) => (
                <div
                  className={
                    "queue-row " +
                    (boot.player.track?.id === track.id ? "current" : "")
                  }
                  key={`${track.id}-${i}`}
                >
                  <button onClick={() => play(track, queue)}>
                    <Cover track={track} size={38} />
                    <span>
                      <strong>{track.title}</strong>
                      <small>{track.artist}</small>
                    </span>
                  </button>
                  <IconButton
                    title="从队列移除"
                    onClick={() =>
                      void queueAction("remove", undefined, track.id)
                    }
                  >
                    <X size={14} />
                  </IconButton>
                </div>
              ))}
              {!queue.length && (
                <Empty title="队列还是空的" detail="播放歌曲或右键添加" />
              )}
            </div>
          </aside>
        )}
      </main>
      <PlayerBar
        player={boot.player}
        command={command}
        onFavorite={() => {
          if (boot.player.track) chooseFavorite(boot.player.track);
        }}
        onQueue={() => setQueueOpen((v) => !v)}
        onImmersive={() => {
          if (boot.player.track) setImmersive(true);
        }}
      />
      {!boot.engineReady && (
        <div className="engine-warning">
          <AlertCircle size={17} />
          <span>音频引擎未就绪：{boot.engineError}</span>
        </div>
      )}
      {notice && (
        <div
          role={notice.error ? "alert" : "status"}
          className={"toast " + (notice.error ? "error-toast" : "")}
        >
          {notice.error ? <AlertCircle size={18} /> : <Check size={18} />}
          <span>{notice.text}</span>
          <IconButton title="关闭提示" onClick={() => setNotice(null)}>
            <X size={15} />
          </IconButton>
        </div>
      )}
      {immersive && (
        <Immersive
          player={boot.player}
          command={command}
          onClose={() => setImmersive(false)}
          onFavorite={() => {
            if (boot.player.track) chooseFavorite(boot.player.track);
          }}
        />
      )}
      {login && (
        <LoginDialog onClose={() => setLogin(false)} onLogin={onLogin} />
      )}
      {picker && (
        <FolderPicker
          {...picker}
          folders={folders}
          onClose={() => setPicker(null)}
          onCreate={() => setCreate(true)}
          onDone={() => {
            const moved = !!picker.from;
            setPicker(null);
            void refreshFolders();
            setReload((v) => v + 1);
            notify(moved ? "已移动到目标收藏夹" : "已添加到收藏夹");
          }}
        />
      )}
      {create && (
        <CreateFolderDialog
          onClose={() => setCreate(false)}
          onDone={() => {
            setCreate(false);
            void refreshFolders();
            notify("收藏夹已创建");
          }}
        />
      )}
      {menu && (
        <ContextMenu x={menu.x} y={menu.y} onClose={() => setMenu(null)}>
          <MenuItem
            icon={<Play size={16} />}
            onClick={() => {
              play(
                menu.track,
                menu.queue ??
                  (page === "search"
                    ? searchTracks
                    : tracks.length
                      ? tracks
                      : [menu.track]),
              );
              setMenu(null);
            }}
          >
            播放
          </MenuItem>
          <MenuItem
            icon={<ListMusic size={16} />}
            onClick={() => {
              void queueAction("next", menu.track);
              setMenu(null);
            }}
          >
            下一首播放
          </MenuItem>
          <MenuItem
            icon={<Plus size={16} />}
            onClick={() => {
              void queueAction("append", menu.track);
              setMenu(null);
            }}
          >
            加入播放队列
          </MenuItem>
          <hr />
          <MenuItem
            icon={<Heart size={16} />}
            onClick={() => {
              chooseFavorite(menu.track);
              setMenu(null);
            }}
          >
            添加到收藏夹
          </MenuItem>
          {!!menu.from && (
            <>
              <MenuItem
                icon={<FolderInput size={16} />}
                onClick={() => {
                  chooseFavorite(menu.track, menu.from);
                  setMenu(null);
                }}
              >
                移动到其他收藏夹
              </MenuItem>
              <MenuItem
                danger
                icon={<Trash2 size={16} />}
                onClick={() => {
                  void removeFavorite(menu.track, menu.from!);
                  setMenu(null);
                }}
              >
                从当前收藏夹删除
              </MenuItem>
            </>
          )}
          <hr />
          <MenuItem
            icon={<DownloadIcon size={16} />}
            onClick={() => {
              void addDownload(menu.track);
              setMenu(null);
            }}
          >
            下载音频
          </MenuItem>
          {menu.track.bvid && (
            <MenuItem
              icon={<ArrowUpRight size={16} />}
              onClick={() => {
                void api("open", {
                  url: "https://www.bilibili.com/video/" + menu.track.bvid,
                });
                setMenu(null);
              }}
            >
              在 Bilibili 打开
            </MenuItem>
          )}
        </ContextMenu>
      )}
      {folderMenu && (
        <ContextMenu
          x={folderMenu.x}
          y={folderMenu.y}
          onClose={() => setFolderMenu(null)}
        >
          <MenuItem
            icon={<ListMusic size={16} />}
            onClick={() => {
              selectFolder({ kind: "folder", id: folderMenu.folder.id });
              setFolderMenu(null);
            }}
          >
            打开收藏夹
          </MenuItem>
          <MenuItem
            icon={<ArrowUpRight size={16} />}
            onClick={() => {
              void api("open", {
                url: `https://www.bilibili.com/medialist/detail/ml${folderMenu.folder.id}`,
              });
              setFolderMenu(null);
            }}
          >
            在 Bilibili 管理
          </MenuItem>
        </ContextMenu>
      )}
    </div>
  );
}
