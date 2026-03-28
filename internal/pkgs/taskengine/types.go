package taskengine

import (
	"context"
	"sync"
	"time"

	"github.com/xxcheng123/cloudpan189-share/internal/pkgs/ptr"
)

// TaskInfo 任务信息
type TaskInfo struct {
	Context   context.Context   `json:"-"`
	Payload   []byte            `json:"payload"`
	ID        string            `json:"id"`        // 任务唯一ID
	Topic     Topic             `json:"topic"`     // 消息主题
	WorkerId  string            `json:"workerId"`  // 处理的Worker ID
	ReceiveAt time.Time         `json:"receiveAt"` // 接收时间
	StartAt   *time.Time        `json:"startAt"`   // 开始时间
	EndAt     *time.Time        `json:"endAt"`     // 结束时间
	Status    string            `json:"status"`    // 状态
	Results   []ProcessorResult `json:"results"`   // 处理器结果
	mu        sync.RWMutex      // 保护状态修改
}

// SetStatus 线程安全地设置状态
func (t *TaskInfo) SetStatus(status string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Status = status

	if status == TaskStatusCompleted || status == TaskStatusFailed || status == TaskStatusCancelled {
		t.EndAt = ptr.Of(time.Now())
	}
}

// GetStatus 线程安全地获取状态
func (t *TaskInfo) GetStatus() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.Status
}

// AddResult 添加处理器结果
func (t *TaskInfo) AddResult(result ProcessorResult) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.Results = append(t.Results, result)
}

// ProcessorResult 处理器执行结果
type ProcessorResult struct {
	ProcessorID string        `json:"processorId"`
	Status      string        `json:"status"`
	Error       string        `json:"error,omitempty"`
	StartTime   time.Time     `json:"startTime"`
	EndTime     time.Time     `json:"endTime"`
	Duration    time.Duration `json:"duration"`
}

type TaskContext struct {
	ctx      context.Context
	cancel   context.CancelFunc
	taskId   string
	taskInfo *TaskInfo
}

// Context 获取上下文
func (tc *TaskContext) Context() context.Context {
	return tc.ctx
}

// Cancel 取消任务
func (tc *TaskContext) Cancel() {
	if tc.cancel != nil {
		tc.cancel()
	}
}

// TaskID 获取任务ID
func (tc *TaskContext) TaskID() string {
	return tc.taskId
}

// TaskInfo 获取任务信息
func (tc *TaskContext) TaskInfo() *TaskInfo {
	return tc.taskInfo
}

// 任务状态常量
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)

type MessageProcessor interface {
	Process(ctx context.Context, message []byte) error
	// ProcessorID 返回处理器唯一标识
	ProcessorID() string
}

type Topic string

func (t Topic) String() string {
	return string(t)
}

// TaskStats 任务统计信息
type TaskStats struct {
	TotalTasks     int64 `json:"totalTasks"`
	CompletedTasks int64 `json:"completedTasks"`
	FailedTasks    int64 `json:"failedTasks"`
	RunningTasks   int64 `json:"runningTasks"`
	PendingTasks   int64 `json:"pendingTasks"`
	mu             *sync.RWMutex
}

// IncrementTotal 增加总任务数
func (s *TaskStats) IncrementTotal() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalTasks++
}

// IncrementCompleted 增加完成任务数
func (s *TaskStats) IncrementCompleted() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CompletedTasks++
}

// IncrementFailed 增加失败任务数
func (s *TaskStats) IncrementFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.FailedTasks++
}

// IncrementRunning 增加运行中任务数
func (s *TaskStats) IncrementRunning() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.RunningTasks++
}

// DecrementRunning 减少运行中任务数
func (s *TaskStats) DecrementRunning() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.RunningTasks--
}

func (s *TaskStats) IncrementPending() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.PendingTasks++
}

func (s *TaskStats) DecrementPending() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.PendingTasks--
}

// GetStats 获取统计信息副本
func (s *TaskStats) GetStats() TaskStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return TaskStats{
		TotalTasks:     s.TotalTasks,
		CompletedTasks: s.CompletedTasks,
		FailedTasks:    s.FailedTasks,
		RunningTasks:   s.RunningTasks,
		PendingTasks:   s.PendingTasks,
	}
}
