import 'package:bilitune/core/theme/app_theme.dart';
import 'package:bilitune/data/models/models.dart';
import 'package:bilitune/features/library/library_page.dart';
import 'package:bilitune/state/providers.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences_platform_interface/in_memory_shared_preferences_async.dart';
import 'package:shared_preferences_platform_interface/shared_preferences_async_platform_interface.dart';

void main() {
  setUp(() {
    SharedPreferencesAsyncPlatform.instance =
        InMemorySharedPreferencesAsync.empty();
  });

  testWidgets('desktop playlist and song panes scroll independently', (
    tester,
  ) async {
    await _pumpLibrary(tester, const Size(1440, 900));

    final playlists = find.byKey(const ValueKey('library-playlists-scroll'));
    final songs = find.byKey(const ValueKey('library-tracks-scroll'));
    final playlistController = tester.widget<ListView>(playlists).controller!;
    final songController = tester.widget<ListView>(songs).controller!;
    final songPosition = tester.getTopLeft(find.text('Song 0'));
    final headingPosition = tester.getTopLeft(find.text('我的歌单'));

    await tester.sendEventToBinding(
      PointerScrollEvent(
        position: tester.getCenter(playlists),
        scrollDelta: const Offset(0, 400),
        kind: PointerDeviceKind.mouse,
      ),
    );
    await tester.pumpAndSettle();

    expect(playlistController.offset, greaterThan(0));
    expect(songController.offset, 0);
    expect(tester.getTopLeft(find.text('Song 0')), songPosition);
    expect(tester.getTopLeft(find.text('我的歌单')), headingPosition);

    final playlistOffset = playlistController.offset;
    await tester.drag(songs, const Offset(0, -300));
    await tester.pumpAndSettle();

    expect(songController.offset, greaterThan(0));
    expect(playlistController.offset, playlistOffset);
    expect(tester.getTopLeft(find.text('我的歌单')), headingPosition);

    // Scrolling beyond the playlist boundary must not move the song pane.
    playlistController.jumpTo(playlistController.position.maxScrollExtent);
    await tester.pump();
    final songOffset = songController.offset;
    await tester.sendEventToBinding(
      PointerScrollEvent(
        position: tester.getCenter(playlists),
        scrollDelta: const Offset(0, 500),
        kind: PointerDeviceKind.mouse,
      ),
    );
    await tester.pumpAndSettle();

    expect(songController.offset, songOffset);
    expect(tester.takeException(), isNull);
  });

  testWidgets('compact layout keeps a usable page scroll after resizing', (
    tester,
  ) async {
    await _pumpLibrary(tester, const Size(1440, 900));

    tester.view.physicalSize = const Size(720, 900);
    await tester.pumpAndSettle();

    expect(
      find.byKey(const ValueKey('library-playlists-scroll')),
      findsNothing,
    );
    expect(find.byKey(const ValueKey('library-tracks-scroll')), findsNothing);
    final page = find.byType(ListView);
    expect(page, findsOneWidget);
    final pagePosition = tester
        .state<ScrollableState>(
          find.descendant(of: page, matching: find.byType(Scrollable)).first,
        )
        .position;
    pagePosition.jumpTo(pagePosition.maxScrollExtent);
    await tester.pumpAndSettle();

    expect(find.text('Song 39').hitTestable(), findsOneWidget);
    expect(tester.takeException(), isNull);

    tester.view.physicalSize = const Size(1440, 900);
    await tester.pumpAndSettle();
    expect(
      find.byKey(const ValueKey('library-playlists-scroll')),
      findsOneWidget,
    );
    expect(find.byKey(const ValueKey('library-tracks-scroll')), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}

Future<void> _pumpLibrary(WidgetTester tester, Size size) async {
  tester.view.devicePixelRatio = 1;
  tester.view.physicalSize = size;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [libraryProvider.overrideWith(_TestLibraryNotifier.new)],
      child: MaterialApp(
        theme: AppTheme.dark(),
        home: const Scaffold(body: LibraryPage()),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

class _TestLibraryNotifier extends LibraryNotifier {
  @override
  LibraryState build() {
    return LibraryState(
      folders: List.generate(
        30,
        (index) => BiliFavoriteFolder(
          mediaId: index + 1,
          fid: index + 1,
          mid: 1,
          title: 'Playlist $index',
          mediaCount: 40,
          gradientSeed: index,
        ),
      ),
      selectedFolderId: 1,
      selectedFolderTracks: List.generate(
        40,
        (index) => Track(
          id: 'song-$index',
          title: 'Song $index',
          artist: 'Artist',
          duration: const Duration(minutes: 3),
          type: ContentType.video,
          gradientSeed: index,
        ),
      ),
    );
  }
}
