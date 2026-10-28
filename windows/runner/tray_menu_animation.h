#ifndef RUNNER_TRAY_MENU_ANIMATION_H_
#define RUNNER_TRAY_MENU_ANIMATION_H_

#include <cmath>
#include <cstdint>

namespace tray_menu {

constexpr int kWidthDip = 180;
constexpr int kPaddingDip = 12;
constexpr int kGapDip = 32;

// Pause at the beginning of each loop so the start remains easy to read.
inline int TitleOffset(uint64_t elapsed_ms, int text_width, int viewport_width,
                       double scale) {
  if (text_width <= viewport_width || scale <= 0) return 0;
  constexpr double kPauseMs = 1000;
  const double speed = 30 * scale;
  const double distance = text_width + kGapDip * scale;
  const double cycle_ms = kPauseMs + distance / speed * 1000;
  const double phase = std::fmod(static_cast<double>(elapsed_ms), cycle_ms);
  if (phase <= kPauseMs) return 0;
  return static_cast<int>((phase - kPauseMs) * speed / 1000);
}

}  // namespace tray_menu

#endif  // RUNNER_TRAY_MENU_ANIMATION_H_
