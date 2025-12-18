package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	token  = os.Getenv("TELEGRAM_TOKEN")
	chatID = os.Getenv("TELEGRAM_CHAT_ID")

	cpuLimit  = getEnvFloat("ALERT_CPU", 50)
	ramLimit  = getEnvFloat("ALERT_RAM", 50)
	diskLimit = getEnvFloat("ALERT_DISK", 50)

	alertCooldown = time.Duration(getEnvInt("ALERT_INTERVAL_MINUTES", 15)) * time.Minute
	summaryHour   = getEnvInt("SUMMARY_HOUR", 9)

	lastAlert time.Time
)

func getEnvFloat(key string, def float64) float64 {
	v, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil {
		return def
	}
	return v
}

func debugNamespace() {
	send("🧪 Debug\n" +
		"Hostname: `" + run("hostname") + "`\n" +
		"Total RAM: `" + run("free -h | grep Mem") + "`\n" +
		"CPU info: `" + run("nproc") + " cores`")
}

func getEnvInt(key string, def int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}
	return v
}

func run(cmd string) string {
	out, _ := exec.Command("bash", "-c", cmd).Output()
	return strings.TrimSpace(string(out))
}

func send(msg string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	body := fmt.Sprintf(`{"chat_id":"%s","text":"%s","parse_mode":"Markdown"}`, chatID, msg)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)
}

func cpuUsage() float64 {
	out := run(`top -bn1 | awk '/Cpu/ {print 100 - $8}'`)
	v, _ := strconv.ParseFloat(out, 64)
	return v
}

func ramUsage() float64 {
	out := run(`free | awk '/Mem:/ {print $3/$2*100}'`)
	v, _ := strconv.ParseFloat(out, 64)
	return v
}

func diskUsage() float64 {
	out := run(`df / | tail -1 | awk '{print $5}' | tr -d '%'`)
	v, _ := strconv.ParseFloat(out, 64)
	return v
}

func topProcs() string {
	return run(`ps -eo pid,comm,%cpu,%mem --sort=-%cpu | head -6`)
}

func shouldAlert() bool {
	return time.Since(lastAlert) > alertCooldown
}

func check() {
	cpu := cpuUsage()
	ram := ramUsage()
	disk := diskUsage()

	var alerts []string

	if cpu >= cpuLimit {
		alerts = append(alerts, fmt.Sprintf("🔥 CPU %.1f%%", cpu))
	}
	if ram >= ramLimit {
		alerts = append(alerts, fmt.Sprintf("🔥 RAM %.1f%%", ram))
	}
	if disk >= diskLimit {
		alerts = append(alerts, fmt.Sprintf("🔥 Disk %.1f%%", disk))
	}

	if len(alerts) > 0 && shouldAlert() {
		lastAlert = time.Now()
		host, _ := os.Hostname()

		send(fmt.Sprintf(
			"🚨 *Resource Alert*\nHost: `%s`\n\n%s\n\n*Top processes:*\n```%s```",
			host,
			strings.Join(alerts, "\n"),
			topProcs(),
		))
	}
}

func dailySummary() {
	host, _ := os.Hostname()
	uptime := run("uptime -p")
	disk := run("df -h / | tail -1")

	send(fmt.Sprintf(
		"📊 *Daily Summary*\n\n"+
			"*Host:* `%s`\n"+
			"*Uptime:* %s\n"+
			"*CPU:* %.1f%%\n"+
			"*RAM:* %.1f%%\n"+
			"*Disk:* `%s`\n\n"+
			"*Top processes:*\n```%s```",
		host,
		uptime,
		cpuUsage(),
		ramUsage(),
		disk,
		topProcs(),
	))
}

func main() {
	if token == "" || chatID == "" {
		fmt.Println("Missing Telegram env vars")
		os.Exit(1)
	}

	host, _ := os.Hostname()
	send("✅ *Server monitor started*\nHost: `" + host + "`")
	debugNamespace() 

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		now := time.Now()
		check()

		if now.Hour() == summaryHour && now.Minute() == 0 {
			dailySummary()
			time.Sleep(61 * time.Second)
		}

		<-ticker.C
	}
}