package main

import "github.com/nlink-jp/stail/internal/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
