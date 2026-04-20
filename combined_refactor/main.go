package main

import (
    "flag"
    "fmt"
    "net/http"
)

func main() {
	flag.IntVar(&listenPort, "port", 13335, "服务监听端口")
	flag.StringVar(&speedTestURL, "url", "speed.cloudflare.com/__down?bytes=99999999", "测速下载地址（不含协议前缀）")
	flag.IntVar(&speedTestWorkers, "speedtest", 5, "默认测速并发")
	flag.BoolVar(&debugMode, "debug", false, "开启调试输出（导出失败明细 CSV）")
	flag.Parse()

    initLocations()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        data, err := staticFiles.ReadFile("index.html")
        if err != nil {
            http.Error(w, "无法加载页面", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        _, _ = w.Write(data)
    })
    http.HandleFunc("/ws", handleWebSocket)

    addr := fmt.Sprintf(":%d", listenPort)
	fmt.Printf("服务启动于 http://localhost:%d\n", listenPort)
	fmt.Printf("当前测速网址: %s\n", speedTestURL)
	fmt.Printf("调试模式: %v\n", debugMode)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("启动失败: %v\n", err)
	}
}
