#ifndef RUNNER_TRAY_MENU_H_
#define RUNNER_TRAY_MENU_H_

#include <windows.h>

#include <optional>
#include <string>

// Native menu drawing is independent of the Flutter surface, including while
// the main window is hidden in the notification area.
class TrayMenu {
 public:
  explicit TrayMenu(HWND owner);
  ~TrayMenu();

  void Update(const std::wstring& title, bool is_playing);
  std::string Show();
  std::optional<LRESULT> HandleMessage(UINT message, WPARAM wparam,
                                       LPARAM lparam);

 private:
  int Scale(int value) const;
  void MeasureTitle();
  void DrawTitle(const DRAWITEMSTRUCT& item);
  void RedrawTitle();
  void SyncTimer();
  void Close();

  HWND owner_;
  HWND popup_ = nullptr;
  HMENU menu_ = nullptr;
  HFONT font_ = nullptr;
  UINT dpi_ = 96;
  std::wstring title_ = L"BiliTune";
  bool is_playing_ = false;
  bool animate_ = true;
  int text_width_ = 0;
  ULONGLONG started_at_ = 0;
};

#endif  // RUNNER_TRAY_MENU_H_
