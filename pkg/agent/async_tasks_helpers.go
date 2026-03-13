package agent

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (m *SquadManager) ensureAsyncTaskForSharedTask(task *SharedTask, sessionID, squadName string) *AsyncTask {
	if task == nil {
		return nil
	}

	m.taskMu.RLock()
	existing := m.asyncTasks[task.ID]
	m.taskMu.RUnlock()
	if existing != nil {
		updated := false
		taskCopy := m.updateAsyncTask(task.ID, func(current *AsyncTask) {
			if strings.TrimSpace(sessionID) != "" && strings.TrimSpace(current.SessionID) == "" {
				current.SessionID = strings.TrimSpace(sessionID)
				updated = true
			}
			if strings.TrimSpace(squadName) != "" && strings.TrimSpace(current.SquadName) == "" {
				current.SquadName = strings.TrimSpace(squadName)
				updated = true
			}
		})
		if updated && strings.TrimSpace(taskCopy.SessionID) != "" {
			m.indexTaskSession(taskCopy.SessionID, taskCopy.ID)
		}
		return taskCopy
	}

	if strings.TrimSpace(squadName) == "" {
		if squad, err := m.store.GetTeam(task.SquadID); err == nil {
			squadName = squad.Name
		}
	}

	asyncTask := &AsyncTask{
		ID:          task.ID,
		SessionID:   strings.TrimSpace(sessionID),
		Kind:        AsyncTaskKindSquad,
		Status:      asyncStatusFromSharedTask(task.Status),
		SquadID:     task.SquadID,
		SquadName:   strings.TrimSpace(squadName),
		CaptainName: task.CaptainName,
		AgentNames:  append([]string(nil), task.AgentNames...),
		Prompt:      task.Prompt,
		AckMessage:  task.AckMessage,
		ResultText:  task.ResultText,
		CreatedAt:   task.CreatedAt,
		StartedAt:   cloneTimePtr(task.StartedAt),
		FinishedAt:  cloneTimePtr(task.FinishedAt),
	}
	m.upsertAsyncTask(asyncTask)
	return asyncTask
}

func (m *SquadManager) upsertAsyncTask(task *AsyncTask) {
	if task == nil {
		return
	}

	m.taskMu.Lock()
	m.asyncTasks[task.ID] = cloneAsyncTask(task)
	m.taskMu.Unlock()
	if strings.TrimSpace(task.SessionID) != "" {
		m.indexTaskSession(task.SessionID, task.ID)
	}
}

func (m *SquadManager) updateAsyncTask(taskID string, mutate func(*AsyncTask)) *AsyncTask {
	m.taskMu.Lock()
	defer m.taskMu.Unlock()

	task := m.asyncTasks[taskID]
	if task == nil {
		task = &AsyncTask{ID: taskID, CreatedAt: time.Now()}
		m.asyncTasks[taskID] = task
	}
	mutate(task)
	if strings.TrimSpace(task.SessionID) != "" {
		m.indexTaskSessionLocked(task.SessionID, task.ID)
	}
	return cloneAsyncTask(task)
}

func (m *SquadManager) emitTaskEvent(taskID string, evt *TaskEvent, terminal bool) {
	if evt == nil {
		return
	}
	evt.ID = uuid.NewString()

	m.taskMu.Lock()
	task := m.asyncTasks[taskID]
	if task != nil {
		evt.TaskID = taskID
		if evt.SessionID == "" {
			evt.SessionID = task.SessionID
		}
		if evt.Kind == "" {
			evt.Kind = task.Kind
		}
		if evt.Status == "" {
			evt.Status = task.Status
		}
		if evt.SquadID == "" {
			evt.SquadID = task.SquadID
		}
		if evt.SquadName == "" {
			evt.SquadName = task.SquadName
		}
		if evt.CaptainName == "" {
			evt.CaptainName = task.CaptainName
		}
		task.Events = appendTaskEvent(task.Events, cloneTaskEvent(evt))
	}
	subs := collectTaskSubscribersLocked(m.taskSubs[taskID])
	if terminal {
		delete(m.taskSubs, taskID)
		delete(m.taskCancels, taskID)
	}
	m.taskMu.Unlock()

	sendTaskEventToSubscribers(subs, evt, terminal)
}

func (m *SquadManager) setTaskCancel(taskID string, cancel context.CancelFunc) {
	m.taskMu.Lock()
	defer m.taskMu.Unlock()
	m.taskCancels[taskID] = cancel
}

func (m *SquadManager) clearTaskCancel(taskID string) {
	m.taskMu.Lock()
	defer m.taskMu.Unlock()
	delete(m.taskCancels, taskID)
}

func (m *SquadManager) indexTaskSession(sessionID, taskID string) {
	m.taskMu.Lock()
	defer m.taskMu.Unlock()
	m.indexTaskSessionLocked(sessionID, taskID)
}

func (m *SquadManager) indexTaskSessionLocked(sessionID, taskID string) {
	for _, existing := range m.sessionTasks[sessionID] {
		if existing == taskID {
			return
		}
	}
	m.sessionTasks[sessionID] = append(m.sessionTasks[sessionID], taskID)
}

func asyncStatusFromSharedTask(status SharedTaskStatus) AsyncTaskStatus {
	switch status {
	case SharedTaskStatusRunning:
		return AsyncTaskStatusRunning
	case SharedTaskStatusCompleted:
		return AsyncTaskStatusCompleted
	case SharedTaskStatusFailed:
		return AsyncTaskStatusFailed
	default:
		return AsyncTaskStatusQueued
	}
}

func appendTaskEvent(events []*TaskEvent, evt *TaskEvent) []*TaskEvent {
	const maxTaskEvents = 200
	events = append(events, evt)
	if len(events) > maxTaskEvents {
		events = append([]*TaskEvent(nil), events[len(events)-maxTaskEvents:]...)
	}
	return events
}

func collectTaskSubscribersLocked(subs map[chan *TaskEvent]struct{}) []chan *TaskEvent {
	if len(subs) == 0 {
		return nil
	}
	out := make([]chan *TaskEvent, 0, len(subs))
	for ch := range subs {
		out = append(out, ch)
	}
	return out
}

func sendTaskEventToSubscribers(subs []chan *TaskEvent, evt *TaskEvent, terminal bool) {
	for _, ch := range subs {
		cloned := cloneTaskEvent(evt)
		select {
		case ch <- cloned:
		case <-time.After(250 * time.Millisecond):
		}
		if terminal {
			close(ch)
		}
	}
}

func cloneAsyncTask(task *AsyncTask) *AsyncTask {
	if task == nil {
		return nil
	}
	cloned := *task
	cloned.AgentNames = append([]string(nil), task.AgentNames...)
	cloned.StartedAt = cloneTimePtr(task.StartedAt)
	cloned.FinishedAt = cloneTimePtr(task.FinishedAt)
	cloned.Events = cloneTaskEvents(task.Events)
	return &cloned
}

func cloneTaskEvents(events []*TaskEvent) []*TaskEvent {
	if len(events) == 0 {
		return nil
	}
	out := make([]*TaskEvent, 0, len(events))
	for _, evt := range events {
		out = append(out, cloneTaskEvent(evt))
	}
	return out
}

func cloneTaskEvent(evt *TaskEvent) *TaskEvent {
	if evt == nil {
		return nil
	}
	cloned := *evt
	cloned.Runtime = cloneAgentEvent(evt.Runtime)
	return &cloned
}

func cloneAgentEvent(evt *Event) *Event {
	if evt == nil {
		return nil
	}
	cloned := *evt
	if evt.ToolArgs != nil {
		cloned.ToolArgs = make(map[string]interface{}, len(evt.ToolArgs))
		for key, value := range evt.ToolArgs {
			cloned.ToolArgs[key] = value
		}
	}
	if evt.StateDelta != nil {
		cloned.StateDelta = make(map[string]interface{}, len(evt.StateDelta))
		for key, value := range evt.StateDelta {
			cloned.StateDelta[key] = value
		}
	}
	return &cloned
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func isTerminalAsyncTaskStatus(status AsyncTaskStatus) bool {
	switch status {
	case AsyncTaskStatusCompleted, AsyncTaskStatusFailed, AsyncTaskStatusCancelled:
		return true
	default:
		return false
	}
}

func slicesSortAsyncTasks(tasks []*AsyncTask) {
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[i].CreatedAt.After(tasks[j].CreatedAt) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
