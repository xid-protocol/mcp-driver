package ssepool

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"
)

type Event struct {
	Data []byte
}

type Connection struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	w      http.ResponseWriter
	f      http.Flusher
}

type thread struct {
	ch   chan Event                 // 稳定队列：thread 级别
	conn atomic.Pointer[Connection] // 当前活跃连接（可被替换）
	// lastActive 记录该 thread 最近一次“活跃”的 Unix 秒级时间戳。
	// 活跃包括：Attach 成功、Post 入队成功、writer 成功写出。
	// 若不在这些位置更新 lastActive，StartCleanup 的 idle 判定会不准确。
	lastActive atomic.Int64
}

// Attach: 将新的 HTTP 连接附着到同一 thread。
// 若已有连接，旧连接会被取消；消息队列不变，新的连接立即接管发送。
func (p *Pool) Attach(threadID string, w http.ResponseWriter) (*Connection, error) {
	f, ok := (w).(http.Flusher)
	if !ok {
		return nil, errors.New("response does not support streaming")
	}
	setSSEHeaders(w)
	f.Flush()

	t := p.getOrCreateThread(threadID)

	ctx, cancel := context.WithCancel(context.Background())
	c := &Connection{Ctx: ctx, Cancel: cancel, w: w, f: f}

	// 替换旧连接
	if old := t.conn.Swap(c); old != nil {
		old.Cancel()
	}

	// 首次创建该 thread 时启动一个单写循环
	if tWasNew := p.startWriterOnce(threadID, t); tWasNew {
		// no-op
	}
	return c, nil
}

// Post: 往指定 thread 入队事件；队列满则丢最老，保证新消息可进入。
func (p *Pool) Post(threadID string, data []byte) {
	t := p.getOrCreateThread(threadID)
	ev := Event{Data: data}
	select {
	case t.ch <- ev:
	default:
		select {
		case <-t.ch: // 丢最老
		default:
		}
		t.ch <- ev
	}
}

func (p *Pool) getOrCreateThread(threadID string) *thread {
	if v, ok := p.threads.Load(threadID); ok {
		return v.(*thread)
	}
	t := &thread{ch: make(chan Event, p.bufSize)}
	if actual, loaded := p.threads.LoadOrStore(threadID, t); loaded {
		return actual.(*thread)
	}
	return t
}

// 确保每个 thread 只启动一次写循环
func (p *Pool) startWriterOnce(threadID string, t *thread) bool {
	type started struct{}
	key := threadID + ":started"
	if _, loaded := p.threads.LoadOrStore(key, started{}); loaded {
		return false
	}
	go p.writerLoop(t)
	return true
}

// 单写循环：从 thread 队列读事件，写到“当前连接”。
// 若无连接，轻量等待；写失败则清空当前连接，等待下一次 Attach。
func (p *Pool) writerLoop(t *thread) {
	for ev := range t.ch {
		for {
			c := t.conn.Load()
			if c == nil {
				time.Sleep(50 * time.Millisecond) // 等待连接附着
				continue
			}
			if err := writeSSE(c.w, c.f, ev); err != nil {
				c.Cancel()
				t.conn.CompareAndSwap(c, nil) // 清空失效连接，等待新连接
				continue
			}
			break
		}
	}
}

func (p *Pool) CloseThread(threadID string) {
	if v, ok := p.threads.LoadAndDelete(threadID); ok {
		t := v.(*thread)
		if c := t.conn.Load(); c != nil {
			c.Cancel()
		}
		// 关闭队列：让 writerLoop 的 `for ev := range t.ch` 自然退出
		close(t.ch)
		// 删除该 thread 的“已启动”标记，便于后续重建时能再次启动 writerLoop
		p.threads.Delete(threadID + ":started")
	}
}

func setSSEHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
}

func writeSSE(w http.ResponseWriter, f http.Flusher, ev Event) error {
	if _, err := w.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := w.Write(ev.Data); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n\n")); err != nil {
		return err
	}
	f.Flush()
	return nil
}

// shouldCleanThread 判断某个 thread 是否“满足清理条件”。
//
// 满足以下全部条件时，会被清理：
//  1. 无活跃连接：当前没有连接附着在该 thread 上（t.conn.Load() == nil）。
//  2. 队列为空：该 thread 的待发送事件队列为空（len(t.ch) == 0）。
//  3. 空闲超时：距离最近一次活跃（Attach/成功入队/成功写出）已超过给定 TTL
//     （nowUnix - t.lastActive.Load() > int64(ttl.Seconds())）。
//
// 参数 nowUnix 应在同一轮扫描前用 time.Now().Unix() 统一获取，以保证判定一致性。
func shouldCleanThread(t *thread, ttl time.Duration) bool {
	now := time.Now().Unix()
	if t == nil {
		return false
	}
	// 条件 1：无活跃连接
	if t.conn.Load() != nil {
		return false
	}
	// 条件 2：队列为空（len 用于有缓冲通道是安全的）
	if len(t.ch) != 0 {
		return false
	}
	// 条件 3：空闲超时
	idleSeconds := now - t.lastActive.Load()
	return idleSeconds > int64(ttl.Seconds())
}

// StartCleanup 启动后台清理协程。
func (p *Pool) StartCleanup(interval time.Duration, ttl time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		p.threads.Range(func(key, value any) bool {
			k, _ := key.(string)
			t := value.(*thread)
			if shouldCleanThread(t, ttl) {
				p.CloseThread(k)
			}
			return true
		})
	}

}
