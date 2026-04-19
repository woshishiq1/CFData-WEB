package main

import (
    "context"
    "embed"
    "net/http"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

const (
    requestURL = "speed.cloudflare.com/cdn-cgi/trace"
)

const (
    timeout     = 1 * time.Second
    maxDuration = 2 * time.Second
)

var (
    //go:embed index.html
    staticFiles embed.FS
)

type location struct {
    Iata      string  `json:"iata"`
    Lat       float64 `json:"lat"`
    Lon       float64 `json:"lon"`
    Cca2      string  `json:"cca2"`
    Region    string  `json:"region"`
    City      string  `json:"city"`
    Region_zh string  `json:"region_zh"`
    Country   string  `json:"country"`
    City_zh   string  `json:"city_zh"`
    Emoji     string  `json:"emoji"`
}

type DataCenterInfo struct {
    DataCenter string
    City       string
    IPCount    int
    MinLatency int
}

type ScanResult struct {
    IP          string
    DataCenter  string
    Region      string
    City        string
    LatencyStr  string
    TCPDuration time.Duration
}

type TestResult struct {
    IP         string
    MinLatency time.Duration
    MaxLatency time.Duration
    AvgLatency time.Duration
    LossRate   float64
    Speed      string
}

type iptestResult struct {
    ipAddr        string
    port          int
    dataCenter    string
    locCode       string
    region        string
    city          string
    latency       string
    tcpDuration   time.Duration
    outboundIP    string
    ipType        string
    visitScheme   string
    tlsVersion    string
    sni           string
    httpVersion   string
    warp          string
    gateway       string
    rbi           string
    kex           string
    timestamp     string
    downloadSpeed float64
}

type csvHeaderPayload struct {
    Headers []string   `json:"headers"`
    Rows    [][]string `json:"rows"`
    File    string     `json:"file"`
}

type resetConfigResult struct {
    Success  bool     `json:"success"`
    Deleted  []string `json:"deleted"`
    Missing  []string `json:"missing"`
    Failed   []string `json:"failed"`
    Reminder string   `json:"reminder"`
}

type appSession struct {
    ws            *websocket.Conn
    wsMutex       sync.Mutex
    taskMutex     sync.Mutex
    isTaskRunning bool
    taskCancel    context.CancelFunc
    wsClosed      bool
    scanMutex     sync.Mutex
    scanResults   []ScanResult
}

type taskStarter func(ctx context.Context, session *appSession)

var (
    locationMap map[string]location

    upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

    processTaskMutex sync.Mutex
    activeTaskCount  int
    configResetMutex sync.Mutex

    listenPort       int
    speedTestURL     string
    speedTestWorkers = 5
)
