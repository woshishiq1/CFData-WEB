package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("WebSocket 升级失败:", err)
        return
    }

    session := &appSession{ws: ws}
    defer func() {
        session.cancelTaskSilently()
        ws.Close()
    }()

    session.sendWSMessage("init_config", map[string]interface{}{
        "speedTestURL":     speedTestURL,
        "speedTestWorkers": speedTestWorkers,
    })

    handlers := map[string]func(json.RawMessage){
        "start_task": func(data json.RawMessage) {
            var params startTaskRequest
            if err := json.Unmarshal(data, &params); err != nil {
                session.sendWSMessage("error", "start_task 参数解析失败")
                return
            }
            if params.Threads <= 0 {
                params.Threads = 100
            }
            session.startTask(func(ctx context.Context, session *appSession) {
                runOfficialTask(ctx, session, params.IPType, params.Threads)
            })
        },
        "start_test": func(data json.RawMessage) {
            var params startTestRequest
            if err := json.Unmarshal(data, &params); err != nil {
                session.sendWSMessage("error", "start_test 参数解析失败")
                return
            }
            if params.Delay < 0 {
                params.Delay = 0
            }
            session.startTask(func(ctx context.Context, session *appSession) {
                runDetailedTest(ctx, session, params.DC, params.Port, params.Delay)
            })
        },
        "start_speed_test": func(data json.RawMessage) {
            var params startSpeedTestRequest
            if err := json.Unmarshal(data, &params); err != nil {
                session.sendWSMessage("error", "start_speed_test 参数解析失败")
                return
            }
            session.startTask(func(ctx context.Context, session *appSession) {
                runSpeedTest(ctx, session, params.IP, params.Port, params.URL)
            })
        },
        "start_nsb_task": func(data json.RawMessage) {
            var params startNSBTaskRequest
            if err := json.Unmarshal(data, &params); err != nil {
                session.sendWSMessage("error", "start_nsb_task 参数解析失败")
                return
            }
            if params.MaxThreads <= 0 {
                params.MaxThreads = speedTestWorkers
            }
            if params.SpeedTest < 0 {
                params.SpeedTest = 0
            }
            if params.Delay < 0 {
                params.Delay = 0
            }
            if strings.TrimSpace(params.SpeedURL) == "" {
                params.SpeedURL = speedTestURL
            }
            if strings.TrimSpace(params.OutFile) == "" {
                params.OutFile = "ip.csv"
            }
            if strings.TrimSpace(params.FileContent) == "" {
                session.sendWSMessage("error", "上传文件为空")
                return
            }
            session.startTask(func(ctx context.Context, session *appSession) {
                runNSBTask(ctx, session, params.FileName, params.FileContent, params.OutFile, params.MaxThreads, params.SpeedTest, params.SpeedURL, params.EnableTLS, params.Delay)
            })
        },
        "stop_task": func(data json.RawMessage) {
            session.stopTask()
        },
        "reset_all_config": func(data json.RawMessage) {
            resetAllConfigFiles(session)
        },
    }

    for {
        _, msg, err := ws.ReadMessage()
        if err != nil {
            break
        }

        var request wsRequest
        if err := json.Unmarshal(msg, &request); err != nil {
            session.sendWSMessage("error", "请求格式错误")
            continue
        }

        handler, ok := handlers[request.Type]
        if !ok {
            session.sendWSMessage("error", "未知请求类型")
            continue
        }
        handler(request.Data)
    }
}
