package models

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type task func() (any, error)
type valueOnlyTask func() any
type errorOnlyTask func() error

type TaskResponse struct {
	val any
	err error
}

type Task struct {
	task    task
	channel chan TaskResponse
}

// TaskQueue serializes database writes through a single worker goroutine to avoid
// SQLite SQLITE_BUSY contention. Use EnQueue (fire-and-forget) for writes where
// the caller doesn't need the result, and EnQueueReturn (blocking) when it does.
type TaskQueue struct {
	tasks chan Task
}

func wrapValueOnlyTask(task valueOnlyTask) task {
	return func() (any, error) {
		return task(), nil
	}
}

func wrapErrorOnlyTask(task errorOnlyTask) task {
	return func() (any, error) {
		return nil, task()
	}
}

func (taskQueue *TaskQueue) runWorker() {
	for task := range taskQueue.tasks {
		channel := task.channel
		result, err := task.task()

		if channel != nil {
			channel <- TaskResponse{val: result, err: err}
		}
	}
}

func (taskQueue *TaskQueue) EnQueue(task task) {
	taskQueue.tasks <- Task{task: task, channel: nil}
}

func (taskQueue *TaskQueue) EnQueueReturn(task task) (any, error) {
	channel := make(chan TaskResponse, 1)
	taskQueue.tasks <- Task{task: task, channel: channel}
	taskResponse := <-channel
	return taskResponse.val, taskResponse.err
}

func (taskQueue *TaskQueue) EnQueueValueOnlyTask(task valueOnlyTask) {
	taskQueue.tasks <- Task{task: wrapValueOnlyTask(task), channel: nil}
}

func (taskQueue *TaskQueue) EnQueueReturnValueOnlyTask(task valueOnlyTask) any {
	channel := make(chan TaskResponse, 1)
	taskQueue.tasks <- Task{task: wrapValueOnlyTask(task), channel: channel}
	taskResponse := <-channel
	return taskResponse.val
}

func (taskQueue *TaskQueue) EnQueueErrorOnlyTask(task errorOnlyTask) {
	taskQueue.tasks <- Task{task: wrapErrorOnlyTask(task), channel: nil}
}

func (taskQueue *TaskQueue) EnQueueReturnErrorOnlyTask(task errorOnlyTask) error {
	channel := make(chan TaskResponse, 1)
	taskQueue.tasks <- Task{task: wrapErrorOnlyTask(task), channel: channel}
	taskResponse := <-channel
	return taskResponse.err
}

// Drain blocks until all previously enqueued tasks have been processed.
// Call this during shutdown after the HTTP server has stopped accepting requests.
func (taskQueue *TaskQueue) Drain() {
	taskQueue.EnQueueReturn(func() (any, error) { return nil, nil })
}

var DBTaskQueue *TaskQueue

const numdbTaskQueueWorkers = 1

func init() {
	DBTaskQueue = &TaskQueue{tasks: make(chan Task, 10)}
	for range numdbTaskQueueWorkers {
		go DBTaskQueue.runWorker()
	}
}

func InitDatabase(driverName string, dataSourceName string, reset bool) {
	if reset {
		slog.Info("--reset-db flag set: dropping and recreating database")
		os.Remove("./chess_site.db")
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if reset {
		dropStmt := `
			DROP TABLE IF EXISTS sessions;
			DROP TABLE IF EXISTS live_matches;
			DROP TABLE IF EXISTS past_matches;
			DROP TABLE IF EXISTS users;
			DROP TABLE IF EXISTS user_ratings;
		`
		_, err = db.Exec(dropStmt)
		if err != nil {
			slog.Error("reset: drop tables", "err", err)
			os.Exit(1)
		}
	}

	schemaPath := filepath.Join("internal", "models", "schema.sql")
	c, ioErr := os.ReadFile(schemaPath)
	if ioErr != nil {
		slog.Error("failed to read schema file", "err", ioErr)
		os.Exit(1)
	}
	sqlStmt := string(c)

	_, err = db.Exec(sqlStmt)
	if err != nil {
		slog.Error("failed to execute schema", "err", err)
		os.Exit(1)
	}

	var sqliteVersion string
	row := db.QueryRow("SELECT sqlite_version();")
	err = row.Scan(&sqliteVersion)

	if err != nil {
		slog.Error("failed to query sqlite version", "err", err)
		os.Exit(1)
	}

	slog.Info("database initialized", "sqliteVersion", sqliteVersion)
}
