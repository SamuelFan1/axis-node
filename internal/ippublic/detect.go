package ippublic

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var defaultURLs = []string{
	"https://api.ipify.org",
	"https://ifconfig.me/ip",
	"https://checkip.amazonaws.com",
}

func Detect() string {
	client := &http.Client{Timeout: 5 * time.Second}
	for _, url := range defaultURLs {
		ip, err := fetchFrom(client, url)
		if err == nil && strings.TrimSpace(ip) != "" {
			return strings.TrimSpace(ip)
		}
	}
	return ""
}

func fetchFrom(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(resp.Body)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	return "", fmt.Errorf("empty response")
}
