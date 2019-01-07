package main

import (
	"fmt"
	"os"
)

var ServerURL = getenv("SERVER_URL", "ws://localhost:8100/")
var ServeAddr = getenv("SERVE_ADDR", "localhost:8100")

func getenv(key, deft string) string {
	v := os.Getenv(key)
	if v == "" {
		return deft
	}
	return v

}

func main() {
	cmd := os.Args[1]
	switch cmd {
	case "server", "serve", "s":
		serve()
	case "client", "c":
		client()
	case "clip":
		for {
			clip()
		}
	default:
		fmt.Println("help:")
		fmt.Println(os.Args[0], "server")
		fmt.Println(os.Args[0], "client")
		fmt.Println(os.Args[0], "clip")
	}
}
