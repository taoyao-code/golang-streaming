package scheduler

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

// VideoCleanupService handles video file cleanup tasks
type VideoCleanupService struct {
	storage   *TaskStorage
	videoDirs []string
	mu        sync.RWMutex
}

// NewVideoCleanupService creates a new video cleanup service
func NewVideoCleanupService(storage *TaskStorage, videoDirs []string) *VideoCleanupService {
	return &VideoCleanupService{
		storage:   storage,
		videoDirs: videoDirs,
	}
}

// AddVideoDeletionTask adds a video for deletion
func (vcs *VideoCleanupService) AddVideoDeletionTask(videoPath string) error {
	return vcs.storage.AddTask("video_deletion", videoPath)
}

// VideoClearDispatcher dispatches video deletion tasks
func (vcs *VideoCleanupService) VideoClearDispatcher(dataChan chan interface{}) error {
	tasks, err := vcs.storage.GetPendingTasks("video_deletion", 3)
	if err != nil {
		log.Printf("Video clear dispatcher error: %v", err)
		return err
	}
	
	if len(tasks) == 0 {
		return errors.New("no pending video deletion tasks")
	}
	
	// Mark tasks as processing and send to executor
	for _, task := range tasks {
		if err := vcs.storage.UpdateTaskStatus(task.ID, "processing"); err != nil {
			log.Printf("Failed to update task status: %v", err)
			continue
		}
		
		dataChan <- task
	}
	
	return nil
}

// VideoClearExecutor executes video deletion tasks
func (vcs *VideoCleanupService) VideoClearExecutor(dataChan chan interface{}) error {
	errorMap := &sync.Map{}
	var wg sync.WaitGroup
	
	// Process all available tasks
	for {
		select {
		case taskInterface := <-dataChan:
			wg.Add(1)
			go func(t interface{}) {
				defer wg.Done()
				
				task, ok := t.(TaskRecord)
				if !ok {
					log.Printf("Invalid task type received")
					return
				}
				
				if err := vcs.deleteVideo(task.Data); err != nil {
					log.Printf("Failed to delete video %s: %v", task.Data, err)
					errorMap.Store(task.ID, err)
					vcs.storage.UpdateTaskStatus(task.ID, "failed")
					return
				}
				
				// Successfully deleted, remove the task
				if err := vcs.storage.RemoveTask(task.ID); err != nil {
					log.Printf("Failed to remove completed task %s: %v", task.ID, err)
					errorMap.Store(task.ID, err)
					return
				}
				
				log.Printf("Successfully deleted video: %s", task.Data)
			}(taskInterface)
			
		default:
			// No more tasks available
			goto waitForCompletion
		}
	}
	
waitForCompletion:
	wg.Wait()
	
	// Check if any errors occurred
	var lastError error
	errorMap.Range(func(k, v interface{}) bool {
		lastError = v.(error)
		return true
	})
	
	return lastError
}

// deleteVideo removes a video file from the filesystem
func (vcs *VideoCleanupService) deleteVideo(videoPath string) error {
	// Check if file exists
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		// File doesn't exist, consider it successfully deleted
		return nil
	}
	
	// Attempt to delete the file
	if err := os.Remove(videoPath); err != nil {
		return fmt.Errorf("failed to delete video file %s: %w", videoPath, err)
	}
	
	return nil
}

// GetStats returns statistics about video cleanup tasks
func (vcs *VideoCleanupService) GetStats() (map[string]interface{}, error) {
	taskStats, err := vcs.storage.GetTaskStats()
	if err != nil {
		return nil, err
	}
	
	stats := map[string]interface{}{
		"video_deletion_tasks": taskStats,
		"configured_directories": len(vcs.videoDirs),
	}
	
	return stats, nil
}

// CleanupOldTasks removes old completed and failed tasks
func (vcs *VideoCleanupService) CleanupOldTasks() error {
	// Clean up tasks older than 24 hours
	return vcs.storage.CleanupCompletedTasks(24 * 60 * 60 * 1000000000) // 24 hours in nanoseconds
}