# ORFix Repair

A Windows desktop utility for fixing selected compatibility issues in GIMI / XXMI character mods.

**Version:** 4.1.0 | **Platform:** Windows 10/11 x64 | **Source:** Go + Wails v2 + React

> This repository provides publicly reviewable, readable application source code. The maintainer reports compiling the Windows v4.1.0 executable locally; that binary has not been independently matched to this source snapshot. Historical status files in the archive predate the local compilation.

## Overview

ORFix Repair scans a selected Mods folder and subfolders, identifies supported INI compatibility problems, and applies targeted fixes. It is intended for certain green/yellow character textures, material and outline issues, selected transparent materials, and possible facial makeup / texture-binding problems.

The tool is conservative: unsupported or ambiguous configurations are skipped for manual review. It does not rebuild character models, rewrite DDS pixels, or guarantee that every visual problem can be fixed.

## Features

- Scan one Mod folder or an entire Mods directory.
- Preview proposed fixes without modifying files.
- Add appropriate ORFix / NNFix calls for strictly recognized texture configurations.
- Handle certain supported TexFx transparent material configurations.
- Detect possible face texture issues and offer an **optional** Face Fix after the player checks in-game.
- Apply strictly matched legacy special-case repairs.
- Create backups before file modifications and support guarded rollback of the latest repair operation.
- Produce scan and repair logs.

## Requirements

- Windows 10 or Windows 11, x64.
- GIMI / XXMI or a compatible 3DMigoto-based Mod environment.
- [ORFix.ini by LeoTools](https://github.com/leotorrez/LeoTools/blob/main/releases/ORFix.ini) must be properly installed for ORFix / NNFix calls to work **in-game**. It is not bundled with ORFix Repair.
  - XXMI users should normally use the launcher-managed ORFix.ini and avoid duplicate copies inside Mods folders.
  - Manual 3DMigoto users should follow [Leo's ORFix Guide](https://github.com/leotorrez/LeoTools/blob/main/guides/ORFixGuide.md).
- Some transparent Mods also require the appropriate [TexFx](https://github.com/SinsOfSeven/TexFx) library.
- The Windows Wails app requires Microsoft WebView2 Runtime.

## How to use

1. Back up your Mods directory.
2. Open ORFix Repair and select a Mod folder or your Mods root.
3. Click **Preview** to inspect possible fixes without writing any changes.
4. Click **Normal Repair** to apply supported fixes. The app backs up modified files first.
5. If a facial texture risk is shown, check the affected character in-game. Only select **Continue Face Fix** when a visible issue is confirmed; otherwise select **Finish**.
6. Test in-game. Use **Rollback** to undo the latest supported repair operation if required.

## Build from source on Windows

The checked-in frontend under `frontend/dist/` is already prepared for embedding into the Wails executable. The project's `go.mod` specifies Go 1.26.0 and Wails v2.17.1.

1. Install Go 1.26+ and the Windows prerequisites for Wails v2.
2. Clone or download this repository to an ordinary local directory.
3. Run `BUILD_WAILS_WINDOWS.cmd` or follow `BUILD_MANUAL_中文说明.txt`.
4. If the build succeeds, the expected output is `build/bin/ORFixRepair_v4_1_0.exe`.

Detailed notes: [Chinese Windows build and validation guide](README_正式Wails构建及验证.md).

## Source code map

- `main.go` — Wails application entrypoint and embedded frontend.
- `app.go` — desktop UI-to-engine bridge.
- `engine/` — INI parser, compatibility rules, Face Fix, backups, rollback, and tests.
- `frontend/dist/` — readable application JavaScript and CSS, bundled React vendor libraries, and UI assets.
- `build/windows/` — Windows icon.
- `.github/workflows/` — GitHub Actions build definition.
- `BUILD_WAILS_WINDOWS.cmd` / `.ps1` — Windows build scripts.

## Limitations and safety

- **ORFix.ini is an external in-game dependency**, not a requirement for launching this application.
- No promise is made that every Mod can be repaired automatically.
- The program's INI scan cannot prove whether a face is visibly broken; check in-game before optional Face Fix.
- Some old Reflection or custom Shader configurations require manual repair.
- Automatic backups and integrity checks are not a substitute for a separate Mods backup.
- `RELEASE_STATUS.json` is a historical packaging snapshot, not confirmation that the maintainer did or did not later compile the EXE.
- Check your local build and the distributed binary for correspondence; builds made in different environments are not necessarily byte-identical.

## Attribution and licensing

This is an independent community project, not affiliated with the game developer, XXMI, LeoTools, or the original Mod authors.

- React / ReactDOM and the adapted glass UI retain their third-party MIT notices in `frontend/dist/vendor/`.
- The app does not include ORFix.ini or TexFx global libraries.
- The included Mod-specific example/fix under `engine/known_fixes/`, character illustrations and application icon may involve third-party rights. **Permission and any separate licenses must be confirmed before claiming they are covered by a project-wide license.**
- The repository is published for source review; a project-wide open-source license has not yet been selected.

See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for dependencies and ownership checks.
