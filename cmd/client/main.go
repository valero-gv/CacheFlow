package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"CacheFlow/internal/client"
)

func main() {
	c, err := client.New("localhost:6379")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("CacheFlow CLI Client")
	fmt.Println("Enter commands (SET/GET/DELETE/EXISTS) or 'exit' to quit")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		cmd := scanner.Text()
		if cmd == "exit" {
			break
		}

		handleCommand(c, cmd)
	}
}

func handleCommand(c *client.Client, cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		fmt.Println("Error: Empty command")
		return
	}

	switch strings.ToUpper(parts[0]) {
	case "SET":
		if len(parts) < 3 {
			fmt.Println("Usage: SET key value [ttl]")
			return
		}
		var ttl time.Duration
		if len(parts) > 3 {
			var err error
			ttl, err = time.ParseDuration(parts[3])
			if err != nil {
				fmt.Printf("Error parsing TTL: %v\n", err)
				return
			}
		}
		err := c.Set(parts[1], parts[2], ttl)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println("OK")
		}

	case "GET":
		if len(parts) != 2 {
			fmt.Println("Usage: GET key")
			return
		}
		value, err := c.Get(parts[1])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else if value == "" {
			fmt.Println("NIL")
		} else {
			fmt.Println(value)
		}

	case "DELETE":
		if len(parts) != 2 {
			fmt.Println("Usage: DELETE key")
			return
		}
		err := c.Delete(parts[1])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println("OK")
		}

	case "EXISTS":
		if len(parts) != 2 {
			fmt.Println("Usage: EXISTS key")
			return
		}
		exists, err := c.Exists(parts[1])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println(exists)
		}

	default:
		fmt.Println("Unknown command. Available commands: SET, GET, DELETE, EXISTS")
	}
}
