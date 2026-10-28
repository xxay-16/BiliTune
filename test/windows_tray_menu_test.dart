import 'dart:io' show Platform;

import 'package:bilitune/core/platform/windows_tray_menu.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('com.wep56.bilitune/tray_menu');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  tearDown(() => messenger.setMockMethodCallHandler(channel, null));

  test(
    'sends complete long Unicode titles without making a string-sized menu',
    () async {
      final title = List.filled(30, '一起唱经典合唱老歌 🎵 & Live').join(' ');
      final calls = <MethodCall>[];
      messenger.setMockMethodCallHandler(channel, (call) async {
        calls.add(call);
        return null;
      });

      await WindowsTrayMenu.update(title: title, isPlaying: true);

      expect(calls.single.method, 'update');
      expect(calls.single.arguments, {'title': title, 'isPlaying': true});
    },
    skip: !Platform.isWindows,
  );

  test(
    'maps every native menu command to its playback or window action',
    () async {
      for (final action in WindowsTrayAction.values) {
        messenger.setMockMethodCallHandler(channel, (call) async {
          expect(call.method, 'show');
          return action.name;
        });
        expect(await WindowsTrayMenu.show(), action);
      }
    },
    skip: !Platform.isWindows,
  );

  test(
    'cancelled and unknown native selections perform no action',
    () async {
      for (final result in <String?>[null, '', 'unknown']) {
        messenger.setMockMethodCallHandler(channel, (_) async => result);
        expect(await WindowsTrayMenu.show(), isNull);
      }
    },
    skip: !Platform.isWindows,
  );

  test(
    'unsupported embedders do not crash on absent native tray methods',
    () async {
      await WindowsTrayMenu.update(title: 'Song', isPlaying: false);
      expect(await WindowsTrayMenu.show(), isNull);
    },
  );
}
