package main

import (
    "encoding/csv"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
)

func initLocations() {
    filename := "locations.json"
    url := "https://www.baipiao.eu.org/cloudflare/locations"
    var locations []location
    var body []byte
    var err error

    if _, err = os.Stat(filename); os.IsNotExist(err) {
        sendLog("本地 locations.json 不存在，正在从服务器下载...")
        content, err := getURLContent(url)
        if err != nil {
            sendLog("获取位置信息失败: " + err.Error())
            return
        }
        body = []byte(content)
        if err := saveToFile(filename, content); err != nil {
            sendLog("保存位置信息失败: " + err.Error())
        }
    } else {
        sendLog(fmt.Sprintf("读取本地 %s 文件...", filename))
        body, err = os.ReadFile(filename)
        if err != nil {
            sendLog("读取位置信息失败: " + err.Error())
            return
        }
    }

    if err := json.Unmarshal(body, &locations); err != nil {
        sendLog("解析位置信息失败: " + err.Error())
        return
    }

    locationMap = make(map[string]location)
    for _, loc := range locations {
        locationMap[loc.Iata] = loc
    }
    fmt.Printf("已加载 %d 个数据中心位置信息\n", len(locationMap))
}

func sendLog(msg string) {
    fmt.Println(msg)
}

func resetAllConfigFiles(session *appSession) {
    configResetMutex.Lock()
    defer configResetMutex.Unlock()

    result := resetConfigResult{
        Success:  true,
        Deleted:  []string{},
        Missing:  []string{},
        Failed:   []string{},
        Reminder: "删掉之后，下次重新运行程序时会自动重新下载。",
    }

    processTaskMutex.Lock()
    hasRunningTasks := activeTaskCount > 0
    processTaskMutex.Unlock()
    if hasRunningTasks {
        result.Success = false
        result.Failed = []string{"当前还有任务在运行（包含其他连接）"}
        session.sendWSMessage("reset_config_result", result)
        session.sendWSMessage("log", "现在还有任务在跑，先停掉再清理本地缓存文件吧")
        return
    }

    files := []string{"ips-v6.txt", "ips-v4.txt", "locations.json"}
    deleted := []string{}
    missing := []string{}
    failed := []string{}

    for _, filename := range files {
        if err := os.Remove(filename); err == nil {
            deleted = append(deleted, filename)
            continue
        } else if os.IsNotExist(err) {
            missing = append(missing, filename)
        } else {
            failed = append(failed, fmt.Sprintf("%s (%v)", filename, err))
        }
    }

    if len(failed) > 0 {
        result.Success = false
    }
    result.Deleted = deleted
    result.Missing = missing
    result.Failed = failed

    session.sendWSMessage("reset_config_result", result)
    if len(failed) > 0 {
        session.sendWSMessage("log", fmt.Sprintf("本地缓存清理失败: %v", failed))
    } else {
        session.sendWSMessage("log", fmt.Sprintf("本地缓存已清理，已删除: %v，原本不存在: %v", deleted, missing))
    }
    session.sendWSMessage("log", "提示：删掉之后，下次重新运行程序时会自动重新下载。")
}

func getIPListContent(filename, apiURL string) (string, error) {
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        return getURLContent(apiURL)
    }
    return getFileContent(filename)
}

func getURLContent(targetURL string) (string, error) {
    resp, err := http.Get(targetURL)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
        return "", fmt.Errorf("请求失败: %s", resp.Status)
    }
    data, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    return string(data), nil
}

func getFileContent(filename string) (string, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return "", err
    }
    return string(data), nil
}

func saveToFile(filename, content string) error {
    return os.WriteFile(filename, []byte(content), 0644)
}

func parseCSVFile(filename string) ([]string, [][]string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, nil, err
    }
    defer file.Close()

    reader := csv.NewReader(file)
    rows, err := reader.ReadAll()
    if err != nil {
        return nil, nil, err
    }
    if len(rows) == 0 {
        return nil, nil, nil
    }

    headers := rows[0]
    return headers, rows[1:], nil
}
