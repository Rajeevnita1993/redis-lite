package main

import (
	"fmt"
	"net"
)

func main() {
	numClients := 10 // Number of concurrent clients
	for i := 0; i < numClients; i++ {
		go func() {
			conn, err := net.Dial("tcp", "localhost:6379")
			if err != nil {
				fmt.Println("Error connecting:", err)
				return
			}
			defer conn.Close()

			// Simulate sending a SET command
			fmt.Fprintf(conn, "SET Key%d Value%d\r\n", i, i)

			// Simulate sending a GET command
			fmt.Fprintf(conn, "GET Key%d\r\n", i)

			// Read and print the response
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				fmt.Println("Error reading response:", err)
				return
			}
			fmt.Println("Response from server:", string(buf[:n]))
		}()
	}

	// Keep the main Goroutine running
	fmt.Scanln()
}
