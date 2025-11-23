package cache

import (
	"testing"
	"time"
)

func TestMemoryManager(t *testing.T) {
	mgr := NewMemoryManager(1 * time.Hour)

	// Test Set and Get
	type testData struct {
		Name  string
		Value int
	}

	original := testData{Name: "test", Value: 42}

	if err := mgr.Set("test-key", original); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	var retrieved testData
	found, err := mgr.Get("test-key", &retrieved)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if !found {
		t.Fatal("Expected to find cached value")
	}

	if retrieved.Name != original.Name || retrieved.Value != original.Value {
		t.Errorf("Retrieved data doesn't match. Got %+v, want %+v", retrieved, original)
	}
}

func TestMemoryManagerExpiration(t *testing.T) {
	mgr := NewMemoryManager(100 * time.Millisecond)

	mgr.Set("test-key", "test-value")

	// Should be found immediately
	var value1 string
	found, _ := mgr.Get("test-key", &value1)
	if !found {
		t.Fatal("Expected to find cached value")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should not be found after expiration
	var value2 string
	found, _ = mgr.Get("test-key", &value2)
	if found {
		t.Error("Expected cache to be expired")
	}
}

func TestMemoryManagerDelete(t *testing.T) {
	mgr := NewMemoryManager(1 * time.Hour)

	mgr.Set("test-key", "test-value")

	if err := mgr.Delete("test-key"); err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	var value string
	found, _ := mgr.Get("test-key", &value)
	if found {
		t.Error("Expected key to be deleted")
	}
}

func TestMemoryManagerClear(t *testing.T) {
	mgr := NewMemoryManager(1 * time.Hour)

	mgr.Set("key1", "value1")
	mgr.Set("key2", "value2")
	mgr.Set("key3", "value3")

	if err := mgr.Clear(); err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	var value string
	found, _ := mgr.Get("key1", &value)
	if found {
		t.Error("Expected all entries to be cleared")
	}
}

func TestMemoryManagerStats(t *testing.T) {
	mgr := NewMemoryManager(100 * time.Millisecond)

	mgr.Set("key1", "value1")
	mgr.Set("key2", "value2")

	total, expired, err := mgr.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 total entries, got %d", total)
	}

	if expired != 0 {
		t.Errorf("Expected 0 expired entries, got %d", expired)
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	total, expired, err = mgr.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 total entries, got %d", total)
	}

	if expired != 2 {
		t.Errorf("Expected 2 expired entries, got %d", expired)
	}
}
