import 'package:bilitune/data/models/models.dart';
import 'package:bilitune/state/providers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  for (final elapsed in [
    Duration.zero,
    const Duration(seconds: 3),
    const Duration(seconds: 4),
    const Duration(hours: 1),
  ]) {
    test('previous switches tracks immediately at position $elapsed', () {
      final container = ProviderContainer(
        overrides: [
          playbackProvider.overrideWith(() => _NavigationPlayback(elapsed)),
        ],
      );
      addTearDown(container.dispose);

      container.read(playbackProvider.notifier).previous();

      final state = container.read(playbackProvider);
      expect(state.track?.id, 'first');
      expect(state.position, Duration.zero);
      expect(state.queue, _queue);
      expect(state.isPlaying, isTrue);
    });
  }

  test(
    'each previous press selects another track, wrapping at the beginning',
    () {
      final container = ProviderContainer(
        overrides: [
          playbackProvider.overrideWith(
            () => _NavigationPlayback(const Duration(minutes: 5)),
          ),
        ],
      );
      addTearDown(container.dispose);
      final notifier = container.read(playbackProvider.notifier);

      notifier.previous();
      expect(container.read(playbackProvider).track?.id, 'first');
      notifier.previous();
      expect(container.read(playbackProvider).track?.id, 'last');
      notifier.previous();
      expect(container.read(playbackProvider).track?.id, 'current');
    },
  );
}

const _queue = [
  Track(
    id: 'first',
    title: 'First',
    artist: 'Artist',
    duration: Duration(minutes: 3),
    type: ContentType.audio,
    gradientSeed: 1,
  ),
  Track(
    id: 'current',
    title: 'Current',
    artist: 'Artist',
    duration: Duration(hours: 2),
    type: ContentType.audio,
    gradientSeed: 2,
  ),
  Track(
    id: 'last',
    title: 'Last',
    artist: 'Artist',
    duration: Duration(minutes: 3),
    type: ContentType.audio,
    gradientSeed: 3,
  ),
];

// Exercise the real previous() selection logic while avoiding audio/network I/O.
class _NavigationPlayback extends PlaybackNotifier {
  _NavigationPlayback(this.elapsed);

  final Duration elapsed;

  @override
  PlaybackState build() => PlaybackState(
    track: _queue[1],
    queue: _queue,
    position: elapsed,
    isPlaying: true,
  );

  @override
  Future<void> playTrack(Track track, {List<Track>? queue}) async {
    state = state.copyWith(track: track, queue: queue, position: Duration.zero);
  }
}
