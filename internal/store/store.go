package store

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"CacheFlow/internal/persistence"
)

// Item represents a cache item with value and optional expiration time
type Item struct {
	Value      interface{}
	Expiration *time.Time
}

// Store represents our key-value store with persistence support
type Store struct {
	mu    sync.RWMutex
	items map[string]Item
	aof   *persistence.AOF
}

// New creates a new Store instance and initializes AOF persistence
func New(aofFilename string) (*Store, error) {
	store := &Store{
		items: make(map[string]Item),
	}

	// Initialize AOF if filename is provided
	if aofFilename != "" {
		aof, err := persistence.New(aofFilename)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize AOF: %w", err)
		}
		store.aof = aof

		// Load data from AOF file
		err = aof.Load(aofFilename, func(command string) error {
			parts := strings.Fields(command)
			if len(parts) == 0 {
				return nil
			}

			cmd := strings.ToUpper(parts[0])
			switch cmd {
			case "SET":
				if len(parts) < 3 {
					return fmt.Errorf("invalid SET command: %s", command)
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
				store.Set(key, value, ttl)
			case "DELETE":
				if len(parts) != 2 {
					return fmt.Errorf("invalid DELETE command: %s", command)
				}
				store.Delete(parts[1])
			default:
				return fmt.Errorf("unknown command: %s", cmd)
			}
			return nil
		})

		if err != nil {
			aof.Close()
			return nil, fmt.Errorf("failed to load data from AOF: %w", err)
		}
	}

	return store, nil
}

// Close properly closes the AOF file if it was opened
func (s *Store) Close() error {
	if s.aof != nil {
		return s.aof.Close()
	}
	return nil
}

// recordCommand formats and persists a command to the AOF file
func (s *Store) recordCommand(parts []string) {
	if s.aof == nil {
		return
	}

	commandString := strings.Join(parts, " ")
	if err := s.aof.Write(commandString); err != nil {
		log.Printf("Failed to record command to AOF: %v", err)
	}
}

// Set adds a value to the store and records the command if AOF is enabled
func (s *Store) Set(key string, value any, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create expiration time if TTL is provided
	var expiration *time.Time
	if ttl > 0 {
		exp := time.Now().Add(ttl)
		expiration = &exp
	}

	// Store the item
	s.items[key] = Item{
		Value:      value,
		Expiration: expiration,
	}

	// Record the command
	parts := []string{"SET", key, fmt.Sprintf("%v", value)}
	if ttl > 0 {
		parts = append(parts, ttl.String())
	}
	s.recordCommand(parts)
}

// Delete removes a value from the store and records the command if AOF is enabled
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, key)
	s.recordCommand([]string{"DELETE", key})
}

// Get retrieves a value from the store
func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[key]
	if !exists {
		return nil, false
	}

	// Check if item has expired
	if item.Expiration != nil && time.Now().After(*item.Expiration) {
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

	// Check if item has expired
	if item.Expiration != nil && time.Now().After(*item.Expiration) {
		return false
	}

	return true
}

// DeleteExpired removes all expired items from the store
func (s *Store) DeleteExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, item := range s.items {
		if item.Expiration != nil && now.After(*item.Expiration) {
			delete(s.items, key)
		}
	}
}
