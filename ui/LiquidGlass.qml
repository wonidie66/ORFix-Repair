import QtQuick
import QtQuick.Controls

Item {
    id: glass
    property real cornerRadius: 18
    property real hoverAmount: 0
    property real pressedAmount: 0
    property real selectedAmount: 0
    property point pointerUV: Qt.point(0.4, 0.2)
    readonly property var host: ApplicationWindow.window
    readonly property Item backdropItem: host ? host.glassBackdrop : null
    readonly property Item backdropTexture: host ? host.glassTexture : null
    readonly property bool optical: host !== null && host !== undefined && host.glassOptics && backdropItem !== null && backdropTexture !== null
    readonly property point backdropOrigin: {
        if (!backdropItem) return Qt.point(0, 0)
        // Read ancestor positions so layout changes invalidate the mapped origin.
        var node = glass
        var tracked = 0
        while (node) {
            tracked += node.x + node.y
            node = node.parent
        }
        return glass.mapToItem(backdropItem, 0, 0)
    }
    Behavior on hoverAmount { NumberAnimation { duration: 150 } }
    Behavior on pressedAmount { NumberAnimation { duration: 95 } }
    Behavior on selectedAmount { NumberAnimation { duration: 140 } }
    Rectangle {
        anchors.fill: parent
        anchors.topMargin: 3
        anchors.leftMargin: -1
        anchors.rightMargin: -1
        anchors.bottomMargin: -3
        radius: glass.cornerRadius
        color: "#10000000"
    }
    // Always render a complete fallback surface; GPU shader is an enhancement.
    Rectangle {
        anchors.fill: parent
        radius: glass.cornerRadius
        gradient: Gradient {
            GradientStop { position: 0.0; color: glass.pressedAmount > 0.5 ? "#d5d5d8" : "#faffffff" }
            GradientStop { position: 0.48; color: glass.selectedAmount > 0.5 ? "#e8e8ea" : "#aaf2f2f4" }
            GradientStop { position: 1.0; color: glass.hoverAmount > 0.5 ? "#f9f9fa" : "#c6dedee1" }
        }
    }
    Rectangle {
        anchors.fill: parent
        radius: glass.cornerRadius
        color: "#007AFF"
        opacity: Math.min(0.22, glass.hoverAmount * 0.18 + glass.selectedAmount * 0.10 + glass.pressedAmount * 0.24)
    }
    Rectangle {
        anchors.fill: parent
        radius: glass.cornerRadius
        gradient: Gradient {
            GradientStop { position: 0.0; color: "#40ffffff" }
            GradientStop { position: 1.0; color: "#0f007AFF" }
        }
        opacity: Math.min(0.35, glass.hoverAmount * 0.28 + glass.selectedAmount * 0.12)
    }
    Loader {
        anchors.fill: parent
        active: glass.optical
        sourceComponent: Component {
            ShaderEffect {
                visible: status !== ShaderEffect.Error
                property var backdrop: glass.backdropTexture
                property vector2d glassSize: Qt.vector2d(Math.max(1,width),Math.max(1,height))
                property vector2d canvasSize: Qt.vector2d(glass.backdropItem ? glass.backdropItem.width : 1,glass.backdropItem ? glass.backdropItem.height : 1)
                property vector2d glassOrigin: Qt.vector2d(glass.backdropOrigin.x,glass.backdropOrigin.y)
                property vector2d pointerUV: Qt.vector2d(glass.pointerUV.x,glass.pointerUV.y)
                property real glassRadius: glass.cornerRadius
                property real hoverAmount: glass.hoverAmount
                property real pressedAmount: glass.pressedAmount
                property real selectedAmount: glass.selectedAmount
                fragmentShader: glassShaderUrl
            }
        }
    }
    Rectangle {
        anchors.fill: parent
        radius: glass.cornerRadius
        color: "transparent"
        border.width: 1
        border.color: glass.hoverAmount > 0.1 || glass.selectedAmount > 0.1 ? "#7a007AFF" : "#f7ffffff"
        Rectangle {
            anchors.fill: parent
            anchors.margins: 1
            radius: Math.max(0,glass.cornerRadius-1)
            color: "transparent"
            border.color: "#26ffffff"
        }
    }
}
