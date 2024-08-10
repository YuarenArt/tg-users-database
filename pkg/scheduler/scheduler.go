package scheduler

import (
	"context"
	"fmt"
	"github.com/robfig/cron"
	"log"
	"os"
	"tg-users-database/pkg/db"
	"time"
)

// FilePath to store the last reset time
const resetTrafficFilePath = "docs/last_reset_time.txt"

const (
	resetTraffic = iota
)

var schedulerPlans = map[int]struct {
	Name     string
	Schedule string
}{
	resetTraffic: {"resetTraffic", "@weekly"},
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
	for _, plan := range schedulerPlans {
		s.RegisterTask(plan.Name, plan.Schedule, s.getTaskRunFunction(plan.Name))
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
	case "resetTraffic":
		return s.checkAndResetTraffic
	default:
		return func() {
			log.Printf("No task function found for %s", name)
		}
	}
}

func (s *Scheduler) checkAndResetTraffic() {
	now := time.Now()
	lastResetTime, err := LastResetTimeFromFile()
	if err != nil {
		log.Printf("Failed to read last reset time: %v", err)
		return
	}

	// Check if the month has changed
	if lastResetTime.Year() != now.Year() || lastResetTime.Month() != now.Month() {
		log.Println("Starts reset user's traffic")
		// Reset traffic for all users
		s.resetAllUserTraffic()

		// Update last reset time to the first day of the current month
		newResetTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, time.Local)
		if err := UpdateLastResetTimeInFile(newResetTime); err != nil {
			log.Printf("Failed to update last reset time: %v", err)
		} else {
			log.Println("Successful update last reset time")
		}
	}
}

func (s *Scheduler) resetAllUserTraffic() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	usernames, err := s.db.AllUsername(ctx)
	if err != nil {
		log.Printf("Failed to get all users: %v", err)
		return
	}
	for _, username := range usernames {
		if err := s.db.ResetUserTraffic(ctx, username); err != nil {
			log.Printf("Failed to reset traffic for user %s: %v", username, err)
		}
	}
}

// LastResetTimeFromFile reads the last reset time from a single file.
// It returns the last reset time stored in the file. If the file does not exist, it returns the zero time value.
func LastResetTimeFromFile() (time.Time, error) {
	if _, err := os.Stat(resetTrafficFilePath); os.IsNotExist(err) {
		file, err := os.Create(resetTrafficFilePath)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to create file: %w", err)
		}
		defer file.Close()

		currentTime := time.Now()
		_, err = file.WriteString(currentTime.Format(time.RFC3339))
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to write current time to file: %w", err)
		}

		return currentTime, nil
	} else if err != nil {
		return time.Time{}, fmt.Errorf("failed to check file existence: %w", err)
	}

	file, err := os.Open(resetTrafficFilePath)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lastResetTimeStr string
	_, err = fmt.Fscanf(file, "%s", &lastResetTimeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read last reset time: %w", err)
	}

	lastResetTime, err := time.Parse(time.RFC3339, lastResetTimeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse last reset time: %w", err)
	}

	return lastResetTime, nil
}

// UpdateLastResetTimeInFile updates the last reset time in the single file.
// It writes the new last reset time to the file.
func UpdateLastResetTimeInFile(lastResetTime time.Time) error {
	file, err := os.Create(resetTrafficFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%s", lastResetTime.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to write last reset time: %w", err)
	}

	return nil
}
