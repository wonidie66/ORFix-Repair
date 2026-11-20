# ORFix Repair V4.1.0 — 真实 Wails 桌面工程

## 本次变更

本版本从 V4.0.1 提取真正的 Wails v2 入口。只保留 `main.go` 与 `app.go` 作为宿主，不包含 V4.0.2 中任何 `edge_main.go`、`edge_host.go`、`localhost` HTTP server 或 `--app` 启动逻辑。窗口由 Wails Windows WebView2 创建。页面的 React/CSS/液态玻璃 SVG、五张派蒙状态图和原 Go 引擎全部嵌入应用资源。

## 已核验与未核验

- 已核验：修复引擎独立测试、Go 静态检查、前端静态资源完整性，及原 GUI 接口调用名称与后端导出方法对应。
- 尚未核验：**Windows Wails EXE 的真实编译与运行**。当前构建环境无法下载 Wails Go 依赖，且无法运行 Windows 桌面。
- 因此，本源码包不是可直接运行的 Windows EXE。请勿将原 Edge 版重命名作为本版发布。

## Windows 构建（推荐）

1. 在 Windows 10/11 上安装 Go **1.26 或更高版本**。Wails v2.17.1 的 `go.mod` 明确要求 Go 1.26。
2. 确保网络允许访问 Go 模块仓库，系统具有 WebView2 Runtime。
3. 解压完整工程到纯英文路径（可减少旧工具链路径问题），双击 `BUILD_WAILS_WINDOWS.cmd`；或者在 PowerShell 中运行 `./BUILD_WAILS_WINDOWS.ps1`。
4. 推荐双击 `BUILD_WAILS_WINDOWS.cmd`，不要直接双击 PS1。新版 CMD 无 BOM、PS1 为 UTF-8 BOM，且会保留错误窗口并生成 `build/logs` 日志。也可参考 `BUILD_MANUAL_中文说明.txt` 逐条手动构建。
5. 编译结束后检查 `build/bin/ORFixRepair_v4_1_0.exe`，并在 Windows 上启动。屏幕应出现带标题栏的 **ORFix Repair** 原生窗口，不能出现 Edge 标签页、地址栏或 `localhost` URL。
6. Windows 任务管理器可能出现 `msedgewebview2.exe` 子进程，这是 WebView2 运行时正常行为，不等于 EXE 在浏览器中打开网页。

## GitHub Actions 构建（无需本机安装 Go）

将源码放入你自己的**私有** GitHub 仓库，保留 `.github/workflows/build-windows.yml`；从 Actions 页面启动 `Build genuine Wails Windows EXE`，下载构建完成的 Artifact。此流程需要仓库所有者授权运行、Github Actions 可用；**当前未在 GitHub Actions 实际执行**。

## 源代码目录

- `main.go`：Wails 主窗口与资源嵌入，不启动浏览器；
- `app.go`：文件夹选择、常规修复、预览、两阶段 Face Fix 和回滚等绑定；
- `engine/`：V4.0.1 经测试的 Go 修复引擎；
- `frontend/dist/`：React、CSS、SVG 玻璃组件及五张结果图片的离线资源；
- `build/windows/icon.ico`：EXE 内嵌图标；
- `BUILD_WAILS_WINDOWS.*`：Windows 本地构建入口；
- `.github/workflows/build-windows.yml`：可复现的 Windows 云端构建工作流。

## 风险和测试边界

修复引擎可能修改旧版凝光的 Blend.buf，不能假定始终仅修改 INI。Face Fix 检测只是静态风险筛选，默认不修改面部；用户进游戏确认异常后才应点击继续。建议备份整个 Mods 文件夹再使用。

原生 Windows 版本的验收需要检查：①启动和关闭窗口；②目录选择；③仅预览不写盘；④正常修复及结果图片切换；⑤等待用户检查面部时不退出；⑥继续 Face Fix；⑦回滚可恢复两阶段原始文件；⑧日志展开后支持滚轮/键盘；⑨DPI 100%/125%/150% 显示。

## 上游资料

- Wails v2: https://v2.wails.io/docs/gettingstarted/installation/
- Wails CLI Build: https://v2.wails.io/docs/reference/cli/
- Wails upstream module v2.17.1: https://github.com/wailsapp/wails/blob/v2.17.1/v2/go.mod
- React: https://react.dev/
- SVG Liquid Glass component (MIT): https://github.com/yurkagon/react-liquid-glass-svg

## 许可

React 与 SVG 玻璃组件的 MIT 许可证保留在 `frontend/dist/vendor/`；MiSans 未内嵌，界面用 Windows 自带微软雅黑。
