# ORFix Repair v4.1.2 — Bilingual Windows build

An open-source, local Windows tool for selected GIMI / XXMI Mod compatibility fixes.

## Download v4.1.2

- **[Windows x64 EXE](https://github.com/wonidie66/ORFix-Repair/releases/download/v4.1.2/ORFixRepair_v4_1_2.exe)**
- [Release notes](https://github.com/wonidie66/ORFix-Repair/releases/tag/v4.1.2)
- SHA-256: See the V4.1.2 GitHub Release page (generated from the published build).

The V4.1.2 release workflow passed the folder-path regression test and Go repair-engine tests, then built and published the Windows x64 EXE. Full in-game rendering verification is still a separate user-side check.

**What's new in v4.1.2:** Fixes a bug where an English UI could replace a selected folder path containing Chinese characters with a generic diagnostic message. Windows drive paths, UNC paths, POSIX absolute paths, and relative paths are now preserved verbatim. The Chinese / English switch, original V4.1.0 layout, and repair engine remain unchanged.

## Requirements
- Windows 10/11 x64, Microsoft WebView2 Runtime.
- External [ORFix.ini](https://github.com/leotorrez/LeoTools/blob/main/releases/ORFix.ini) for ORFix / NNFix commands to work in-game. GIMI/XXMI global installation details: [Leo's guide](https://github.com/leotorrez/LeoTools/blob/main/guides/ORFixGuide.md).
- TexFx-equipped Mods may require [TexFx](https://github.com/SinsOfSeven/TexFx).

## How to use
1. Select a Mod directory or Mods root.
2. Optionally use Preview (no modifications).
3. Run Normal Repair. Backups are created for modified files.
4. Review any face warning **in-game**; Continue Face Fix only after confirming the problem.
5. Test in-game. Roll back the last batch if necessary.

## Language
Press **中文** or **EN** in the top-right header at any time. Preference is stored in the local WebView profile. The language switch affects the application UI, not Mod files or the repair engine's saved raw logs. System file-dialog chrome can follow the Windows display language.

## Build
Go 1.26+, Wails v2.17.1. Prebuilt frontend assets are in `frontend/dist/`; see `BUILD_MANUAL_中文说明.txt` or `BUILD_WAILS_WINDOWS.cmd`. Output: `build/bin/ORFixRepair_v4_1_2.exe`.

## Source structure
- `main.go`, `app.go` — Wails host and bridge.
- `engine/` — original Mod scan/repair and rollback code.
- `frontend/dist/app.js` — React UI and event workflow.
- `frontend/dist/language.js` — bilingual React translation layer.
- `frontend/dist/language.svg` — language icon supplied for this version.
- `frontend/dist/style.css` — V4.1.0 styles + compact language switch styling.
- `frontend/dist/vendor/` — vendor JS and MIT license notices.

## Safety and licensing
The software is provided as an independent community utility, not endorsed by the game or XXMI. ORFix.ini and TexFx are not bundled. The public source is reviewable; rights to individual Mod-derived examples and artwork should be verified separately before applying a repository-wide license. See `THIRD_PARTY_NOTICES.md`.
