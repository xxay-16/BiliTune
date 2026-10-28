# BiliTune

<div align="center">
  <img src="assest/logo.png" alt="BiliTune Logo" width="160" />
</div>

### 这是什么？
```
模仿Youtube music的思路、Spotify的前端制作的哔哩哔哩音乐播放器
```
### 有什么特点？
```
- 支持Windows、Android双端（或许还会支持更多？）
- 把B站账号的收藏夹当作歌单，在app内可以创建新的收藏夹，添加视频进入收藏
- 从视频字幕或网易yun等平台获取歌词
- flutter制作，多端支持优异，内存占用小
- 好看的ui/ux，轻松易懂的使用方式
- 尊重Bilibili bot协议，严格限制api调用频率，避免为上游增加压力以及触发风控
```

## 截图
<table>
  <tr>
    <td align="center"><img src="assest/桌面1.png" alt="Windows 截图 1" /></td>
    <td align="center"><img src="assest/桌面2.png" alt="Windows 截图 2" /></td>
    <td align="center"><img src="assest/桌面3.png" alt="Windows 截图 3" /></td>
  </tr>
  <tr>
    <td align="center"><img src="assest/安卓1.png" alt="Android 截图 1" /></td>
    <td align="center"><img src="assest/安卓2.png" alt="Android 截图 2" /></td>
    <td align="center"><img src="assest/安卓3.png" alt="Android 截图 3" /></td>
  </tr>
</table>

- 在静态网页 [预览](https://wep-56.github.io/BiliTune/) 桌面端与移动端

## Getting Started
### 非开发者
前往 [Release](https://github.com/WEP-56/BiliTune/releases) 下载对应设备的安装包。

- Windows：下载 `BiliTune-*-setup.exe`，按安装向导安装。
- Android：下载 `BiliTune-*-release.apk`，允许从浏览器或文件管理器安装应用后直接安装。
- 首次使用：进入设置页登录 B 站账号，应用会同步收藏夹作为歌单使用；未登录时也可以使用搜索、最近播放和本地下载相关功能。

### 开发者
本项目基于 Flutter，当前主要目标平台为 Windows 和 Android。

```bash
git clone https://github.com/WEP-56/BiliTune.git
cd BiliTune
flutter pub get
flutter run -d windows
```

Android 调试请先连接设备或启动模拟器：

```bash
flutter devices
flutter run -d <device-id>
```

常用检查与构建命令：

```bash
flutter analyze
flutter test
flutter build windows --release
flutter build apk --release
```

Release 包由 GitHub Actions 在推送匹配 `pubspec.yaml` 版本的 tag 时自动构建，例如 `version: 0.0.8+2` 可使用 `v0.0.8` 或 `v0.0.8+2`。

### Windows 自动打包

推送到 `main` 或 `master` 后，会自动运行独立的 [Windows Build](https://github.com/xxay-16/BiliTune/actions/workflows/windows-build.yml) 工作流。也可以在该页面点击 **Run workflow** 手动打包，不需要 Android 签名密钥。

构建会使用 Flutter 3.41.9 和锁定的依赖，先运行静态检查和测试，再生成 Windows x64 安装版 `.exe`、免安装版 `.zip` 和 `SHA256SUMS.txt`。打开成功的构建记录，在摘要或 **Artifacts** 区域下载文件，产物保留 30 天；文件名包含版本号和提交短哈希。

免安装版请完整解压后运行 `bilitune.exe`。本地已有 Release 构建和 Inno Setup 时，也可运行以下命令生成相同格式的包（`-Version` 可自行指定）：

```powershell
./scripts/package-windows.ps1 -Version 0.1.4-5
```

只生成免安装版时，追加 `-SkipInstaller`。

## 致谢
参考了以下优秀项目

api来自：[Nemo2011/bilibili-api](https://github.com/Nemo2011/bilibili-api)
