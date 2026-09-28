package main

import (
	"embed"
	"log"

	"anyzone/internal/core/assets"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend
var frontendAssets embed.FS

func main() {
	app, err := NewApp()
	if err != nil {
		log.Fatalf("应用初始化错误: %v", err)
	}

	appIcon := assets.GetAppIconPNG()
	assets.EnsureLinuxDesktopIntegration()

	err = wails.Run(&options.App{
		Title:             "AnyZone",
		Width:             1200,
		Height:            800,
		MinWidth:          960,
		MinHeight:         640,
		AssetServer: &assetserver.Options{
			Assets: frontendAssets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 23, B: 42, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			Icon:             appIcon,
			ProgramName:      "anyzone",
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.Mica,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
	})

	if err != nil {
		log.Fatalf("启动 AnyZone 失败: %v", err)
	}
}
