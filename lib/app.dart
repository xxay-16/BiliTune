import 'dart:async';
import 'dart:io' show Platform;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:tray_manager/tray_manager.dart';
import 'package:window_manager/window_manager.dart';

import 'core/router/app_router.dart';
import 'core/theme/app_theme.dart';
import 'core/platform/windows_hotkeys.dart';
import 'core/platform/windows_tray_menu.dart';
import 'shared/widgets/startup_reveal.dart';
import 'state/providers.dart';

/// Root widget: wires the router and the dark/light themes, switching theme
/// mode reactively from [themeModeProvider].
class BiliTuneApp extends ConsumerStatefulWidget {
  const BiliTuneApp({super.key});

  @override
  ConsumerState<BiliTuneApp> createState() => _BiliTuneAppState();
}

class _BiliTuneAppState extends ConsumerState<BiliTuneApp>
    with WindowListener, TrayListener {
  bool _windowsControlsReady = false;
  _TrayMenuSnapshot? _lastTrayMenuSnapshot;
  PlaybackNotifier? _playbackNotifier;

  @override
  void initState() {
    super.initState();
    _playbackNotifier = ref.read(playbackProvider.notifier);
    if (Platform.isWindows) {
      windowManager.addListener(this);
      trayManager.addListener(this);
      WindowsHotkeyBridge.instance.setActionHandler(_handleHotkeyAction);
      unawaited(_initWindowsControls());
    }
  }

  @override
  void dispose() {
    if (Platform.isWindows) {
      final playbackNotifier = _playbackNotifier;
      if (playbackNotifier != null) {
        unawaited(playbackNotifier.saveNow());
      }
      windowManager.removeListener(this);
      trayManager.removeListener(this);
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final themeMode = ref.watch(themeModeProvider);
    ref.listen(playbackProvider, (_, next) {
      if (Platform.isWindows) unawaited(_syncTray(next));
    });
    ref.listen(windowsHotkeysProvider, (_, next) {
      if (Platform.isWindows) unawaited(_syncWindowsHotkeys(next));
    });

    return MaterialApp.router(
      title: 'BiliTune',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light(),
      darkTheme: AppTheme.dark(),
      themeMode: themeMode,
      routerConfig: appRouter,
      builder: (context, child) {
        final content = child ?? const SizedBox.shrink();
        final app = Stack(
          children: [
            content,
            const Positioned.fill(child: StartupReveal()),
          ],
        );
        return Platform.isWindows ? ExcludeSemantics(child: app) : app;
      },
    );
  }

  Future<void> _initWindowsControls() async {
    await windowManager.setPreventClose(true);
    await trayManager.setIcon('assest/logo.ico');
    await trayManager.setToolTip('BiliTune');
    _windowsControlsReady = true;
    await _syncTray(ref.read(playbackProvider));
    await _syncWindowsHotkeys(ref.read(windowsHotkeysProvider));
  }

  Future<void> _syncWindowsHotkeys(List<WindowsHotkeyBinding> bindings) async {
    if (!Platform.isWindows) return;
    await WindowsHotkeyBridge.instance.sync(bindings);
  }

  Future<void> _handleHotkeyAction(WindowsHotkeyAction action) async {
    switch (action) {
      case WindowsHotkeyAction.playPause:
        await ref.read(playbackProvider.notifier).togglePlay();
      case WindowsHotkeyAction.previousTrack:
        ref.read(playbackProvider.notifier).previous();
      case WindowsHotkeyAction.nextTrack:
        ref.read(playbackProvider.notifier).next();
      case WindowsHotkeyAction.toggleWindow:
        await _toggleWindow();
    }
  }

  Future<void> _syncTray(PlaybackState playback) async {
    if (!_windowsControlsReady) return;
    final track = playback.track;
    final title = track?.title ?? 'BiliTune';
    final snapshot = _TrayMenuSnapshot(
      trackId: track?.id,
      title: title,
      artist: track?.artist,
      isPlaying: playback.isPlaying,
    );
    if (snapshot == _lastTrayMenuSnapshot) return;
    _lastTrayMenuSnapshot = snapshot;

    await trayManager.setToolTip(
      track == null ? 'BiliTune' : '$title - ${track.artist}',
    );
    await WindowsTrayMenu.update(title: title, isPlaying: playback.isPlaying);
  }

  Future<void> _showTrayMenu() async {
    final action = await WindowsTrayMenu.show();
    if (!mounted) return;
    switch (action) {
      case WindowsTrayAction.previous:
        await _handleHotkeyAction(WindowsHotkeyAction.previousTrack);
      case WindowsTrayAction.playPause:
        await _handleHotkeyAction(WindowsHotkeyAction.playPause);
      case WindowsTrayAction.next:
        await _handleHotkeyAction(WindowsHotkeyAction.nextTrack);
      case WindowsTrayAction.show:
        await _showWindow();
      case WindowsTrayAction.hide:
        await windowManager.hide();
      case WindowsTrayAction.exit:
        await _exitApp();
      case null:
        break;
    }
  }

  Future<void> _showWindow() async {
    await windowManager.show();
    await windowManager.focus();
  }

  Future<void> _toggleWindow() async {
    if (await windowManager.isVisible()) {
      await windowManager.hide();
      return;
    }
    await _showWindow();
  }

  Future<void> _exitApp() async {
    await ref.read(playbackProvider.notifier).saveNow();
    await windowManager.setPreventClose(false);
    await trayManager.destroy();
    await windowManager.destroy();
  }

  @override
  Future<void> onWindowClose() async {
    final preventClose = await windowManager.isPreventClose();
    if (!preventClose) return;

    await ref.read(playbackProvider.notifier).saveNow();

    switch (ref.read(windowCloseBehaviorProvider)) {
      case WindowCloseBehavior.minimize:
        await windowManager.minimize();
      case WindowCloseBehavior.tray:
        await windowManager.hide();
      case WindowCloseBehavior.exit:
        await _exitApp();
    }
  }

  @override
  void onTrayIconMouseDown() {
    unawaited(_showWindow());
  }

  @override
  void onTrayIconRightMouseDown() {
    unawaited(_showTrayMenu());
  }
}

class _TrayMenuSnapshot {
  const _TrayMenuSnapshot({
    required this.trackId,
    required this.title,
    required this.artist,
    required this.isPlaying,
  });

  final String? trackId;
  final String title;
  final String? artist;
  final bool isPlaying;

  @override
  bool operator ==(Object other) {
    return other is _TrayMenuSnapshot &&
        other.trackId == trackId &&
        other.title == title &&
        other.artist == artist &&
        other.isPlaying == isPlaying;
  }

  @override
  int get hashCode => Object.hash(trackId, title, artist, isPlaying);
}
