package main

import (
	"fmt"
	"os"
)

func main() {
	cmd := os.Args[1]
	switch cmd {
	case "server", "serve", "s":
		serve()
	case "client", "c":
		client()
	default:
		fmt.Println("help:")
		fmt.Println(os.Args[0], "server")
		fmt.Println(os.Args[0], "client")
	}
}
