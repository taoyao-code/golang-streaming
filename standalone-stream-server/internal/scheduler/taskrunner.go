package scheduler

import (
	"sync"
	"time"
)

// TaskRunner represents a background task execution engine
type TaskRunner struct {
	controller chan string
	errorChan  chan string
	dataChan   chan interface{}
	dataSize   int
	longLived  bool
	dispatcher TaskFunc
	executor   TaskFunc
	mu         sync.RWMutex
	running    bool
}

// TaskFunc defines the signature for task functions
type TaskFunc func(dataChan chan interface{}) error

// Channel control constants
const (
	ReadyToDispatch = "d"
	ReadyToExecute  = "e"
	Close          = "c"
)

// NewTaskRunner creates a new task runner instance
func NewTaskRunner(size int, longLived bool, dispatcher, executor TaskFunc) *TaskRunner {
	return &TaskRunner{
		controller: make(chan string, 1),
		errorChan:  make(chan string, 1),
		dataChan:   make(chan interface{}, size),
		dataSize:   size,
		longLived:  longLived,
		dispatcher: dispatcher,
		executor:   executor,
	}
}

// Start begins the task runner lifecycle
func (tr *TaskRunner) Start() {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	if tr.running {
		return
	}
	tr.running = true
	
	tr.controller <- ReadyToDispatch
	go tr.startDispatch()
}

// Stop gracefully stops the task runner
func (tr *TaskRunner) Stop() {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	if !tr.running {
		return
	}
	tr.running = false
	
	tr.errorChan <- Close
}

// IsRunning returns whether the task runner is currently active
func (tr *TaskRunner) IsRunning() bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.running
}

// startDispatch handles the main task runner loop
func (tr *TaskRunner) startDispatch() {
	defer func() {
		if !tr.longLived {
			close(tr.controller)
			close(tr.dataChan)
			close(tr.errorChan)
		}
		tr.mu.Lock()
		tr.running = false
		tr.mu.Unlock()
	}()

	for {
		select {
		case c := <-tr.controller:
			if c == ReadyToDispatch {
				err := tr.dispatcher(tr.dataChan)
				if err != nil {
					tr.errorChan <- Close
				} else {
					tr.controller <- ReadyToExecute
				}
			}
			
			if c == ReadyToExecute {
				err := tr.executor(tr.dataChan)
				if err != nil {
					tr.errorChan <- Close
				} else {
					tr.controller <- ReadyToDispatch
				}
			}
			
		case e := <-tr.errorChan:
			if e == Close {
				return
			}
		}
	}
}

// Worker manages timed execution of task runners
type Worker struct {
	ticker   *time.Ticker
	runner   *TaskRunner
	interval time.Duration
	mu       sync.RWMutex
	running  bool
}

// NewWorker creates a new worker with specified interval
func NewWorker(interval time.Duration, runner *TaskRunner) *Worker {
	return &Worker{
		interval: interval,
		runner:   runner,
	}
}

// Start begins the worker's timed execution
func (w *Worker) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.running {
		return
	}
	w.running = true
	w.ticker = time.NewTicker(w.interval)
	
	go w.startWorker()
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if !w.running {
		return
	}
	w.running = false
	
	if w.ticker != nil {
		w.ticker.Stop()
	}
	w.runner.Stop()
}

// IsRunning returns whether the worker is currently active
func (w *Worker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// startWorker handles the worker's main loop
func (w *Worker) startWorker() {
	defer func() {
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
	}()
	
	for {
		select {
		case <-w.ticker.C:
			if w.runner != nil {
				go w.runner.Start()
			}
		}
	}
}