# ORFix Repair V4.1.2 — Wails Windows x64 构建与验收记录

## 版本变更

V4.1.2 在 V4.1.1 双语版本基础上修复中文目录路径在英文界面被误替换为诊断兜底文案的问题。翻译层先识别 Windows 盘符路径、UNC 路径、POSIX 绝对路径和相对路径，再执行文本翻译；路径及其中的中文字符保持原样。原页面结构、窗口布局和 Go 修复规则保持不变。

## V4.1.2 修复说明

回归测试 `frontend/tests/language.test.cjs` 覆盖 Windows 中文目录、正斜杠盘符路径、UNC 网络路径、POSIX 绝对路径、相对路径，同时检查 Browse Folder 等 UI 标签仍正常翻译、未知诊断文本仍使用安全兜底提示。构建工作流会在 Windows runner 中运行该测试。

## 构建与验收状态

V4.1.2 的 Windows EXE 由新发布工作流生成后，才可在下方记录 Release 地址、文件哈希和实际构建状态。

- GitHub Actions workflow：`Publish ORFix Repair v4.1.2`
- Workflow run：成功（`completed / success`）
- Windows 构建平台：`windows-latest`
- Go：1.26.x
- Wails：v2.17.1
- EXE：`ORFixRepair_v4_1_2.exe`
- PE：PE32+ Windows x86-64 GUI executable
- Go build info：go1.26.10、GOOS=windows、GOARCH=amd64、desktop / wv2runtime.download / production tags
- SHA-256：`825a8dfb14901643585eb3a2f9632fe1a907e26d551db2fb03b093c3fc42633d`
- Release：<https://github.com/wonidie66/ORFix-Repair/releases/tag/v4.1.2>

以上证明 Windows x64 Wails 构建已由 GitHub Actions 完成并发布。**它不等同于在用户的 Windows 设备上完成 GUI 交互验收，也不等同于游戏内 Mod 画面验收。**

## 构建步骤（自行复现）

1. Windows 10/11 x64，安装 Go 1.26.x、Wails v2.17.1 所需 Windows 构建工具及 Microsoft WebView2 Runtime。
2. 克隆本仓库，确认 `go.mod`、`wails.json`、`frontend/dist/` 和构建脚本完整。
3. 在项目根目录执行 `BUILD_WAILS_WINDOWS.cmd`，或按 `BUILD_WAILS_WINDOWS.ps1` 操作。
4. 检查 `build/bin/ORFixRepair_v4_1_2.exe` 是否生成。
5. 使用 `Get-FileHash -Algorithm SHA256` 核验文件哈希。
6. 在 Windows 实机启动并按“原生窗口、目录选择、Preview、Normal Repair、Face Fix 二阶段、Rollback、语言切换、重启后语言记忆、DPI 100/125/150”清单验收。

## 关键源码文件

- `frontend/tests/language.test.cjs`：国际化路径保留回归测试。
- `.github/workflows/publish-v4.1.2.yml`：V4.1.2 Windows EXE 构建和发布工作流。

- `main.go`：Wails 主窗口和前端资源嵌入。
- `app.go`：Wails 方法桥接、目录选择和当前语言对话框标题。
- `engine/`：原 Mod 扫描、兼容修复、Face Fix、备份与回滚规则。
- `frontend/dist/app.js`：React 页面与操作流程。
- `frontend/dist/language.js`：双语词典、语言状态、React 渲染期翻译和偏好存储。
- `frontend/dist/language.svg`：语言按钮图标。
- `frontend/dist/style.css`：原界面样式与新增语言控件样式。
- `.github/workflows/build-windows.yml`、`.github/workflows/publish-v4.1.2.yml`：Windows 构建/发布工作流。

## 测试边界

Chromium 自动化已覆盖初始页面、预览、回滚、面部风险、Face Fix 完成、正常修复成功、失败和重复失败等状态；9 个实际 UI 场景通过。localStorage 持久化检查在 about:blank 测试上下文受 SecurityError 限制，尚需在 Wails WebView 中实测。真实 Windows GUI、DPI 和游戏内渲染仍需设备验收。

## 安全与第三方依赖

- ORFix.ini 是 ORFix / NNFix 生效所需的外部游戏环境依赖，不随本项目打包；XXMI 用户应优先按 XXMI 管理方式安装。
- TexFx Mod 可能需要对应 TexFx 全局库。
- Face Fix 静态风险不是游戏故障确认；玩家应先进游戏检查，再决定是否继续。
- 修复前仍建议备份整个 Mods 目录。
- React / ReactDOM 与 SVG glass 组件许可文件保留在 `frontend/dist/vendor/`。
- 角色专用 INI 样例和图像素材的再分发权利应分别核实；项目级许可证不能自动覆盖第三方素材。
