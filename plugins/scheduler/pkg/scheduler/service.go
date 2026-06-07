package scheduler

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/issueye/goscript/sdk/gtp"
)

const ModuleName = "@plugin/scheduler"

type Service struct {
	mu     sync.Mutex
	nextID int64
	tasks  map[string]*Task
	events chan gtp.Frame
}

type Task struct {
	ID         string
	Name       string
	Payload    gtp.Value
	DelayMS    int64
	IntervalMS int64
	Repeat     int
	Fired      int
	NextRun    time.Time
	CreatedAt  time.Time
	Cancelled  bool
	timer      *time.Timer
}

func NewService() *Service {
	return &Service{
		tasks:  make(map[string]*Task),
		events: make(chan gtp.Frame, 128),
	}
}

func (s *Service) Events() <-chan gtp.Frame {
	return s.events
}

func (s *Service) Handle(frame gtp.Frame) gtp.Frame {
	if frame.Type != "call" {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("scheduler expects call frames"))
	}
	if frame.Module != "" && frame.Module != ModuleName {
		return gtp.ErrorResult(frame.ID, gtp.NotFoundError("unknown module %s", frame.Module))
	}
	switch frame.Method {
	case "schedule":
		return s.handleSchedule(frame)
	case "cancel":
		return s.handleCancel(frame)
	case "list":
		return s.handleList(frame)
	case "clear":
		return s.handleClear(frame)
	default:
		return gtp.ErrorResult(frame.ID, gtp.NotFoundError("unknown scheduler method %s", frame.Method))
	}
}

func (s *Service) Run(in io.Reader, out io.Writer) error {
	decoder := gtp.NewDecoder(in)
	encoder := gtp.NewEncoder(out)
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for event := range s.events {
			_ = encoder.Encode(event)
		}
	}()
	defer func() {
		s.Clear()
		close(s.events)
		<-writerDone
	}()

	for {
		frame, err := decoder.Decode()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if frame.Type == "hello" {
			if err := encoder.Encode(ReadyFrame(frame.ID)); err != nil {
				return err
			}
			continue
		}
		if err := encoder.Encode(s.Handle(frame)); err != nil {
			return err
		}
	}
}

func ReadyFrame(id string) gtp.Frame {
	return gtp.Frame{
		Version:      gtp.Version,
		ID:           id,
		Type:         "ready",
		Service:      "scheduler",
		Capabilities: []string{"call", "event"},
		Modules: map[string][]string{
			ModuleName: {"schedule", "cancel", "list", "clear"},
		},
	}
}

func (s *Service) handleSchedule(frame gtp.Frame) gtp.Frame {
	opts, errObj := gtp.RequiredObjectArg(frame.Args, 0, "options")
	if errObj != nil {
		return gtp.ErrorResult(frame.ID, errObj)
	}
	delayMS, ok := gtp.NumberField(opts, "delayMs")
	if !ok || delayMS < 0 {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("options.delayMs must be a non-negative number"))
	}
	intervalMS, _ := gtp.NumberField(opts, "intervalMs")
	if intervalMS < 0 {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("options.intervalMs must be a non-negative number"))
	}
	repeat := 1
	if repeatValue, ok := gtp.NumberField(opts, "repeat"); ok {
		repeat = int(repeatValue)
	}
	if intervalMS > 0 && repeat == 1 {
		repeat = -1
	}
	name, _ := gtp.StringField(opts, "name")
	payload, ok := gtp.Field(opts, "payload")
	if !ok {
		payload = gtp.Null()
	}

	task := s.createTask(name, payload, int64(delayMS), int64(intervalMS), repeat)
	return gtp.OKResult(frame.ID, taskValue(task))
}

func (s *Service) handleCancel(frame gtp.Frame) gtp.Frame {
	if len(frame.Args) < 1 {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("task id is required"))
	}
	id, ok := frame.Args[0].StringValue()
	if !ok {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("task id must be a string"))
	}
	cancelled := s.Cancel(id)
	return gtp.OKResult(frame.ID, gtp.Object(map[string]gtp.Value{
		"id":        gtp.String(id),
		"cancelled": gtp.Bool(cancelled),
	}))
}

func (s *Service) handleList(frame gtp.Frame) gtp.Frame {
	return gtp.OKResult(frame.ID, gtp.Array(s.ListValues()))
}

func (s *Service) handleClear(frame gtp.Frame) gtp.Frame {
	count := s.Clear()
	return gtp.OKResult(frame.ID, gtp.Object(map[string]gtp.Value{"count": gtp.Number(float64(count))}))
}

func (s *Service) createTask(name string, payload gtp.Value, delayMS, intervalMS int64, repeat int) *Task {
	s.mu.Lock()
	s.nextID++
	id := fmt.Sprintf("task-%d", s.nextID)
	task := &Task{
		ID:         id,
		Name:       name,
		Payload:    payload,
		DelayMS:    delayMS,
		IntervalMS: intervalMS,
		Repeat:     repeat,
		CreatedAt:  time.Now(),
	}
	task.NextRun = task.CreatedAt.Add(time.Duration(delayMS) * time.Millisecond)
	s.tasks[id] = task
	s.mu.Unlock()

	task.timer = time.AfterFunc(time.Duration(delayMS)*time.Millisecond, func() {
		s.fire(id)
	})
	return task
}

func (s *Service) fire(id string) {
	var event gtp.Frame
	var rescheduleAfter time.Duration
	s.mu.Lock()
	task, ok := s.tasks[id]
	if !ok || task.Cancelled {
		s.mu.Unlock()
		return
	}
	task.Fired++
	event = gtp.Frame{
		Version: gtp.Version,
		ID:      fmt.Sprintf("evt-%s-%d", task.ID, task.Fired),
		Type:    "event",
		Module:  ModuleName,
		Event:   "trigger",
		Data:    ptr(taskValue(task)),
	}
	shouldRepeat := task.IntervalMS > 0 && (task.Repeat < 0 || task.Fired < task.Repeat)
	if shouldRepeat {
		rescheduleAfter = time.Duration(task.IntervalMS) * time.Millisecond
		task.NextRun = time.Now().Add(rescheduleAfter)
	} else {
		delete(s.tasks, id)
	}
	s.mu.Unlock()

	select {
	case s.events <- event:
	default:
	}

	if rescheduleAfter > 0 {
		s.mu.Lock()
		if task, ok := s.tasks[id]; ok && !task.Cancelled {
			task.timer = time.AfterFunc(rescheduleAfter, func() { s.fire(id) })
		}
		s.mu.Unlock()
	}
}

func (s *Service) Cancel(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return false
	}
	task.Cancelled = true
	if task.timer != nil {
		task.timer.Stop()
	}
	delete(s.tasks, id)
	return true
}

func (s *Service) Clear() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := len(s.tasks)
	for _, task := range s.tasks {
		task.Cancelled = true
		if task.timer != nil {
			task.timer.Stop()
		}
	}
	s.tasks = make(map[string]*Task)
	return count
}

func (s *Service) ListValues() []gtp.Value {
	s.mu.Lock()
	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	s.mu.Unlock()

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})
	values := make([]gtp.Value, len(tasks))
	for i, task := range tasks {
		values[i] = taskValue(task)
	}
	return values
}

func taskValue(task *Task) gtp.Value {
	return gtp.Object(map[string]gtp.Value{
		"id":         gtp.String(task.ID),
		"name":       gtp.String(task.Name),
		"payload":    task.Payload,
		"delayMs":    gtp.Number(float64(task.DelayMS)),
		"intervalMs": gtp.Number(float64(task.IntervalMS)),
		"repeat":     gtp.Number(float64(task.Repeat)),
		"fired":      gtp.Number(float64(task.Fired)),
		"nextRun":    gtp.String(task.NextRun.Format(time.RFC3339Nano)),
		"createdAt":  gtp.String(task.CreatedAt.Format(time.RFC3339Nano)),
	})
}

func ptr[T any](v T) *T { return &v }
