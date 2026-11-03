# Third-party software

BiliTune Desktop dynamically loads the unmodified `libmpv-2.dll`. It can be replaced by a compatible x64 libmpv client API v2 build. No license restrictions are added to replacing or debugging this library.

- Wails **3.0.0**: MIT, https://github.com/wailsapp/wails/tree/v3.0.0
- Go and golang.org/x libraries: BSD-style, https://go.dev/LICENSE
- React / React DOM **18.3.1**: MIT, https://github.com/facebook/react/tree/v18.3.1
- Lucide icons: ISC, https://github.com/lucide-icons/lucide
- node-qrcode: MIT, https://github.com/soldair/node-qrcode
- libmpv Windows audio build **2023-09-24**, mpv commit **652a1dd**: https://github.com/media-kit/libmpv-win32-audio-build/releases/tag/2023-09-24
- mpv source and copyright: https://github.com/mpv-player/mpv/tree/652a1dd
- Windows build recipes and dependency source retrieval: https://github.com/media-kit/libmpv-win32-audio-build

The bundled audio DLL is the same pinned runtime used by the previous desktop release. Its FFmpeg configuration contains `--disable-gpl --disable-nonfree --enable-version3`; the LGPL license texts, mpv copyright notice and notices for the bundled frontend/Go dependencies are included in `licenses/`. Consult the upstream build recipes for linked codec/network library sources and their individual licenses.

Runtime download and extracted DLL SHA-256 hashes are pinned and verified by `scripts/fetch-mpv.ps1`. BiliTune sources are available at https://github.com/xxay-16/BiliTune .
