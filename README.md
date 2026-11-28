# ORFix Repair V5.0.0

A standalone offline repair utility for compatible Genshin Impact character mods using 3DMigoto / GIMI / XXMI. This version replaces the previous Go/Wails interface and engine with Python and native PySide6/Qt Quick QML.

## Features
- Improved ORFix/NNFix material detection, including compatible multi-slot, conditional and variant bindings.
- Separate, confirmation-based facial repair; preview, automatic backup and rollback.
- Detailed text and JSON diagnostics, with rule identifiers and repair evidence.
- Native Liquid Glass-inspired interface, English/Chinese language switching and native folder selection.
- Reduced Windows executable size (46.8 MB reported by the maintainer on the standard Slim2 build).

## Usage
Select a mod folder, choose Start Repair or Preview, then check the result in game. Use facial repair only when the corresponding problem is observed. Rollback restores the latest backed-up batch if integrity checks pass. Use a copy of the mod when testing unfamiliar configurations.

To run the source, use 64-bit Python 3.11–3.13 and install the dependencies:

```sh
python -m pip install -r requirements.txt
python main.py
```

The optional optical shader is compiled with the installed Qt qsb tool; `python main.py --safe-ui` uses the native fallback. The public repository contains source and runtime resources only; private build scripts, test fixtures, and research records are intentionally not distributed here. The Windows executable, if distributed, runs without a separately installed Python environment.

## Limitations
Only supported, sufficiently identified configurations are changed. This is not a universal mod repair or a universal transparency fix. FrameAnalysis advanced diagnostics is not included in V5.0.0. The maintainer reports that the standard Windows build works in general; compatibility with all mods and systems is not guaranteed.

External ORFix/NNFix and TexFx libraries must be provided by the supported mod framework. See THIRD_PARTY_NOTICES.md for attribution and licensing boundaries.

## 中文说明
V5.0.0 使用 Python + 原生 QML 重构；保留正常修复、独立面部修复、预览、备份回滚及双语界面，改进复杂材质、条件分支和日志诊断。请在游戏内确认结果；不保证所有 Mod 或所有渲染场景均能修复。公开仓库仅提供源代码及必要运行资源，不包含内部构建脚本、研发文档和测试样本。FrameAnalysis 高级诊断尚未集成。
