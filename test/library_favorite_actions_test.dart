import 'dart:async';

import 'package:bilitune/core/network/bili_cookie_store.dart';
import 'package:bilitune/core/network/bili_wbi_signer.dart';
import 'package:bilitune/core/theme/app_theme.dart';
import 'package:bilitune/data/models/models.dart';
import 'package:bilitune/data/repositories/bili_music_repository.dart';
import 'package:bilitune/data/services/bili_api_service.dart';
import 'package:bilitune/features/library/library_page.dart';
import 'package:bilitune/shared/widgets/favorite_folder_dialogs.dart';
import 'package:bilitune/state/providers.dart';
import 'package:dio/dio.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:shared_preferences_platform_interface/in_memory_shared_preferences_async.dart';
import 'package:shared_preferences_platform_interface/shared_preferences_async_platform_interface.dart';

void main() {
  late Dio dio;
  late BiliMusicRepository repository;
  late List<RequestOptions> requests;
  late int mutationCode;
  late bool failFolderRefresh;
  Completer<void>? mutationGate;

  setUp(() async {
    SharedPreferencesAsyncPlatform.instance =
        InMemorySharedPreferencesAsync.empty();
    requests = [];
    mutationCode = 0;
    failFolderRefresh = false;
    mutationGate = null;
    final cookies = BiliCookieStore(SharedPreferencesAsync());
    await cookies.saveCredential(_credential);
    dio = Dio();
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (request, handler) async {
          requests.add(request);
          Map<String, dynamic> response;
          if (request.method == 'POST') {
            await mutationGate?.future;
            response = {
              'code': mutationCode,
              'message': mutationCode == 0 ? 'OK' : 'Request denied',
              'data': <String, dynamic>{},
            };
          } else if (request.uri.path.endsWith('/created/list-all')) {
            response = failFolderRefresh
                ? {'code': -503, 'message': 'Refresh unavailable'}
                : {
                    'code': 0,
                    'data': {
                      'list': [
                        _folderJson(1, 'Source', 1),
                        // The destination already contains the moved track.
                        _folderJson(2, 'Target', 5),
                        _folderJson(3, 'Hidden', 9),
                      ],
                    },
                  };
          } else {
            handler.reject(
              DioException(
                requestOptions: request,
                error: 'Unexpected request',
              ),
            );
            return;
          }
          handler.resolve(
            Response(requestOptions: request, statusCode: 200, data: response),
          );
        },
      ),
    );
    repository = BiliMusicRepository(
      BiliApiService(dio, cookies, BiliWbiSigner(dio)),
    );
  });

  tearDown(() => dio.close(force: true));

  Future<ProviderContainer> pumpLibrary(WidgetTester tester) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(1440, 900);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetPhysicalSize);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authProvider.overrideWith(_SignedInAuth.new),
          libraryProvider.overrideWith(_FavoriteLibrary.new),
          biliMusicRepositoryProvider.overrideWithValue(repository),
        ],
        child: MaterialApp(
          theme: AppTheme.dark(),
          home: const Scaffold(body: LibraryPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    return ProviderScope.containerOf(tester.element(find.byType(LibraryPage)));
  }

  test(
    'move submits source and destination in a single authenticated request',
    () async {
      await repository.moveTrackToFavoriteFolder(
        track: _songA,
        fromMediaId: 1,
        toMediaId: 2,
      );

      expect(requests, hasLength(1));
      expect(requests.single.method, 'POST');
      expect(requests.single.uri.path, '/x/v3/fav/resource/deal');
      expect(requests.single.data, containsPair('rid', 101));
      expect(requests.single.data, containsPair('add_media_ids', '2'));
      expect(requests.single.data, containsPair('del_media_ids', '1'));
      expect(requests.single.data, containsPair('csrf', 'csrf'));
    },
  );

  test('moving into the source folder makes no request', () async {
    await repository.moveTrackToFavoriteFolder(
      track: _songA,
      fromMediaId: 1,
      toMediaId: 1,
    );

    expect(requests, isEmpty);
  });

  testWidgets('the existing add dialog adds without removing from the source', (
    tester,
  ) async {
    final container = await pumpLibrary(tester);
    unawaited(
      showAddToFavoriteDialog(tester.element(find.byType(LibraryPage)), _songA),
    );
    await tester.pumpAndSettle();
    final dialog = find.byType(Dialog);
    expect(
      find.descendant(of: dialog, matching: find.text('Source')),
      findsOneWidget,
    );
    await tester.tap(
      find.descendant(of: dialog, matching: find.text('Target')),
    );
    await tester.pumpAndSettle();

    expect(requests.single.data, containsPair('add_media_ids', '2'));
    expect(requests.single.data, containsPair('del_media_ids', ''));
    expect(container.read(libraryProvider).selectedFolderTracks, hasLength(2));
    expect(find.byType(Dialog), findsNothing);
    await _expireToast(tester);
  });

  testWidgets('right click deletes only the selected song from its folder', (
    tester,
  ) async {
    final container = await pumpLibrary(tester);
    await _openMenu(tester);
    expect(container.read(playbackProvider).track, isNull);
    expect(find.text('移动到其他收藏夹'), findsOneWidget);

    await tester.tap(find.text('从当前收藏夹删除'));
    await tester.pumpAndSettle();

    final deletion = requests
        .where((request) => request.method == 'POST')
        .single;
    expect(deletion.uri.path, '/x/v3/fav/resource/batch-del');
    expect(deletion.data, containsPair('media_id', 1));
    expect(deletion.data, containsPair('resources', '101:2'));
    final library = container.read(libraryProvider);
    expect(library.selectedFolderTracks.map((track) => track.id), ['song-b']);
    expect(library.trackKeyword, 'Song');
    expect(library.folders.map((folder) => folder.mediaId), [1, 2]);
    expect(library.folders.map((folder) => folder.mediaCount), [1, 5]);
    expect(library.isUpdatingFavorites, isFalse);
    expect(container.read(playbackProvider).track, isNull);
    await _expireToast(tester);
  });

  testWidgets(
    'move excludes source and prevents repeated clicks while pending',
    (tester) async {
      final container = await pumpLibrary(tester);
      mutationGate = Completer<void>();
      await _openMenu(tester);
      await tester.tap(find.text('移动到其他收藏夹'));
      await tester.pumpAndSettle();

      final dialog = find.byType(Dialog);
      expect(
        find.descendant(of: dialog, matching: find.text('Source')),
        findsNothing,
      );
      final target = find.descendant(of: dialog, matching: find.text('Target'));
      await tester.tap(target);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(container.read(libraryProvider).isUpdatingFavorites, isTrue);
      expect(
        container.read(libraryProvider).selectedFolderTracks,
        hasLength(2),
      );
      await tester.tap(target);
      await tester.pump();
      expect(
        requests.where((request) => request.method == 'POST'),
        hasLength(1),
      );

      mutationGate!.complete();
      await tester.pumpAndSettle();

      final library = container.read(libraryProvider);
      expect(find.byType(Dialog), findsNothing);
      expect(library.selectedFolderTracks.map((track) => track.id), ['song-b']);
      expect(library.folders.map((folder) => folder.mediaCount), [1, 5]);
      expect(library.trackKeyword, 'Song');
      expect(library.isUpdatingFavorites, isFalse);
      expect(container.read(playbackProvider).track, isNull);
      await _expireToast(tester);
    },
  );

  testWidgets('failed move retains the song and displays an error for retry', (
    tester,
  ) async {
    final container = await pumpLibrary(tester);
    mutationCode = -403;
    await _openMenu(tester);
    await tester.tap(find.text('移动到其他收藏夹'));
    await tester.pumpAndSettle();
    await tester.tap(
      find.descendant(of: find.byType(Dialog), matching: find.text('Target')),
    );
    await tester.pumpAndSettle();

    expect(find.byType(Dialog), findsOneWidget);
    expect(find.textContaining('Request denied'), findsWidgets);
    expect(container.read(libraryProvider).selectedFolderTracks, hasLength(2));
    expect(container.read(libraryProvider).isUpdatingFavorites, isFalse);
    expect(requests, hasLength(1));

    mutationCode = 0;
    await tester.tap(
      find.descendant(of: find.byType(Dialog), matching: find.text('Target')),
    );
    await tester.pumpAndSettle();
    expect(find.byType(Dialog), findsNothing);
    expect(container.read(libraryProvider).selectedFolderTracks, hasLength(1));
    await _expireToast(tester);
  });

  testWidgets('failed deletion keeps the song and reports the failure', (
    tester,
  ) async {
    final container = await pumpLibrary(tester);
    mutationCode = -403;
    await _openMenu(tester);
    await tester.tap(find.text('从当前收藏夹删除'));
    await tester.pumpAndSettle();

    expect(container.read(libraryProvider).selectedFolderTracks, hasLength(2));
    expect(container.read(libraryProvider).isUpdatingFavorites, isFalse);
    expect(find.textContaining('删除失败'), findsOneWidget);
    expect(requests, hasLength(1));
    await _expireToast(tester);
  });

  testWidgets('metadata refresh failure does not undo a successful deletion', (
    tester,
  ) async {
    final container = await pumpLibrary(tester);
    failFolderRefresh = true;
    await _openMenu(tester);
    await tester.tap(find.text('从当前收藏夹删除'));
    await tester.pumpAndSettle();

    final library = container.read(libraryProvider);
    expect(library.selectedFolderTracks.map((track) => track.id), ['song-b']);
    expect(library.errorMessage, contains('操作已完成'));
    expect(library.isUpdatingFavorites, isFalse);
    expect(requests.where((request) => request.method == 'POST'), hasLength(1));
    expect(find.text('已从当前收藏夹删除'), findsOneWidget);
    await _expireToast(tester);
  });
}

Future<void> _openMenu(WidgetTester tester) async {
  await tester.tap(
    find.text('Song A'),
    buttons: kSecondaryMouseButton,
    kind: PointerDeviceKind.mouse,
  );
  await tester.pumpAndSettle();
}

Future<void> _expireToast(WidgetTester tester) async {
  await tester.pump(const Duration(seconds: 4));
  await tester.pumpAndSettle();
}

const _credential = BiliCredential(
  sessdata: 'session',
  biliJct: 'csrf',
  mid: 7,
);
const _songA = Track(
  id: 'song-a',
  title: 'Song A',
  artist: 'Artist',
  duration: Duration(minutes: 3),
  type: ContentType.video,
  gradientSeed: 1,
  aid: 101,
);
const _songB = Track(
  id: 'song-b',
  title: 'Song B',
  artist: 'Artist',
  duration: Duration(minutes: 3),
  type: ContentType.video,
  gradientSeed: 2,
  aid: 102,
);

class _SignedInAuth extends AuthNotifier {
  @override
  AuthState build() => const AuthState(credential: _credential);
}

class _FavoriteLibrary extends LibraryNotifier {
  @override
  LibraryState build() => const LibraryState(
    folders: [
      BiliFavoriteFolder(
        mediaId: 1,
        fid: 1,
        mid: 7,
        title: 'Source',
        mediaCount: 2,
        gradientSeed: 1,
      ),
      BiliFavoriteFolder(
        mediaId: 2,
        fid: 2,
        mid: 7,
        title: 'Target',
        mediaCount: 5,
        gradientSeed: 2,
      ),
    ],
    selectedFolderId: 1,
    selectedFolderTracks: [_songA, _songB],
    trackKeyword: 'Song',
  );
}

Map<String, dynamic> _folderJson(int id, String title, int count) => {
  'id': id,
  'fid': id,
  'mid': 7,
  'title': title,
  'media_count': count,
};
