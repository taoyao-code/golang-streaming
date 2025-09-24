package scheduler

import (
	"os"
	"sync"
	"testing"
	"time"
)

// TestVideoCleanupService_CleanupOldTasks tests the fixed time duration usage
func TestVideoCleanupService_CleanupOldTasks(t *testing.T) {
	// Create a temporary storage for testing
	tempDir := t.TempDir()
	storage := NewTaskStorage(tempDir)

	// Create video cleanup service
	vcs := NewVideoCleanupService(storage, []string{tempDir})

	// Add a test task
	err := vcs.AddVideoDeletionTask("test-video.mp4")
	if err != nil {
		t.Fatalf("Failed to add deletion task: %v", err)
	}

	// Test the cleanup method (should not error with the fixed time.Duration usage)
	err = vcs.CleanupOldTasks()
	if err != nil {
		t.Errorf("CleanupOldTasks failed: %v", err)
	}
}

// TestWorker_StartStop tests the fixed worker shutdown mechanism
func TestWorker_StartStop(t *testing.T) {
	// Create a simple task runner for testing
	callCount := 0
	mu := sync.Mutex{}
	
	dispatcher := func(dataChan chan interface{}) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil
	}
	
	executor := func(dataChan chan interface{}) error {
		return nil
	}
	
	runner := NewTaskRunner(10, false, dispatcher, executor)
	worker := NewWorker(50*time.Millisecond, runner)
	
	// Start the worker
	worker.Start()
	
	// Let it run for a short time
	time.Sleep(150 * time.Millisecond)
	
	// Stop the worker
	worker.Stop()
	
	// Give it time to stop
	time.Sleep(100 * time.Millisecond)
	
	// Verify it's not running
	if worker.IsRunning() {
		t.Error("Worker should not be running after Stop()")
	}
	
	// Check that at least one task execution occurred
	mu.Lock()
	if callCount == 0 {
		t.Error("Expected at least one task execution")
	}
	mu.Unlock()
}

// TestTaskRunner_StartStop tests the task runner lifecycle
func TestTaskRunner_StartStop(t *testing.T) {
	executed := false
	mu := sync.Mutex{}
	
	dispatcher := func(dataChan chan interface{}) error {
		mu.Lock()
		executed = true
		mu.Unlock()
		return nil
	}
	
	executor := func(dataChan chan interface{}) error {
		return nil
	}
	
	runner := NewTaskRunner(10, false, dispatcher, executor)
	
	// Start the runner
	runner.Start()
	
	// Give it time to execute
	time.Sleep(50 * time.Millisecond)
	
	// Stop the runner
	runner.Stop()
	
	// Verify it's not running
	if runner.IsRunning() {
		t.Error("TaskRunner should not be running after Stop()")
	}
	
	// Check that the dispatcher was executed
	mu.Lock()
	if !executed {
		t.Error("Expected dispatcher to be executed")
	}
	mu.Unlock()
}

// TestVideoCleanupService_deleteVideo tests the video deletion functionality
func TestVideoCleanupService_deleteVideo(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewTaskStorage(tempDir)

	vcs := NewVideoCleanupService(storage, []string{tempDir})

	// Create a test file
	testFile := tempDir + "/test-video.mp4"
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()

	// Test deleting existing file
	err = vcs.deleteVideo(testFile)
	if err != nil {
		t.Errorf("Failed to delete existing file: %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should have been deleted")
	}

	// Test deleting non-existent file (should not error)
	err = vcs.deleteVideo(testFile + ".nonexistent")
	if err != nil {
		t.Errorf("Deleting non-existent file should not error: %v", err)
	}
}