//go:build !windows

package player

import "fmt"

func New() (Engine, error) {
	return nil, fmt.Errorf("此发行版面向 Windows 桌面，请使用 Windows 安装包")
}
