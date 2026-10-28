#include "tray_menu.h"

#include "tray_menu_animation.h"

namespace {

constexpr UINT kTitle = 0xB170;
constexpr UINT kPrevious = 0xB171;
constexpr UINT kPlayPause = 0xB172;
constexpr UINT kNext = 0xB173;
constexpr UINT kShow = 0xB174;
constexpr UINT kHide = 0xB175;
constexpr UINT kExit = 0xB176;
constexpr UINT_PTR kAnimationTimer = 0xB177;

std::string ActionForCommand(UINT command) {
  switch (command) {
    case kPrevious: return "previous";
    case kPlayPause: return "playPause";
    case kNext: return "next";
    case kShow: return "show";
    case kHide: return "hide";
    case kExit: return "exit";
    default: return "";
  }
}

}  // namespace

TrayMenu::TrayMenu(HWND owner) : owner_(owner) {}

TrayMenu::~TrayMenu() {
  Close();
}

int TrayMenu::Scale(int value) const {
  return MulDiv(value, static_cast<int>(dpi_), 96);
}

void TrayMenu::Update(const std::wstring& title, bool is_playing) {
  if (title_ != title) {
    title_ = title.empty() ? L"BiliTune" : title;
    started_at_ = GetTickCount64();
    if (menu_) MeasureTitle();
  }
  is_playing_ = is_playing;
  if (menu_) {
    MENUITEMINFOW info{};
    info.cbSize = sizeof(info);
    info.fMask = MIIM_STRING;
    info.dwTypeData = const_cast<wchar_t*>(
        is_playing_ ? L"\u6682\u505c" : L"\u64ad\u653e");
    SetMenuItemInfoW(menu_, kPlayPause, FALSE, &info);
    SyncTimer();
    RedrawTitle();
  }
}

std::string TrayMenu::Show() {
  if (menu_) return "";
  dpi_ = GetDpiForWindow(owner_);
  if (dpi_ == 0) dpi_ = 96;

  NONCLIENTMETRICSW metrics{};
  metrics.cbSize = sizeof(metrics);
  if (SystemParametersInfoForDpi(SPI_GETNONCLIENTMETRICS, sizeof(metrics),
                                 &metrics, 0, dpi_)) {
    font_ = CreateFontIndirectW(&metrics.lfMenuFont);
  }
  BOOL animations = TRUE;
  SystemParametersInfoW(SPI_GETCLIENTAREAANIMATION, 0, &animations, 0);
  animate_ = animations != FALSE;

  menu_ = CreatePopupMenu();
  if (!menu_) {
    Close();
    return "";
  }
  MENUINFO menu_info{};
  menu_info.cbSize = sizeof(menu_info);
  menu_info.fMask = MIM_STYLE;
  menu_info.dwStyle = MNS_NOCHECK;
  SetMenuInfo(menu_, &menu_info);

  AppendMenuW(menu_, MF_OWNERDRAW | MF_DISABLED, kTitle,
              reinterpret_cast<LPCWSTR>(this));
  AppendMenuW(menu_, MF_SEPARATOR, 0, nullptr);
  AppendMenuW(menu_, MF_STRING, kPrevious, L"\u4e0a\u4e00\u9996");
  AppendMenuW(menu_, MF_STRING, kPlayPause,
              is_playing_ ? L"\u6682\u505c" : L"\u64ad\u653e");
  AppendMenuW(menu_, MF_STRING, kNext, L"\u4e0b\u4e00\u9996");
  AppendMenuW(menu_, MF_SEPARATOR, 0, nullptr);
  AppendMenuW(menu_, MF_STRING, kShow,
              L"\u663e\u793a\u4e3b\u7a97\u53e3");
  AppendMenuW(menu_, MF_STRING, kHide,
              L"\u9690\u85cf\u5230\u6258\u76d8");
  AppendMenuW(menu_, MF_SEPARATOR, 0, nullptr);
  AppendMenuW(menu_, MF_STRING, kExit, L"\u9000\u51fa");

  MeasureTitle();
  started_at_ = GetTickCount64();
  SyncTimer();
  POINT cursor{};
  GetCursorPos(&cursor);
  // A tray menu must own foreground focus to dismiss on an outside click.
  SetForegroundWindow(owner_);
  const UINT command = static_cast<UINT>(TrackPopupMenuEx(
      menu_, TPM_RETURNCMD | TPM_BOTTOMALIGN | TPM_LEFTALIGN | TPM_RIGHTBUTTON,
      cursor.x, cursor.y, owner_, nullptr));
  Close();
  PostMessageW(owner_, WM_NULL, 0, 0);
  return ActionForCommand(command);
}

void TrayMenu::MeasureTitle() {
  HDC dc = GetDC(owner_);
  const int saved = SaveDC(dc);
  SelectObject(dc, font_ ? font_ : GetStockObject(DEFAULT_GUI_FONT));
  SIZE size{};
  GetTextExtentPoint32W(dc, title_.c_str(), static_cast<int>(title_.size()), &size);
  text_width_ = size.cx;
  RestoreDC(dc, saved);
  ReleaseDC(owner_, dc);
}

void TrayMenu::DrawTitle(const DRAWITEMSTRUCT& item) {
  const int saved = SaveDC(item.hDC);
  FillRect(item.hDC, &item.rcItem, GetSysColorBrush(COLOR_MENU));
  SelectObject(item.hDC, font_ ? font_ : GetStockObject(DEFAULT_GUI_FONT));
  SetBkMode(item.hDC, TRANSPARENT);
  SetTextColor(item.hDC, GetSysColor(COLOR_GRAYTEXT));
  RECT clip = item.rcItem;
  clip.left += Scale(tray_menu::kPaddingDip);
  clip.right -= Scale(tray_menu::kPaddingDip);
  IntersectClipRect(item.hDC, clip.left, clip.top, clip.right, clip.bottom);

  const int available = clip.right - clip.left;
  const int offset = animate_
      ? tray_menu::TitleOffset(GetTickCount64() - started_at_, text_width_,
                               available, static_cast<double>(dpi_) / 96)
      : 0;
  RECT text = clip;
  text.left -= offset;
  if (text_width_ > available && animate_) {
    text.right = text.left + text_width_ + Scale(4);
    DrawTextW(item.hDC, title_.c_str(), static_cast<int>(title_.size()), &text,
               DT_SINGLELINE | DT_VCENTER | DT_NOPREFIX);
    OffsetRect(&text, text_width_ + Scale(tray_menu::kGapDip), 0);
  }
  DrawTextW(item.hDC, title_.c_str(), static_cast<int>(title_.size()), &text,
             DT_SINGLELINE | DT_VCENTER | DT_NOPREFIX |
                 (animate_ ? 0 : DT_END_ELLIPSIS));
  RestoreDC(item.hDC, saved);
}

void TrayMenu::RedrawTitle() {
  if (!menu_ || !popup_ || !IsWindow(popup_)) return;
  RECT rect{};
  if (!GetMenuItemRect(owner_, menu_, 0, &rect)) return;
  MapWindowPoints(HWND_DESKTOP, popup_, reinterpret_cast<POINT*>(&rect), 2);
  InvalidateRect(popup_, &rect, FALSE);
  UpdateWindow(popup_);
}

void TrayMenu::SyncTimer() {
  if (menu_ && animate_ &&
      text_width_ > Scale(tray_menu::kWidthDip - 2 * tray_menu::kPaddingDip)) {
    SetTimer(owner_, kAnimationTimer, 33, nullptr);
  } else {
    KillTimer(owner_, kAnimationTimer);
  }
}

void TrayMenu::Close() {
  KillTimer(owner_, kAnimationTimer);
  popup_ = nullptr;
  if (menu_) DestroyMenu(menu_);
  menu_ = nullptr;
  if (font_) DeleteObject(font_);
  font_ = nullptr;
}

std::optional<LRESULT> TrayMenu::HandleMessage(UINT message, WPARAM wparam,
                                              LPARAM lparam) {
  if (!menu_) return std::nullopt;
  if (message == WM_MEASUREITEM) {
    auto* item = reinterpret_cast<MEASUREITEMSTRUCT*>(lparam);
    if (item && item->CtlType == ODT_MENU &&
        item->itemData == reinterpret_cast<ULONG_PTR>(this)) {
      item->itemWidth = static_cast<UINT>(Scale(tray_menu::kWidthDip));
      item->itemHeight = static_cast<UINT>(Scale(30));
      return TRUE;
    }
  } else if (message == WM_DRAWITEM) {
    auto* item = reinterpret_cast<DRAWITEMSTRUCT*>(lparam);
    if (item && item->CtlType == ODT_MENU &&
        item->itemData == reinterpret_cast<ULONG_PTR>(this)) {
      DrawTitle(*item);
      return TRUE;
    }
  } else if (message == WM_ENTERIDLE && wparam == MSGF_MENU) {
    popup_ = reinterpret_cast<HWND>(lparam);
    return 0;
  } else if (message == WM_TIMER && wparam == kAnimationTimer) {
    RedrawTitle();
    return 0;
  }
  return std::nullopt;
}
