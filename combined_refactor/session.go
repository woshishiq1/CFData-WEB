package main

import (
    "context"
)

func (s *appSession) sendWSMessage(msgType string, data interface{}) {
    s.wsMutex.Lock()
    defer s.wsMutex.Unlock()
    if s.wsClosed {
        return
    }
    msg := map[string]interface{}{
        "type": msgType,
        "data": data,
    }
    if err := s.ws.WriteJSON(msg); err != nil {
        s.wsClosed = true
        sendLog("WebSocket 发送失败: " + err.Error())
    }
}

func (s *appSession) startTask(run taskStarter) {
    ctx, cancel := context.WithCancel(context.Background())
    started := s.beginTask(cancel)
    if !started {
        cancel()
        s.sendWSMessage("error", "已有任务正在运行，请等待完成后再试")
        return
    }
    go run(ctx, s)
}

func (s *appSession) stopTask() {
    s.cancelTask(true)
}

func (s *appSession) cancelTaskSilently() {
    s.cancelTask(false)
}

func (s *appSession) cancelTask(withLog bool) {
    s.taskMutex.Lock()
    cancel := s.taskCancel
    s.taskMutex.Unlock()
    if cancel != nil {
        cancel()
        if withLog {
            s.sendWSMessage("log", "已发送强制终止信号，正在清理当前任务...")
        }
    }
}

func (s *appSession) beginTask(cancel context.CancelFunc) bool {
    s.taskMutex.Lock()
    defer s.taskMutex.Unlock()
    if s.isTaskRunning {
        return false
    }
    s.isTaskRunning = true
    s.taskCancel = cancel
    processTaskMutex.Lock()
    activeTaskCount++
    processTaskMutex.Unlock()
    return true
}

func (s *appSession) endTask() {
    s.taskMutex.Lock()
    wasRunning := s.isTaskRunning
    defer s.taskMutex.Unlock()
    s.isTaskRunning = false
    s.taskCancel = nil
    if wasRunning {
        processTaskMutex.Lock()
        if activeTaskCount > 0 {
            activeTaskCount--
        }
        processTaskMutex.Unlock()
    }
}
