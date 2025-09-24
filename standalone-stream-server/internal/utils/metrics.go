package utils

import (
"time"

"github.com/prometheus/client_golang/prometheus"
"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
// HTTP request metrics
HTTPRequestsTotal = promauto.NewCounterVec(
prometheus.CounterOpts{
Name: "http_requests_total",
Help: "Total number of HTTP requests",
},
[]string{"method", "endpoint", "status"},
)

HTTPRequestDuration = promauto.NewHistogramVec(
prometheus.HistogramOpts{
Name:    "http_request_duration_seconds",
Help:    "HTTP request duration in seconds",
Buckets: prometheus.DefBuckets,
},
[]string{"method", "endpoint"},
)

// Video streaming metrics
VideoStreamsTotal = promauto.NewCounterVec(
prometheus.CounterOpts{
Name: "video_streams_total",
Help: "Total number of video streams served",
},
[]string{"directory", "status"},
)

VideoStreamDuration = promauto.NewHistogramVec(
prometheus.HistogramOpts{
Name:    "video_stream_duration_seconds",
Help:    "Video stream duration in seconds",
Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
},
[]string{"directory"},
)

// System metrics
ActiveConnections = promauto.NewGauge(
prometheus.GaugeOpts{
Name: "active_connections_total",
Help: "Number of active connections",
},
)

VideoFilesTotal = promauto.NewGaugeVec(
prometheus.GaugeOpts{
Name: "video_files_total",
Help: "Total number of video files by directory",
},
[]string{"directory"},
)

// Scheduler metrics
SchedulerTasksTotal = promauto.NewCounterVec(
prometheus.CounterOpts{
Name: "scheduler_tasks_total",
Help: "Total number of scheduler tasks",
},
[]string{"task_type", "status"},
)

SchedulerWorkerStatus = promauto.NewGaugeVec(
prometheus.GaugeOpts{
Name: "scheduler_worker_status",
Help: "Scheduler worker status (1=running, 0=stopped)",
},
[]string{"worker_name"},
)
)

// RecordHTTPRequest records an HTTP request metric
func RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordVideoStream records a video stream metric
func RecordVideoStream(directory, status string, duration time.Duration) {
VideoStreamsTotal.WithLabelValues(directory, status).Inc()
if status == "success" {
VideoStreamDuration.WithLabelValues(directory).Observe(duration.Seconds())
}
}

// UpdateActiveConnections updates the active connections gauge
func UpdateActiveConnections(count int) {
ActiveConnections.Set(float64(count))
}

// UpdateVideoFilesCount updates the video files count for a directory
func UpdateVideoFilesCount(directory string, count int) {
VideoFilesTotal.WithLabelValues(directory).Set(float64(count))
}

// RecordSchedulerTask records a scheduler task metric
func RecordSchedulerTask(taskType, status string) {
SchedulerTasksTotal.WithLabelValues(taskType, status).Inc()
}

// UpdateSchedulerWorkerStatus updates scheduler worker status
func UpdateSchedulerWorkerStatus(workerName string, running bool) {
var value float64
if running {
value = 1
}
SchedulerWorkerStatus.WithLabelValues(workerName).Set(value)
}
