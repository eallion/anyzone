package main

import (
	"embed"
	"log"

	"os"

	"anyzone/internal/core/assets"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func init() {
	// WebKitGTK 4.1 在现代 Linux (Ubuntu Wayland / Mesa / NVIDIA) 环境中，DMA-BUF 渲染管线极易引发崩溃
	// 默认设置 WEBKIT_DISABLE_DMABUF_RENDERER=1 保障 GUI 稳定渲染启动
	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
}

// AppVersion 应用编译版本号，CI 编译时可通过 -ldflags "-X main.AppVersion=x.y.z" 动态覆盖
var AppVersion = "1.0.0"

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
