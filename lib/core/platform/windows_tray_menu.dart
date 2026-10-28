import 'dart:io' show Platform;

import 'package:flutter/services.dart';

enum WindowsTrayAction { previous, playPause, next, show, hide, exit }

class WindowsTrayMenu {
  WindowsTrayMenu._();

  static const _channel = MethodChannel('com.wep56.bilitune/tray_menu');

  static Future<void> update({
    required String title,
    required bool isPlaying,
  }) async {
    if (!Platform.isWindows) return;
    try {
      await _channel.invokeMethod<void>('update', {
        'title': title,
        'isPlaying': isPlaying,
      });
    } on MissingPluginException {
      // Other embedders and widget tests do not host the Windows menu.
    }
  }

  static Future<WindowsTrayAction?> show() async {
    if (!Platform.isWindows) return null;
    try {
      final action = await _channel.invokeMethod<String>('show');
      for (final candidate in WindowsTrayAction.values) {
        if (candidate.name == action) return candidate;
      }
    } on MissingPluginException {
      // Other embedders and widget tests do not host the Windows menu.
    }
    return null;
  }
}
