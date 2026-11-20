# 构建脚本修正说明

已确认旧脚本存在编码隐患：PS1 为 UTF-8 无 BOM，Windows PowerShell 5.1 可能按 ANSI 解码，造成中文乱码；CMD 文件带 UTF-8 BOM，CMD 对 BOM 处理不稳定。

本次修正：
- CMD 改为纯 ASCII、无 BOM；
- PS1 改为 UTF-8 BOM + CRLF；
- PS1 增加 try/catch/finally 和 Read-Host，避免双击后闪退；
- 保存完整构建日志至 build/logs；
- 明确显示 Go 版本、GOPROXY、GOPATH 和每一步失败的退出码；
- Go 1.26.4 满足 Wails v2.17.1 要求，无需重新安装 Go。

这次只修正构建脚本和说明，没有改动 Wails 入口、React 页面或 Go 修复引擎。
