# ORFix Repair v4.2.0 — Bilingual Windows build

An open-source local Windows utility for selected GIMI / XXMI Mod compatibility repairs.

## Download v4.2.0

- **[Windows x64 EXE](https://github.com/wonidie66/ORFix-Repair/releases/download/v4.2.0/ORFixRepair_v4_2_0.exe)**
- [Release notes](https://github.com/wonidie66/ORFix-Repair/releases/tag/v4.2.0)
- SHA-256: `84b13d2a1e21299afb7ff4d255a4a203e6d628caa9fc19176bc2e445de9e8988`

## What's new in v4.2.0

- Adds a generic, signature-based four-slot material recognition and NNFix rule for the verified Diffuse / LightMap / MetalMap / ShadowRamp layout.
- The rule is independent of character names, IB hashes and first-index values; unsupported or incomplete layouts are skipped for manual review.
- Existing ORFix / NNFix, TexFx, Ningguang legacy migration, backup, rollback, Face Fix and bilingual UI workflows are retained.

## Requirements

- Windows 10/11 x64 and Microsoft WebView2 Runtime.
- Go 1.26+ and Wails v2.17.1 to build from source.
- External [ORFix.ini](https://github.com/leotorrez/LeoTools/blob/main/releases/ORFix.ini) for ORFix / NNFix commands to work in-game.
- TexFx-equipped Mods may require [TexFx](https://github.com/SinsOfSeven/TexFx).

## How to use

1. Select a Mod directory or Mods root.
2. Optionally run Preview (no modifications).
3. Run Normal Repair. Backups are created for modified files.
4. Review any Face Fix warning in-game before continuing.
5. Test in-game and use Rollback if needed.

## Source

The source code and regression tests are included in this repository. See `V4.2.0_变更说明.md` and `THIRD_PARTY_NOTICES.md`.
