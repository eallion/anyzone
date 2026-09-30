package assets

import (
	"os"
	"testing"
)

func TestGetAppIconPNG(t *testing.T) {
	data := GetAppIconPNG()
	if len(data) == 0 {
		t.Fatal("图标数据为空")
	}

	// 验证 build/appicon.png 是否已正确生成落盘
	info, err := os.Stat("../../../build/appicon.png")
	if err != nil {
		// 尝试当前工作路径
		info, err = os.Stat("build/appicon.png")
	}
	if err == nil && info.Size() == 0 {
		t.Fatal("生成的 appicon.png 大小为 0")
	}
}
