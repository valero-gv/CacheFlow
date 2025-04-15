package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"CacheFlow/internal/store"
)

// Server represents our cache server
type Server struct {
	store *store.Store
	addr  string
}

// New creates a new Server instance
func New(addr string) (*Server, error) {
	log.Println("Initializing server...")

	aofFilename := "aof.log"

	storage, err := store.New(aofFilename)
	if err != nil {
		log.Printf("Failed to initialize store with AOF '%s': %v", aofFilename, err)
		return nil, fmt.Errorf("store initialization failed: %w", err)
	}
	log.Println("Store initialized successfully.")

	server := &Server{
		store: storage,
		addr:  addr,
	}
	log.Printf("Server configured for address %s", addr)

	return server, nil
}

// Start starts the server and listens for incoming connections
func (s *Server) Start() error {
	log.Println("Starting server listener...")
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Printf("Failed to start listener: %v", err)
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer listener.Close()
	defer func() {
		log.Println("Attempting to close store...")
		if err := s.store.Close(); err != nil {
			log.Printf("Error closing store: %v", err)
		} else {
			log.Println("Store closed successfully.")
		}
	}()

	log.Printf("Server listening on %s", s.addr)

	// Start a goroutine to periodically clean up expired items
	go func() {
		log.Println("Background cleanup routine started.")
		for {
			s.store.DeleteExpired()
			time.Sleep(1 * time.Minute)
		}
	}()

	log.Println("Entering accept loop...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		log.Printf("Accepted connection from %s", conn.RemoteAddr())

		// Handle each connection in a separate goroutine
		go s.handleConnection(conn)
	}
}

// handleConnection processes a single client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		log.Printf("Closing connection from %s", conn.RemoteAddr())
		conn.Close()
	}()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		// Read command from client
		cmd, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading command from %s: %v", conn.RemoteAddr(), err)
			return
		}

		// Process command and send response
		response := s.handleCommand(strings.TrimSpace(cmd))
		if _, err := writer.WriteString(response + "\n"); err != nil {
			log.Printf("Error writing response to %s: %v", conn.RemoteAddr(), err)
			return
		}
		writer.Flush()
	}
}

// handleCommand processes a single command and returns the response
func (s *Server) handleCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "ERROR: Empty command"
	}

	switch strings.ToUpper(parts[0]) {
	case "SET":
		if len(parts) < 3 {
			return "ERROR: SET requires key and value"
		}
		key := parts[1]
		var value string
		var ttl time.Duration
		if len(parts) > 3 {
			parsedTTL, err := time.ParseDuration(parts[len(parts)-1])
			if err == nil {
				ttl = parsedTTL
				value = strings.Join(parts[2:len(parts)-1], " ")
			} else {
				value = strings.Join(parts[2:], " ")
			}
		} else {
			value = strings.Join(parts[2:], " ")
		}
		s.store.Set(key, value, ttl)
		return "OK"

	case "GET":
		if len(parts) != 2 {
			return "ERROR: GET requires key"
		}
		value, exists := s.store.Get(parts[1])
		if !exists {
			return "NIL"
		}
		return fmt.Sprintf("%v", value)

	case "DELETE":
		if len(parts) != 2 {
			return "ERROR: DELETE requires key"
		}
		s.store.Delete(parts[1])
		return "OK"

	case "EXISTS":
		if len(parts) != 2 {
			return "ERROR: EXISTS requires key"
		}
		if s.store.Exists(parts[1]) {
			return "1"
		}
		return "0"

	default:
		return "ERROR: Unknown command"
	}
}
