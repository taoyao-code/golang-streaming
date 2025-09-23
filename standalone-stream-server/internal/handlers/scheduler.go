package handlers

import (
	"standalone-stream-server/internal/models"
	"standalone-stream-server/internal/scheduler"

	"github.com/gofiber/fiber/v2"
)

// SchedulerHandler handles scheduler-related requests
type SchedulerHandler struct {
	config           *models.Config
	schedulerService *scheduler.SchedulerService
}

// NewSchedulerHandler creates a new scheduler handler
func NewSchedulerHandler(config *models.Config, schedulerService *scheduler.SchedulerService) *SchedulerHandler {
	return &SchedulerHandler{
		config:           config,
		schedulerService: schedulerService,
	}
}

// GetStats returns scheduler statistics
func (sh *SchedulerHandler) GetStats(c *fiber.Ctx) error {
	stats := sh.schedulerService.GetStats()
	
	return c.JSON(fiber.Map{
		"scheduler": stats,
		"timestamp": fiber.Map{
			"unix":   c.Context().Time().Unix(),
			"format": c.Context().Time().Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

// AddVideoDeletionTask schedules a video for deletion
func (sh *SchedulerHandler) AddVideoDeletionTask(c *fiber.Ctx) error {
	videoID := c.Params("video-id")
	if videoID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Video ID is required",
		})
	}
	
	// For now, we'll use the video ID as the path
	// In a real implementation, you'd look up the actual file path
	videoPath := videoID
	
	if err := sh.schedulerService.AddVideoDeletionTask(videoPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to schedule video deletion",
			"details": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"message":  "Video deletion scheduled successfully",
		"video_id": videoID,
	})
}

// Start starts the scheduler service
func (sh *SchedulerHandler) Start(c *fiber.Ctx) error {
	if err := sh.schedulerService.Start(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to start scheduler service",
			"details": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"message": "Scheduler service started successfully",
	})
}

// Stop stops the scheduler service
func (sh *SchedulerHandler) Stop(c *fiber.Ctx) error {
	if err := sh.schedulerService.Stop(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to stop scheduler service",
			"details": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"message": "Scheduler service stopped successfully",
	})
}

// Status returns the current status of the scheduler service
func (sh *SchedulerHandler) Status(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"running": sh.schedulerService.IsRunning(),
		"stats":   sh.schedulerService.GetStats(),
	})
}