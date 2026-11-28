import QtQuick
import QtQuick.Controls
Button {
    id: control
    property bool primary: false
    property string glyph: ""
    property string iconName: glyph === "▱" ? "folder" : glyph === "✧" ? "sparkle" : glyph === "◉" ? "eye" : glyph === "↶" ? "refresh" : ""
    property int cornerRadius: 24
    property bool accentBlue: false
    implicitHeight: 44
    font.family: "Microsoft YaHei UI"
    font.pixelSize: 13
    font.bold: true
    opacity: enabled ? 1 : 0.48
    hoverEnabled: true
    scale: down ? 0.974 : 1.0
    Behavior on scale { NumberAnimation { duration: 125; easing.type: Easing.OutCubic } }
    HoverHandler { id: pointer; enabled: control.enabled }
    background: LiquidGlass {
        cornerRadius: Math.min(control.cornerRadius, height/2)
        selectedAmount: control.primary ? 1 : 0
        hoverAmount: control.hovered ? 1 : 0
        pressedAmount: control.down ? 1 : 0
        pointerUV: Qt.point(pointer.point.position.x/Math.max(1,control.width),pointer.point.position.y/Math.max(1,control.height))
        Rectangle {
            anchors.fill: parent
            anchors.margins: -2
            radius: Math.min(control.cornerRadius+2,height/2)
            color: "transparent"
            border.width: control.visualFocus ? 2 : 0
            border.color: "#737378"
        }
    }
    contentItem: Item {
        Row {
            anchors.centerIn: parent
            spacing: 8
            Image {
                visible: control.iconName !== ""
                width: visible ? 18 : 0
                height: 18
                source: visible ? ("assets/" + control.iconName + "-" + ((control.hovered || control.accentBlue) ? "blue" : "graphite") + ".png") : ""
                fillMode: Image.PreserveAspectFit
            }
            Text { text: control.text; color: control.hovered ? "#007AFF" : (control.primary ? "#202024" : "#45454a"); font: control.font; width: Math.min(implicitWidth, Math.max(20, control.width - (control.iconName ? 38 : 16))); elide: Text.ElideRight; height: 18; verticalAlignment: Text.AlignVCenter; textFormat: Text.PlainText }
        }
    }
}
