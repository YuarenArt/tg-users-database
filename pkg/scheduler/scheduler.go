package scheduler

import (
	"log"

	"github.com/YuarenArt/tg-users-database/pkg/db"

	"github.com/robfig/cron"
)

const (
	resetTraffic       = "resetTraffic"
	checkSubscriptions = "checkSubscriptions"
)

var schedulerPlans = map[string]string{
	resetTraffic:       "@weekly",
	checkSubscriptions: "@daily",
}

// Task represents a task to be executed by the scheduler
type Task struct {
	Name     string
	Schedule string
	Run      func()
}

// Scheduler is a struct that holds the cron scheduler and a list of tasks
type Scheduler struct {
	cron  *cron.Cron
	tasks []Task
	db    *db.Database
}

// NewScheduler creates a new Scheduler instance
func NewScheduler(db *db.Database) *Scheduler {
	s := &Scheduler{
		cron:  cron.New(),
		tasks: []Task{},
		db:    db,
	}

	// Initialize and register tasks
	s.initializeTasks()

	return s
}

// Start initializes and starts the scheduler
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// initializeTasks registers provided tasks using the schedulerPlans map
func (s *Scheduler) initializeTasks() {
	for name, schedule := range schedulerPlans {
		s.RegisterTask(name, schedule, s.getTaskRunFunction(name))
	}
}

// RegisterTask adds a task to the scheduler
func (s *Scheduler) RegisterTask(name, schedule string, run func()) {
	task := Task{
		Name:     name,
		Schedule: schedule,
		Run:      run,
	}
	s.tasks = append(s.tasks, task)

	if err := s.cron.AddFunc(schedule, run); err != nil {
		log.Printf("Failed to add task %s to the scheduler: %v", name, err)
	}
}

// getTaskRunFunction returns the appropriate function to run based on the task name
func (s *Scheduler) getTaskRunFunction(name string) func() {
	switch name {
	case resetTraffic:
		return s.checkAndResetTraffic
	case checkSubscriptions:
		return s.checkAndUpdateSubscriptions
	default:
		return func() {
			log.Printf("No task function found for %s", name)
		}
	}
}
