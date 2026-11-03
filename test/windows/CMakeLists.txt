cmake_minimum_required(VERSION 3.14)
project(bilitune_hotkey_tests LANGUAGES CXX)
enable_testing()

add_executable(windows_hotkeys_test
  ../windows_hotkeys_native_test.cpp
  ../../windows/runner/windows_hotkeys.cpp
)
target_compile_features(windows_hotkeys_test PRIVATE cxx_std_17)
target_compile_definitions(windows_hotkeys_test PRIVATE UNICODE _UNICODE NOMINMAX)
target_compile_options(windows_hotkeys_test PRIVATE /W4 /WX /EHsc)
target_link_libraries(windows_hotkeys_test PRIVATE user32)
add_test(NAME windows_hotkey_registration COMMAND windows_hotkeys_test)
