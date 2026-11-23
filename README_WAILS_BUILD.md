# ORFix Repair V4.2.0 — Wails Windows x64 构建说明

## 本版本新增内容

V4.2.0 在 V4.1.2 基础上新增一条**通用四槽位材质 NNFIX 规则**，不绑定任何角色名称、IB Hash 或 `match_first_index`。规则仅识别以下完整材质语义顺序：

- `ps-t0`：Diffuse
- `ps-t1`：LightMap
- `ps-t2`：MetalMap
- `ps-t3`：ShadowRamp

自动修复前还必须满足：四个资源引用具有相同前缀和数值变体；四个资源定义均存在；对应文件均存在且非空；四个槽位在同一个未被分支、绘制或命令调用打断的绑定组内；不存在重复槽位或第五个及以上的额外槽位。满足全部条件后，才在对应绘制前插入 `run = CommandList\global\ORFix\NNFix`。

该规则面向**具有上述相同材质结构的不同角色 Mod**。它不是“所有四槽位材质都使用 NNFIX”的宽泛判断；NormalMap、未知槽位、资源缺失、命名不一致或复杂绑定均继续保守跳过。只修改 INI 中的修复命令，不修改贴图引用、DDS、IB、BUF、Face Fix 或其他 Mod 资源。

本源码包供本地构建与实机测试使用，V4.2.0 尚未发布 EXE 或 GitHub Release。

## 构建环境

- Windows 10/11 x64
- Go 1.26.x（用户已安装的 Go 1.26.4 满足要求）
- Wails v2.17.1（构建脚本会安装锁定版本）
- Microsoft WebView2 Runtime（运行 EXE 所需）

## 本地构建

1. 将源码 ZIP 解压到简单路径，例如 `C:\ORFixBuild\ORFixRepair_V4.2.0_Wails`。
2. 进入工程根目录。
3. 双击 `BUILD_WAILS_WINDOWS.cmd`，不要直接双击 PS1。
4. 脚本会下载依赖、运行 `go test -count=1 ./engine`、安装 Wails CLI 并构建原生 Windows x64 EXE。
5. 输出文件：`build\bin\ORFixRepair_v4_2_0.exe`。
6. 构建日志保存在 `build\logs`。

也可以在 CMD 中手动执行：

```bat
go version
go mod download
go test -count=1 ./engine
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1
for /f "delims=" %G in ('go env GOPATH') do set "GOPATH=%G"
"%GOPATH%\bin\wails.exe" build -platform windows/amd64 -s -clean
dir build\bin\ORFixRepair_v4_2_0.exe
```

## 测试范围

源码新增通用四槽位规则的跨角色正向、资源缺失、槽位语义错误、资源前缀不一致、额外槽位、已有修复、重复运行、诊断与增强扫描集成测试；同时保留 V4.1.2 的中英文路径回归测试与原有 Go 引擎测试。

自动化测试通过不等于 Windows 原生窗口或游戏画面已验收。请在本地 EXE 构建后，分别测试 Preview、Normal Repair、Face Fix 二阶段、Rollback、中英文切换以及至少两个不同角色、但采用相同四槽位材质结构的 Mod 实际画面。

## 原有功能与安全边界

- 原页面布局、窗口尺寸、按钮位置与液态玻璃视觉样式不变。
- ORFix.ini / TexFx 仍为外部游戏侧依赖，不随本工具打包。
- Face Fix 仍为先提示风险、由用户确认后才执行的二阶段流程。
- 所有修改仍通过原有备份、校验与回滚机制完成。
- 本规则不自动修改模型、贴图像素、贴图槽位或未知 Shader。
