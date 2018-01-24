package main

import (
	"./dfslib"
	"fmt"
)

func main() {
	fmt.Println("Creating dir")
	dfslib.CreateLocalFileStore("./tmp/dfs-dev/")
}