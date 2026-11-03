#include "flutter_window.h"

#include <optional>
#include <string>

#include "flutter/generated_plugin_registrant.h"
#include <flutter/standard_method_codec.h>
#include <shobjidl.h>

#include "utils.h"

namespace {

using flutter::EncodableList;
using flutter::EncodableMap;
using flutter::EncodableValue;

const EncodableValue* FindMapValue(const EncodableMap& map,
                                   const char* key) {
  auto it = map.find(EncodableValue(std::string(key)));
  return it == map.end() ? nullptr : &it->second;
}

std::string StringFromValue(const EncodableValue* value) {
  if (!value) {
    return std::string();
  }
  const auto* text = std::get_if<std::string>(value);
  return text == nullptr ? std::string() : *text;
}

std::wstring Utf16FromUtf8(const std::string& value) {
  if (value.empty()) return L"";
  const int length = MultiByteToWideChar(
      CP_UTF8, 0, value.data(), static_cast<int>(value.size()), nullptr, 0);
  std::wstring result(static_cast<size_t>(length), L'\0');
  MultiByteToWideChar(CP_UTF8, 0, value.data(), static_cast<int>(value.size()),
                      result.data(), length);
  return result;
}

UINT ModifiersFromList(const EncodableList* values) {
  UINT modifiers = MOD_NOREPEAT;
  if (!values) {
    return modifiers;
  }
  for (const auto& value : *values) {
    const auto modifier = StringFromValue(&value);
    if (modifier == "control") {
      modifiers |= MOD_CONTROL;
    } else if (modifier == "alt") {
      modifiers |= MOD_ALT;
    } else if (modifier == "shift") {
      modifiers |= MOD_SHIFT;
    } else if (modifier == "win") {
      modifiers |= MOD_WIN;
    }
  }
  return modifiers;
}

}  // namespace

FlutterWindow::FlutterWindow(const flutter::DartProject& project)
    : project_(project) {}

FlutterWindow::~FlutterWindow() {}

bool FlutterWindow::OnCreate() {
  if (!Win32Window::OnCreate()) {
    return false;
  }

  RECT frame = GetClientArea();

  // The size here must match the window dimensions to avoid unnecessary surface
  // creation / destruction in the startup path.
  flutter_controller_ = std::make_unique<flutter::FlutterViewController>(
      frame.right - frame.left, frame.bottom - frame.top, project_);
  // Ensure that basic setup of the controller was successful.
  if (!flutter_controller_->engine() || !flutter_controller_->view()) {
    return false;
  }
  RegisterPlugins(flutter_controller_->engine());
  ConfigureHotkeyChannel();
  ConfigureDownloadChannel();
  ConfigureTrayChannel();
  SetChildContent(flutter_controller_->view()->GetNativeWindow());

  flutter_controller_->engine()->SetNextFrameCallback([&]() {
    this->Show();
  });

  // Flutter can complete the first frame before the "show window" callback is
  // registered. The following call ensures a frame is pending to ensure the
  // window is shown. It is a no-op if the first frame hasn't completed yet.
  flutter_controller_->ForceRedraw();

  return true;
}

void FlutterWindow::OnDestroy() {
  hotkeys_ = nullptr;
  media_controls_ = nullptr;
  hotkey_channel_ = nullptr;
  download_channel_ = nullptr;
  tray_channel_ = nullptr;
  tray_menu_ = nullptr;
  if (flutter_controller_) {
    flutter_controller_ = nullptr;
  }

  Win32Window::OnDestroy();
}

LRESULT
FlutterWindow::MessageHandler(HWND hwnd, UINT const message,
                              WPARAM const wparam,
                              LPARAM const lparam) noexcept {
  // Handle our commands before plugin delegation so another window procedure
  // cannot swallow global hotkeys or keyboard media commands.
  if (hotkeys_) {
    std::string action;
    if (message == WM_HOTKEY) {
      action = hotkeys_->ActionForHotkey(static_cast<int>(wparam));
    } else if (message == WM_APPCOMMAND || message == kWindowsMediaCommand) {
      action = hotkeys_->ActionForAppCommand(lparam);
    }
    if (!action.empty()) {
      EmitHotkeyAction(action);
      return message == WM_HOTKEY ? 0 : TRUE;
    }
  }
  if (tray_menu_) {
    const auto result = tray_menu_->HandleMessage(message, wparam, lparam);
    if (result) return *result;
  }
  // Give Flutter, including plugins, an opportunity to handle window messages.
  if (flutter_controller_) {
    std::optional<LRESULT> result =
        flutter_controller_->HandleTopLevelWindowProc(hwnd, message, wparam,
                                                      lparam);
    if (result) {
      return *result;
    }
  }

  switch (message) {
    case WM_FONTCHANGE:
      flutter_controller_->engine()->ReloadSystemFonts();
      break;
  }

  return Win32Window::MessageHandler(hwnd, message, wparam, lparam);
}

void FlutterWindow::ConfigureTrayChannel() {
  tray_menu_ = std::make_shared<TrayMenu>(GetHandle());
  tray_channel_ =
      std::make_unique<flutter::MethodChannel<flutter::EncodableValue>>(
          flutter_controller_->engine()->messenger(),
          "com.wep56.bilitune/tray_menu",
          &flutter::StandardMethodCodec::GetInstance());
  tray_channel_->SetMethodCallHandler(
      [this](const flutter::MethodCall<flutter::EncodableValue>& call,
             std::unique_ptr<flutter::MethodResult<flutter::EncodableValue>>
                 result) {
        const auto menu = tray_menu_;
        if (!menu) {
          result->Success(EncodableValue());
          return;
        }
        if (call.method_name() == "update") {
          const auto* arguments = call.arguments();
          const auto* map = arguments == nullptr
              ? nullptr : std::get_if<EncodableMap>(arguments);
          if (map == nullptr) {
            result->Error("invalid_arguments", "Expected tray playback state");
            return;
          }
          const auto* playing_value = FindMapValue(*map, "isPlaying");
          const auto* playing = playing_value == nullptr
              ? nullptr : std::get_if<bool>(playing_value);
          menu->Update(Utf16FromUtf8(StringFromValue(FindMapValue(*map, "title"))),
                       playing != nullptr && *playing);
          result->Success();
        } else if (call.method_name() == "show") {
          const std::string action = menu->Show();
          result->Success(action.empty() ? EncodableValue() : EncodableValue(action));
        } else {
          result->NotImplemented();
        }
      });
}

void FlutterWindow::ConfigureHotkeyChannel() {
  media_controls_ = std::make_unique<WindowsMediaControls>(GetHandle());
  hotkeys_ = std::make_unique<WindowsHotkeys>(GetHandle(), WindowsHotkeys::Register{},
      WindowsHotkeys::Unregister{}, !media_controls_->available());
  hotkey_channel_ =
      std::make_unique<flutter::MethodChannel<flutter::EncodableValue>>(
          flutter_controller_->engine()->messenger(),
          "com.wep56.bilitune/hotkeys",
          &flutter::StandardMethodCodec::GetInstance());

  hotkey_channel_->SetMethodCallHandler(
      [this](const flutter::MethodCall<flutter::EncodableValue>& call,
             std::unique_ptr<flutter::MethodResult<flutter::EncodableValue>>
                 result) {
        if (call.method_name() == "updatePlayback") {
          const auto* arguments = call.arguments();
          const auto* map = arguments == nullptr ? nullptr : std::get_if<EncodableMap>(arguments);
          if (map == nullptr) {
            result->Error("invalid_arguments", "Expected playback state");
            return;
          }
          const auto* playing_value = FindMapValue(*map, "isPlaying");
          const auto* playing = playing_value == nullptr ? nullptr : std::get_if<bool>(playing_value);
          const auto* track_value = FindMapValue(*map, "hasTrack");
          const auto* has_track = track_value == nullptr ? nullptr : std::get_if<bool>(track_value);
          media_controls_->Update(Utf16FromUtf8(StringFromValue(FindMapValue(*map, "title"))),
              Utf16FromUtf8(StringFromValue(FindMapValue(*map, "artist"))),
              playing != nullptr && *playing, has_track != nullptr && *has_track);
          result->Success();
          return;
        }
        if (call.method_name() == "suspendHotkeys") {
          hotkeys_->Suspend();
          media_controls_->Suspend(true);
          result->Success();
          return;
        }
        if (call.method_name() != "setHotkeys") {
          result->NotImplemented();
          return;
        }

        const auto* arguments = call.arguments();
        const auto* arguments_map = arguments == nullptr
                                        ? nullptr
                                        : std::get_if<EncodableMap>(arguments);
        const auto* bindings_value = arguments_map == nullptr
                                         ? nullptr
                                         : FindMapValue(*arguments_map,
                                                        "bindings");
        const auto* bindings = bindings_value == nullptr
                                   ? nullptr
                                   : std::get_if<EncodableList>(
                                         bindings_value);

        std::vector<HotkeyBinding> requested;
        if (bindings != nullptr) {
          for (const auto& binding_value : *bindings) {
            const auto* binding = std::get_if<EncodableMap>(&binding_value);
            if (binding == nullptr) {
              continue;
            }

            const auto action = StringFromValue(FindMapValue(*binding,
                                                             "action"));
            const auto key = StringFromValue(FindMapValue(*binding, "key"));
            const auto* modifiers_value = FindMapValue(*binding, "modifiers");
            const auto* modifiers = modifiers_value == nullptr
                                        ? nullptr
                                        : std::get_if<EncodableList>(
                                              modifiers_value);

            requested.push_back({action, key, ModifiersFromList(modifiers)});
          }
        }

        const auto registration = hotkeys_->SetBindings(requested);
        media_controls_->Suspend(false);
        EncodableList failures;
        for (const auto& failure : registration.failures) {
          failures.push_back(EncodableValue(EncodableMap{
              {EncodableValue("action"), EncodableValue(failure.action)},
              {EncodableValue("key"), EncodableValue(failure.key)},
              {EncodableValue("errorCode"), EncodableValue(static_cast<int>(failure.error_code))},
              {EncodableValue("isDefault"), EncodableValue(failure.is_default)},
          }));
        }
        result->Success(EncodableValue(EncodableMap{
            {EncodableValue("registered"), EncodableValue(registration.registered)},
            {EncodableValue("failures"), EncodableValue(failures)},
        }));
      });
}

void FlutterWindow::ConfigureDownloadChannel() {
  download_channel_ =
      std::make_unique<flutter::MethodChannel<flutter::EncodableValue>>(
          flutter_controller_->engine()->messenger(),
          "com.wep56.bilitune/downloads",
          &flutter::StandardMethodCodec::GetInstance());

  download_channel_->SetMethodCallHandler(
      [this](const flutter::MethodCall<flutter::EncodableValue>& call,
             std::unique_ptr<flutter::MethodResult<flutter::EncodableValue>>
                 result) {
        if (call.method_name() != "pickDirectory") {
          result->NotImplemented();
          return;
        }

        IFileOpenDialog* dialog = nullptr;
        HRESULT hr = CoCreateInstance(CLSID_FileOpenDialog, nullptr,
                                      CLSCTX_INPROC_SERVER,
                                      IID_PPV_ARGS(&dialog));
        if (FAILED(hr) || dialog == nullptr) {
          result->Success(EncodableValue());
          return;
        }

        DWORD options = 0;
        dialog->GetOptions(&options);
        dialog->SetOptions(options | FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM |
                           FOS_PATHMUSTEXIST);

        hr = dialog->Show(GetHandle());
        if (FAILED(hr)) {
          dialog->Release();
          result->Success(EncodableValue());
          return;
        }

        IShellItem* item = nullptr;
        hr = dialog->GetResult(&item);
        if (SUCCEEDED(hr) && item != nullptr) {
          PWSTR path = nullptr;
          hr = item->GetDisplayName(SIGDN_FILESYSPATH, &path);
          if (SUCCEEDED(hr) && path != nullptr) {
            result->Success(EncodableValue(Utf8FromUtf16(path)));
            CoTaskMemFree(path);
          } else {
            result->Success(EncodableValue());
          }
          item->Release();
        } else {
          result->Success(EncodableValue());
        }

        dialog->Release();
      });
}

void FlutterWindow::EmitHotkeyAction(const std::string& action) {
  if (!hotkey_channel_ || action.empty()) {
    return;
  }
  hotkey_channel_->InvokeMethod(
      "onHotkeyPressed",
      std::make_unique<EncodableValue>(EncodableMap{
          {EncodableValue("action"), EncodableValue(action)},
      }));
}
