import { useState } from "react";
import {
  ArrowUpRight,
  FolderOpen,
  Keyboard,
  LoaderCircle,
  LogOut,
  RefreshCw,
  Trash2,
  X,
} from "lucide-react";
import { api } from "./api";
import { IconButton } from "./components";
import { HotkeyDialog, hotkeyLabel, hotkeyNames } from "./dialogs";
import type { Bootstrap, Settings as Prefs } from "./types";

export function SettingsPage({
  bootstrap,
  onUpdate,
  notify,
  onLogin,
  onLogout,
}: {
  bootstrap: Bootstrap;
  onUpdate: (data: Bootstrap) => void;
  notify: (text: string, error?: boolean) => void;
  onLogin: () => void;
  onLogout: () => void;
}) {
  const prefs = bootstrap.settings;
  const [busy, setBusy] = useState(false);
  const [recording, setRecording] = useState("");
  async function update(patch: Partial<Prefs>) {
    setBusy(true);
    try {
      await api("settings", { ...prefs, ...patch });
      onUpdate(await api<Bootstrap>("bootstrap"));
    } catch (err) {
      notify((err as Error).message, true);
    } finally {
      setBusy(false);
    }
  }
  async function directory() {
    try {
      const result = await api<{ path: string }>("dialog/directory", {});
      if (result.path) await update({ downloadDir: result.path });
    } catch (err) {
      notify((err as Error).message, true);
    }
  }
  async function checkUpdates() {
    setBusy(true);
    try {
      const result = await api<{
        available: boolean;
        version: string;
        url?: string;
      }>("update");
      if (result.available && result.url) {
        notify("发现新版本 " + result.version);
        await api("open", { url: result.url });
      } else notify("当前已是最新桌面版");
    } catch (err) {
      notify((err as Error).message, true);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="page settings-page">
      <header className="page-heading">
        <div>
          <span className="eyebrow">让音乐更合心意</span>
          <h1>设置</h1>
        </div>
        {busy && <LoaderCircle className="spin" size={20} />}
      </header>
      <div className="settings-scroll">
        <section className="settings-section">
          <h2>外观与播放</h2>
          <Setting title="主题" detail="选择 BiliTune 的界面外观">
            <select
              aria-label="主题"
              value={prefs.theme}
              onChange={(e) => update({ theme: e.target.value })}
            >
              <option value="dark">深色</option>
              <option value="light">浅色</option>
              <option value="system">跟随系统</option>
            </select>
          </Setting>
          <Setting
            title="音质偏好"
            detail="最高可用音质取决于 Bilibili 账号权限与原始音轨"
          >
            <select
              aria-label="音质偏好"
              value={prefs.audioQuality}
              onChange={(e) => update({ audioQuality: e.target.value })}
            >
              {[
                ["auto", "自动 · 最高可用"],
                ["lossless", "优先无损 FLAC"],
                ["high", "高品质"],
                ["medium", "标准"],
                ["low", "节省流量"],
              ].map(([v, t]) => (
                <option key={v} value={v}>
                  {t}
                </option>
              ))}
            </select>
          </Setting>
          <Setting title="播放速度" detail="同时适用于歌曲和长视频音轨">
            <select
              aria-label="播放速度"
              value={prefs.playbackRate}
              onChange={(e) => update({ playbackRate: Number(e.target.value) })}
            >
              {[0.5, 0.75, 1, 1.25, 1.5, 1.75, 2].map((v) => (
                <option value={v} key={v}>
                  {v} 倍
                </option>
              ))}
            </select>
          </Setting>
          <Setting title="歌词来源" detail="站内音频歌词、视频字幕与 LRCLIB">
            <select
              aria-label="歌词来源"
              value={prefs.lyricsSource}
              onChange={(e) => update({ lyricsSource: e.target.value })}
            >
              {[
                ["auto", "自动匹配"],
                ["biliFirst", "优先站内"],
                ["lrclibFirst", "优先 LRCLIB"],
                ["biliOnly", "仅站内歌词 / 字幕"],
                ["lrclibOnly", "仅 LRCLIB"],
              ].map(([v, t]) => (
                <option key={v} value={v}>
                  {t}
                </option>
              ))}
            </select>
          </Setting>
          <Setting title="播放历史数量" detail="超过上限时自动保留最近记录">
            <select
              value={prefs.historyLimit}
              aria-label="播放历史数量"
              onChange={(e) => update({ historyLimit: Number(e.target.value) })}
            >
              {[20, 50, 100, 200, 500].map((v) => (
                <option key={v} value={v}>
                  {v} 首
                </option>
              ))}
            </select>
          </Setting>
        </section>
        <section className="settings-section">
          <h2>本地下载</h2>
          <Setting
            title="下载目录"
            detail={prefs.downloadDir || "默认：下载 / BiliTune"}
          >
            <button className="button outline" onClick={directory}>
              <FolderOpen size={16} />
              选择目录
            </button>
          </Setting>
          {prefs.downloadDir && (
            <button
              className="text-button"
              onClick={() => update({ downloadDir: "" })}
            >
              恢复默认目录
            </button>
          )}
          <Setting
            title="同时下载数量"
            detail="暂停后可继续下载，完成文件保留在本机"
          >
            <select
              value={prefs.downloadConcurrency}
              aria-label="同时下载数量"
              onChange={(e) =>
                update({ downloadConcurrency: Number(e.target.value) })
              }
            >
              {[1, 2, 3, 4].map((v) => (
                <option key={v} value={v}>
                  {v} 个任务
                </option>
              ))}
            </select>
          </Setting>
        </section>
        <section className="settings-section">
          <h2>桌面与快捷键</h2>
          <Setting title="开机自启动" detail="登录 Windows 后在托盘中启动">
            <input
              aria-label="开机自启动"
              type="checkbox"
              role="switch"
              checked={prefs.autoStart}
              onChange={(e) => update({ autoStart: e.target.checked })}
            />
          </Setting>
          {bootstrap.hotkeyErrors.autoStart && (
            <p className="error">{bootstrap.hotkeyErrors.autoStart}</p>
          )}
          <Setting title="关闭主窗口时" detail="托盘状态下音乐和快捷键继续可用">
            <select
              aria-label="关闭主窗口时"
              value={prefs.closeBehavior}
              onChange={(e) => update({ closeBehavior: e.target.value })}
            >
              <option value="tray">隐藏到托盘</option>
              <option value="minimize">最小化</option>
              <option value="exit">退出程序</option>
            </select>
          </Setting>
          <p className="section-note">
            键盘媒体键默认可用。上一曲始终直接切歌；停止播放会回到曲首并保留队列。
          </p>
          {Object.entries(hotkeyNames).map(([action, name]) => (
            <Setting
              key={action}
              title={name}
              detail={
                bootstrap.hotkeyErrors[action] ||
                "全局组合键，窗口隐藏时同样生效"
              }
              error={!!bootstrap.hotkeyErrors[action]}
            >
              <span className="hotkey-label">
                {hotkeyLabel(prefs.hotkeys.find((h) => h.action === action))}
              </span>
              <IconButton
                title={"录制" + name + "快捷键"}
                onClick={() => setRecording(action)}
              >
                <Keyboard size={18} />
              </IconButton>
              <IconButton
                title={"清除" + name + "自定义快捷键"}
                disabled={!prefs.hotkeys.some((h) => h.action === action)}
                onClick={() =>
                  update({
                    hotkeys: prefs.hotkeys.filter((h) => h.action !== action),
                  })
                }
              >
                <X size={16} />
              </IconButton>
            </Setting>
          ))}
        </section>
        <section className="settings-section">
          <h2>账号与数据</h2>
          <Setting
            title={bootstrap.account?.name || "尚未登录 Bilibili"}
            detail={
              bootstrap.account
                ? "MID " + bootstrap.account.mid
                : "登录后可同步收藏夹"
            }
          >
            {bootstrap.account ? (
              <button className="button outline" onClick={onLogout}>
                <LogOut size={16} />
                退出登录
              </button>
            ) : (
              <button className="button primary" onClick={onLogin}>
                登录账号
              </button>
            )}
          </Setting>
          <Setting title="播放历史" detail="仅清除 BiliTune 的本地记录">
            <button
              className="button outline"
              onClick={async () => {
                try {
                  await api("history/clear", {});
                  onUpdate(await api<Bootstrap>("bootstrap"));
                  notify("播放历史已清空");
                } catch (err) {
                  notify((err as Error).message, true);
                }
              }}
            >
              <Trash2 size={16} />
              清空记录
            </button>
          </Setting>
        </section>
        <section className="settings-section">
          <h2>关于 BiliTune</h2>
          <Setting
            title={"BiliTune Desktop " + bootstrap.version}
            detail="Go + Wails 3 · 桌面音频播放器"
          >
            <button
              className="button outline"
              disabled={busy}
              onClick={checkUpdates}
            >
              <RefreshCw size={16} />
              检查更新
            </button>
          </Setting>
          <button
            className="text-button"
            onClick={() =>
              api("open", { url: "https://github.com/xxay-16/BiliTune" })
            }
          >
            GitHub 项目 <ArrowUpRight size={15} />
          </button>
          {bootstrap.migrated && (
            <p className="section-note">
              旧版的登录信息、设置和播放历史已导入，原始数据文件保持原样。
            </p>
          )}
        </section>
        <p className="settings-bottom">为每一次聆听，留一点自己的空间。</p>
      </div>
      {recording && (
        <HotkeyDialog
          action={recording}
          settings={prefs}
          onClose={() => setRecording("")}
          onSaved={(data) => {
            onUpdate(data);
            notify(
              data.hotkeyErrors[recording] || "快捷键已保存",
              !!data.hotkeyErrors[recording],
            );
          }}
        />
      )}
    </div>
  );
}

function Setting({
  title,
  detail,
  children,
  error = false,
}: {
  title: string;
  detail?: string;
  children: React.ReactNode;
  error?: boolean;
}) {
  return (
    <div className="setting-row">
      <div>
        <strong>{title}</strong>
        {detail && <p className={error ? "error" : ""}>{detail}</p>}
      </div>
      <div className="setting-control">{children}</div>
    </div>
  );
}
