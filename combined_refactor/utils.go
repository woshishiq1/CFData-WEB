package main

import (
    "bufio"
    "fmt"
    "io"
    "math/rand"
    "net"
    "os"
    "path/filepath"
    "strings"
)

func readNonEmptyLines(reader io.Reader) ([]string, error) {
    scanner := bufio.NewScanner(reader)
    var lines []string
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }
        lines = append(lines, line)
    }
    return lines, scanner.Err()
}

func parseIPList(content string) ([]string, error) {
    return readNonEmptyLines(strings.NewReader(content))
}

func readIPs(filename string) ([]string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    return readNonEmptyLines(file)
}

func parseTraceResponse(body string) map[string]string {
    result := make(map[string]string)
    lines := strings.Split(body, "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }
        parts := strings.SplitN(line, "=", 2)
        if len(parts) == 2 {
            result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
        }
    }
    return result
}

func getIPType(ip string) string {
    if ip == "" {
        return "未知"
    }
    parsedIP := net.ParseIP(ip)
    if parsedIP == nil {
        return "无效IP"
    }
    if parsedIP.To4() != nil {
        return "IPv4"
    }
    return "IPv6"
}

func safeFilename(name string) string {
    name = strings.TrimSpace(name)
    if name == "" {
        return "ip.csv"
    }
    base := filepath.Base(name)
    if strings.TrimSpace(base) == "" {
        return "ip.csv"
    }
    return base
}

func getRandomIPv4s(ipList []string) []string {
    var randomIPs []string
    for _, subnet := range ipList {
        baseIP := strings.TrimSuffix(subnet, "/24")
        octets := strings.Split(baseIP, ".")
        if len(octets) != 4 {
            continue
        }
        octets[3] = fmt.Sprintf("%d", rand.Intn(256))
        randomIPs = append(randomIPs, strings.Join(octets, "."))
    }
    return randomIPs
}

func getRandomIPv6s(ipList []string) []string {
    var randomIPs []string
    for _, subnet := range ipList {
        baseIP := strings.TrimSuffix(subnet, "/48")
        sections := strings.Split(baseIP, ":")
        if len(sections) < 3 {
            continue
        }
        sections = sections[:3]
        for i := 0; i < 5; i++ {
            sections = append(sections, fmt.Sprintf("%x", rand.Intn(65536)))
        }
        randomIPs = append(randomIPs, strings.Join(sections, ":"))
    }
    return randomIPs
}
