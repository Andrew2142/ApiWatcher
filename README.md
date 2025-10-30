# API Watcher

A lightweight Go-Based API Monitoring and Replay System

## Features

- Real-time website/API monitoring
- Record user interactions with visible browser
- Replay snapshots to verify functionality
- Email alerts for failures
- Dashboard with uptime statistics
- SSH remote monitoring

## Requirements

**This application requires Linux, WSL2, or native macOS/Linux. Windows is not supported.**

## Installation

Follow these steps in order:

### 1. Install Go 1.25.3+

**Linux (Ubuntu/Debian):**
```bash
# Download and extract
wget https://go.dev/dl/go1.25.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.3.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**macOS (Intel/x86_64):**
```bash
# Download and extract
curl -O https://go.dev/dl/go1.25.3.darwin-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.3.darwin-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc
source ~/.zshrc
```

**macOS (Apple Silicon/ARM64):**
```bash
# Download and extract
curl -O https://go.dev/dl/go1.25.3.darwin-arm64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.3.darwin-arm64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc
source ~/.zshrc
```

**Verify installation:**
```bash
go version  # Should show: go version go1.25.3 ...
```

### 2. Install Chromium

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install chromium-browser
```

**macOS:**
```bash
brew install chromium
```

### 3. Install Wails

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Verify: `wails version`

### 4. Clone & Setup

```bash
git clone <repository-url>
cd ApiWatcher
cd frontend
npm install
cd ..
```

## Running

**Development (with start.sh):**
```bash
chmod +x start.sh
./start.sh
```

This will:
- Build the daemon
- Start the daemon on port 9876
- Start the frontend dev server on port 9901
- Launch Wails

**Production Build:**
```bash
wails build
```

## Quick Start

1. Run the app: `./start.sh`
2. Choose **Local Mode** to monitor your machine
3. Go to **Websites** tab and add URLs to monitor
4. Go to **Settings** to configure:
   - Worker sleep time (check interval in minutes)
   - Headless browser mode (enable to hide browser windows)
   - Email alerts (SMTP configuration)

## Usage

### 1. Add Websites
1. Go to **Websites** tab
2. Add URLs you want to monitor
3. Click **Save**

### 2. Record Snapshots (Optional)
1. Navigate to **Snapshots** tab
2. Select a website from the list
3. Click **Start Recording**
4. A browser window opens - perform your user interactions
5. Press ENTER when done to save the snapshot

### 3. Replay Snapshots
1. Go to **Snapshots** tab
2. Select a recorded snapshot
3. Click **Replay** to watch it run automatically

### 4. Start Background Monitoring
1. Go to **Dashboard** tab
2. Choose your websites to monitor
3. Click **Start Monitoring**
4. The app will check your websites periodically
5. Browser windows will open based on your **Headless Browser Mode** setting in Settings

## Settings

- **Worker Sleep Time** - Minutes between checks (1-1440)
- **Headless Browser Mode** - Hide browser windows during monitoring
- **Email Alerts** - Configure SMTP for failure notifications

## Project Structure

```
ApiWatcher/
├── cmd/                      # Entry points
├── frontend/                 # React UI
├── internal/                 # Core packages
│   ├── config/              # Settings
│   ├── daemon/              # Background service
│   ├── monitor/             # Website checking
│   ├── snapshot/            # Recording & replay
│   ├── remote/              # SSH support
│   └── email/               # Notifications
├── app.go                    # Wails backend
└── main.go                   # Entry point
```

## Troubleshooting

**Chromium not found:**
- Ensure chromium/chrome is installed and in PATH
- Check: `which chromium-browser` or `which google-chrome`

**Wails not found:**
- Run: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Add `$HOME/go/bin` to PATH

**Port 9876 in use:**
- Change daemon port or kill process using it

**Settings not saving:**
- Check `~/.url-checker/app-settings.json` exists
- Verify file permissions

## Data Storage

- `~/.url-checker/snapshots/` - Recordings
- `~/.url-checker/saved-configs/` - Configurations
- `~/.url-checker/app-settings.json` - Settings
- `~/.apiwatcher/logs/` - Daemon logs

## License

MIT - See LICENSE file

---

Built by machines.
