import 'dart:io' show Platform;

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

enum WindowsHotkeyAction {
  playPause,
  previousTrack,
  nextTrack,
  toggleWindow,
  stop,
  play,
  pause,
}

const windowsHotkeyActions = <WindowsHotkeyAction>[
  WindowsHotkeyAction.playPause,
  WindowsHotkeyAction.previousTrack,
  WindowsHotkeyAction.nextTrack,
  WindowsHotkeyAction.stop,
  WindowsHotkeyAction.toggleWindow,
];

extension WindowsHotkeyActionLabel on WindowsHotkeyAction {
  String get label => switch (this) {
    WindowsHotkeyAction.playPause => '播放 / 暂停',
    WindowsHotkeyAction.previousTrack => '上一首',
    WindowsHotkeyAction.nextTrack => '下一首',
    WindowsHotkeyAction.toggleWindow => '隐藏 / 显示主窗口',
    WindowsHotkeyAction.stop => '停止播放',
    WindowsHotkeyAction.play => '播放',
    WindowsHotkeyAction.pause => '暂停',
  };

  String get description => switch (this) {
    WindowsHotkeyAction.playPause => '全局控制当前曲目的播放状态',
    WindowsHotkeyAction.previousTrack => '回到上一首或当前曲目开头',
    WindowsHotkeyAction.nextTrack => '播放队列中的下一首',
    WindowsHotkeyAction.toggleWindow => '显示前台窗口，或隐藏到托盘',
    WindowsHotkeyAction.stop => '停止当前曲目并回到开头',
    WindowsHotkeyAction.play => '继续播放当前曲目',
    WindowsHotkeyAction.pause => '暂停当前曲目',
  };
}

WindowsHotkeyAction? windowsHotkeyActionFromName(String? value) {
  for (final action in WindowsHotkeyAction.values) {
    if (action.name == value) return action;
  }
  return null;
}

enum HotkeyModifier { control, alt, shift, win }

extension HotkeyModifierLabel on HotkeyModifier {
  String get label => switch (this) {
    HotkeyModifier.control => 'Ctrl',
    HotkeyModifier.alt => 'Alt',
    HotkeyModifier.shift => 'Shift',
    HotkeyModifier.win => 'Win',
  };
}

HotkeyModifier? hotkeyModifierFromName(String? value) {
  for (final modifier in HotkeyModifier.values) {
    if (modifier.name == value) return modifier;
  }
  return null;
}

class WindowsHotkeyBinding {
  const WindowsHotkeyBinding({
    required this.action,
    required this.key,
    this.modifiers = const <HotkeyModifier>{},
  });

  final WindowsHotkeyAction action;
  final String key;
  final Set<HotkeyModifier> modifiers;

  bool get isSet => key.isNotEmpty;

  String get displayLabel {
    if (!isSet) return '未设置';
    final parts = <String>[
      for (final modifier in HotkeyModifier.values)
        if (modifiers.contains(modifier)) modifier.label,
      _keyLabel(key),
    ];
    return parts.join(' + ');
  }

  String get signature {
    final modifierNames = modifiers.map((item) => item.name).toList()..sort();
    return [...modifierNames, key].join('+');
  }

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'action': action.name,
      'key': key,
      'modifiers': modifiers.map((item) => item.name).toList(growable: false),
    };
  }

  Map<String, dynamic> toNativeJson() => toJson();

  factory WindowsHotkeyBinding.fromJson(Map<String, dynamic> json) {
    final action = windowsHotkeyActionFromName(json['action']?.toString());
    if (action == null) {
      throw FormatException('Unknown hotkey action: ${json['action']}');
    }
    return WindowsHotkeyBinding(
      action: action,
      key: json['key']?.toString() ?? '',
      modifiers:
          (json['modifiers'] as List?)
              ?.map((item) => hotkeyModifierFromName(item.toString()))
              .whereType<HotkeyModifier>()
              .toSet() ??
          const <HotkeyModifier>{},
    );
  }

  static String _keyLabel(String key) {
    return switch (key) {
      'Space' => 'Space',
      'Enter' => 'Enter',
      'Escape' => 'Esc',
      'Backspace' => 'Backspace',
      'Tab' => 'Tab',
      'ArrowLeft' => 'Left',
      'ArrowRight' => 'Right',
      'ArrowUp' => 'Up',
      'ArrowDown' => 'Down',
      'PageUp' => 'Page Up',
      'PageDown' => 'Page Down',
      'MediaPlayPause' => 'Media Play/Pause',
      'MediaNextTrack' => 'Media Next',
      'MediaPreviousTrack' => 'Media Previous',
      'MediaStop' => 'Media Stop',
      _ => key,
    };
  }
}

WindowsHotkeyBinding? windowsHotkeyFromKeyEvent(
  WindowsHotkeyAction action,
  KeyEvent event,
) {
  if (event is! KeyDownEvent && event is! KeyRepeatEvent) return null;
  if (_isModifierKey(event.logicalKey)) return null;

  final key = _hotkeyToken(event.logicalKey);
  if (key == null) return null;

  final keyboard = HardwareKeyboard.instance;
  return WindowsHotkeyBinding(
    action: action,
    key: key,
    modifiers: <HotkeyModifier>{
      if (keyboard.isControlPressed) HotkeyModifier.control,
      if (keyboard.isAltPressed) HotkeyModifier.alt,
      if (keyboard.isShiftPressed) HotkeyModifier.shift,
      if (keyboard.isMetaPressed) HotkeyModifier.win,
    },
  );
}

bool _isModifierKey(LogicalKeyboardKey key) {
  return key == LogicalKeyboardKey.control ||
      key == LogicalKeyboardKey.controlLeft ||
      key == LogicalKeyboardKey.controlRight ||
      key == LogicalKeyboardKey.alt ||
      key == LogicalKeyboardKey.altLeft ||
      key == LogicalKeyboardKey.altRight ||
      key == LogicalKeyboardKey.shift ||
      key == LogicalKeyboardKey.shiftLeft ||
      key == LogicalKeyboardKey.shiftRight ||
      key == LogicalKeyboardKey.meta ||
      key == LogicalKeyboardKey.metaLeft ||
      key == LogicalKeyboardKey.metaRight;
}

String? _hotkeyToken(LogicalKeyboardKey key) {
  final namedKeys = <LogicalKeyboardKey, String>{
    LogicalKeyboardKey.numpad0: 'Numpad0',
    LogicalKeyboardKey.numpad1: 'Numpad1',
    LogicalKeyboardKey.numpad2: 'Numpad2',
    LogicalKeyboardKey.numpad3: 'Numpad3',
    LogicalKeyboardKey.numpad4: 'Numpad4',
    LogicalKeyboardKey.numpad5: 'Numpad5',
    LogicalKeyboardKey.numpad6: 'Numpad6',
    LogicalKeyboardKey.numpad7: 'Numpad7',
    LogicalKeyboardKey.numpad8: 'Numpad8',
    LogicalKeyboardKey.numpad9: 'Numpad9',
    LogicalKeyboardKey.numpadAdd: 'NumpadAdd',
    LogicalKeyboardKey.numpadSubtract: 'NumpadSubtract',
    LogicalKeyboardKey.numpadMultiply: 'NumpadMultiply',
    LogicalKeyboardKey.numpadDivide: 'NumpadDivide',
    LogicalKeyboardKey.numpadDecimal: 'NumpadDecimal',
    LogicalKeyboardKey.numpadEnter: 'Enter',
    LogicalKeyboardKey.semicolon: 'Semicolon',
    LogicalKeyboardKey.equal: 'Equal',
    LogicalKeyboardKey.comma: 'Comma',
    LogicalKeyboardKey.minus: 'Minus',
    LogicalKeyboardKey.period: 'Period',
    LogicalKeyboardKey.slash: 'Slash',
    LogicalKeyboardKey.backquote: 'Backquote',
    LogicalKeyboardKey.bracketLeft: 'BracketLeft',
    LogicalKeyboardKey.backslash: 'Backslash',
    LogicalKeyboardKey.bracketRight: 'BracketRight',
    LogicalKeyboardKey.quote: 'Quote',
  };
  if (namedKeys.containsKey(key)) return namedKeys[key];
  final label = key.keyLabel.toUpperCase();
  if (RegExp(r'^[A-Z]$').hasMatch(label)) return label;
  if (RegExp(r'^[0-9]$').hasMatch(label)) return label;
  if (RegExp(r'^F([1-9]|1[0-9]|2[0-4])$').hasMatch(label)) return label;

  if (key == LogicalKeyboardKey.space) return 'Space';
  if (key == LogicalKeyboardKey.enter) return 'Enter';
  if (key == LogicalKeyboardKey.escape) return 'Escape';
  if (key == LogicalKeyboardKey.backspace) return 'Backspace';
  if (key == LogicalKeyboardKey.tab) return 'Tab';
  if (key == LogicalKeyboardKey.arrowLeft) return 'ArrowLeft';
  if (key == LogicalKeyboardKey.arrowRight) return 'ArrowRight';
  if (key == LogicalKeyboardKey.arrowUp) return 'ArrowUp';
  if (key == LogicalKeyboardKey.arrowDown) return 'ArrowDown';
  if (key == LogicalKeyboardKey.home) return 'Home';
  if (key == LogicalKeyboardKey.end) return 'End';
  if (key == LogicalKeyboardKey.pageUp) return 'PageUp';
  if (key == LogicalKeyboardKey.pageDown) return 'PageDown';
  if (key == LogicalKeyboardKey.insert) return 'Insert';
  if (key == LogicalKeyboardKey.delete) return 'Delete';
  if (key == LogicalKeyboardKey.mediaPlayPause) return 'MediaPlayPause';
  if (key == LogicalKeyboardKey.mediaTrackNext) return 'MediaNextTrack';
  if (key == LogicalKeyboardKey.mediaTrackPrevious) {
    return 'MediaPreviousTrack';
  }
  if (key == LogicalKeyboardKey.mediaStop) return 'MediaStop';
  return null;
}

class WindowsHotkeyFailure {
  const WindowsHotkeyFailure({
    required this.action,
    required this.errorCode,
    required this.isDefault,
  });

  final WindowsHotkeyAction action;
  final int errorCode;
  final bool isDefault;

  String get message {
    final prefix = isDefault ? '媒体键' : '此组合键';
    if (errorCode == 1409) {
      return '$prefix已被其他程序或另一个 BiliTune 窗口占用，请关闭冲突程序或改用其他组合键。';
    }
    return '$prefix未能注册（Windows 错误 $errorCode），请重新录制其他组合键。';
  }
}

class WindowsHotkeyRegistration {
  const WindowsHotkeyRegistration({
    this.registered = 0,
    this.failures = const [],
  });

  final int registered;
  final List<WindowsHotkeyFailure> failures;

  factory WindowsHotkeyRegistration.fromNative(Object? value) {
    if (value is! Map) return const WindowsHotkeyRegistration();
    final failures = <WindowsHotkeyFailure>[];
    final raw = value['failures'];
    if (raw is List) {
      for (final item in raw.whereType<Map>()) {
        final action = windowsHotkeyActionFromName(item['action']?.toString());
        if (action == null) continue;
        failures.add(
          WindowsHotkeyFailure(
            action: action,
            errorCode: (item['errorCode'] as num?)?.toInt() ?? -1,
            isDefault: item['isDefault'] == true,
          ),
        );
      }
    }
    return WindowsHotkeyRegistration(
      registered: (value['registered'] as num?)?.toInt() ?? 0,
      failures: failures,
    );
  }

  String? errorFor(WindowsHotkeyAction action) {
    final matching = failures
        .where((failure) => failure.action == action)
        .toList();
    if (matching.isEmpty) return null;
    return matching
        .firstWhere(
          (failure) => !failure.isDefault,
          orElse: () => matching.first,
        )
        .message;
  }
}

class WindowsHotkeyBridge {
  WindowsHotkeyBridge._();

  @visibleForTesting
  WindowsHotkeyBridge.forTesting();

  static final instance = WindowsHotkeyBridge._();
  static const _channel = MethodChannel('com.wep56.bilitune/hotkeys');
  bool _suspended = false;
  Future<void> _pending = Future<void>.value();
  (String, String, bool, bool)? _lastPlayback;

  Future<T> _enqueue<T>(Future<T> Function() operation) {
    final next = _pending.then((_) => operation());
    _pending = next.then<void>((_) {}, onError: (Object _, StackTrace _) {});
    return next;
  }

  void setActionHandler(
    Future<void> Function(WindowsHotkeyAction action) onAction,
  ) {
    if (!Platform.isWindows) return;
    _lastPlayback = null;
    _channel.setMethodCallHandler((call) async {
      if (call.method != 'onHotkeyPressed') return null;
      final arguments = call.arguments;
      final actionName = arguments is Map
          ? arguments['action']?.toString()
          : arguments?.toString();
      final action = windowsHotkeyActionFromName(actionName);
      if (action != null) await onAction(action);
      return null;
    });
  }

  Future<WindowsHotkeyRegistration?> sync(
    Iterable<WindowsHotkeyBinding> bindings,
  ) {
    final requested = bindings.toList(growable: false);
    return _enqueue(() async {
      if (!Platform.isWindows || _suspended) return null;
      try {
        final result = await _channel
            .invokeMethod<Object?>('setHotkeys', <String, Object?>{
              'bindings': requested
                  .where((binding) => binding.isSet)
                  .map((binding) => binding.toNativeJson())
                  .toList(growable: false),
            });
        return WindowsHotkeyRegistration.fromNative(result);
      } on MissingPluginException {
        // The Windows runner owns this channel. Ignoring this keeps tests and
        // unsupported embedders from failing during startup.
        return null;
      }
    });
  }

  Future<void> updatePlayback({
    required String title,
    required String artist,
    required bool isPlaying,
    required bool hasTrack,
  }) {
    final snapshot = (title, artist, isPlaying, hasTrack);
    return _enqueue(() async {
      if (!Platform.isWindows || snapshot == _lastPlayback) return;
      try {
        await _channel.invokeMethod<void>('updatePlayback', {
          'title': title,
          'artist': artist,
          'isPlaying': isPlaying,
          'hasTrack': hasTrack,
        });
        _lastPlayback = snapshot;
      } on MissingPluginException {
        // The Windows runner owns the system media session.
      }
    });
  }

  Future<void> suspend() async {
    _suspended = true;
    await _enqueue(() async {
      if (!Platform.isWindows) return;
      try {
        await _channel.invokeMethod<void>('suspendHotkeys');
      } on MissingPluginException {
        // Test embedders do not own system hotkeys.
      }
    });
  }

  Future<WindowsHotkeyRegistration?> resume(
    Iterable<WindowsHotkeyBinding> bindings,
  ) {
    _suspended = false;
    return sync(bindings);
  }
}
