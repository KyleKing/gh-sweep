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

	if err := mgr.Set("test-key", "test-value"); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	// Should be found immediately
	var value1 string
	found, err := mgr.Get("test-key", &value1)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if !found {
		t.Fatal("Expected to find cached value")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should not be found after expiration
	var value2 string
	found, err = mgr.Get("test-key", &value2)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if found {
		t.Error("Expected cache to be expired")
	}
}

func TestMemoryManagerDelete(t *testing.T) {
	mgr := NewMemoryManager(1 * time.Hour)

	if err := mgr.Set("test-key", "test-value"); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	if err := mgr.Delete("test-key"); err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	var value string
	found, err := mgr.Get("test-key", &value)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if found {
		t.Error("Expected key to be deleted")
	}
}

func TestMemoryManagerClear(t *testing.T) {
	mgr := NewMemoryManager(1 * time.Hour)

	if err := mgr.Set("key1", "value1"); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}
	if err := mgr.Set("key2", "value2"); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}
	if err := mgr.Set("key3", "value3"); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	if err := mgr.Clear(); err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	var value string
	found, err := mgr.Get("key1", &value)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if found {
		t.Error("Expected all entries to be cleared")
	}
}

func TestMemoryManagerStats(t *testing.T) {
	mgr := NewMemoryManager(100 * time.Millisecond)

	if err := mgr.Set("key1", "value1"); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}
	if err := mgr.Set("key2", "value2"); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

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

func TestMemoryManagerCleanExpired(t *testing.T) {
	mgr := NewMemoryManager(100 * time.Millisecond)

	if err := mgr.Set("fresh", "value"); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if err := mgr.Set("still-fresh", "value"); err != nil {
		t.Fatalf("Failed to set: %v", err)
	}

	if err := mgr.CleanExpired(); err != nil {
		t.Fatalf("CleanExpired() error = %v", err)
	}

	total, expired, err := mgr.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if total != 1 || expired != 0 {
		t.Errorf("Stats() after CleanExpired = (total=%d, expired=%d), want (1, 0)", total, expired)
	}
}

func TestMemoryManagerClose(t *testing.T) {
	mgr := NewMemoryManager(time.Hour)

	if err := mgr.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
