# Server Resource Monitor (Telegram Alerts)

A lightweight Go-based Linux server monitoring agent that sends
human-readable alerts and daily summaries to Telegram.

Designed to run directly on the host using systemd
(no Docker, no agents, no dependencies).



# ✨ Features

- CPU, RAM, Disk monitoring
- Spike alerts with cooldown (anti-spam)
- Top offending processes (CPU + RAM)
- Daily summary report
- Telegram notifications
- Very low resource usage
- systemd-managed (auto-restart, boot-safe)



# 🧠 Requirements

- Linux server
- Go (>= 1.13)
- systemd
- Telegram bot token & chat ID



# 🚀 Installation

## 1. Clone the repository
```bash
cd /opt
git clone https://github.com/joeVenner/go-server-resources-tracker.git
cd go-server-resources-tracker
```



## 2. Build the binary
```
go build -o server-monitor
```



## 3. Create environment config
```
sudo nano /etc/server-monitor.env

TELEGRAM_TOKEN=YOUR_TELEGRAM_TOKEN
TELEGRAM_CHAT_ID=YOUR_CHAT_ID

ALERT_CPU=50
ALERT_RAM=50
ALERT_DISK=50
ALERT_INTERVAL_MINUTES=15
SUMMARY_HOUR=9

sudo chmod 600 /etc/server-monitor.env
```

## 4. Create systemd service
```
sudo nano /etc/systemd/system/server-monitor.service

[Unit]
Description=Server Resource Monitor (Telegram Alerts)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/opt/go-server-resources-tracker/server-monitor
EnvironmentFile=/etc/server-monitor.env
Restart=always
RestartSec=5
User=root
Nice=10
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```


## 5. Enable & start service
```
sudo systemctl daemon-reload
sudo systemctl enable server-monitor
sudo systemctl start server-monitor
```


### ✅ Verification

Check status:

```
systemctl status server-monitor
```

View logs:

```
journalctl -u server-monitor -f
```

You should receive a Telegram message:

### ✅ Server monitor started
Host: your-server-name


### 🧪 Testing alerts

Trigger CPU load:

``` 
yes > /dev/null &
sleep 30
killall yes 
```


### 🧹 Uninstall

```
sudo systemctl stop server-monitor
sudo systemctl disable server-monitor
sudo rm /etc/systemd/system/server-monitor.service
sudo rm /etc/server-monitor.env
```



## 📜 License MIT