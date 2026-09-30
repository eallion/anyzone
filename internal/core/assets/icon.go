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
		// 确保项目根目录与当前运行目录下的 build/appicon.png 均得到同步落盘
		for _, candidate := range []string{"build", "../../../build", "../../build"} {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				_ = os.WriteFile(filepath.Join(candidate, "appicon.png"), iconData, 0644)
			}
		}
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

	// 如果系统级 /usr/share/applications/anyzone.desktop 已存在，无需重复在用户目录写入，避免覆盖包管理器配置
	if _, err := os.Stat("/usr/share/applications/anyzone.desktop"); err == nil {
		return
	}

	// 2. 写入开发环境 Desktop Entry，解决 Ubuntu GNOME Shell (Wayland/X11) dev 模式下匹配不到 StartupWMClass 导致 Dock 无图标
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
Exec=env WEBKIT_DISABLE_DMABUF_RENDERER=1 ` + execPath + `
Icon=anyzone
Terminal=false
Categories=Network;Development;
StartupWMClass=AnyZone
`
	_ = os.WriteFile(desktopPath, []byte(desktopEntry), 0644)
}

// generateIconPNG 绘制符合现代桌面规范的标准圆角矩形应用图标 (上下左右严格几何对称，四边轮廓清晰)
func generateIconPNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	// 背景设为全透明
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)

	center := float64(size) / 2.0
	// 圆角矩形几何定义: 居中 432x432，四周留 40px 安全内边距，圆角半径 80px (工整端庄)
	rectWidth := 432.0
	rectHeight := 432.0
	halfW := rectWidth / 2.0
	halfH := rectHeight / 2.0
	cornerR := 80.0

	// 1. 绘制超平滑抗锯齿圆角矩形底衬与均质双层微光边框 (消除深色背景下的轮廓视差)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px := float64(x) + 0.5
			py := float64(y) + 0.5

			// 计算像素点到圆角矩形中心的相对位移（变换到第一象限）
			qx := math.Abs(px-center) - (halfW - cornerR)
			qy := math.Abs(py-center) - (halfH - cornerR)

			// Signed Distance Function (SDF)
			var dist float64
			if qx > 0 && qy > 0 {
				dist = math.Sqrt(qx*qx+qy*qy) - cornerR
			} else {
				dist = math.Max(qx, qy) - cornerR
			}

			if dist > 0.5 {
				continue
			}

			// 外边缘亚像素抗锯齿 (0.0 ~ 1.0)
			alphaFactor := 1.0
			if dist > -0.5 {
				alphaFactor = 0.5 - dist
			}

			// 底色渐变: 从左上深灰蓝 (#1e293b) 到右下深青黑 (#0f172a)，四角均保持可见明度
			t := (px + py) / (float64(size) * 2.0)
			bgR := float64(0x1e)*(1.0-t) + float64(0x0f)*t
			bgG := float64(0x29)*(1.0-t) + float64(0x17)*t
			bgB := float64(0x3b)*(1.0-t) + float64(0x2a)*t

			// 四周均质轮廓描边 (Border 宽度 3.0px)
			if dist >= -3.0 {
				bFactor := (dist + 3.0) / 3.0
				// 上方青色、下方微紫、四周边框均清晰
				edgeR := float64(0x38)*(1.0-t) + float64(0x81)*t
				edgeG := float64(0xbd)*(1.0-t) + float64(0x8c)*t
				edgeB := float64(0xf8)*(1.0-t) + float64(0xf8)*t

				bgR = bgR*(1.0-bFactor*0.7) + edgeR*(bFactor*0.7)
				bgG = bgG*(1.0-bFactor*0.7) + edgeG*(bFactor*0.7)
				bgB = bgB*(1.0-bFactor*0.7) + edgeB*(bFactor*0.7)
			}

			img.SetRGBA(x, y, color.RGBA{
				R: uint8(bgR),
				G: uint8(bgG),
				B: uint8(bgB),
				A: uint8(alphaFactor * 255.0),
			})
		}
	}

	// 2. 绘制上下严格对称的 AnyZone 核心图形 (Z 主体 + 双向菱形拓扑，彻底杜绝上大下小)
	offsetY := 75.0
	offsetX := 95.0

	// Z 字顶横与底横（完全等长、等粗、关于中心严格上下镜像）
	drawSmoothLine(img, center-offsetX, center-offsetY, center+offsetX, center-offsetY, 24, color.RGBA{0x38, 0xbd, 0xf8, 255}) // 顶横 (极光青)
	drawSmoothLine(img, center-offsetX, center+offsetY, center+offsetX, center+offsetY, 24, color.RGBA{0xa8, 0x55, 0xf7, 255}) // 底横 (幻影紫)

	// Z 字中心对角斜梁
	drawSmoothLine(img, center+offsetX-5, center-offsetY, center-offsetX+5, center+offsetY, 26, color.RGBA{0x63, 0x66, 0xf1, 255}) // 斜梁 (电光紫蓝)

	// 中心水平能量跨梁 (Zone 分割基准)
	drawSmoothLine(img, center-50, center, center+50, center, 14, color.RGBA{0x38, 0xbd, 0xf8, 255})

	// 上下对称的双向钻石网格拓扑连线 (纯白高光连杆)
	drawSmoothLine(img, center, center-offsetY, center-50, center, 14, color.RGBA{0xff, 0xff, 0xff, 255})
	drawSmoothLine(img, center, center-offsetY, center+50, center, 14, color.RGBA{0xff, 0xff, 0xff, 255})
	drawSmoothLine(img, center, center+offsetY, center-50, center, 14, color.RGBA{0xff, 0xff, 0xff, 255})
	drawSmoothLine(img, center, center+offsetY, center+50, center, 14, color.RGBA{0xff, 0xff, 0xff, 255})

	// 3. 绘制全对称 DNS 拓扑节点 (关于 (center, center) 中心对称)
	// 四角端点节点
	drawSmoothCircle(img, center-offsetX, center-offsetY, 11, color.RGBA{0x38, 0xbd, 0xf8, 255})
	drawSmoothCircle(img, center+offsetX, center-offsetY, 11, color.RGBA{0x38, 0xbd, 0xf8, 255})
	drawSmoothCircle(img, center-offsetX, center+offsetY, 11, color.RGBA{0xa8, 0x55, 0xf7, 255})
	drawSmoothCircle(img, center+offsetX, center+offsetY, 11, color.RGBA{0xa8, 0x55, 0xf7, 255})

	// 上下对称枢纽节点
	drawSmoothCircle(img, center, center-offsetY, 12, color.RGBA{0xff, 0xff, 0xff, 255})
	drawSmoothCircle(img, center, center+offsetY, 12, color.RGBA{0xff, 0xff, 0xff, 255})

	// 核心正中央枢纽节点
	drawSmoothCircle(img, center, center, 14, color.RGBA{0xff, 0xff, 0xff, 255})

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func drawSmoothCircle(img *image.RGBA, cx, cy float64, r float64, col color.RGBA) {
	margin := r + 1.5
	minX := int(math.Max(0, cx-margin))
	maxX := int(math.Min(float64(img.Rect.Dx()-1), cx+margin))
	minY := int(math.Max(0, cy-margin))
	maxY := int(math.Min(float64(img.Rect.Dy()-1), cy+margin))

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5
			py := float64(y) + 0.5
			dist := math.Sqrt((px-cx)*(px-cx) + (py-cy)*(py-cy))

			if dist > r+0.5 {
				continue
			}

			edgeAlpha := 1.0
			if dist > r-0.5 {
				edgeAlpha = (r + 0.5 - dist)
			}
			srcA := float64(col.A) / 255.0 * edgeAlpha

			dst := img.RGBAAt(x, y)
			dstA := float64(dst.A) / 255.0
			outA := srcA + dstA*(1.0-srcA)
			if outA > 0 {
				outR := (float64(col.R)*srcA + float64(dst.R)*dstA*(1.0-srcA)) / outA
				outG := (float64(col.G)*srcA + float64(dst.G)*dstA*(1.0-srcA)) / outA
				outB := (float64(col.B)*srcA + float64(dst.B)*dstA*(1.0-srcA)) / outA
				img.SetRGBA(x, y, color.RGBA{
					R: uint8(math.Round(outR)),
					G: uint8(math.Round(outG)),
					B: uint8(math.Round(outB)),
					A: uint8(math.Round(outA * 255.0)),
				})
			}
		}
	}
}

func drawSmoothLine(img *image.RGBA, x0, y0, x1, y1, width float64, col color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	lenLine := math.Sqrt(dx*dx + dy*dy)
	if lenLine == 0 {
		return
	}
	halfW := width / 2.0
	margin := halfW + 1.5

	minX := int(math.Max(0, math.Min(x0, x1)-margin))
	maxX := int(math.Min(float64(img.Rect.Dx()-1), math.Max(x0, x1)+margin))
	minY := int(math.Max(0, math.Min(y0, y1)-margin))
	maxY := int(math.Min(float64(img.Rect.Dy()-1), math.Max(y0, y1)+margin))

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5 - x0
			py := float64(y) + 0.5 - y0

			// 投影点在线段上的比例 t
			t := (px*dx + py*dy) / (lenLine * lenLine)
			tClamped := math.Max(0.0, math.Min(1.0, t))

			// 点到有限线段的最短欧氏距离（自动获得完美的圆角端点）
			closestX := tClamped * dx
			closestY := tClamped * dy
			dist := math.Sqrt((px-closestX)*(px-closestX) + (py-closestY)*(py-closestY))

			if dist > halfW+0.5 {
				continue
			}

			// 边缘抗锯齿 Alpha
			edgeAlpha := 1.0
			if dist > halfW-0.5 {
				edgeAlpha = (halfW + 0.5 - dist)
			}
			srcA := float64(col.A) / 255.0 * edgeAlpha

			// Alpha 混合到背景
			dst := img.RGBAAt(x, y)
			dstA := float64(dst.A) / 255.0
			outA := srcA + dstA*(1.0-srcA)
			if outA > 0 {
				outR := (float64(col.R)*srcA + float64(dst.R)*dstA*(1.0-srcA)) / outA
				outG := (float64(col.G)*srcA + float64(dst.G)*dstA*(1.0-srcA)) / outA
				outB := (float64(col.B)*srcA + float64(dst.B)*dstA*(1.0-srcA)) / outA
				img.SetRGBA(x, y, color.RGBA{
					R: uint8(math.Round(outR)),
					G: uint8(math.Round(outG)),
					B: uint8(math.Round(outB)),
					A: uint8(math.Round(outA * 255.0)),
				})
			}
		}
	}
}
