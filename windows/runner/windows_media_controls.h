#ifndef RUNNER_WINDOWS_MEDIA_CONTROLS_H_
#define RUNNER_WINDOWS_MEDIA_CONTROLS_H_

#include <windows.h>
#include <memory>
#include <string>

constexpr UINT kWindowsMediaCommand = WM_APP + 560;

class WindowsMediaControls {
 public:
  explicit WindowsMediaControls(HWND owner);
  ~WindowsMediaControls();
  bool available() const;
  void Update(const std::wstring& title, const std::wstring& artist,
              bool is_playing, bool has_track);
  void Suspend(bool suspended);

 private:
  struct Impl;
  std::unique_ptr<Impl> impl_;
};

#endif  // RUNNER_WINDOWS_MEDIA_CONTROLS_H_
