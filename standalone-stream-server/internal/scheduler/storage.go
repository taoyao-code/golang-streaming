package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TaskRecord represents a scheduled task
type TaskRecord struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Data      string    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"` // pending, processing, completed, failed
}

// TaskStorage handles persistence of task records
type TaskStorage struct {
	dataDir string
	mu      sync.RWMutex
}

// NewTaskStorage creates a new task storage instance
func NewTaskStorage(dataDir string) *TaskStorage {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create task storage directory: %v", err))
	}
	
	return &TaskStorage{
		dataDir: dataDir,
	}
}

// AddTask adds a new task to the storage
func (ts *TaskStorage) AddTask(taskType, data string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	task := TaskRecord{
		ID:        fmt.Sprintf("%d_%s", time.Now().UnixNano(), taskType),
		Type:      taskType,
		Data:      data,
		CreatedAt: time.Now(),
		Status:    "pending",
	}
	
	filename := filepath.Join(ts.dataDir, fmt.Sprintf("%s.json", task.ID))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create task file: %w", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(task); err != nil {
		return fmt.Errorf("failed to encode task: %w", err)
	}
	
	return nil
}

// GetPendingTasks retrieves a limited number of pending tasks
func (ts *TaskStorage) GetPendingTasks(taskType string, limit int) ([]TaskRecord, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	var tasks []TaskRecord
	
	// Read all files in the data directory
	files, err := os.ReadDir(ts.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read task directory: %w", err)
	}
	
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			task, err := ts.readTaskFile(filepath.Join(ts.dataDir, file.Name()))
			if err != nil {
				continue // Skip corrupted files
			}
			
			if task.Type == taskType && task.Status == "pending" {
				tasks = append(tasks, task)
				if len(tasks) >= limit {
					break
				}
			}
		}
	}
	
	return tasks, nil
}

// UpdateTaskStatus updates the status of a task
func (ts *TaskStorage) UpdateTaskStatus(taskID, status string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	filename := filepath.Join(ts.dataDir, fmt.Sprintf("%s.json", taskID))
	
	// Read current task
	task, err := ts.readTaskFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read task: %w", err)
	}
	
	// Update status
	task.Status = status
	
	// Write back to file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to update task file: %w", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(task); err != nil {
		return fmt.Errorf("failed to encode updated task: %w", err)
	}
	
	return nil
}

// RemoveTask removes a task from storage
func (ts *TaskStorage) RemoveTask(taskID string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	filename := filepath.Join(ts.dataDir, fmt.Sprintf("%s.json", taskID))
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove task: %w", err)
	}
	
	return nil
}

// CleanupCompletedTasks removes completed and failed tasks older than the specified duration
func (ts *TaskStorage) CleanupCompletedTasks(olderThan time.Duration) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	cutoff := time.Now().Add(-olderThan)
	
	files, err := os.ReadDir(ts.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read task directory: %w", err)
	}
	
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			task, err := ts.readTaskFile(filepath.Join(ts.dataDir, file.Name()))
			if err != nil {
				continue
			}
			
			if (task.Status == "completed" || task.Status == "failed") && task.CreatedAt.Before(cutoff) {
				os.Remove(filepath.Join(ts.dataDir, file.Name()))
			}
		}
	}
	
	return nil
}

// GetTaskStats returns statistics about tasks
func (ts *TaskStorage) GetTaskStats() (map[string]int, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	stats := map[string]int{
		"pending":    0,
		"processing": 0,
		"completed":  0,
		"failed":     0,
		"total":      0,
	}
	
	files, err := os.ReadDir(ts.dataDir)
	if err != nil {
		return stats, fmt.Errorf("failed to read task directory: %w", err)
	}
	
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			task, err := ts.readTaskFile(filepath.Join(ts.dataDir, file.Name()))
			if err != nil {
				continue
			}
			
			stats[task.Status]++
			stats["total"]++
		}
	}
	
	return stats, nil
}

// readTaskFile reads a task from a JSON file
func (ts *TaskStorage) readTaskFile(filename string) (TaskRecord, error) {
	var task TaskRecord
	
	file, err := os.Open(filename)
	if err != nil {
		return task, err
	}
	defer file.Close()
	
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&task); err != nil {
		return task, err
	}
	
	return task, nil
}