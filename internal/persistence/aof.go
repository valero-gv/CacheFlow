package persistence

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// AOF represents the Append-Only File persistence mechanism
type AOF struct {
	file      *os.File
	writer    *bufio.Writer
	isLoading bool
	mu        sync.Mutex
}

// New creates a new AOF instance
func New(filename string) (*AOF, error) {
	if filename == "" {
		return &AOF{}, nil // AOF disabled
	}

	log.Printf("Initializing AOF persistence: %s", filename)

	// Open file in append and create mode
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open AOF file %s for writing: %w", filename, err)
	}

	// Check file integrity
	if err := checkAOFIntegrity(filename); err != nil {
		file.Close()
		return nil, fmt.Errorf("AOF file integrity check failed: %w", err)
	}

	return &AOF{
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

// checkAOFIntegrity verifies the integrity of the AOF file
func checkAOFIntegrity(filename string) error {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open AOF file for integrity check: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Check command format
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return fmt.Errorf("invalid command format at line %d: %s", lineNumber, line)
		}

		cmd := strings.ToUpper(parts[0])
		switch cmd {
		case "SET":
			if len(parts) < 3 {
				return fmt.Errorf("invalid SET command at line %d: %s", lineNumber, line)
			}
		case "DELETE":
			if len(parts) != 2 {
				return fmt.Errorf("invalid DELETE command at line %d: %s", lineNumber, line)
			}
		default:
			return fmt.Errorf("unknown command at line %d: %s", lineNumber, cmd)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("error scanning AOF file: %w", err)
	}

	return nil
}

// Load loads data from the AOF file
func (a *AOF) Load(filename string, handler func(command string) error) error {
	if filename == "" {
		return nil // AOF disabled
	}

	log.Println("Loading data from AOF file...")

	// Check file integrity before loading
	if err := checkAOFIntegrity(filename); err != nil {
		return fmt.Errorf("AOF file integrity check failed before loading: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open AOF file %s for reading: %w", filename, err)
	}
	defer file.Close()

	a.isLoading = true
	defer func() { a.isLoading = false }()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			log.Printf("Skipping empty command in AOF file at line %d", lineNumber)
			continue
		}

		if err := handler(line); err != nil {
			return fmt.Errorf("error processing command at line %d: %w", lineNumber, err)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("error reading AOF file %s: %w", filename, err)
	}

	log.Println("Finished loading data from AOF file.")
	return nil
}

// Write writes a command to the AOF file and syncs with disk
func (a *AOF) Write(command string) error {
	if a.file == nil {
		return nil // AOF disabled
	}

	if a.isLoading {
		return nil // Skip writing during loading
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Write to buffer
	_, err := a.writer.WriteString(command + "\n")
	if err != nil {
		log.Printf("ERROR writing to AOF file: %v", err)
		return err
	}

	// Flush buffer to file
	if err := a.writer.Flush(); err != nil {
		log.Printf("ERROR flushing AOF file: %v", err)
		return err
	}

	// Force sync with disk
	if err := a.file.Sync(); err != nil {
		log.Printf("ERROR syncing AOF file: %v", err)
		return err
	}

	return nil
}

// Close closes the AOF file
func (a *AOF) Close() error {
	if a.file == nil {
		return nil // AOF disabled
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	log.Printf("Closing AOF file...")
	if err := a.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush AOF file: %w", err)
	}

	if err := a.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync AOF file: %w", err)
	}

	return a.file.Close()
}
