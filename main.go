package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// Frontend resources are embedded into the executable. No Node.js, Python,
// external CMD script, or separate HTML file is required on the target PC.
//go:embed all:frontend/dist
var frontend embed.FS

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:            "ORFix Repair v4.2.0",
		Width:            830,
		Height:           570,
		MinWidth:         780,
		MinHeight:        550,
		DisableResize:    false,
		AssetServer:      &assetserver.Options{Assets: frontend},
		BackgroundColour: &options.RGBA{R: 248, G: 251, B: 255, A: 255},
		OnStartup:        app.startup,
		Bind:             []interface{}{app},
		Windows: &windows.Options{
			Theme: windows.Light,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
