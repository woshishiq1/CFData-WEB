package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/url"
	"strings"
)

const autoSpeedURLValue = "auto"

var encodedAutoSpeedURLs = []string{
	"Y2YuMDkwMjI3Lnh5ei9fX2Rvd24/Ynl0ZXM9OTk5OTk5OTk=",
	"aHR0cHM6Ly9ub2RlanMub3JnL2Rpc3QvdjIyLjIxLjEvbm9kZS12MjIuMjEuMS50YXIuZ3o=",
	"aHR0cHM6Ly9ub2RlanMub3JnL2Rpc3QvdjIzLjExLjEvbm9kZS12MjMuMTEuMS50YXIuZ3o=",
	"aHR0cHM6Ly9ub2RlanMub3JnL2Rpc3QvdjI0LjEyLjAvbm9kZS12MjQuMTIuMC50YXIuZ3o=",
	"aHR0cHM6Ly9yZWdpc3RyeS5ucG1qcy5vcmcvb25ueHJ1bnRpbWUtbm9kZS8tL29ubnhydW50aW1lLW5vZGUtMS4yMy4yLnRneg==",
	"aHR0cHM6Ly9yZWdpc3RyeS5ucG1qcy5vcmcvb25ueHJ1bnRpbWUtbm9kZS8tL29ubnhydW50aW1lLW5vZGUtMS4yMy4wLnRneg==",
	"aHR0cHM6Ly9yZWdpc3RyeS5ucG1qcy5vcmcvb25ueHJ1bnRpbWUtbm9kZS8tL29ubnhydW50aW1lLW5vZGUtMS4yMi4wLnRneg==",
	"aHR0cHM6Ly9ub2RlanMub3JnL2Rpc3QvdjIxLjcuMy9ub2RlLXYyMS43LjMudGFyLmd6",
	"aHR0cHM6Ly9ub2RlanMub3JnL2Rpc3QvdjIwLjE5LjYvbm9kZS12MjAuMTkuNi50YXIuZ3o=",
	"aHR0cHM6Ly9ub2RlanMub3JnL2Rpc3QvdjE4LjIwLjgvbm9kZS12MTguMjAuOC50YXIuZ3o=",
	"aHR0cHM6Ly9yZXBvLmFuYWNvbmRhLmNvbS9taW5pY29uZGEvTWluaWNvbmRhMy1weTMxMF8yNC4xMS4xLTAtTGludXgteDg2XzY0LnNo",
	"aHR0cHM6Ly9yZXBvLmFuYWNvbmRhLmNvbS9taW5pY29uZGEvTWluaWNvbmRhMy1weTMxMV8yNC4xMS4xLTAtTGludXgteDg2XzY0LnNo",
	"aHR0cHM6Ly9yZXBvLmFuYWNvbmRhLmNvbS9taW5pY29uZGEvTWluaWNvbmRhMy1weTMxMl8yNC4xMS4xLTAtTGludXgteDg2XzY0LnNo",
	"aHR0cHM6Ly9yZXBvLmFuYWNvbmRhLmNvbS9taW5pY29uZGEvTWluaWNvbmRhMy1sYXRlc3QtTGludXgteDg2XzY0LnNo",
}

func resolveSpeedTestURL(rawURL string) string {
	value := strings.TrimSpace(rawURL)
	if value == "" {
		value = strings.TrimSpace(speedTestURL)
	}
	if strings.EqualFold(value, autoSpeedURLValue) || value == "自动选择" {
		return pickAutoSpeedURL()
	}
	return value
}

func pickAutoSpeedURL() string {
	if len(encodedAutoSpeedURLs) == 0 {
		return speedTestURL
	}
	idx := 0
	if n, err := rand.Int(rand.Reader, big.NewInt(int64(len(encodedAutoSpeedURLs)))); err == nil {
		idx = int(n.Int64())
	}
	decoded, err := base64.StdEncoding.DecodeString(encodedAutoSpeedURLs[idx])
	if err != nil || len(decoded) == 0 {
		return speedTestURL
	}
	return string(decoded)
}

func parseSpeedTestURL(rawURL string, fallbackScheme string) (*url.URL, error) {
	value := resolveSpeedTestURL(rawURL)
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("测速地址为空")
	}

	fallbackScheme = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(fallbackScheme)), "://")
	if fallbackScheme != "http" && fallbackScheme != "https" {
		fallbackScheme = "https"
	}

	lowerValue := strings.ToLower(value)
	if strings.HasPrefix(value, "//") {
		value = fallbackScheme + ":" + value
	} else if !strings.HasPrefix(lowerValue, "http://") && !strings.HasPrefix(lowerValue, "https://") {
		value = fallbackScheme + "://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return nil, err
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("不支持的测速地址协议: %s", parsed.Scheme)
	}
	if strings.TrimSpace(parsed.Hostname()) == "" {
		return nil, fmt.Errorf("测速地址缺少域名")
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed, nil
}
