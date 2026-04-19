package main

import (
    "context"
    "encoding/csv"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "sync"
    "sort"
    "time"
)

func scanNSBEntry(ctx context.Context, item string, enableTLS bool, delay int) *iptestResult {
    parts := strings.Fields(item)
    if len(parts) != 2 {
        return nil
    }
    ipAddr := parts[0]
    portStr := parts[1]
    port, err := strconv.Atoi(portStr)
    if err != nil {
        return nil
    }

    dialer := &net.Dialer{Timeout: timeout, KeepAlive: 0}
    start := time.Now()
    conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ipAddr, strconv.Itoa(port)))
    if err != nil {
        return nil
    }
    defer conn.Close()

    tcpDuration := time.Since(start)
    if delay > 0 && tcpDuration.Milliseconds() > int64(delay) {
        return nil
    }

    protocol := "http://"
    if enableTLS {
        protocol = "https://"
    }

    start = time.Now()
    client := http.Client{
        Transport: &http.Transport{
            Dial: func(network, addr string) (net.Conn, error) {
                return conn, nil
            },
        },
        Timeout: timeout,
    }
    req, err := http.NewRequestWithContext(ctx, "GET", protocol+requestURL, nil)
    if err != nil {
        return nil
    }
    req.Header.Set("User-Agent", "Mozilla/5.0")
    req.Close = true

    resp, err := client.Do(req)
    if err != nil {
        return nil
    }
    duration := time.Since(start)
    if duration > maxDuration {
        resp.Body.Close()
        return nil
    }

    bodyData, err := io.ReadAll(resp.Body)
    resp.Body.Close()
    if err != nil {
        return nil
    }

    trace := parseTraceResponse(string(bodyData))
    if _, ok := trace["uag"]; !ok || trace["uag"] != "Mozilla/5.0" {
        return nil
    }

    dataCenter := trace["colo"]
    locCode := trace["loc"]
    if dataCenter == "" {
        return nil
    }

    loc := locationMap[dataCenter]
    return &iptestResult{
        ipAddr:      ipAddr,
        port:        port,
        dataCenter:  dataCenter,
        locCode:     locCode,
        region:      loc.Region,
        city:        loc.City,
        latency:     fmt.Sprintf("%d ms", tcpDuration.Milliseconds()),
        tcpDuration: tcpDuration,
        outboundIP:  trace["ip"],
        ipType:      getIPType(trace["ip"]),
        visitScheme: trace["visit_scheme"],
        tlsVersion:  trace["tls"],
        sni:         trace["sni"],
        httpVersion: trace["http"],
        warp:        trace["warp"],
        gateway:     trace["gateway"],
        rbi:         trace["rbi"],
        kex:         trace["kex"],
        timestamp:   trace["ts"],
    }
}

func sortNSBResults(results []iptestResult, speedTest int) {
    if speedTest > 0 {
        sort.Slice(results, func(i, j int) bool {
            return results[i].downloadSpeed > results[j].downloadSpeed
        })
        return
    }

    sort.Slice(results, func(i, j int) bool {
        return results[i].tcpDuration < results[j].tcpDuration
    })
}

func runNSBDownloadSpeed(ctx context.Context, ip string, port int, enableTLS bool, testURL string) float64 {
    if strings.TrimSpace(testURL) == "" {
        testURL = speedTestURL
    }

    scheme := "http://"
    if enableTLS {
        scheme = "https://"
    }
    if !strings.HasPrefix(testURL, "http://") && !strings.HasPrefix(testURL, "https://") {
        testURL = scheme + testURL
    }

    parsedURL, err := url.Parse(testURL)
    if err != nil {
        return 0
    }

    client := http.Client{
        Transport: &http.Transport{
            DialContext: func(c context.Context, network, addr string) (net.Conn, error) {
                dialer := &net.Dialer{Timeout: 5 * time.Second}
                return dialer.DialContext(c, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
            },
            TLSHandshakeTimeout: 10 * time.Second,
        },
        Timeout: 15 * time.Second,
    }

    fullURL := fmt.Sprintf("%s%s%s", scheme, parsedURL.Host, parsedURL.RequestURI())

    req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
    if err != nil {
        return 0
    }
    req.Header.Set("User-Agent", "Mozilla/5.0")

    start := time.Now()
    resp, err := client.Do(req)
    if err != nil {
        return 0
    }
    defer resp.Body.Close()

    written, err := io.Copy(io.Discard, resp.Body)
    if err != nil {
        return 0
    }
    duration := time.Since(start)
    if duration <= 0 {
        return 0
    }

    return float64(written) / duration.Seconds() / 1024
}

func runNSBTask(ctx context.Context, session *appSession, fileName, fileContent, outFile string, maxThreads, speedTest int, speedURL string, enableTLS bool, delay int) {
    defer session.endTask()

    session.sendWSMessage("log", fmt.Sprintf("开始非标优选：%s", fileName))

    tmpFile, err := os.CreateTemp("", "cfdata-nsb-*.txt")
    if err != nil {
        session.sendWSMessage("error", "无法创建临时文件: "+err.Error())
        return
    }
    tmpPath := tmpFile.Name()
    defer os.Remove(tmpPath)

    if _, err := io.WriteString(tmpFile, fileContent); err != nil {
        tmpFile.Close()
        session.sendWSMessage("error", "写入临时文件失败: "+err.Error())
        return
    }
    if err := tmpFile.Close(); err != nil {
        session.sendWSMessage("error", "关闭临时文件失败: "+err.Error())
        return
    }

    ips, err := readIPs(tmpPath)
    if err != nil {
        session.sendWSMessage("error", "解析上传文件失败: "+err.Error())
        return
    }
    if len(ips) == 0 {
        session.sendWSMessage("error", "上传文件中未找到有效的 ip 端口行")
        return
    }

    session.sendWSMessage("log", fmt.Sprintf("共读取 %d 条 ip 端口，开始延迟检测", len(ips)))

    nsbResults := make([]iptestResult, 0, len(ips))
    resMutex := &sync.Mutex{}
    total := len(ips)
    wasCanceled := runBoundedWorkers(ctx, total, maxThreads, 1, func(current, total int) {
        reportNSBProgress(session, "scan", current, total, "延迟扫描")
    }, func(idx int) {
        item := ips[idx]
        select {
        case <-ctx.Done():
            return
        default:
        }

        res := scanNSBEntry(ctx, item, enableTLS, delay)
        if res == nil {
            return
        }

        resMutex.Lock()
        nsbResults = append(nsbResults, *res)
        resMutex.Unlock()
    })

    if wasCanceled || ctx.Err() != nil {
        session.sendWSMessage("log", "检测到停止命令，非标优选延迟扫描已强制终止")
        session.sendWSMessage("error", "任务已被手动终止，未生成最终结果")
        return
    }

    if len(nsbResults) == 0 {
        session.sendWSMessage("error", "未发现有效 IP")
        return
    }

    if speedTest > 0 {
        session.sendWSMessage("log", fmt.Sprintf("开始测速：%d 条记录，线程数=%d", len(nsbResults), speedTest))

        total := len(nsbResults)
        speedCanceled := runBoundedWorkers(ctx, total, speedTest, 1, func(current, total int) {
            reportNSBProgress(session, "speed", current, total, "速度测试")
        }, func(idx int) {
            select {
            case <-ctx.Done():
                return
            default:
            }

            res := &nsbResults[idx]
            res.downloadSpeed = runNSBDownloadSpeed(ctx, res.ipAddr, res.port, enableTLS, speedURL)
        })
        if speedCanceled {
            wasCanceled = true
        }
    }

    if wasCanceled || ctx.Err() != nil {
        session.sendWSMessage("log", "检测到停止命令，非标测速任务已强制终止")
        session.sendWSMessage("error", "测速任务已被手动终止，不再进行结果整理")
        return
    }

    sortNSBResults(nsbResults, speedTest)

    if err := writeNSBCSV(outFile, nsbResults, speedTest, enableTLS); err != nil {
        session.sendWSMessage("error", "导出 CSV 失败: "+err.Error())
        return
    }

    headers, rows, err := parseCSVFile(outFile)
    if err != nil {
        session.sendWSMessage("error", "读取导出 CSV 失败: "+err.Error())
        return
    }

    session.sendWSMessage("nsb_csv_complete", csvHeaderPayload{Headers: headers, Rows: rows, File: outFile})
    session.sendWSMessage("log", fmt.Sprintf("非标优选完成，结果文件: %s", outFile))
}

func writeNSBCSV(outFile string, results []iptestResult, speedTest int, enableTLS bool) error {
    outFile = safeFilename(outFile)
    file, err := os.Create(outFile)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    includeSpeed := speedTest > 0
    if err := writer.Write(nsbCSVHeaders(includeSpeed)); err != nil {
        return err
    }

    for _, res := range results {
        if err := writer.Write(nsbCSVRow(res, includeSpeed, enableTLS)); err != nil {
            return err
        }
    }

    return nil
}

func nsbCSVHeaders(includeSpeed bool) []string {
    headers := []string{"IP", "端口", "TLS", "数据中心", "源IP位置", "地区", "城市", "网络延迟"}
    if includeSpeed {
        headers = append(headers, "下载速度")
    }
    headers = append(headers, "出站IP", "IP类型", "访问协议", "TLS版本", "SNI", "HTTP版本", "WARP", "Gateway", "RBI", "密钥交换", "时间戳")
    return headers
}

func nsbCSVRow(res iptestResult, includeSpeed bool, enableTLS bool) []string {
    row := []string{
        res.ipAddr,
        strconv.Itoa(res.port),
        strconv.FormatBool(enableTLS),
        res.dataCenter,
        res.locCode,
        res.region,
        res.city,
        res.latency,
    }
    if includeSpeed {
        row = append(row, fmt.Sprintf("%.0f kB/s", res.downloadSpeed))
    }
    row = append(row,
        res.outboundIP,
        res.ipType,
        res.visitScheme,
        res.tlsVersion,
        res.sni,
        res.httpVersion,
        res.warp,
        res.gateway,
        res.rbi,
        res.kex,
        res.timestamp,
    )
    return row
}
