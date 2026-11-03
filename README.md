# BiliTune Desktop

<p align="center">
  <img src="frontend/public/logo.png" alt="BiliTune" width="100" />
</p>

[![Windows Build](https://github.com/xxay-16/BiliTune/actions/workflows/windows-build.yml/badge.svg)](https://github.com/xxay-16/BiliTune/actions/workflows/windows-build.yml)

基于 **Go + Wails 3.0.0** 的哔哩哔哩桌面音乐播放器。当前提供 Windows 10/11 x64 版，不再包含 Flutter、Android 或手机版构建。

Go 负责 Bilibili 接口、播放与下载调度、本地数据和 Windows 系统集成；React + TypeScript 构建桌面界面，音频由 libmpv 播放。

## 0.2.2 更新

- 点击搜索结果、歌单、播放栏、推荐卡片或队列中的 UP 主名字，进入播放器内的 UP 主主页。
- 将 UP 主公开的 Bilibili 合集、系列统一展示为专辑，支持完整歌曲列表、播放全部、专辑内搜索和右键管理。
- 旧历史和下载记录缺少 UP 主 UID 时，查询补齐后保存；前进 / 后退会保留 UP 主及专辑的页面身份。
- Windows 自动打包同时提供免安装 ZIP 和安装版 EXE，上传前核对 ZIP 内容、校验值及程序对应的 Git 提交。

## 界面预览

![BiliTune 浅色桌面界面](assest/desktop-home.png)

预览图使用示例音乐和账号。实际歌单、封面和昵称来自 Bilibili 与你的登录账号。

新版采用暖灰侧栏、浅色推荐首页、封面叠层横幅和底部播放栏；支持前进后退、侧栏搜索与深浅主题切换。

## 下载与安装

在 [Windows Build](https://github.com/xxay-16/BiliTune/actions/workflows/windows-build.yml) 选择最近一次成功的构建，在页面底部的 **Artifacts** 下载产物。下载 Actions 产物需要登录 GitHub。

| 文件 | 用途 |
| --- | --- |
| `BiliTune-Setup-<版本>-<提交>.exe` | Windows 安装包，按向导安装 |
| `BiliTune-<版本>-<提交>-windows-x64-portable.zip` | 便携版，完整解压后运行 `bilitune.exe` |
| `SHA256SUMS.txt` | 核对安装包和便携包的 SHA-256 校验值 |

文件名包含版本号和 Git 提交的前 7 位，便于确认正在试用的构建。正式标签版本发布后，也可在 [Releases](https://github.com/xxay-16/BiliTune/releases) 下载。

运行需要 **Windows 10/11 x64** 和 **Microsoft Edge WebView2 Runtime**。便携版请保留 `bilitune.exe` 同目录的 `libmpv-2.dll`。

## 开始使用

1. 启动 BiliTune，通过右上角或设置页登录 Bilibili，支持二维码和 Cookie 登录。
2. 打开“我的歌单”，选择收藏夹。应用会读取全部分页，收藏夹列表与歌曲列表可以分别滚动。
3. 单击歌曲播放；右键可添加到收藏夹、移动到其他收藏夹、从当前收藏夹删除、下载音轨或加入播放队列。
4. 在设置页录制全局组合快捷键。键盘媒体键可控制播放、暂停、停止、上一曲和下一曲。
5. 隐藏到托盘后可继续播放；右键托盘图标打开固定窄宽度菜单，长歌曲名会自动滚动。
6. 点击歌曲下方的 UP 主名字，在主页打开专辑；使用“播放全部”将整张专辑加入本次播放队列。

## 功能

- 二维码或 Cookie 登录；凭据仅保存在本机，并使用 Windows DPAPI 加密。
- 收藏夹作为歌单，完整读取全部分页；支持搜索歌单、创建收藏夹、歌曲右键添加、移动和删除。
- 收藏夹列表与歌曲列表独立滚动；大列表按可见范围渲染。
- 点击歌曲下方的 UP 主名字，打开站内主页；可查看专辑、全部作品，或前往 B 站主页。
- UP 主的 B 站合集和系列统一作为专辑展示，完整加载专辑歌曲，支持播放全部、搜索与右键管理。
- 发现音乐、视频搜索与 BV / AV / AU 号直达。
- Go 管理音频播放、队列、进度、循环、随机、倍速、音量和本地文件。
- 站内歌词 / 字幕与 LRCLIB 歌词，沉浸播放界面。
- 音频下载、并发限制、暂停续传、播放下载内容、打开文件位置。
- Windows 系统媒体键、自定义全局快捷键、冲突提示；上一曲始终直接切换。
- 托盘后台播放，180 DIP 固定宽度菜单，长标题自动横向滚动。
- 默认浅色外观，保留深色主题；支持开机启动、关闭窗口行为和版本检查。

## UP 主与专辑

在搜索结果、歌单、推荐卡片、播放栏和队列中点击 UP 主名字，即可打开播放器内的 UP 主主页。主页的“专辑”展示该 UP 主公开的 Bilibili 合集与系列；打开后可播放全部歌曲、搜索专辑内容，或把歌曲加入队列、收藏夹和下载任务。

使用顶部前进 / 后退按钮可在搜索、UP 主和专辑页面之间返回。“B 站主页”会在浏览器打开对应 UP 主的真实空间地址。

新加载的歌曲保存 UP 主 UID；旧历史和下载记录缺少 UID 时，会通过视频 / 音频信息补齐并保存。如果 Bilibili 拒绝这次查询，会提示登录后重试。专辑列表和歌曲列表均读取全部分页，不以单页 20 / 30 条为上限。

## 从 Flutter 版升级

先从系统托盘退出旧版，再启动新版本，以免两个播放器同时响应媒体键。

0.2.1 首次启动会切换到新版浅色外观；之后在设置里选择的主题会跨重启保留。

首次启动会自动导入旧版登录信息、设置、播放位置、队列、历史、下载任务和隐藏收藏夹设置。仅在新数据目录还没有 `state.json` 时执行导入，旧文件保持原样。

| 数据 | 默认位置 |
| --- | --- |
| 旧版设置 | `%APPDATA%\com.wep56\bilitune\shared_preferences.json` |
| 新版设置、队列和历史 | `%APPDATA%\com.wep56\bilitune\desktop\state.json` |
| 新版登录凭据 | `%APPDATA%\com.wep56\bilitune\desktop\session.dat` |
| 默认音频下载目录 | `%USERPROFILE%\Downloads\BiliTune` |

登录凭据由 Windows DPAPI 加密保存在本机。未完成的下载任务在重启后以暂停状态恢复，可手动继续；移除下载记录会保留音频文件。

## 常见问题

**Bilibili 请求失败 / HTTP 412**

检查 Bilibili 登录状态，稍后刷新重试。线上接口受账号权限、网络和 Bilibili 风控影响；未登录的在线查询在开发验证中出现过 HTTP 412，登录后的同步与收藏夹修改还需在实际账号环境中验证。

**快捷键无响应或冲突**

先退出托盘中残留的旧版 BiliTune，再检查设置页的快捷键冲突提示。组合键被其他程序占用时，可重新录制。部分键盘的媒体键需要配合 Fn 使用。

**上一曲和停止播放的行为**

无论已经播放多久，上一曲都会直接切换到队列中的上一首。停止播放会回到曲首并保留队列。

**音频区收藏管理**

当前收藏夹管理支持视频音轨。音频区 AU 条目支持直达、播放与下载，其收藏管理需要在 Bilibili 中操作。

## 构建 Windows

与 CI 对齐使用以下依赖：

| 依赖 | 版本 / 用途 |
| --- | --- |
| Go | **1.27.3**，编译后端和桌面程序 |
| Wails | **3.0.0**，构建脚本自动安装对应 CLI |
| Node.js | **24**，构建前端 |
| 7-Zip | 解压音频运行库 |
| Inno Setup | **6 或 7**，生成 EXE 安装包时使用 |

Go 桌面程序使用 `CGO_ENABLED=0` 构建；脚本会下载固定版本的 libmpv，并核对压缩包和 DLL 的 SHA-256。

```powershell
git clone https://github.com/xxay-16/BiliTune.git
cd BiliTune

./scripts/build-windows.ps1
./bin/bilitune.exe
./scripts/package-windows.ps1
```

程序输出到 `bin`，分发产物位于 `dist/windows`，包括 EXE 安装包、便携 ZIP 和校验文件。仅生成便携版可运行：

```powershell
./scripts/package-windows.ps1 -SkipInstaller
```

打包脚本会保留已有产物；重复打包可通过 `-Version` 指定新的文件名版本，或用 `-OutputDir` 指定其他输出目录。

打包后可检查压缩包内容与原始构建文件是否一致：

```powershell
./scripts/verify-windows-package.ps1 -Version 0.2.2
# 仅生成免安装包时，校验也加上同名开关：
./scripts/verify-windows-package.ps1 -Version 0.2.2 -SkipInstaller
```

如果打包时指定了其他 `-Version` / `-OutputDir`，校验时使用相同的 `-Version`，并通过 `-PackageDir` 指向对应目录。

## 自动构建与发布

[Windows Build 工作流](.github/workflows/windows-build.yml) 会在推送 `main` / `master`、提交 Pull Request 或手动触发时执行测试、编译和 Windows 打包，产物保留 **30 天**。

每次构建会检查：

1. Go、前端、依赖锁文件与 Windows 资源中的应用版本一致。
2. Go / 前端代码格式、后端回归测试、前端测试与类型检查构建。UP 主 UID 关联、旧记录补齐及专辑完整分页测试包含在后端测试中。
3. 随包音频引擎的真实解码、暂停、进度跳转与错误反馈。
4. 安装包和 ZIP 的 SHA-256 与校验文件一致，ZIP 中的程序、音频 DLL 完整且与本次构建逐项比对通过，程序对应本次干净的 Git 提交。

上述检查通过后才上传产物。构建结果摘要优先提供免安装 ZIP 下载入口，同时列出安装包、校验文件、提交号、ZIP 大小和文件数量；无需把开发环境或自己的 Bilibili 登录信息上传到 GitHub。

推送与 `internal/model/model.go` 中版本号一致的标签（例如 `v0.2.2`）时，工作流在构建成功后发布 GitHub Release，附带安装包、便携包和校验文件。日常分支推送的安装包从 Actions 下载。

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

测试覆盖 UP 主身份关联、旧记录 UID 补齐、完整专辑 / 合集分页、完整歌单分页、WBI 签名、原子移动、迁移、登录隔离、续传、上一曲、加载取消、加载中暂停、续播、真实解码和 Windows SMTC / 快捷键注册。二维码登录、账号权限和线上收藏夹修改仍依赖 Bilibili 服务及登录状态。

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
