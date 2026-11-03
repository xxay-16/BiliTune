#include <windows.h>

// CMake's Release configuration must still execute the regression assertions.
#ifdef NDEBUG
#undef NDEBUG
#endif
#include <cassert>
#include <iostream>
#include <tuple>
#include <vector>

#include "../windows/runner/windows_hotkeys.h"

int main() {
  using Attempt = std::tuple<int, UINT, UINT>;
  std::vector<Attempt> attempts;
  std::vector<int> removed;
  UINT blocked_key = 0;
  const auto register_key = [&](HWND, int id, UINT modifiers, UINT key) -> DWORD {
    attempts.emplace_back(id, modifiers, key);
    return key == blocked_key ? ERROR_HOTKEY_ALREADY_REGISTERED : ERROR_SUCCESS;
  };
  const auto unregister_key = [&](HWND, int id) { removed.push_back(id); };

  {
    WindowsHotkeys keys(nullptr, register_key, unregister_key);
    const auto defaults = keys.SetBindings({});
    assert(defaults.registered == 4 && defaults.failures.empty());
    assert(keys.ActionForHotkey(5600) == "playPause");
    assert(keys.ActionForHotkey(5601) == "previousTrack");
    assert(keys.ActionForHotkey(5602) == "nextTrack");
    assert(keys.ActionForHotkey(5603) == "stop");
    assert((std::get<1>(attempts.front()) & MOD_NOREPEAT) != 0);

    attempts.clear();
    removed.clear();
    blocked_key = 'P';
    const auto result = keys.SetBindings({
        {"playPause", "P", MOD_CONTROL | MOD_ALT},
        {"nextTrack", "ArrowRight", MOD_CONTROL | MOD_ALT},
    });
    assert(removed.size() == 4);
    assert(result.registered == 5 && result.failures.size() == 1);
    assert(result.failures[0].action == "playPause");
    assert(result.failures[0].error_code == ERROR_HOTKEY_ALREADY_REGISTERED);
    assert(!result.failures[0].is_default);
    assert(keys.ActionForHotkey(5600).empty());
    assert(keys.ActionForHotkey(5601) == "nextTrack");
    assert(std::get<1>(attempts.front()) == (MOD_CONTROL | MOD_ALT | MOD_NOREPEAT));

    blocked_key = 0;
    const auto remapped = keys.SetBindings({{"previousTrack", "MediaNextTrack", 0}});
    assert(remapped.registered == 4 && remapped.failures.empty());
    assert(keys.ActionForHotkey(5600) == "previousTrack");
    assert(keys.ActionForAppCommand(static_cast<LPARAM>(APPCOMMAND_MEDIA_STOP) << 16) == "stop");
    assert(keys.ActionForAppCommand(static_cast<LPARAM>(APPCOMMAND_MEDIA_PLAY) << 16) == "play");
    assert(keys.ActionForAppCommand(static_cast<LPARAM>(APPCOMMAND_MEDIA_PAUSE) << 16) == "pause");
    assert(keys.ActionForAppCommand(static_cast<LPARAM>(APPCOMMAND_VOLUME_UP) << 16).empty());

    keys.Suspend();
    assert(keys.ActionForHotkey(5600).empty());
    assert(keys.ActionForAppCommand(static_cast<LPARAM>(APPCOMMAND_MEDIA_PLAY_PAUSE) << 16).empty());
    assert(keys.SetBindings({}).registered == 4);
    assert(keys.ActionForHotkey(5600) == "playPause");

    const auto invalid = keys.SetBindings({{"playPause", "F99999999999999999", 0}});
    assert(invalid.failures.size() == 1);
    assert(invalid.failures[0].error_code == ERROR_INVALID_PARAMETER);
  }

  {
    attempts.clear();
    WindowsHotkeys keys(nullptr, register_key, unregister_key, false);
    assert(keys.SetBindings({}).registered == 0);
    assert(attempts.empty());
    assert(keys.ActionForAppCommand(static_cast<LPARAM>(APPCOMMAND_MEDIA_NEXTTRACK) << 16) == "nextTrack");
    const auto result = keys.SetBindings({{"stop", "MediaStop", 0}});
    assert(result.registered == 1 && attempts.empty());
    assert(keys.ActionForAppCommand(static_cast<LPARAM>(APPCOMMAND_MEDIA_STOP) << 16) == "stop");
  }

  assert(WindowsHotkeys::VirtualKey("0") == '0');
  assert(WindowsHotkeys::VirtualKey("Numpad0") == VK_NUMPAD0);
  assert(WindowsHotkeys::VirtualKey("Numpad9") == VK_NUMPAD9);
  assert(WindowsHotkeys::VirtualKey("F24") == VK_F24);
  assert(WindowsHotkeys::VirtualKey("F0") == 0);
  assert(WindowsHotkeys::VirtualKey("Period") == VK_OEM_PERIOD);
  std::cout << "Windows hotkey registration, media commands, conflict and recording checks passed.\n";
}
