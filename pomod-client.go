package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: pomod-client [toggle|finish|status]")
		return
	}

	conn, err := net.Dial("unix", "/tmp/pomod.sock")
	if err != nil {
		fmt.Println("daemon not running?")
		return
	}
	defer conn.Close()

	conn.Write([]byte(os.Args[1]))

	if os.Args[1] == "status" {
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		fmt.Println(string(buf[:n]))
	}
}

