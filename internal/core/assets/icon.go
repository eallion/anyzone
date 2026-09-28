package assets

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sync"
)

var (
	iconOnce sync.Once
	iconData []byte
)

// GetAppIconPNG 获取 512x512 AnyZone 高清 PNG 图标字节流
func GetAppIconPNG() []byte {
	iconOnce.Do(func() {
		iconData = generateIconPNG(512)
		// 确保 build/appicon.png 也落盘，供 Wails 打包脚本使用
		_ = os.MkdirAll("build", 0755)
		_ = os.WriteFile(filepath.Join("build", "appicon.png"), iconData, 0644)
	})
	return iconData
}

// EnsureLinuxDesktopIntegration 为 Ubuntu / Linux 环境自动注册 Dock 图标与 Desktop Entry
func EnsureLinuxDesktopIntegration() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	iconBytes := GetAppIconPNG()

	// 1. 写入用户本地高分图标库 (~/.local/share/icons/hicolor/512x512/apps/anyzone.png)
	iconDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "512x512", "apps")
	_ = os.MkdirAll(iconDir, 0755)
	iconPath := filepath.Join(iconDir, "anyzone.png")
	_ = os.WriteFile(iconPath, iconBytes, 0644)

	// 2. 写入应用 Desktop Entry，解决 Ubuntu GNOME Shell (Wayland/X11) dev 模式下匹配不到 StartupWMClass 导致 Dock 无图标
	appsDir := filepath.Join(home, ".local", "share", "applications")
	_ = os.MkdirAll(appsDir, 0755)
	desktopPath := filepath.Join(appsDir, "anyzone.desktop")

	execPath, err := os.Executable()
	if err != nil || execPath == "" {
		execPath = "anyzone"
	}

	desktopEntry := `[Desktop Entry]
Version=1.0
Type=Application
Name=AnyZone
GenericName=DNS Management
Comment=Universal DNS Management Desktop Client
Exec=` + execPath + `
Icon=anyzone
Terminal=false
Categories=Network;Development;
StartupWMClass=anyzone
`
	_ = os.WriteFile(desktopPath, []byte(desktopEntry), 0644)
}

// generateIconPNG 使用 Go 原生图像库绘制 AnyZone 科技感应用图标
func generateIconPNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	// 背景设为全透明
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)

	center := float64(size) / 2.0
	radius := float64(size)*0.44

	// 1. 绘制圆角六边科技盾形底衬
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			dist := math.Sqrt(dx*dx + dy*dy)

			// 柔和六边形距离计算
			angle := math.Atan2(dy, dx)
			hexDist := dist * math.Cos(math.Mod(angle, math.Pi/3.0)-math.Pi/6.0)

			if hexDist <= radius {
				// 渐变计算: 从左上 (#1e293b) 到右下 (#0f172a)
				t := (float64(x+y) / float64(size*2))
				r := uint8(float64(0x1e)*(1-t) + float64(0x0f)*t)
				g := uint8(float64(0x29)*(1-t) + float64(0x17)*t)
				b := uint8(float64(0x3b)*(1-t) + float64(0x2a)*t)

				// 边缘渐变光芒描边
				edge := radius - hexDist
				if edge < 10 {
					edgeFactor := (10 - edge) / 10.0
					// 渐变青紫边框: #38bdf8 到 #a855f7
					r = uint8(float64(r)*(1-edgeFactor) + float64(0x63)*edgeFactor)
					g = uint8(float64(g)*(1-edgeFactor) + float64(0x66)*edgeFactor)
					b = uint8(float64(b)*(1-edgeFactor) + float64(0xf1)*edgeFactor)
				}
				img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}

	// 2. 绘制 AnyZone 标志性 A 与 Z 折线
	drawLine(img, center-100, center-80, center+100, center-80, 22, color.RGBA{0x38, 0xbd, 0xf8, 255})  // Z 顶横
	drawLine(img, center+90, center-80, center-90, center+80, 24, color.RGBA{0x63, 0x66, 0xf1, 255})   // Z 斜杠
	drawLine(img, center-100, center+80, center+100, center+80, 22, color.RGBA{0xa8, 0x55, 0xf7, 255}) // Z 底横

	// A 的顶针与白色横梁
	drawLine(img, center, center-120, center-65, center+40, 16, color.RGBA{0xff, 0xff, 0xff, 240})
	drawLine(img, center, center-120, center+65, center+40, 16, color.RGBA{0xff, 0xff, 0xff, 240})
	drawLine(img, center-40, center, center+40, center, 14, color.RGBA{0x38, 0xbd, 0xf8, 255})

	// 3. 绘制节点光斑圆点
	drawCircle(img, int(center), int(center-120), 12, color.RGBA{0xff, 0xff, 0xff, 255})
	drawCircle(img, int(center-100), int(center-80), 9, color.RGBA{0x38, 0xbd, 0xf8, 255})
	drawCircle(img, int(center+100), int(center-80), 9, color.RGBA{0xa8, 0x55, 0xf7, 255})
	drawCircle(img, int(center-100), int(center+80), 9, color.RGBA{0x38, 0xbd, 0xf8, 255})
	drawCircle(img, int(center+100), int(center+80), 9, color.RGBA{0xa8, 0x55, 0xf7, 255})
	drawCircle(img, int(center), int(center), 12, color.RGBA{0xff, 0xff, 0xff, 255})

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func drawCircle(img *image.RGBA, cx, cy, r int, col color.RGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r*r {
				if x >= 0 && x < img.Rect.Dx() && y >= 0 && y < img.Rect.Dy() {
					img.SetRGBA(x, y, col)
				}
			}
		}
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1, width float64, col color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	lenLine := math.Sqrt(dx*dx + dy*dy)
	if lenLine == 0 {
		return
	}
	nx := -dy / lenLine
	ny := dx / lenLine
	halfW := width / 2.0

	minX := int(math.Max(0, math.Min(x0, x1)-width))
	maxX := int(math.Min(float64(img.Rect.Dx()-1), math.Max(x0, x1)+width))
	minY := int(math.Max(0, math.Min(y0, y1)-width))
	maxY := int(math.Min(float64(img.Rect.Dy()-1), math.Max(y0, y1)+width))

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px := float64(x) - x0
			py := float64(y) - y0
			proj := (px*dx + py*dy) / lenLine
			if proj >= 0 && proj <= lenLine {
				dist := math.Abs(px*nx + py*ny)
				if dist <= halfW {
					img.SetRGBA(x, y, col)
				}
			}
		}
	}
}
