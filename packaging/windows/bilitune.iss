#define MyAppName "BiliTune"
#define MyAppExeName "bilitune.exe"
#ifndef AppVersion
#define AppVersion "0.2.0"
#endif
#ifndef SourceDir
#define SourceDir "..\..\bin"
#endif
#ifndef OutputDir
#define OutputDir "..\..\dist\windows"
#endif
[Setup]
AppId={{9B806D84-4E60-42B0-9F48-75CF019395E4}
AppName={#MyAppName}
AppVersion={#AppVersion}
AppPublisher=BiliTune contributors
AppPublisherURL=https://github.com/xxay-16/BiliTune
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
OutputDir={#OutputDir}
OutputBaseFilename=BiliTune-Setup-{#AppVersion}
SetupIconFile=icon.ico
UninstallDisplayIcon={app}\{#MyAppExeName}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0
CloseApplications=yes
[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
[Files]
Source: "{#SourceDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs
[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon
[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent
