package store

import (
	"bufio"   // For reading AOF file line by line
	"fmt"     // For error formatting
	"io"      // For io.EOF
	"log"     // For logging AOF errors/info
	"os"      // For file operations
	"strings" // For joining command parts
	"sync"
	"time"
)

// Item represents a cache item
type Item struct {
	Value      interface{}
	Expiration *time.Time
}

// aofPersister handles writing commands to the Append-Only File.
type aofPersister struct {
	file *os.File
	mu   sync.Mutex // To protect concurrent writes to the file
}

// Store represents our key-value store
type Store struct {
	mu        sync.RWMutex
	items     map[string]Item
	persister *aofPersister // Handles AOF writing
	isLoading bool          // Flag to prevent recording commands during AOF load
}

// New creates a new Store instance and initializes AOF persistence.
func New(aofFilename string) (*Store, error) {
	var persister *aofPersister
	var aofFile *os.File // Keep file handle for loading
	var err error

	if aofFilename != "" {
		log.Printf("Initializing AOF persistence: %s", aofFilename)
		// Open file for appending/creating and WRITING
		aofFile, err = os.OpenFile(aofFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open AOF file %s for writing: %w", aofFilename, err)
		}
		persister = &aofPersister{
			file: aofFile,
		}
	} else {
		log.Println("AOF persistence is disabled (no filename provided).")
	}

	store := &Store{
		items:     make(map[string]Item),
		persister: persister,
		isLoading: false, // Start with loading false
	}

	// Load data from AOF file if persistence is enabled
	if persister != nil {
		store.isLoading = true // Set loading flag
		log.Println("Loading data from AOF file...")
		err = store.loadFromAOF(aofFilename)
		store.isLoading = false // Unset loading flag
		if err != nil {
			// Close the write handle if loading failed
			persister.file.Close()
			return nil, fmt.Errorf("failed to load data from AOF file %s: %w", aofFilename, err)
		}
		log.Println("Finished loading data from AOF file.")
	}

	return store, nil
}

// loadFromAOF reads the AOF file and replays commands.
func (s *Store) loadFromAOF(aofFilename string) error {
	file, err := os.OpenFile(aofFilename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open AOF file %s for reading: %w", aofFilename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if line == "" {
			continue // Skip empty lines
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			log.Printf("Skipping empty command in AOF file at line %d", lineNumber)
			continue
		}

		command := strings.ToUpper(parts[0])
		// Execute command directly on the store (recordCommand will be skipped due to isLoading flag)
		switch command {
		case "SET":
			if len(parts) < 3 {
				log.Printf("Invalid SET command in AOF file at line %d: %s", lineNumber, line)
				continue
			}
			key := parts[1]
			var value string
			var ttl time.Duration
			if len(parts) > 3 {
				parsedTTL, err := time.ParseDuration(parts[len(parts)-1])
				if err == nil {
					ttl = parsedTTL
					// Adjust value reconstruction if TTL was parsed
					value = strings.Join(parts[2:len(parts)-1], " ")
				} else {
					// Assume last part is part of the value if not a valid duration
					value = strings.Join(parts[2:], " ")
				}
			} else {
				value = strings.Join(parts[2:], " ")
			}
			s.Set(key, value, ttl) // Call regular Set, recordCommand will check isLoading
		case "DELETE":
			if len(parts) != 2 {
				log.Printf("Invalid DELETE command in AOF file at line %d: %s", lineNumber, line)
				continue
			}
			key := parts[1]
			s.Delete(key) // Call regular Delete
		default:
			log.Printf("Skipping unknown command '%s' in AOF file at line %d", command, lineNumber)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("error reading AOF file %s: %w", aofFilename, err)
	}

	return nil
}

// Close correctly closes the AOF file if it was opened.
func (s *Store) Close() error {
	if s.persister != nil && s.persister.file != nil {
		log.Printf("Closing AOF file...")
		return s.persister.file.Close()
	}
	return nil
}

// write appends a command to the AOF file.
func (ap *aofPersister) write(command string) error {
	if ap == nil || ap.file == nil {
		return nil // AOF disabled
	}

	ap.mu.Lock() // Lock to ensure only one goroutine writes at a time
	defer ap.mu.Unlock()

	// Append the command string followed by a newline
	_, err := ap.file.WriteString(command + "\n")
	if err != nil {
		log.Printf("ERROR writing to AOF file: %v", err)
		// Consider more robust error handling: retry? mark store as dirty?
	}
	// Optional: Force sync to disk immediately (slower but safer)
	// Usually, OS buffering is sufficient, but for high durability, Sync can be used.
	// if err == nil {
	//  err = ap.file.Sync() // Force kernel buffer to disk
	//  if err != nil {
	//      log.Printf("ERROR syncing AOF file: %v", err)
	//  }
	// }

	return err // Return the error from WriteString (or Sync if enabled)
}

// recordCommand formats and persists a command to the AOF file.
func (s *Store) recordCommand(parts []string) {
	// Skip recording if the store is currently loading from AOF
	if s.isLoading {
		return
	}
	if s.persister == nil {
		return // AOF disabled
	}
	// Reconstruct the command string simply for now
	// TODO: Consider a more robust serialization format later (like RESP)
	// We need to handle values with spaces correctly.
	// For SET key value [ttl], parts = ["SET", "key", "value part1", "value part2", "ttl"]
	// A simple Join(" ") might not be perfectly reconstructable without knowing command structure.
	// Let's just join for now, assuming simple values or accepting limitation.
	commandString := strings.Join(parts, " ")
	err := s.persister.write(commandString)
	if err != nil {
		// Decide how to handle AOF write errors. Log? Panic? Ignore?
		log.Printf("Failed to record command to AOF: %v", err)
	}
}

// Set adds a value to the store and records the command if AOF is enabled.
func (s *Store) Set(key string, value any, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var expiration *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiration = &t
	}

	s.items[key] = Item{
		Value:      value,
		Expiration: expiration,
	}

	// Record command after successful in-memory update
	parts := []string{"SET", key, fmt.Sprintf("%v", value)}
	if ttl > 0 {
		parts = append(parts, ttl.String())
	}

	s.recordCommand(parts)
}

// Delete removes a value from the store and records the command if AOF is enabled.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if key exists before recording delete (optional, but good practice)
	_, exists := s.items[key]
	if exists {
		delete(s.items, key)
		// Record command only if deletion happened
		parts := []string{"DELETE", key}
		s.recordCommand(parts)
	}

}

// Get retrieves a value from the store
func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[key]
	if !ok {
		return nil, false
	}
	// Check if the item has expired
	if item.Expiration != nil && time.Now().After(*item.Expiration) {
		//delete(s.items, key) // Let's keep this commented out as discussed
		//s.Delete(key)
		return nil, false
	}

	return item.Value, true
}

// Exists checks if a key exists in the store
func (s *Store) Exists(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[key]
	if !exists {
		return false
	}

	// Check if the item has expired
	if item.Expiration != nil && time.Now().After(*item.Expiration) {
		return false
	}

	return true
}

// DeleteExpired removes all expired items from the store.
// NOTE: DeleteExpired modifies the store but is NOT directly triggered by a client command.
// Therefore, we should NOT record individual DELETEs here in the AOF.
// The AOF only records client-initiated commands (SET, DELETE). Expired items
// will naturally be filtered out during replay or by the running server.
func (s *Store) DeleteExpired() {
	now := time.Now()
	s.mu.Lock() // Need a write lock to delete items
	defer s.mu.Unlock()

	for key, item := range s.items {
		if item.Expiration != nil && now.After(*item.Expiration) {
			delete(s.items, key)
		}
	}
}
