import 'dart:io' show Platform;

import 'package:bilitune/app.dart';
import 'package:bilitune/core/platform/windows_hotkeys.dart';
import 'package:bilitune/data/models/models.dart';
import 'package:bilitune/state/providers.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences_platform_interface/in_memory_shared_preferences_async.dart';
import 'package:shared_preferences_platform_interface/shared_preferences_async_platform_interface.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('com.wep56.bilitune/hotkeys');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  late WindowsHotkeyBridge bridge;

  setUp(() {
    bridge = WindowsHotkeyBridge.forTesting();
    SharedPreferencesAsyncPlatform.instance =
        InMemorySharedPreferencesAsync.empty();
  });

  tearDown(() async {
    await bridge.resume(const []);
    messenger.setMockMethodCallHandler(channel, null);
  });

  test(
    'registration results expose conflicts instead of silently accepting them',
    () async {
      messenger.setMockMethodCallHandler(channel, (call) async {
        expect(call.method, 'setHotkeys');
        expect((call.arguments as Map)['bindings'], [
          {
            'action': 'playPause',
            'key': 'P',
            'modifiers': ['control', 'alt'],
          },
        ]);
        return {
          'registered': 3,
          'failures': [
            {'action': 'playPause', 'errorCode': 1409, 'isDefault': false},
          ],
        };
      });

      final result = await bridge.sync(const [
        WindowsHotkeyBinding(
          action: WindowsHotkeyAction.playPause,
          key: 'P',
          modifiers: {HotkeyModifier.control, HotkeyModifier.alt},
        ),
      ]);

      expect(result?.registered, 3);
      expect(
        result?.errorFor(WindowsHotkeyAction.playPause),
        contains('已被其他程序'),
      );
      expect(result?.errorFor(WindowsHotkeyAction.nextTrack), isNull);
      messenger.setMockMethodCallHandler(channel, (_) async => null);
    },
    skip: !Platform.isWindows,
  );

  test(
    'recording suspends hotkeys and restores the requested bindings',
    () async {
      final calls = <MethodCall>[];
      messenger.setMockMethodCallHandler(channel, (call) async {
        calls.add(call);
        return {'registered': 1};
      });
      await bridge.suspend();
      final ignored = await bridge.sync(const [
        WindowsHotkeyBinding(action: WindowsHotkeyAction.nextTrack, key: 'F8'),
      ]);
      expect(ignored, isNull);
      expect(calls.map((call) => call.method), ['suspendHotkeys']);

      await bridge.resume(const [
        WindowsHotkeyBinding(
          action: WindowsHotkeyAction.previousTrack,
          key: 'F7',
        ),
      ]);
      expect(calls.map((call) => call.method), [
        'suspendHotkeys',
        'setHotkeys',
      ]);
      expect((calls.last.arguments as Map)['bindings'], [
        {'action': 'previousTrack', 'key': 'F7', 'modifiers': <String>[]},
      ]);
    },
    skip: !Platform.isWindows,
  );

  test(
    'system media metadata preserves Unicode and ignores position-only changes',
    () async {
      bridge.setActionHandler((_) async {});
      final calls = <MethodCall>[];
      messenger.setMockMethodCallHandler(channel, (call) async {
        calls.add(call);
        return null;
      });
      for (var i = 0; i < 3; i++) {
        await bridge.updatePlayback(
          title: '歌曲 🎵',
          artist: '歌手',
          isPlaying: true,
          hasTrack: true,
        );
      }
      expect(calls, hasLength(1));
      expect(calls.single.method, 'updatePlayback');
      expect(calls.single.arguments, {
        'title': '歌曲 🎵',
        'artist': '歌手',
        'isPlaying': true,
        'hasTrack': true,
      });
      await bridge.updatePlayback(
        title: '歌曲 🎵',
        artist: '歌手',
        isPlaying: false,
        hasTrack: true,
      );
      expect(calls, hasLength(2));
    },
    skip: !Platform.isWindows,
  );

  testWidgets(
    'recorder distinguishes number-pad keys and accepts punctuation',
    (tester) async {
      await tester.sendKeyDownEvent(LogicalKeyboardKey.controlLeft);
      await tester.sendKeyDownEvent(LogicalKeyboardKey.numpad1);
      final binding = windowsHotkeyFromKeyEvent(
        WindowsHotkeyAction.nextTrack,
        const KeyDownEvent(
          physicalKey: PhysicalKeyboardKey.numpad1,
          logicalKey: LogicalKeyboardKey.numpad1,
          timeStamp: Duration.zero,
        ),
      );
      expect(binding?.key, 'Numpad1');
      expect(binding?.modifiers, {HotkeyModifier.control});
      await tester.sendKeyUpEvent(LogicalKeyboardKey.numpad1);
      await tester.sendKeyUpEvent(LogicalKeyboardKey.controlLeft);
      final period = windowsHotkeyFromKeyEvent(
        WindowsHotkeyAction.previousTrack,
        const KeyDownEvent(
          physicalKey: PhysicalKeyboardKey.period,
          logicalKey: LogicalKeyboardKey.period,
          timeStamp: Duration.zero,
        ),
      );
      expect(period?.key, 'Period');
    },
  );

  testWidgets(
    'native media and custom commands reach the playback controller',
    (tester) async {
      messenger.setMockMethodCallHandler(
        channel,
        (_) async => {'registered': 4, 'failures': []},
      );
      final playback = _RecordingPlayback();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [playbackProvider.overrideWith(() => playback)],
          child: const BiliTuneApp(),
        ),
      );
      await tester.pump(const Duration(seconds: 3));
      await tester.pump(const Duration(milliseconds: 500));

      for (final action in [
        'playPause',
        'previousTrack',
        'nextTrack',
        'stop',
        'play',
        'pause',
        'unknown',
      ]) {
        final pending = messenger.handlePlatformMessage(
          channel.name,
          const StandardMethodCodec().encodeMethodCall(
            MethodCall('onHotkeyPressed', {'action': action}),
          ),
          null,
        );
        await tester.pump();
        await pending;
      }
      expect(playback.actions, [
        'playPause',
        'previousTrack',
        'nextTrack',
        'stop',
        'play',
        'pause',
      ]);
    },
    skip: !Platform.isWindows,
  );

  test(
    'stop preserves the selected track and queue while returning to the start',
    () async {
      final container = ProviderContainer(
        overrides: [playbackProvider.overrideWith(_SeededPlayback.new)],
      );
      addTearDown(container.dispose);
      final notifier = container.read(playbackProvider.notifier);
      await notifier.setPlaying(
        true,
      ); // A direct Play must not pause an already-playing track.
      expect(container.read(playbackProvider).isPlaying, isTrue);
      await notifier.stop();
      final state = container.read(playbackProvider);
      expect(state.isPlaying, isFalse);
      expect(state.position, Duration.zero);
      expect(state.track?.id, 'test-song');
      expect(state.queue, hasLength(1));
      await notifier.setPlaying(false);
      expect(container.read(playbackProvider).isPlaying, isFalse);
    },
  );
}

const _track = Track(
  id: 'test-song',
  title: 'Test song',
  artist: 'Artist',
  duration: Duration(minutes: 3),
  type: ContentType.audio,
  gradientSeed: 1,
);

class _RecordingPlayback extends PlaybackNotifier {
  final actions = <String>[];
  @override
  PlaybackState build() => const PlaybackState(track: _track, queue: [_track]);
  @override
  Future<void> togglePlay() async {
    actions.add('playPause');
  }

  @override
  void previous() {
    actions.add('previousTrack');
  }

  @override
  void next() {
    actions.add('nextTrack');
  }

  @override
  Future<void> setPlaying(bool playing) async {
    actions.add(playing ? 'play' : 'pause');
  }

  @override
  Future<void> stop() async {
    actions.add('stop');
  }

  @override
  Future<void> saveNow() async {}
}

class _SeededPlayback extends PlaybackNotifier {
  @override
  PlaybackState build() => const PlaybackState(
    track: _track,
    queue: [_track],
    isPlaying: true,
    position: Duration(seconds: 30),
  );
}
