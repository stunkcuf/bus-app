package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

func main() {
	// Try to find a way to clear rate limit
	// Since we can't directly access the running server's memory,
	// let's check if there's an admin endpoint or we need to restart
	
	fmt.Println("Rate limiting issue detected.")
	fmt.Println("The server is blocking requests due to too many attempts.")
	fmt.Println("\nTo fix this, you need to:")
	fmt.Println("1. Stop the current server (find PID 23684 and kill it)")
	fmt.Println("2. Restart the server")
	fmt.Println("\nOr wait 15 minutes for the rate limit to reset.")
	
	// Also, let's check what IP is being used
	resp, err := http.Get("http://localhost:5002/health")
	if err == nil {
		defer resp.Body.Close()
		fmt.Printf("\nHealth check status: %d\n", resp.StatusCode)
	}
}