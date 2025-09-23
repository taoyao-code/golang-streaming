package scheduler

import (
	"log"
	"path/filepath"
	"standalone-stream-server/internal/models"
	"sync"
	"time"
)

// SchedulerService manages all background tasks and workers
type SchedulerService struct {
	config             *models.Config
	storage            *TaskStorage
	videoCleanupService *VideoCleanupService
	workers            map[string]*Worker
	taskRunners        map[string]*TaskRunner
	mu                 sync.RWMutex
	running            bool
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(config *models.Config) *SchedulerService {
	// Create task storage directory
	dataDir := filepath.Join(".", "data", "tasks")
	storage := NewTaskStorage(dataDir)
	
	// Extract video directories from config
	var videoDirs []string
	for _, dir := range config.Video.Directories {
		if dir.Enabled {
			videoDirs = append(videoDirs, dir.Path)
		}
	}
	
	videoCleanupService := NewVideoCleanupService(storage, videoDirs)
	
	return &SchedulerService{
		config:              config,
		storage:             storage,
		videoCleanupService: videoCleanupService,
		workers:             make(map[string]*Worker),
		taskRunners:         make(map[string]*TaskRunner),
	}
}

// Start initializes and starts all background services
func (ss *SchedulerService) Start() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	if ss.running {
		return nil
	}
	
	log.Println("Starting scheduler service...")
	
	// Create video cleanup task runner
	videoCleanupRunner := NewTaskRunner(
		3,    // buffer size
		true, // long-lived
		ss.videoCleanupService.VideoClearDispatcher,
		ss.videoCleanupService.VideoClearExecutor,
	)
	ss.taskRunners["video_cleanup"] = videoCleanupRunner
	
	// Create worker for video cleanup (runs every 30 seconds)
	videoCleanupWorker := NewWorker(30*time.Second, videoCleanupRunner)
	ss.workers["video_cleanup"] = videoCleanupWorker
	
	// Create cleanup worker for old tasks (runs every hour)
	cleanupTaskRunner := NewTaskRunner(
		1,    // buffer size
		true, // long-lived
		ss.cleanupDispatcher,
		ss.cleanupExecutor,
	)
	ss.taskRunners["cleanup"] = cleanupTaskRunner
	
	cleanupWorker := NewWorker(1*time.Hour, cleanupTaskRunner)
	ss.workers["cleanup"] = cleanupWorker
	
	// Start all workers
	for name, worker := range ss.workers {
		worker.Start()
		log.Printf("Started %s worker", name)
	}
	
	ss.running = true
	log.Println("Scheduler service started successfully")
	
	return nil
}

// Stop gracefully shuts down all background services
func (ss *SchedulerService) Stop() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	if !ss.running {
		return nil
	}
	
	log.Println("Stopping scheduler service...")
	
	// Stop all workers
	for name, worker := range ss.workers {
		worker.Stop()
		log.Printf("Stopped %s worker", name)
	}
	
	ss.running = false
	log.Println("Scheduler service stopped successfully")
	
	return nil
}

// IsRunning returns whether the scheduler service is running
func (ss *SchedulerService) IsRunning() bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.running
}

// AddVideoDeletionTask schedules a video for deletion
func (ss *SchedulerService) AddVideoDeletionTask(videoPath string) error {
	return ss.videoCleanupService.AddVideoDeletionTask(videoPath)
}

// GetStats returns statistics about the scheduler service
func (ss *SchedulerService) GetStats() map[string]interface{} {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	
	stats := map[string]interface{}{
		"running":        ss.running,
		"active_workers": len(ss.workers),
		"task_runners":   len(ss.taskRunners),
	}
	
	// Add worker status
	workerStats := make(map[string]bool)
	for name, worker := range ss.workers {
		workerStats[name] = worker.IsRunning()
	}
	stats["worker_status"] = workerStats
	
	// Add task runner status
	runnerStats := make(map[string]bool)
	for name, runner := range ss.taskRunners {
		runnerStats[name] = runner.IsRunning()
	}
	stats["runner_status"] = runnerStats
	
	// Add video cleanup stats
	if videoStats, err := ss.videoCleanupService.GetStats(); err == nil {
		stats["video_cleanup"] = videoStats
	}
	
	return stats
}

// cleanupDispatcher handles dispatching cleanup tasks
func (ss *SchedulerService) cleanupDispatcher(dataChan chan interface{}) error {
	// Send a cleanup signal
	dataChan <- "cleanup_old_tasks"
	return nil
}

// cleanupExecutor handles executing cleanup tasks
func (ss *SchedulerService) cleanupExecutor(dataChan chan interface{}) error {
	// Process cleanup tasks
	for {
		select {
		case task := <-dataChan:
			if taskStr, ok := task.(string); ok && taskStr == "cleanup_old_tasks" {
				if err := ss.videoCleanupService.CleanupOldTasks(); err != nil {
					log.Printf("Failed to cleanup old tasks: %v", err)
					return err
				}
				log.Println("Successfully cleaned up old tasks")
			}
		default:
			return nil
		}
	}
}