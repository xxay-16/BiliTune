# BiliTune Desktop

基于 **Go + Wails 3.0.0** 的哔哩哔哩桌面音乐播放器。当前提供 Windows 10/11 x64 版，不再包含 Flutter、Android 或手机版构建。

## 使用

在 [GitHub Actions](https://github.com/xxay-16/BiliTune/actions/workflows/windows-build.yml) 下载最新构建的 Windows 安装包或便携 ZIP。正式标签版本在 [Releases](https://github.com/xxay-16/BiliTune/releases) 发布。需要 Microsoft Edge WebView2 Runtime；多数 Windows 10/11 已自带。

- 二维码或 Cookie 登录；凭据仅保存在本机，并使用 Windows DPAPI 加密。
- 收藏夹作为歌单，完整读取全部分页；支持搜索歌单、创建收藏夹、歌曲右键添加、移动和删除。
- 收藏夹列表与歌曲列表独立滚动；大列表按可见范围渲染。
- 发现音乐、视频搜索与 BV / AV / AU 号直达。
- Go 管理音频播放、队列、进度、循环、随机、倍速、音量和本地文件。
- 站内歌词 / 字幕与 LRCLIB 歌词，沉浸播放界面。
- 音频下载、并发限制、暂停续传、播放下载内容、打开文件位置。
- Windows 系统媒体键、自定义全局快捷键、冲突提示；上一曲始终直接切换。
- 托盘后台播放，180 DIP 固定宽度菜单，长标题自动横向滚动。
- 深浅主题、开机启动、关闭窗口行为、版本检查。

升级前请从托盘退出旧版，避免两个播放器同时响应媒体键。首次启动自动读取旧版 `%APPDATA%/com.wep56/bilitune/shared_preferences.json`，导入设置、登录、播放位置、队列、历史和下载任务。原文件保持原样；新数据位于其下的 `desktop` 文件夹。下载文件不会因删除记录而删除。

## 构建 Windows

依赖：Go **1.27.3**、Node.js **24**、7-Zip；生成安装包还需 Inno Setup 6/7。Go 代码不依赖 CGo；音频 DLL 按固定 SHA-256 下载校验。

```powershell
./scripts/build-windows.ps1
./bin/bilitune.exe
./scripts/package-windows.ps1 -Version 0.2.0
```

产物位于 `dist/windows`，包括 EXE 安装包、便携 ZIP 和 SHA-256 文件。仅生成便携版可加 `-SkipInstaller`。每次推送 main 自动执行测试并打包，产物保留 30 天；推送与版本号一致的 `v0.2.0` 标签会发布 Release。

## 开发与验证

```powershell
npm --prefix frontend ci
npm --prefix frontend run build
# 首次取运行库
./scripts/fetch-mpv.ps1
# 使用临时编译程序时也能从 bin 读取音频 DLL
go run . --data-dir .codex/dev-data

go test ./internal/...
npm --prefix frontend test
```

已安装 Wails CLI 时可用 `wails3 dev -config packaging/config.yml` 监听 Go 文件并重新构建。前端开发可另开 `npm --prefix frontend run dev`，后端使用 `go run . --preview 127.0.0.1:9246 --data-dir .codex/preview-data`；该预览入口仅用于开发，桌面发布版通过 Wails 内置资源服务通信，不监听本地网络端口。

真实音频与 Windows 原生集成测试：

```powershell
$env:BILITUNE_MPV_PATH = (Resolve-Path bin/libmpv-2.dll).Path
$env:BILITUNE_TEST_MPV = '1'
$env:BILITUNE_TEST_NATIVE = '1'
go test -count=1 ./internal/player ./internal/native
# 有 MinGW GCC 时还可运行竞争检测：
$env:CGO_ENABLED = '1'
go test -race ./internal/...
```

测试覆盖完整歌单分页、WBI 签名、原子移动、迁移、登录隔离、续传、上一曲、加载取消、加载中暂停、续播、真实解码和 Windows SMTC / 快捷键注册。二维码登录、账号权限和线上收藏夹修改仍依赖 Bilibili 服务及登录状态。

## 结构

- `main.go`：Wails 桌面窗口、托盘、原生对话框和生命周期。
- `internal/bili`：Bilibili API、WBI、歌单、音源与歌词。
- `internal/app`：播放与下载调度、本地接口。
- `internal/player`：Go 调用 libmpv 音频引擎。
- `internal/native`：Windows SMTC、全局快捷键与自启动。
- `internal/store`：原子保存、DPAPI、旧版数据导入。
- `frontend`：React + TypeScript 桌面界面。
- `packaging`、`scripts`：Windows 构建、校验和与安装包。

Flutter 版本保留在 Git 分支 `backup/flutter-before-wails3`。第三方组件与音频运行库说明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。
