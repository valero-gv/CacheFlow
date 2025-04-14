package store

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"
)

// Helper function to create a test store with a unique AOF file
// and ensure cleanup.
func createTestStore(t *testing.T) (*Store, string) {
	// Create a unique temp filename for AOF
	// Note: In Go 1.17+, t.TempDir() is better for creating temp directories.
	// For simplicity here, we just use a unique filename pattern in the current dir.
	// Warning: This might leave files if tests panic badly before defer runs.
	// A safer approach involves t.TempDir() or explicit cleanup in setup/teardown.
	aofFilename := fmt.Sprintf("test_%s_%d.aof", t.Name(), time.Now().UnixNano())

	s, err := New(aofFilename)
	if err != nil {
		// We need t.Fatalf here because without a store, the test cannot proceed.
		t.Fatalf("Failed to create store with AOF file %s: %v", aofFilename, err)
	}

	// Schedule cleanup actions using t.Cleanup (Go 1.14+) which is preferred over defer in tests
	// as it runs even if t.Fatal is called. If using older Go, use defer *before* t.Fatalf check.
	t.Cleanup(func() {
		err := s.Close()
		if err != nil {
			// Log closing error but don't fail the test just for this
			t.Logf("Warning: error closing store's AOF file %s: %v", aofFilename, err)
		}
		err = os.Remove(aofFilename)
		if err != nil && !os.IsNotExist(err) { // Don't worry if file is already gone
			t.Logf("Warning: error removing AOF file %s: %v", aofFilename, err)
		}
	})

	return s, aofFilename // Return store and filename (filename might be useful for AOF test)
}

// TestSetGet tests basic Set and Get operations without TTL.
func TestSetGet(t *testing.T) {
	s, _ := createTestStore(t) // Use helper to create store and setup cleanup

	// Test case 1: String value
	key1 := "mykey"
	value1 := "myvalue"
	s.Set(key1, value1, 0) // 0 TTL means no expiration

	retrievedVal1, exists1 := s.Get(key1)
	if !exists1 {
		t.Errorf("TestSetGet: Expected key '%s' to exist, but it doesn't", key1)
	}
	// Type assertion to check if the retrieved value is a string
	strVal1, ok1 := retrievedVal1.(string)
	if !ok1 {
		t.Errorf("TestSetGet: Expected value for key '%s' to be a string, but got %T", key1, retrievedVal1)
	} else if strVal1 != value1 {
		t.Errorf("TestSetGet: Expected value '%s' for key '%s', but got '%s'", value1, key1, strVal1)
	}

	// Test case 2: Integer value
	key2 := "numkey"
	value2 := 12345
	s.Set(key2, value2, 0)

	retrievedVal2, exists2 := s.Get(key2)
	if !exists2 {
		t.Errorf("TestSetGet: Expected key '%s' to exist, but it doesn't", key2)
	}
	intVal2, ok2 := retrievedVal2.(int)
	if !ok2 {
		t.Errorf("TestSetGet: Expected value for key '%s' to be an int, but got %T", key2, retrievedVal2)
	} else if intVal2 != value2 {
		t.Errorf("TestSetGet: Expected value %d for key '%s', but got %d", value2, key2, intVal2)
	}

	// Test case 3: Byte slice value
	key3 := "datakey"
	value3 := []byte("some data")
	s.Set(key3, value3, 0)

	retrievedVal3, exists3 := s.Get(key3)
	if !exists3 {
		t.Errorf("TestSetGet: Expected key '%s' to exist, but it doesn't", key3)
	}
	byteVal3, ok3 := retrievedVal3.([]byte)
	if !ok3 {
		t.Errorf("TestSetGet: Expected value for key '%s' to be a []byte, but got %T", key3, retrievedVal3)
	} else if !bytes.Equal(byteVal3, value3) { // Use bytes.Equal for comparing slices
		t.Errorf("TestSetGet: Expected value %v for key '%s', but got %v", value3, key3, byteVal3)
	}

	// Test case 4: Overwriting a key
	newValue1 := "new value"
	s.Set(key1, newValue1, 0) // Overwrite key1
	retrievedVal1_overwrite, exists1_overwrite := s.Get(key1)
	if !exists1_overwrite {
		t.Errorf("TestSetGet: Expected key '%s' to exist after overwrite, but it doesn't", key1)
	}
	strVal1_overwrite, ok1_overwrite := retrievedVal1_overwrite.(string)
	if !ok1_overwrite {
		t.Errorf("TestSetGet: Expected overwritten value for key '%s' to be a string, but got %T", key1, retrievedVal1_overwrite)
	} else if strVal1_overwrite != newValue1 {
		t.Errorf("TestSetGet: Expected overwritten value '%s' for key '%s', but got '%s'", newValue1, key1, strVal1_overwrite)
	}

	// Test case 5: Getting a non-existent key
	keyNonExistent := "nokey"
	_, existsNonExistent := s.Get(keyNonExistent)
	if existsNonExistent {
		t.Errorf("TestSetGet: Expected key '%s' not to exist, but it does", keyNonExistent)
	}
}

// TestTTL tests the time-to-live functionality.
func TestTTL(t *testing.T) {
	s, _ := createTestStore(t) // Use helper to create store and setup cleanup

	keyTTL := "ttl_key"
	valueTTL := "temporary value"
	ttlDuration := 100 * time.Millisecond // Short TTL for testing

	// 1. Set key with TTL
	s.Set(keyTTL, valueTTL, ttlDuration)

	// 2. Check immediately - should exist
	retrievedVal1, exists1 := s.Get(keyTTL)
	if !exists1 {
		t.Errorf("TestTTL: Expected key '%s' to exist immediately after set, but it doesn't", keyTTL)
	}
	strVal1, ok1 := retrievedVal1.(string)
	if !ok1 {
		t.Errorf("TestTTL: Expected value for key '%s' to be a string, got %T", keyTTL, retrievedVal1)
	} else if strVal1 != valueTTL {
		t.Errorf("TestTTL: Expected value '%s' for key '%s', got '%s'", valueTTL, keyTTL, strVal1)
	}

	existsCheck1 := s.Exists(keyTTL)
	if !existsCheck1 {
		t.Errorf("TestTTL: Exists check failed for key '%s' immediately after set", keyTTL)
	}

	// 3. Wait for TTL to expire
	time.Sleep(ttlDuration + 50*time.Millisecond) // Wait slightly longer than TTL

	// 4. Check after TTL expiration - should not exist
	_, exists2 := s.Get(keyTTL)
	if exists2 {
		t.Errorf("TestTTL: Expected key '%s' to be expired, but Get still finds it", keyTTL)
	}

	existsCheck2 := s.Exists(keyTTL)
	if existsCheck2 {
		t.Errorf("TestTTL: Exists check failed for key '%s' after expiration", keyTTL)
	}

	// 5. Test key without TTL - should persist
	keyNoTTL := "no_ttl_key"
	valueNoTTL := "permanent value"
	s.Set(keyNoTTL, valueNoTTL, 0) // No TTL

	// Wait a bit, less than the previous TTL duration
	time.Sleep(50 * time.Millisecond)

	retrievedVal3, exists3 := s.Get(keyNoTTL)
	if !exists3 {
		t.Errorf("TestTTL: Expected key '%s' (no TTL) to exist, but it doesn't", keyNoTTL)
	}
	strVal3, ok3 := retrievedVal3.(string)
	if !ok3 {
		t.Errorf("TestTTL: Expected value for key '%s' to be a string, got %T", keyNoTTL, retrievedVal3)
	} else if strVal3 != valueNoTTL {
		t.Errorf("TestTTL: Expected value '%s' for key '%s', got '%s'", valueNoTTL, keyNoTTL, strVal3)
	}
}

// TestDelete tests the Delete operation.
func TestDelete(t *testing.T) {
	s, _ := createTestStore(t) // Use helper to create store and setup cleanup

	key1 := "delete_me"
	value1 := "some value"
	key2 := "keep_me"
	value2 := "another value"

	// Set two keys
	s.Set(key1, value1, 0)
	s.Set(key2, value2, 0)

	// Check they both exist initially
	if _, exists := s.Get(key1); !exists {
		t.Fatalf("TestDelete: Pre-condition failed. Key '%s' should exist before delete.", key1)
	}
	if _, exists := s.Get(key2); !exists {
		t.Fatalf("TestDelete: Pre-condition failed. Key '%s' should exist before delete.", key2)
	}

	// Delete key1
	s.Delete(key1)

	// Check key1 is gone
	if _, exists := s.Get(key1); exists {
		t.Errorf("TestDelete: Expected key '%s' to be deleted, but it still exists.", key1)
	}
	if s.Exists(key1) { // Also check with Exists
		t.Errorf("TestDelete: Expected Exists check for key '%s' to return false after delete.", key1)
	}

	// Check key2 still exists
	val2, exists2 := s.Get(key2)
	if !exists2 {
		t.Errorf("TestDelete: Expected key '%s' to still exist after deleting key '%s', but it doesn't.", key2, key1)
	}
	strVal2, ok2 := val2.(string)
	if !ok2 || strVal2 != value2 {
		t.Errorf("TestDelete: Value for key '%s' seems corrupted after deleting key '%s'. Got: %v", key2, key1, val2)
	}

	// Test deleting a non-existent key (should not panic or error)
	keyNonExistent := "already_gone"
	// Ensure it doesn't exist first
	if _, exists := s.Get(keyNonExistent); exists {
		t.Fatalf("TestDelete: Pre-condition failed. Key '%s' should not exist before attempting delete.", keyNonExistent)
	}
	// Attempt to delete
	s.Delete(keyNonExistent)
	// Check it still doesn't exist (no side effects)
	if _, exists := s.Get(keyNonExistent); exists {
		t.Errorf("TestDelete: Deleting a non-existent key '%s' caused it to exist somehow.", keyNonExistent)
	}
}

// TestDeleteExpired tests the background cleanup function.
func TestDeleteExpired(t *testing.T) {
	s, _ := createTestStore(t) // Use helper to create store and setup cleanup

	keyExpired := "expired_key"
	keyNotExpired := "not_expired_key"
	keyNoTTL := "no_ttl_key"

	ttlShort := 50 * time.Millisecond
	ttlLong := 200 * time.Millisecond

	// Set keys: one expired soon, one later, one never
	s.Set(keyExpired, "value1", ttlShort)
	s.Set(keyNotExpired, "value2", ttlLong)
	s.Set(keyNoTTL, "value3", 0) // No TTL

	// Wait for the short TTL to expire, but not the long one
	time.Sleep(ttlShort + 20*time.Millisecond) // Wait a bit longer than ttlShort

	// Run the cleanup function
	s.DeleteExpired()

	// Check results after first cleanup
	if _, exists := s.Get(keyExpired); exists {
		t.Errorf("TestDeleteExpired: Key '%s' should have been deleted by DeleteExpired, but still exists.", keyExpired)
	}
	if _, exists := s.Get(keyNotExpired); !exists {
		t.Errorf("TestDeleteExpired: Key '%s' should NOT have been deleted yet, but it's gone.", keyNotExpired)
	}
	if _, exists := s.Get(keyNoTTL); !exists {
		t.Errorf("TestDeleteExpired: Key '%s' (no TTL) should NOT have been deleted, but it's gone.", keyNoTTL)
	}

	// Wait for the long TTL to expire
	// We already waited ttlShort + 20ms, need to wait remaining time for ttlLong
	remainingWait := ttlLong - (ttlShort + 20*time.Millisecond)
	if remainingWait > 0 {
		time.Sleep(remainingWait + 20*time.Millisecond) // Wait a bit longer than ttlLong total
	}

	// Run the cleanup function again
	s.DeleteExpired()

	// Check results after second cleanup
	if _, exists := s.Get(keyNotExpired); exists {
		t.Errorf("TestDeleteExpired: Key '%s' should have been deleted by the second DeleteExpired run, but still exists.", keyNotExpired)
	}
	if _, exists := s.Get(keyNoTTL); !exists {
		t.Errorf("TestDeleteExpired: Key '%s' (no TTL) should still exist after second cleanup, but it's gone.", keyNoTTL)
	}
}

// Add other test functions here later, e.g., TestTTL, TestDelete, TestExists, TestDeleteExpired
