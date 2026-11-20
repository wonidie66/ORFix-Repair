# Third-party credits and permissions

Publishing this repository does not assert that every included third-party asset is owned by the maintainer.

## Bundled software libraries

- React / ReactDOM — MIT licensed. Upstream: https://react.dev/ . Bundled notice: `frontend/dist/vendor/LICENSE_React_MIT.txt`.
- SVG liquid-glass adaptation — based on `yurkagon/react-liquid-glass-svg` (MIT). Upstream: https://github.com/yurkagon/react-liquid-glass-svg . Bundled notice: `frontend/dist/vendor/LICENSE_Glass_MIT.txt`.

## External libraries and references (not bundled as game-global fixes)

- Wails: https://wails.io/
- Go: https://go.dev/
- Microsoft WebView2 Runtime: https://learn.microsoft.com/microsoft-edge/webview2/
- LeoTools / ORFix.ini and its guide: https://github.com/leotorrez/LeoTools
- TexFx: https://github.com/SinsOfSeven/TexFx
- Gi-FaceFix: https://github.com/Sanddino00/Gi-FaceFix
- GI Model Importer: https://github.com/SilentNightSound/GI-Model-Importer

## Distribution-rights checks

- `engine/known_fixes/Citlali_high_priest_verified_v2.ini` is a bundled character-specific patched Mod configuration. Its original author and permission to redistribute the full INI have not been independently confirmed. It is embedded by `engine/recipes.go`; deleting it without modifying the Go source would break the build.
- `frontend/dist/assets/` and `build/windows/icon.ico` may include game-derived artwork. Confirm permission to distribute these assets; a license for your own application code does not cover third-party art.
- A locally built v4.1.0 EXE should be checked against the source snapshot distributed here.

ORFix.ini, TexFx, game files, and separately authored Mod assets are not relicensed by this repository. A project-level license should only be added once the rights to the original project code and exceptions are clear.
