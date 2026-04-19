package main

import (
    "context"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "strconv"
    "strings"
    "time"
)

func runSpeedTest(ctx context.Context, session *appSession, ip string, port int, customURL string) {
    defer session.endTask()

    session.sendWSMessage("log", fmt.Sprintf("开始对 IP %s 端口 %d 进行测速...", ip, port))

    scheme := "http"
    if port == 443 || port == 2053 || port == 2083 || port == 2087 || port == 2096 || port == 8443 {
        scheme = "https"
    }

    testURL := speedTestURL
    if customURL != "" {
        testURL = customURL
    }
    if !strings.HasPrefix(testURL, "http://") && !strings.HasPrefix(testURL, "https://") {
        testURL = scheme + "://" + testURL
    }

    parsedURL, err := url.Parse(testURL)
    if err != nil {
        session.sendWSMessage("speed_test_result", map[string]string{"ip": ip, "speed": "URL解析错误"})
        return
    }

    client := http.Client{
        Transport: &http.Transport{
            Dial: func(network, addr string) (net.Conn, error) {
                return net.Dial("tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
            },
            TLSHandshakeTimeout: 10 * time.Second,
        },
        Timeout: 15 * time.Second,
    }

    fullURL := fmt.Sprintf("%s://%s%s", scheme, parsedURL.Hostname(), parsedURL.RequestURI())
    req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
    if err != nil {
        session.sendWSMessage("speed_test_result", map[string]string{"ip": ip, "speed": "请求构造错误"})
        session.sendWSMessage("log", "测速失败: "+err.Error())
        return
    }
    req.Header.Set("User-Agent", "Mozilla/5.0")

    start := time.Now()
    resp, err := client.Do(req)
    if err != nil {
        session.sendWSMessage("speed_test_result", map[string]string{"ip": ip, "speed": "连接错误"})
        session.sendWSMessage("log", "测速失败: "+err.Error())
        return
    }
    defer resp.Body.Close()

    buf := make([]byte, 32*1024)
    var totalBytes int64
    var maxSpeed float64

    timeout := time.After(5 * time.Second)
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    lastBytes := int64(0)
    lastTime := start
    done := false
    for !done {
        select {
        case <-ctx.Done():
            session.sendWSMessage("log", "测速任务已终止")
            return
        case <-timeout:
            done = true
        case now := <-ticker.C:
            duration := now.Sub(lastTime).Seconds()
            if duration > 0 {
                diff := totalBytes - lastBytes
                currentSpeed := float64(diff) / duration / 1024 / 1024
                if currentSpeed > maxSpeed {
                    maxSpeed = currentSpeed
                }
            }
            lastBytes = totalBytes
            lastTime = now
        default:
            n, err := resp.Body.Read(buf)
            if n > 0 {
                totalBytes += int64(n)
            }
            if err != nil {
                if err == io.EOF {
                    done = true
                    continue
                }
                done = true
            }
        }
    }

    if totalBytes <= 0 || maxSpeed <= 0 {
        session.sendWSMessage("speed_test_result", map[string]string{"ip": ip, "speed": "0 kB/s"})
        return
    }

    duration := time.Since(start).Seconds()
    if duration == 0 {
        duration = 1
    }
    realSpeed := float64(totalBytes) / duration / 1024

    speedStr := fmt.Sprintf("%.2f MB/s", realSpeed/1024)
    if maxSpeed > realSpeed/1024 {
        speedStr = fmt.Sprintf("%.2f MB/s", maxSpeed)
    }

    session.sendWSMessage("speed_test_result", map[string]string{"ip": ip, "speed": speedStr})
    session.sendWSMessage("log", fmt.Sprintf("IP %s 测速完成: %s", ip, speedStr))
}
