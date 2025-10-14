# API Watcher

A lightweight Go-based website monitoring tool that checks your websites for failing API calls and sends alert emails when issues are detected.

---

## Features

* Monitors multiple websites automatically
* Detects failing API calls (HTTP 4xx/5xx responses)
* Ignores static assets like `.js`, `.css`, `.png`, etc.
* Sends email alerts via SMTP
* Prevents duplicate alerts by storing alert timestamps
* Configurable via `~/.url-checker/config.json` and `.env` file

---

## Installation

### Prerequisites

* Go 1.19+
* Google Chrome or Chromium installed

### Setup

1. Clone the repository:

```bash
git clone https://github.com/yourusername/url-checker.git
cd url-checker
```

2. Install dependencies:

```bash
go mod tidy
```

3. Create a `.env` file in the project root:

```env
SMTP_FROM=your-email@example.com
SMTP_USER=your-smtp-username
SMTP_PASS=your-smtp-password
SMTP_HOST=mail.smtp2go.com
SMTP_PORT=587
WORKER_SLEEP=10
```

4. Build the app:

```bash
go build -o url-checker
```

5. Run the app:

```bash
./url-checker
```

---

## Configuration

### Config File

* Location: `~/.url-checker/config.json`
* Stores websites to monitor and alert email

Example:

```json
{
  "email": "alerts@example.com",
  "websites": [
    "https://example.com",
    "https://api.example.com"
  ]
}
```

---

### Email Alerts

The app uses SMTP credentials from `.env`.
Update `SMTP_FROM`, `SMTP_USER`, `SMTP_PASS`, `SMTP_HOST`, and `SMTP_PORT` accordingly.

---

## How it Works

1. Loads configuration from `config.json` and `.env`
2. Uses headless Chrome (`chromedp`) to visit each website
3. Captures network requests and checks their HTTP status
4. Sends alert emails if any request fails
5. Avoids duplicate alerts by checking the last alert time (default 5 hours)

---

## Logs & Persistence

* Alert timestamps saved in: `~/.url-checker/alert_log.json`
* Console output shows info, warnings, and errors

---

## Contributing

1. Fork the repository
2. Create a branch for your feature: `git checkout -b feature/my-feature`
3. Commit your changes: `git commit -m "Add my feature"`
4. Push to your branch: `git push origin feature/my-feature`
5. Open a pull request

---

## License

MIT License © Legend



INSTALL DAEMON ON SERVER
 
# 1. Navigate to your project (you're probably close)
cd ~/dev/ApiWatcher

# 2. Build the daemon
go build -o apiwatcher-daemon ./cmd/apiwatcher-daemon

# 3. Move to the correct location
mkdir -p ~/.apiwatcher/bin
mv apiwatcher-daemon ~/.apiwatcher/bin/

# 4. Restart daemon
pkill -f apiwatcher-daemon
chmod +x ~/.apiwatcher/bin/apiwatcher-daemon
nohup ~/.apiwatcher/bin/apiwatcher-daemon > ~/.apiwatcher/logs/daemon.log 2>&1 &

# 5. Watch logs
tail -f ~/.apiwatcher/logs/daemon.log