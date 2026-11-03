export interface Track {
  id: string;
  title: string;
  artist: string;
  artistMid?: number;
  artistAvatar?: string;
  durationMs: number;
  type: string;
  coverUrl?: string;
  bvid?: string;
  aid?: number;
  cid?: number;
  audioId?: number;
  sourceUrl?: string;
  localPath?: string;
  playCount?: number;
}
export interface Artist {
  mid: number;
  name: string;
  avatar?: string;
  banner?: string;
  signature?: string;
  followers?: number;
  videoCount?: number;
}
export interface Album {
  id: number;
  type: "season" | "series";
  mid: number;
  title: string;
  coverUrl?: string;
  intro?: string;
  count: number;
  artist?: string;
}
export interface AlbumDetail {
  album: Album;
  tracks: Track[];
}
export interface ArtistVideos {
  tracks: Track[];
  page: number;
  total: number;
  hasMore: boolean;
}
export interface Folder {
  id: number;
  title: string;
  count: number;
  coverUrl?: string;
  intro?: string;
  private: boolean;
}
export interface Account {
  mid: number;
  name: string;
  avatar: string;
  vip: boolean;
}
export interface Hotkey {
  action: string;
  key: string;
  modifiers: string[];
}
export interface Settings {
  theme: string;
  closeBehavior: string;
  audioQuality: string;
  lyricsSource: string;
  playbackRate: number;
  historyLimit: number;
  downloadDir: string;
  downloadConcurrency: number;
  hotkeys: Hotkey[];
  autoStart: boolean;
}
export interface Player {
  track: Track | null;
  playing: boolean;
  position: number;
  duration: number;
  buffered: number;
  buffering: boolean;
  volume: number;
  repeat: string;
  shuffle: boolean;
  quality: string;
  error?: string;
  queueVersion: number;
  queueCount: number;
}
export interface Download {
  id: string;
  track: Track;
  status: string;
  received: number;
  total: number;
  path?: string;
  error?: string;
}
export interface Bootstrap {
  version: string;
  settings: Settings;
  account: Account | null;
  player: Player;
  history: Track[];
  searchHistory: string[];
  migrated: boolean;
  engineReady: boolean;
  engineError?: string;
  hotkeyErrors: Record<string, string>;
}
export interface Lyric {
  time: number;
  text: string;
}
export interface Shelf {
  title: string;
  tracks: Track[];
}
