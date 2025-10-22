package main

import (
	_ "embed"

	"github.com/haierkeys/singbox-subscribe-convert/cmd"
)

//go:embed config/config.yaml
var c string

func main() {
	cmd.Execute(c)
}
