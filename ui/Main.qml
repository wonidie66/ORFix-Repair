import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
ApplicationWindow {
    id: win
    objectName: "ORFixNativeWindow"
    visible: true
    width: 830; height: 570; minimumWidth: 780; minimumHeight: 550
    title: "ORFix Repair V5.0.0"
    color: "#f5f5f7"
    font.family: "Microsoft YaHei UI"
    property bool logsOpen: false
    property string phase: backend.phase
    property int stage: phase === "idle" ? 0 : phase === "risk" ? 2 : phase.indexOf("face") === 0 ? 3 : 1
    function tr2(zh,en) { return backend.language === "en" ? en : zh }
    property var states: ({
        idle: [tr2("准备就绪","Ready"), tr2("开始修复前，请选择 Mod 文件夹","Choose your Mod folder"), tr2("工具将检查贴图配置并修复符合条件的兼容问题。所有实际修改均会备份。","Checks known compatibility rules and backs up every modified file."), ""],
        working: [tr2("正在处理","Working"), tr2("正在扫描 Mod 配置…","Checking Mod configuration…"), tr2("正在分析材质、资源与修复条件。","Analyzing materials, resources and repair conditions."), ""],
        success: [tr2("处理完成","Completed"), tr2("正常修复流程已完成","Normal repair completed"), tr2("请进入游戏检查显示效果。静态修复并不代表所有渲染问题已经解决。","Check in-game appearance. Static repair is not a visual guarantee."), "success"],
        risk: [tr2("需要确认","Review needed"), tr2("检测到疑似面部兼容风险","Possible face compatibility issue"), tr2("请先进入游戏检查面部显示。若面部异常，再继续 Face Fix。","Check the face in-game first. Continue Face Fix only if needed."), "face-risk"],
        faceWorking: [tr2("正在处理","Working"), tr2("正在修复面部绑定…","Repairing face bindings…"), tr2("仅修改符合条件的面部贴图配置。","Only eligible face bindings are modified."), ""],
        faceSuccess: [tr2("修复完成","Completed"), tr2("面部兼容修复已完成","Face compatibility repair completed"), tr2("请返回游戏检查面部显示，所有修改已建立备份。","Check the face in-game. All changes have backups."), "face-done"],
        failed: [tr2("需要检查","Review needed"), tr2("本次修复未完全成功","Repair needs review"), tr2("查看详细日志及错误信息；可恢复已修改的文件。","Review the detailed log and available backups."), "failure"],
        faceFailed: [tr2("需要检查","Review needed"), tr2("面部修复未完成","Face repair not completed"), tr2("没有可执行候选或处理遇到问题，请查看日志。","No eligible change or an error occurred. Review the log."), "failure"],
        preview: [tr2("扫描结束","Scan complete"), tr2("预览完成，未修改文件","Preview complete — no Mod edits"), tr2("本次只记录修复候选及潜在风险。","Candidates and potential risks are recorded without Mod edits."), ""],
        rollback: [tr2("操作完成","Completed"), tr2("已恢复最近批次备份","Latest batch restored"), tr2("恢复数量及文件详情已记录在日志中。","Restored files are listed in the log."), ""]
    })
    property var cfg: states[phase] || states.idle
    property Item glassBackdrop: backdropLayer
    property Item glassTexture: backdropCapture
    property bool glassOptics: glassShaderReady
    Item {
        id: backdropLayer
        anchors.fill: parent
        Rectangle { anchors.fill: parent; color: "#f5f5f7" }
        Image { anchors.fill: parent; source: "assets/silver-backdrop.png"; fillMode: Image.Stretch }
    }
    ShaderEffectSource {
        id: backdropCapture
        sourceItem: win.glassOptics ? backdropLayer : null
        visible: false
        live: true
        hideSource: false
        textureMirroring: ShaderEffectSource.MirrorVertically
    }
    Item {
        anchors.fill: parent
        ColumnLayout {
            anchors.fill: parent; spacing: 0
            Rectangle {
                Layout.fillWidth: true; Layout.preferredHeight: 70; color: "#40ffffff"
                RowLayout {
                    anchors.fill: parent; anchors.leftMargin: 32; anchors.rightMargin: 32; spacing: 11
                    Image { source: "assets/logo.png"; Layout.preferredWidth: 42; Layout.preferredHeight: 42; fillMode: Image.PreserveAspectFit }
                    ColumnLayout { spacing: 3; Text { text: "ORFix Repair"; color: "#202024"; font.pixelSize: 17; font.bold: true } Text { text: "V5.0.0"; color: "#74747a"; font.pixelSize: 12 } }
                    Item { Layout.fillWidth: true }
                    LiquidGlass {
                        cornerRadius: 20; Layout.preferredWidth: 132; Layout.preferredHeight: 40
                        RowLayout { anchors.fill: parent; anchors.margins: 5; spacing: 4
                            Image { source: "assets/language.png"; Layout.preferredWidth: 20; Layout.preferredHeight: 20; fillMode: Image.PreserveAspectFit }
                            GlassButton { text: "中文"; primary: backend.language === "zh"; cornerRadius: 9; Layout.fillWidth: true; Layout.preferredHeight: 28; onClicked: backend.setLanguage("zh") }
                            GlassButton { text: "EN"; primary: backend.language === "en"; cornerRadius: 9; Layout.fillWidth: true; Layout.preferredHeight: 28; onClicked: backend.setLanguage("en") }
                        }
                    }
                }
            }
            Rectangle { Layout.fillWidth: true; Layout.preferredHeight: 1; color: "#25a0a0a5" }
            ColumnLayout {
                Layout.fillWidth: true; Layout.fillHeight: true; Layout.leftMargin: 32; Layout.rightMargin: 32; Layout.topMargin: 24; Layout.bottomMargin: 11; spacing: 12
                Text { text: tr2("修复 Mod 贴图与兼容问题","Repair Mod textures and compatibility"); font.pixelSize: 21; font.bold: true; color: "#202024"; Layout.fillWidth: true; elide: Text.ElideRight }
                RowLayout {
                    Layout.fillWidth: true; spacing: 11
                    LiquidGlass { Layout.fillWidth: true; Layout.preferredHeight: 45; cornerRadius: 22
                        RowLayout { anchors.fill: parent; anchors.margins: 12; spacing: 10
                            Image { source: "assets/folder-graphite.png"; Layout.preferredWidth: 19; Layout.preferredHeight: 19 }
                            Text { text: backend.folder || tr2("选择你的 GIMI / Mods 文件夹","Choose your GIMI / Mods folder"); textFormat: Text.PlainText; Layout.fillWidth: true; elide: Text.ElideMiddle; color: "#515156"; font.pixelSize: 13 }
                        }
                        ToolTip.visible: pathMouse.containsMouse; ToolTip.text: backend.folder
                        MouseArea { id: pathMouse; anchors.fill: parent; hoverEnabled: true; acceptedButtons: Qt.NoButton }
                    }
                    GlassButton { text: tr2("浏览目录","Browse Folder"); glyph: "▱"; accentBlue: true; Layout.preferredWidth: 134; Layout.preferredHeight: 45; enabled: !backend.busy && phase !== "risk"; onClicked: backend.browse() }
                }
                RowLayout {
                    visible: phase !== "risk" && phase !== "faceWorking"; Layout.fillWidth: true; spacing: 11
                    GlassButton { Layout.fillWidth: true; Layout.preferredHeight: 44; primary: true; accentBlue: true; glyph: "✧"; text: backend.busy ? tr2("正在修复…","Working…") : tr2("开始正常修复","Start Repair"); enabled: !backend.busy; onClicked: { win.logsOpen=false; backend.execute("normal") } }
                    GlassButton { Layout.fillWidth: true; Layout.preferredHeight: 44; glyph: "◉"; text: tr2("仅预览","Preview"); enabled: !backend.busy; onClicked: { win.logsOpen=false; backend.execute("preview") } }
                    GlassButton { Layout.fillWidth: true; Layout.preferredHeight: 44; glyph: "↶"; text: tr2("回滚最近批次","Rollback Latest"); enabled: !backend.busy; onClicked: { win.logsOpen=false; backend.execute("rollback") } }
                }
                RowLayout {
                    visible: !win.logsOpen; Layout.alignment: Qt.AlignHCenter; Layout.preferredHeight: 40; spacing: 11
                    Repeater { model: [tr2("常规修复","Normal repair"),tr2("游戏中检查","Check in game"),tr2("可选 Face Fix","Optional Face Fix")]
                        delegate: RowLayout { required property int index; required property string modelData; spacing: 6
                            Rectangle { Layout.preferredWidth: 18; Layout.preferredHeight: 18; radius: 9; color: win.stage>=index+1 ? "#dbdbdf" : "#f6f6f7"; border.color: "#c6c6cb"; Text { anchors.centerIn: parent; text: index+1; color: win.stage>=index+1 ? "#36363b":"#75757b"; font.pixelSize: 11; font.bold: true } }
                            Text { text: modelData; color: win.stage>=index+1 ? "#434348":"#77777d"; font.pixelSize: 12 }
                            Rectangle { visible: index<2; Layout.preferredWidth: 37; Layout.preferredHeight: 1; color: "#c7c7cc" }
                        }
                    }
                }
                LiquidGlass {
                    visible: !win.logsOpen; Layout.fillWidth: true; Layout.fillHeight: true; Layout.minimumHeight: 147; Layout.maximumHeight: 172; cornerRadius: 23
                    RowLayout { anchors.fill: parent; anchors.margins: 20; spacing: 10
                        ColumnLayout { Layout.fillWidth: true; spacing: 7
                            Text { text: "●  " + cfg[0]; color: phase.indexOf("Failed")>=0 || phase === "failed" ? "#444449" : phase === "risk" ? "#55555a" : "#444449"; font.pixelSize: 12; font.bold: true }
                            Text { text: cfg[1]; Layout.fillWidth: true; wrapMode: Text.WordWrap; color: "#202024"; font.pixelSize: 20; font.bold: true }
                            Text { text: cfg[2]; Layout.fillWidth: true; wrapMode: Text.WordWrap; color: "#5b5b60"; font.pixelSize: 13 }
                            RowLayout { visible: phase === "risk"; spacing: 9
                                GlassButton { primary: true; accentBlue: true; text: tr2("继续修复 Face Fix","Continue Face Fix"); Layout.preferredWidth: 165; Layout.preferredHeight: 34; onClicked: backend.execute("face") }
                                GlassButton { text: tr2("完成退出","Finish & Exit"); Layout.preferredWidth: 110; Layout.preferredHeight: 34; onClicked: backend.finish() }
                            }
                            RowLayout { visible: phase !== "idle" && phase !== "working" && phase !== "faceWorking" && phase !== "risk"; spacing: 12
                                Text { text: backend.scanned + tr2(" 个 INI 已扫描"," INIs scanned") + "  |  " + backend.modified + tr2(" 个文件已处理"," files processed"); color: "#606066"; font.pixelSize: 12 }
                                GlassButton { text: tr2("完成退出","Finish & Exit"); Layout.preferredWidth: 105; Layout.preferredHeight: 28; onClicked: backend.finish() }
                            }
                        }
                        Item { Layout.preferredWidth: 142; Layout.fillHeight: true
                            Image { anchors.centerIn: parent; width: 142; height: 142; visible: cfg[3] !== ""; source: cfg[3] !== "" ? "assets/"+cfg[3]+".png" : ""; fillMode: Image.PreserveAspectFit }
                            Rectangle { visible: cfg[3] === ""; anchors.centerIn: parent; width: 78; height: 78; radius: 35; border.color: "#ffffff"; color: "#e3e3e6"; Image { anchors.centerIn: parent; source: "assets/sparkle-blue.png"; width: 34; height: 34 } }
                        }
                    }
                }
                RowLayout { Layout.fillWidth: true; Layout.preferredHeight: 38; spacing: 8
                    Button {
                        id: logButton
                        flat: true
                        onClicked: win.logsOpen = !win.logsOpen
                        background: Item {}
                        contentItem: RowLayout {
                            spacing: 6
                            Text {
                                text: tr2("详细日志","Detailed log")
                                color: "#49494e"
                                font.pixelSize: 13
                                font.bold: true
                                verticalAlignment: Text.AlignVCenter
                            }
                            Rectangle {
                                Layout.preferredWidth: 16
                                Layout.preferredHeight: 16
                                radius: 8
                                color: win.logsOpen ? "#e7f1ff" : "#eef1f5"
                                border.color: win.logsOpen ? "#b7d5ff" : "#dde1e7"
                                Text {
                                    anchors.centerIn: parent
                                    text: win.logsOpen ? "▴" : "▾"
                                    color: win.logsOpen ? "#007AFF" : "#70757d"
                                    font.pixelSize: 9
                                    font.bold: true
                                }
                            }
                        }
                    }
                    Text { text: tr2("修复过程与备份记录","Repair and backup records"); font.pixelSize: 12; color: "#737379" }
                    Item { Layout.fillWidth: true }
                    RowLayout {
                        spacing: 6
                        Rectangle {
                            Layout.preferredWidth: 16
                            Layout.preferredHeight: 16
                            radius: 8
                            color: "#007AFF"
                            Text {
                                anchors.centerIn: parent
                                text: "✓"
                                color: "white"
                                font.pixelSize: 10
                                font.bold: true
                            }
                        }
                        Text { text: tr2("自动备份 · 支持回滚","Backup · Rollback"); color: "#66666b"; font.pixelSize: 12 }
                    }
                }
                ScrollView {
                    visible: win.logsOpen; Layout.fillWidth: true; Layout.fillHeight: true; Layout.minimumHeight: 130; clip: true; ScrollBar.vertical.policy: ScrollBar.AsNeeded
                    TextArea {
                        text: backend.logText || tr2("暂无日志。", "No log yet.")
                        readOnly: true
                        wrapMode: TextEdit.Wrap
                        selectByMouse: true
                        textFormat: TextEdit.PlainText
                        font.pixelSize: 13
                        color: "#515156"
                        background: Rectangle {
                            radius: 13
                            color: "#f7f7f8"
                            border.color: "#d6d6da"
                        }
                        padding: 12
                    }
                }
                Item { visible: !win.logsOpen; Layout.fillHeight: true; Layout.minimumHeight: 0 }
            }
        }
    }
}
