#ifndef RUNNER_WINDOWS_HOTKEYS_H_
#define RUNNER_WINDOWS_HOTKEYS_H_

#include <windows.h>

#include <functional>
#include <map>
#include <string>
#include <vector>

struct HotkeyBinding {
  std::string action;
  std::string key;
  UINT modifiers = 0;
};

struct HotkeyFailure {
  std::string action;
  std::string key;
  DWORD error_code;
  bool is_default;
};

struct HotkeyRegistration {
  int registered = 0;
  std::vector<HotkeyFailure> failures;
};

class WindowsHotkeys {
 public:
  using Register = std::function<DWORD(HWND, int, UINT, UINT)>;
  using Unregister = std::function<void(HWND, int)>;

  explicit WindowsHotkeys(HWND owner, Register register_key = {},
                          Unregister unregister_key = {}, bool media_key_fallback = true);
  ~WindowsHotkeys();
  HotkeyRegistration SetBindings(const std::vector<HotkeyBinding>& bindings);
  void Suspend();
  std::string ActionForHotkey(int id) const;
  std::string ActionForAppCommand(LPARAM lparam) const;
  static UINT VirtualKey(const std::string& token);

 private:
  void Clear();

  HWND owner_;
  Register register_;
  Unregister unregister_;
  std::map<int, std::string> actions_;
  bool suspended_ = false;
  bool media_key_fallback_;
  std::map<UINT, std::string> media_overrides_;
};

#endif  // RUNNER_WINDOWS_HOTKEYS_H_
