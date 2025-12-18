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

/*
=====================================================
Configuration (from environment variables)
=====================================================
*/

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

/*
=====================================================
Helpers
=====================================================
*/

func getEnvFloat(key string, def float64) float64 {
	v, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil {
		return def
	}
	return v
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
	body := fmt.Sprintf(
		`{"chat_id":"%s","text":"%s","parse_mode":"Markdown"}`,
		chatID,
		msg,
	)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)
}

func nowUTC() string {
	return time.Now().UTC().Format("2006-01-02 15:04 UTC")
}

/*
=====================================================
System metrics
=====================================================
*/

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

func ramHuman() string {
	return run(`free -h | awk '/Mem:/ {print $3 " / " $2}'`)
}

func diskUsage() float64 {
	out := run(`df / | tail -1 | awk '{print $5}' | tr -d '%'`)
	v, _ := strconv.ParseFloat(out, 64)
	return v
}

func topProcs() string {
	return run(`ps -eo pid,comm,%cpu,%mem --sort=-%cpu | head -6`)
}

/*
=====================================================
Alert logic
=====================================================
*/

func shouldAlert() bool {
	return time.Since(lastAlert) > alertCooldown
}

func sendResourceAlert(cpu, ram, disk float64) {
	host := run("hostname")

	body := fmt.Sprintf(
		"🖥 *Host:* `%s`\n"+
			"⏰ *Time:* %s\n\n"+
			"⚠️ *Thresholds exceeded:*\n"+
			"• CPU Usage: %.1f%%\n"+
			"• RAM Usage: %.1f%%\n"+
			"• Disk Usage: %.1f%%\n\n"+
			"🔥 *Top Processes:*\n```%s```",
		host,
		nowUTC(),
		cpu,
		ram,
		disk,
		topProcs(),
	)

	send("🚨 *Server Resource Alert*\n\n" + body)
}

func checkResources() {
	cpu := cpuUsage()
	ram := ramUsage()
	disk := diskUsage()

	triggered := false

	if cpu >= cpuLimit || ram >= ramLimit || disk >= diskLimit {
		triggered = true
	}

	if triggered && shouldAlert() {
		lastAlert = time.Now()
		sendResourceAlert(cpu, ram, disk)
	}
}

/*
=====================================================
Daily summary
=====================================================
*/

func dailySummary() {
	host := run("hostname")
	uptime := run("uptime -p")
	cores := run("nproc")

	body := fmt.Sprintf(
		"🖥 *Host:* `%s`\n"+
			"⏰ *Time:* %s\n"+
			"⏱ *Uptime:* %s\n"+
			"🧠 *CPU Cores:* %s\n\n"+
			"📈 *Current Usage:*\n"+
			"• CPU: %.1f%%\n"+
			"• RAM: %s\n"+
			"• Disk: %.1f%%\n\n"+
			"🔥 *Top Processes:*\n```%s```",
		host,
		nowUTC(),
		uptime,
		cores,
		cpuUsage(),
		ramHuman(),
		diskUsage(),
		topProcs(),
	)

	send("📊 *Daily Server Summary*\n\n" + body)
}

/*
=====================================================
Debug (can be removed later)
=====================================================
*/

func debugNamespace() {
	send(
		"🧪 *Debug Information*\n\n" +
			"Hostname: `" + run("hostname") + "`\n" +
			"RAM: `" + run("free -h | grep Mem") + "`\n" +
			"CPU Cores: `" + run("nproc") + "`",
	)
}

/*
=====================================================
Main
=====================================================
*/

func main() {
	if token == "" || chatID == "" {
		fmt.Println("Missing TELEGRAM_TOKEN or TELEGRAM_CHAT_ID")
		os.Exit(1)
	}

	host := run("hostname")
	send("✅ *Server monitor started*\n🖥 Host: `" + host + "`")

	// One-time debug (safe to remove later)
	debugNamespace()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		now := time.Now()

		checkResources()

		if now.Hour() == summaryHour && now.Minute() == 0 {
			dailySummary()
			time.Sleep(61 * time.Second) // prevent duplicate summary
		}

		<-ticker.C
	}
}