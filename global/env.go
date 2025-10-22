package global

import (
	"github.com/haierkeys/singbox-subscribe-convert/pkg/fileurl"
)

var (
	// 程序执行目录
	ROOT string
	Name string = "Singbox Subscribe Convert"
)

func init() {

	filename := fileurl.GetExePath()
	ROOT = filename + "/"

}
