#include "windows_media_controls.h"

#include <SystemMediaTransportControlsInterop.h>
#include <roapi.h>
#include <winrt/Windows.Foundation.h>
#include <winrt/Windows.Media.h>

using namespace winrt::Windows::Media;

struct WindowsMediaControls::Impl {
  SystemMediaTransportControls controls{nullptr};
  winrt::event_token button_token{};
  bool has_track = false;
  bool suspended = false;
  bool apartment_initialized = false;
};

WindowsMediaControls::WindowsMediaControls(HWND owner)
    : impl_(std::make_unique<Impl>()) {
  try {
    winrt::check_hresult(RoInitialize(RO_INIT_SINGLETHREADED));
    impl_->apartment_initialized = true;
    const auto factory = winrt::get_activation_factory<
        SystemMediaTransportControls, ISystemMediaTransportControlsInterop>();
    winrt::check_hresult(factory->GetForWindow(
        owner, winrt::guid_of<SystemMediaTransportControls>(),
        winrt::put_abi(impl_->controls)));
    impl_->controls.IsEnabled(false);
    impl_->controls.IsPlayEnabled(true);
    impl_->controls.IsPauseEnabled(true);
    impl_->controls.IsStopEnabled(true);
    impl_->controls.IsPreviousEnabled(true);
    impl_->controls.IsNextEnabled(true);
    impl_->button_token = impl_->controls.ButtonPressed(
        [owner](const auto&, const SystemMediaTransportControlsButtonPressedEventArgs& event) {
          UINT command = 0;
          switch (event.Button()) {
            case SystemMediaTransportControlsButton::Play: command = APPCOMMAND_MEDIA_PLAY; break;
            case SystemMediaTransportControlsButton::Pause: command = APPCOMMAND_MEDIA_PAUSE; break;
            case SystemMediaTransportControlsButton::Stop: command = APPCOMMAND_MEDIA_STOP; break;
            case SystemMediaTransportControlsButton::Previous: command = APPCOMMAND_MEDIA_PREVIOUSTRACK; break;
            case SystemMediaTransportControlsButton::Next: command = APPCOMMAND_MEDIA_NEXTTRACK; break;
            default: return;
          }
          // WinRT callbacks may run off the platform thread. Route through the
          // owner window before invoking the Flutter method channel.
          PostMessageW(owner, kWindowsMediaCommand, 0,
                        static_cast<LPARAM>(command) << 16);
        });
  } catch (const winrt::hresult_error&) {
    impl_->controls = nullptr;
  }
}

WindowsMediaControls::~WindowsMediaControls() {
  if (available()) {
    try {
      impl_->controls.ButtonPressed(impl_->button_token);
      impl_->controls.IsEnabled(false);
    } catch (const winrt::hresult_error&) {
    }
    impl_->controls = nullptr;
  }
  if (impl_->apartment_initialized) RoUninitialize();
}

bool WindowsMediaControls::available() const {
  return impl_->controls != nullptr;
}

void WindowsMediaControls::Update(const std::wstring& title,
                                   const std::wstring& artist,
                                   bool is_playing, bool has_track) {
  if (!available()) return;
  try {
    impl_->has_track = has_track;
    impl_->controls.IsEnabled(has_track && !impl_->suspended);
    impl_->controls.PlaybackStatus(!has_track ? MediaPlaybackStatus::Closed :
        (is_playing ? MediaPlaybackStatus::Playing : MediaPlaybackStatus::Paused));
    const auto display = impl_->controls.DisplayUpdater();
    display.Type(MediaPlaybackType::Music);
    display.MusicProperties().Title(title);
    display.MusicProperties().Artist(artist);
    display.Update();
  } catch (const winrt::hresult_error&) {
    // A shell restart must not interrupt audio playback.
  }
}

void WindowsMediaControls::Suspend(bool suspended) {
  if (!available()) return;
  impl_->suspended = suspended;
  try {
    impl_->controls.IsEnabled(impl_->has_track && !suspended);
  } catch (const winrt::hresult_error&) {
  }
}
