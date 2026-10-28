import 'package:bilitune/core/network/bili_api_exception.dart';
import 'package:bilitune/core/network/bili_cookie_store.dart';
import 'package:bilitune/core/network/bili_wbi_signer.dart';
import 'package:bilitune/data/repositories/bili_music_repository.dart';
import 'package:bilitune/data/services/bili_api_service.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:shared_preferences_platform_interface/in_memory_shared_preferences_async.dart';
import 'package:shared_preferences_platform_interface/shared_preferences_async_platform_interface.dart';

void main() {
  late Dio dio;
  late BiliMusicRepository repository;
  late List<RequestOptions> requests;
  late List<Map<String, dynamic>> responses;

  setUp(() {
    SharedPreferencesAsyncPlatform.instance =
        InMemorySharedPreferencesAsync.empty();
    requests = [];
    responses = [];
    dio = Dio();
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          if (responses.isEmpty) {
            handler.reject(
              DioException(
                requestOptions: options,
                error: 'Unexpected extra page request',
              ),
            );
            return;
          }
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: responses.removeAt(0),
            ),
          );
        },
      ),
    );
    repository = BiliMusicRepository(
      BiliApiService(
        dio,
        BiliCookieStore(SharedPreferencesAsync()),
        BiliWbiSigner(dio),
      ),
    );
  });

  tearDown(() => dio.close(force: true));

  test(
    'loads every page in order for a folder with more than 20 tracks',
    () async {
      responses.addAll([
        _page(1, 20, hasMore: true),
        _page(21, 20, hasMore: true),
        _page(41, 5, hasMore: false),
      ]);

      final tracks = await repository.favoriteFolderTracks(123);

      expect(tracks.map((track) => track.aid), List.generate(45, (i) => i + 1));
      expect(requests.map((request) => request.uri.queryParameters['pn']), [
        '1',
        '2',
        '3',
      ]);
      expect(
        requests.every((request) => request.uri.queryParameters['ps'] == '20'),
        isTrue,
      );
    },
  );

  test('stops at a full final page when has_more is false', () async {
    responses.addAll([
      _page(1, 20, hasMore: true),
      _page(21, 20, hasMore: false),
    ]);

    expect(await repository.favoriteFolderTracks(123), hasLength(40));
    expect(requests, hasLength(2));
  });

  test('continues a short page when the API reports more results', () async {
    responses.addAll([_page(1, 3, hasMore: true), _page(4, 2, hasMore: false)]);

    expect(await repository.favoriteFolderTracks(123), hasLength(5));
    expect(requests, hasLength(2));
  });

  test('supports numeric pagination flags', () async {
    responses.addAll([_page(1, 3, hasMore: 1), _page(4, 20, hasMore: 0)]);

    expect(await repository.favoriteFolderTracks(123), hasLength(23));
    expect(requests, hasLength(2));
  });

  test('uses page length when the pagination flag is absent', () async {
    responses.addAll([_page(1, 20), _page(21, 4)]);

    expect(await repository.favoriteFolderTracks(123), hasLength(24));
    expect(requests, hasLength(2));
  });

  test('stops on an empty page even if has_more is true', () async {
    responses.addAll([
      _page(1, 20, hasMore: true),
      _page(21, 0, hasMore: true),
    ]);

    expect(await repository.favoriteFolderTracks(123), hasLength(20));
    expect(requests, hasLength(2));
  });

  test('returns an empty list for an empty folder', () async {
    responses.add(_page(1, 0, hasMore: false));

    expect(await repository.favoriteFolderTracks(123), isEmpty);
    expect(requests, hasLength(1));
  });

  test('preserves search and ordering options on every page', () async {
    responses.addAll([_page(1, 2, hasMore: true), _page(3, 1, hasMore: false)]);

    final tracks = await repository.favoriteFolderTracks(
      456,
      page: 2,
      pageSize: 2,
      order: 'pubtime',
      keyword: '音乐',
    );

    expect(tracks, hasLength(3));
    expect(requests.map((request) => request.uri.queryParameters['pn']), [
      '2',
      '3',
    ]);
    for (final request in requests) {
      expect(request.uri.queryParameters, containsPair('media_id', '456'));
      expect(request.uri.queryParameters, containsPair('ps', '2'));
      expect(request.uri.queryParameters, containsPair('order', 'pubtime'));
      expect(request.uri.queryParameters, containsPair('keyword', '音乐'));
    }
  });

  test(
    'reports later page failures instead of returning a partial playlist',
    () async {
      responses.addAll([
        _page(1, 20, hasMore: true),
        {'code': -403, 'message': 'Access denied'},
      ]);

      await expectLater(
        repository.favoriteFolderTracks(123),
        throwsA(isA<BiliApiException>()),
      );
      expect(requests, hasLength(2));
    },
  );
}

Map<String, dynamic> _page(int firstId, int count, {Object? hasMore}) {
  return {
    'code': 0,
    'data': {
      'has_more': ?hasMore,
      'medias': count == 0
          ? null
          : List.generate(count, (index) {
              final id = firstId + index;
              return {
                'id': id,
                'title': 'Song $id',
                'type': 2,
                'duration': 180,
                'upper': {'name': 'Artist $id'},
              };
            }),
    },
  };
}
