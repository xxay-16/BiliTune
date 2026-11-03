import { useCallback, useEffect, useState } from "react";
import QRCode from "qrcode";
import {
  Check,
  FolderPlus,
  Lock,
  LoaderCircle,
  RefreshCw,
  ShieldCheck,
  Smartphone,
  Keyboard,
} from "lucide-react";
import { api, keyToken } from "./api";
import { Cover, Modal, SearchField, IconButton } from "./components";
import type {
  Account,
  Folder,
  Hotkey,
  Settings,
  Track,
  Bootstrap,
} from "./types";

export function LoginDialog({
  onClose,
  onLogin,
}: {
  onClose: () => void;
  onLogin: (account: Account) => void;
}) {
  const [tab, setTab] = useState<"qr" | "cookie">("qr");
  const [url, setURL] = useState("");
  const [key, setKey] = useState("");
  const [status, setStatus] = useState("正在生成二维码…");
  const [expired, setExpired] = useState(false);
  const [error, setError] = useState("");
  const [cookie, setCookie] = useState("");
  const [busy, setBusy] = useState(false);
  const [nonce, setNonce] = useState(0);
  useEffect(() => {
    if (tab !== "qr") return;
    let canceled = false;
    setError("");
    setExpired(false);
    setURL("");
    setStatus("正在生成二维码…");
    api<{ url: string; qrcode_key: string }>("qr")
      .then(async (data) => {
        const image = await QRCode.toDataURL(data.url, {
          width: 240,
          margin: 2,
          errorCorrectionLevel: "M",
        });
        if (!canceled) {
          setURL(image);
          setKey(data.qrcode_key);
          setStatus("使用哔哩哔哩 App 扫码登录");
        }
      })
      .catch((err) => {
        if (!canceled) setError(err.message);
      });
    return () => {
      canceled = true;
    };
  }, [tab, nonce]);
  useEffect(() => {
    if (!key || tab !== "qr" || expired) return;
    let canceled = false,
      polling = false;
    const timer = setInterval(async () => {
      if (polling) return;
      polling = true;
      try {
        const result = await api<{ code: number; account?: Account }>(
          "qr/poll?key=" + encodeURIComponent(key),
        );
        if (canceled) return;
        if (result.code === 0 && result.account) {
          clearInterval(timer);
          onLogin(result.account);
        } else if (result.code === 86038) {
          setExpired(true);
          setStatus("二维码已过期");
        } else if (result.code === 86090 || result.code === 86102) {
          setStatus("已扫码，请在手机上确认");
        } else setStatus("使用哔哩哔哩 App 扫码登录");
      } catch (err) {
        if (!canceled) setError((err as Error).message);
      } finally {
        polling = false;
      }
    }, 2000);
    return () => {
      canceled = true;
      clearInterval(timer);
    };
  }, [key, tab, expired, onLogin]);
  async function submit() {
    setBusy(true);
    setError("");
    try {
      const account = await api<Account>("login", { cookie });
      setCookie("");
      onLogin(account);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal title="登录 Bilibili" onClose={onClose}>
      <p className="muted">同步你的收藏夹，在 BiliTune 里继续聆听。</p>
      <div className="segmented">
        <button
          className={tab === "qr" ? "selected" : ""}
          onClick={() => setTab("qr")}
        >
          <Smartphone size={16} />
          扫码登录
        </button>
        <button
          className={tab === "cookie" ? "selected" : ""}
          onClick={() => setTab("cookie")}
        >
          <Keyboard size={16} />
          Cookie 登录
        </button>
      </div>
      {tab === "qr" ? (
        <>
          <div className="qr-box">
            {url ? (
              <img src={url} alt="Bilibili 登录二维码" />
            ) : (
              <LoaderCircle size={30} className="spin" />
            )}
            {expired && (
              <button
                className="qr-refresh"
                onClick={() => {
                  setKey("");
                  setNonce((n) => n + 1);
                }}
              >
                <RefreshCw />
                刷新二维码
              </button>
            )}
          </div>
          <p className="center muted">{status}</p>
        </>
      ) : (
        <>
          <label className="field">
            完整 Cookie
            <textarea
              autoComplete="off"
              spellCheck={false}
              value={cookie}
              onChange={(e) => setCookie(e.target.value)}
              placeholder="SESSDATA=…; bili_jct=…; DedeUserID=…"
              rows={4}
            />
          </label>
          <p className="small muted">
            需要包含 SESSDATA。收藏夹管理还需要 bili_jct。
          </p>
          <button
            className="button primary full"
            disabled={busy || !cookie.trim()}
            onClick={submit}
          >
            {busy ? (
              <LoaderCircle className="spin" size={18} />
            ) : (
              <Check size={17} />
            )}
            登录
          </button>
        </>
      )}
      {error && <p className="error">{error}</p>}
      <p className="privacy-note">
        <ShieldCheck size={15} />
        登录凭据仅保存在本机，Windows 使用系统加密保护。
      </p>
    </Modal>
  );
}

export function FolderPicker({
  track,
  from,
  folders,
  onClose,
  onDone,
  onCreate,
}: {
  track: Track;
  from?: number;
  folders: Folder[];
  onClose: () => void;
  onDone: () => void;
  onCreate: () => void;
}) {
  const [query, setQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const targets = folders.filter(
    (f) => f.id !== from && f.title.toLowerCase().includes(query.toLowerCase()),
  );
  async function select(folder: Folder) {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await api("favorite", { track, from: from || 0, to: folder.id });
      onDone();
    } catch (err) {
      setError((err as Error).message);
      setBusy(false);
    }
  }
  return (
    <Modal
      title={from ? "移动到其他收藏夹" : "添加到收藏夹"}
      onClose={() => {
        if (!busy) onClose();
      }}
    >
      <div className="picker-track">
        <Cover track={track} size={48} />
        <div>
          <strong>{track.title}</strong>
          <small>{track.artist}</small>
        </div>
      </div>
      <SearchField value={query} onChange={setQuery} placeholder="搜索收藏夹" />
      <div className="folder-picker">
        {targets.map((folder) => (
          <button
            disabled={busy}
            key={folder.id}
            onClick={() => select(folder)}
          >
            <Cover folder={folder} size={38} />
            <span>
              <strong>{folder.title}</strong>
              <small>{folder.count} 首</small>
            </span>
            {folder.private && <Lock size={15} />}
          </button>
        ))}
        {targets.length === 0 && (
          <p className="muted center">没有其他可用收藏夹</p>
        )}
      </div>
      {error && <p className="error">{error}</p>}
      {busy && (
        <p className="loading">
          <LoaderCircle className="spin" size={17} />
          正在更新收藏夹…
        </p>
      )}
      <button
        className="button outline full"
        disabled={busy}
        onClick={onCreate}
      >
        <FolderPlus size={17} />
        新建收藏夹
      </button>
    </Modal>
  );
}

export function CreateFolderDialog({
  onClose,
  onDone,
}: {
  onClose: () => void;
  onDone: () => void;
}) {
  const [title, setTitle] = useState("");
  const [intro, setIntro] = useState("");
  const [privateFolder, setPrivate] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit() {
    if (!title.trim() || busy) return;
    setBusy(true);
    try {
      await api("folder", { title, intro, private: privateFolder });
      onDone();
    } catch (err) {
      setError((err as Error).message);
      setBusy(false);
    }
  }
  return (
    <Modal
      title="新建收藏夹"
      onClose={() => {
        if (!busy) onClose();
      }}
    >
      <label className="field">
        名称
        <input
          autoFocus
          maxLength={80}
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="给这份收藏取个名字"
        />
      </label>
      <label className="field">
        简介
        <textarea
          rows={3}
          maxLength={300}
          value={intro}
          onChange={(e) => setIntro(e.target.value)}
          placeholder="可选"
        />
      </label>
      <label className="checkbox">
        <input
          type="checkbox"
          checked={privateFolder}
          onChange={(e) => setPrivate(e.target.checked)}
        />
        设为私密收藏夹
      </label>
      {error && <p className="error">{error}</p>}
      <button
        className="button primary full"
        disabled={busy || !title.trim()}
        onClick={submit}
      >
        {busy ? (
          <LoaderCircle className="spin" size={17} />
        ) : (
          <FolderPlus size={17} />
        )}
        创建收藏夹
      </button>
    </Modal>
  );
}

export const hotkeyNames: Record<string, string> = {
  playPause: "播放 / 暂停",
  previousTrack: "上一曲",
  nextTrack: "下一曲",
  stop: "停止播放",
  toggleWindow: "显示 / 隐藏窗口",
};
export function hotkeyLabel(binding?: Hotkey) {
  if (!binding) return "未设置自定义组合键";
  const names: Record<string, string> = {
    control: "Ctrl",
    alt: "Alt",
    shift: "Shift",
    win: "Win",
    Space: "Space",
    ArrowLeft: "←",
    ArrowRight: "→",
    ArrowUp: "↑",
    ArrowDown: "↓",
    MediaPlayPause: "媒体播放/暂停",
    MediaNextTrack: "媒体下一曲",
    MediaPreviousTrack: "媒体上一曲",
    MediaStop: "媒体停止",
  };
  return [
    ...binding.modifiers.map((m) => names[m] || m),
    names[binding.key] || binding.key,
  ].join(" + ");
}
export function HotkeyDialog({
  action,
  settings,
  onClose,
  onSaved,
}: {
  action: string;
  settings: Settings;
  onClose: () => void;
  onSaved: (bootstrap: Bootstrap) => void;
}) {
  const [binding, setBinding] = useState<Hotkey | null>(null);
  const [error, setError] = useState("");
  const [ready, setReady] = useState(false);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    let disposed = false;
    api("hotkeys/record", { recording: true })
      .then(() => {
        if (!disposed) setReady(true);
      })
      .catch((err) => {
        if (!disposed) setError(err.message);
      });
    return () => {
      disposed = true;
      void api("hotkeys/record", { recording: false }).catch(() => {});
    };
  }, []);
  useEffect(() => {
    if (!ready) return;
    const listener = (event: KeyboardEvent) => {
      event.preventDefault();
      event.stopImmediatePropagation();
      const key = keyToken(event);
      if (!key) return;
      setBinding({
        action,
        key,
        modifiers: [
          ...(event.ctrlKey ? ["control"] : []),
          ...(event.altKey ? ["alt"] : []),
          ...(event.shiftKey ? ["shift"] : []),
          ...(event.metaKey ? ["win"] : []),
        ],
      });
    };
    const blur = () => onClose();
    window.addEventListener("keydown", listener, true);
    window.addEventListener("blur", blur);
    return () => {
      window.removeEventListener("keydown", listener, true);
      window.removeEventListener("blur", blur);
    };
  }, [ready, action, onClose]);
  async function save() {
    if (!binding) return;
    setBusy(true);
    try {
      const signature = (h: Hotkey) =>
        h.key + "|" + [...h.modifiers].sort().join("|");
      const hotkeys = settings.hotkeys.filter(
        (h) => h.action !== action && signature(h) !== signature(binding),
      );
      await api("settings", { ...settings, hotkeys: [...hotkeys, binding] });
      await api("hotkeys/record", { recording: false });
      onSaved(await api<Bootstrap>("bootstrap"));
      onClose();
    } catch (err) {
      setError((err as Error).message);
      setBusy(false);
    }
  }
  return (
    <Modal title={"录制快捷键 · " + hotkeyNames[action]} onClose={onClose}>
      <p className="muted">
        按下需要的组合键，然后保存。旧快捷键会在录制结束后恢复。
      </p>
      <div className="hotkey-capture">
        {binding ? hotkeyLabel(binding) : ready ? "等待按键…" : "正在准备…"}
      </div>
      {error && <p className="error">{error}</p>}
      <button
        className="button primary full"
        disabled={!binding || busy}
        onClick={save}
      >
        保存组合键
      </button>
    </Modal>
  );
}
