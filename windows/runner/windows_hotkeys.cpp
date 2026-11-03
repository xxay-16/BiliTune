#include "windows_hotkeys.h"

#include <set>
#include <utility>

namespace {
const HotkeyBinding kMediaKeys[] = {
    {"playPause", "MediaPlayPause", 0},
    {"previousTrack", "MediaPreviousTrack", 0},
    {"nextTrack", "MediaNextTrack", 0},
    {"stop", "MediaStop", 0},
};
}  // namespace

WindowsHotkeys::WindowsHotkeys(HWND owner, Register register_key,
                               Unregister unregister_key, bool media_key_fallback)
    : owner_(owner), register_(std::move(register_key)),
      unregister_(std::move(unregister_key)), media_key_fallback_(media_key_fallback) {
  if (!register_) {
    register_ = [](HWND window, int id, UINT modifiers, UINT key) -> DWORD {
      if (RegisterHotKey(window, id, modifiers, key)) return ERROR_SUCCESS;
      const DWORD error = GetLastError();
      return error == ERROR_SUCCESS ? ERROR_GEN_FAILURE : error;
    };
  }
  if (!unregister_) {
    unregister_ = [](HWND window, int id) { UnregisterHotKey(window, id); };
  }
}

WindowsHotkeys::~WindowsHotkeys() { Clear(); }

HotkeyRegistration WindowsHotkeys::SetBindings(
    const std::vector<HotkeyBinding>& bindings) {
  Clear();
  suspended_ = false;
  HotkeyRegistration result;
  std::set<std::pair<UINT, UINT>> assigned;
  int id = 5600;
  const auto add = [&](const HotkeyBinding& binding, bool is_default) {
    const UINT key = VirtualKey(binding.key);
    const UINT modifiers = binding.modifiers | MOD_NOREPEAT;
    const auto signature = std::make_pair(modifiers, key);
    // Explicit bindings take precedence over the same default media key.
    if (is_default && assigned.count(signature)) return;
    DWORD error = ERROR_INVALID_PARAMETER;
    if (!binding.action.empty() && key != 0) {
      if (assigned.insert(signature).second) {
        if (!media_key_fallback_ && (binding.modifiers & ~MOD_NOREPEAT) == 0 &&
            key >= VK_MEDIA_NEXT_TRACK && key <= VK_MEDIA_PLAY_PAUSE) {
          media_overrides_[key] = binding.action;
          result.registered++;
          return;
        }
        error = register_(owner_, id, modifiers, key);
      } else {
        error = ERROR_HOTKEY_ALREADY_REGISTERED;
      }
    }
    if (error == ERROR_SUCCESS) {
      actions_[id] = binding.action;
      result.registered++;
    } else {
      result.failures.push_back(
          {binding.action, binding.key, error, is_default});
    }
    id++;
  };
  for (const auto& binding : bindings) add(binding, false);
  if (media_key_fallback_) {
    for (const auto& binding : kMediaKeys) add(binding, true);
  }
  return result;
}

void WindowsHotkeys::Clear() {
  for (const auto& entry : actions_) unregister_(owner_, entry.first);
  actions_.clear();
  media_overrides_.clear();
}

void WindowsHotkeys::Suspend() {
  suspended_ = true;
  Clear();
}

std::string WindowsHotkeys::ActionForHotkey(int id) const {
  const auto found = actions_.find(id);
  return found == actions_.end() ? "" : found->second;
}

std::string WindowsHotkeys::ActionForAppCommand(LPARAM lparam) const {
  if (suspended_) return "";
  UINT key = 0;
  switch (GET_APPCOMMAND_LPARAM(lparam)) {
    case APPCOMMAND_MEDIA_PLAY_PAUSE: key = VK_MEDIA_PLAY_PAUSE; break;
    case APPCOMMAND_MEDIA_PREVIOUSTRACK: key = VK_MEDIA_PREV_TRACK; break;
    case APPCOMMAND_MEDIA_NEXTTRACK: key = VK_MEDIA_NEXT_TRACK; break;
    case APPCOMMAND_MEDIA_STOP: key = VK_MEDIA_STOP; break;
  }
  const auto overridden = media_overrides_.find(key);
  if (overridden != media_overrides_.end()) return overridden->second;
  switch (GET_APPCOMMAND_LPARAM(lparam)) {
    case APPCOMMAND_MEDIA_PLAY_PAUSE: return "playPause";
    case APPCOMMAND_MEDIA_PLAY: return "play";
    case APPCOMMAND_MEDIA_PAUSE: return "pause";
    case APPCOMMAND_MEDIA_PREVIOUSTRACK: return "previousTrack";
    case APPCOMMAND_MEDIA_NEXTTRACK: return "nextTrack";
    case APPCOMMAND_MEDIA_STOP: return "stop";
    default: return "";
  }
}

UINT WindowsHotkeys::VirtualKey(const std::string& key) {
  if (key.size() == 1) {
    const char ch = key[0];
    if (ch >= 'A' && ch <= 'Z') return static_cast<UINT>(ch);
    if (ch >= 'a' && ch <= 'z') return static_cast<UINT>(ch - 'a' + 'A');
    if (ch >= '0' && ch <= '9') return static_cast<UINT>(ch);
  }
  if (key.size() >= 2 && key.size() <= 3 && key[0] == 'F') {
    int index = 0;
    for (size_t i = 1; i < key.size(); i++) {
      if (key[i] < '0' || key[i] > '9') return 0;
      index = index * 10 + key[i] - '0';
    }
    if (index >= 1 && index <= 24) return VK_F1 + index - 1;
  }
  if (key.size() == 7 && key.compare(0, 6, "Numpad") == 0 &&
      key[6] >= '0' && key[6] <= '9') {
    return VK_NUMPAD0 + key[6] - '0';
  }
  static const std::map<std::string, UINT> keys = {
      {"Space", VK_SPACE}, {"Enter", VK_RETURN}, {"Escape", VK_ESCAPE},
      {"Backspace", VK_BACK}, {"Tab", VK_TAB}, {"ArrowLeft", VK_LEFT},
      {"ArrowRight", VK_RIGHT}, {"ArrowUp", VK_UP}, {"ArrowDown", VK_DOWN},
      {"Home", VK_HOME}, {"End", VK_END}, {"PageUp", VK_PRIOR},
      {"PageDown", VK_NEXT}, {"Insert", VK_INSERT}, {"Delete", VK_DELETE},
      {"MediaPlayPause", VK_MEDIA_PLAY_PAUSE}, {"MediaStop", VK_MEDIA_STOP},
      {"MediaNextTrack", VK_MEDIA_NEXT_TRACK},
      {"MediaPreviousTrack", VK_MEDIA_PREV_TRACK},
      {"NumpadAdd", VK_ADD}, {"NumpadSubtract", VK_SUBTRACT},
      {"NumpadMultiply", VK_MULTIPLY}, {"NumpadDivide", VK_DIVIDE},
      {"NumpadDecimal", VK_DECIMAL},
      {"Semicolon", VK_OEM_1}, {"Equal", VK_OEM_PLUS}, {"Comma", VK_OEM_COMMA},
      {"Minus", VK_OEM_MINUS}, {"Period", VK_OEM_PERIOD}, {"Slash", VK_OEM_2},
      {"Backquote", VK_OEM_3}, {"BracketLeft", VK_OEM_4},
      {"Backslash", VK_OEM_5}, {"BracketRight", VK_OEM_6}, {"Quote", VK_OEM_7},
  };
  const auto found = keys.find(key);
  return found == keys.end() ? 0 : found->second;
}
