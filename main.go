package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type LogConfig struct {
	Name string
	Path string
}

var (
	logs          []LogConfig
	systemMetrics []string
	allowedHosts  []string
	allowAll      bool
)

func main() {
	port := flag.Int("p", 8080, "Port")
	logsFlag := flag.String("logs", "", "name:path,name:path")
	metricsFlag := flag.String("system_metrics", "", "cpu,memory,disk")
	hostsFlag := flag.String("allowed_hosts", "*", "Allowed IPs: * or ip1,ip2,ip3")
	flag.Parse()

	logs = parseLogConfig(*logsFlag)
	systemMetrics = parseList(*metricsFlag)
	parseAllowedHosts(*hostsFlag)

	for _, log := range logs {
		http.HandleFunc("/"+log.Name, protect(logHandler(log.Path)))
	}
	http.HandleFunc("/metrics", protect(metricsHandler))
	http.HandleFunc("/health", protect(healthHandler))

	fmt.Printf("agent started on :%d\n", *port)
	if allowAll {
		fmt.Println("allowed_hosts: *")
	} else {
		fmt.Printf("allowed_hosts: %v\n", allowedHosts)
	}
	http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
}

func parseAllowedHosts(s string) {
	if s == "*" || s == "" {
		allowAll = true
		return
	}
	allowedHosts = strings.Split(s, ",")
}

func protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !allowAll {
			ip := getClientIP(r)
			if !isAllowed(ip) {
				http.Error(w, "forbidden", 403)
				return
			}
		}
		next(w, r)
	}
}

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isAllowed(ip string) bool {
	for _, allowed := range allowedHosts {
		if allowed == ip {
			return true
		}
	}
	return false
}

func parseLogConfig(s string) []LogConfig {
	var configs []LogConfig
	if s == "" {
		return configs
	}
	for _, item := range strings.Split(s, ",") {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) == 2 {
			configs = append(configs, LogConfig{Name: parts[0], Path: parts[1]})
		}
	}
	return configs
}

func parseList(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func logHandler(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lines := 20
		if l := r.URL.Query().Get("lines"); l != "" {
			if n, err := strconv.Atoi(l); err == nil {
				lines = n
			}
		}
		if lines > 100 {
			lines = 100
		}

		content, err := tailFile(path, lines)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(content))
	}
}

func tailFile(path string, n int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return strings.Join(lines, "\n"), nil
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"timestamp": time.Now().Unix(),
	}

	for _, m := range systemMetrics {
		switch m {
		case "cpu":
			data["cpu"] = getCPU()
		case "memory":
			data["memory"] = getMemory()
		case "disk":
			data["disk"] = getDisks()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

func getCPU() map[string]interface{} {
	idle1, total1 := readCPUStat()
	time.Sleep(100 * time.Millisecond)
	idle2, total2 := readCPUStat()

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)
	used := 100.0 * (1.0 - idleDelta/totalDelta)

	return map[string]interface{}{
		"used_percent": round(used),
		"free_percent": round(100.0 - used),
	}
}

func readCPUStat() (idle, total uint64) {
	data, _ := os.ReadFile("/proc/stat")
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			for i := 1; i < len(fields); i++ {
				val, _ := strconv.ParseUint(fields[i], 10, 64)
				total += val
				if i == 4 {
					idle = val
				}
			}
			break
		}
	}
	return
}

func getMemory() []map[string]interface{} {
	var info syscall.Sysinfo_t
	syscall.Sysinfo(&info)

	total := info.Totalram / 1024 / 1024
	free := info.Freeram / 1024 / 1024
	used := total - free

	swapTotal := info.Totalswap / 1024 / 1024
	swapFree := info.Freeswap / 1024 / 1024
	swapUsed := swapTotal - swapFree

	return []map[string]interface{}{
		{
			"type":     "ram",
			"total_mb": total,
			"used_mb":  used,
			"free_mb":  free,
		},
		{
			"type":     "swap",
			"total_mb": swapTotal,
			"used_mb":  swapUsed,
			"free_mb":  swapFree,
		},
	}
}

func getDisks() []map[string]interface{} {
	var disks []map[string]interface{}

	data, _ := os.ReadFile("/proc/mounts")
	lines := strings.Split(string(data), "\n")

	seen := make(map[string]bool)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		device := fields[0]
		mount := fields[1]

		// skip non-local disks
		if !strings.HasPrefix(device, "/dev/") {
			continue
		}
		// skip NAS/network mounts
		if strings.Contains(device, "nfs") || strings.Contains(mount, "nfs") {
			continue
		}
		// skip duplicates
		if seen[device] {
			continue
		}
		seen[device] = true

		var stat syscall.Statfs_t
		if err := syscall.Statfs(mount, &stat); err != nil {
			continue
		}

		total := stat.Blocks * uint64(stat.Bsize) / 1024 / 1024 / 1024
		free := stat.Bfree * uint64(stat.Bsize) / 1024 / 1024 / 1024
		used := total - free

		disks = append(disks, map[string]interface{}{
			"mount":    mount,
			"total_gb": total,
			"used_gb":  used,
			"free_gb":  free,
		})
	}
	return disks
}

func round(f float64) float64 {
	return float64(int(f*100)) / 100
}
