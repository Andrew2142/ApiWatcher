# API Watcher

A powerful website and API monitoring application with interactive snapshot recording and replay capabilities. Monitor your infrastructure in real-time, record user interactions, and replay them to verify functionality.

## Key Features

- **Real-time Monitoring** - Continuously monitor website and API availability with configurable check intervals
- **Interactive Snapshots** - Record user interactions (clicks, form fills, navigation) using a visible browser
- **Snapshot Replay** - Replay recorded snapshots in a headless or visible browser to verify functionality
- **SSH Remote Monitoring** - Monitor APIs and websites on remote servers via SSH tunneling
- **Email Alerts** - Receive SMTP-based email notifications when websites go down or experience issues
- **Dashboard Analytics** - View uptime statistics (1h, 24h, 7d), response times, and health metrics
- **Headless Browser Mode** - Toggle between visible and headless browser for recordings and replays
- **Configuration Presets** - Save and load monitoring configurations for quick switching
- **Cross-Platform Desktop App** - Built with Go and React for a modern UI experience

## Tech Stack

**Backend:**
- Go 1.25.1+ with Wails framework for desktop UI
- ChromeDP for browser automation and snapshot recording
- SSH support via golang.org/x/crypto for remote monitoring
- Custom daemon architecture for background monitoring

**Frontend:**
- React 18.2.0 with modern hooks
- TailwindCSS for responsive styling
- ag-grid for data tables and statistics
- FontAwesome icons
- Webpack for bundling

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go** 1.25.1 or higher ([Download](https://go.dev/dl))
- **Node.js & npm** 14+ ([Download](https://nodejs.org))
- **Wails CLI** (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- **Google Chrome/Chromium** (required for snapshot recording and replay)
- **Git** (for cloning the repository)

Optional:
- SSH capability for remote monitoring
- SMTP server credentials for email alerts

## Installation & Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd ApiWatcher
```

### 2. Install Frontend Dependencies

```bash
cd frontend
npm install
cd ..
```

### 3. Install Wails

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 4. Build & Run

**Quick Start (Development):**
```bash
wails dev
```

This will:
- Build the daemon
- Start the frontend dev server with hot reload
- Launch the Wails application

**Production Build:**
```bash
wails build
```

The compiled executable will be in the `build/` directory.

### 5. Configuration (Optional)

Create a `.env` file in the project root for SMTP email configuration:

```env
SMTP_HOST=mail.smtp2go.com
SMTP_PORT=2525
SMTP_USER=your-api-key
SMTP_PASS=your-password
SMTP_FROM=sender@example.com
WORKER_SLEEP=10
```

Alternatively, configure SMTP settings through the application UI: **Settings → Email (SMTP)**.

## Project Structure

```
ApiWatcher/
├── cmd/
│   ├── apiwatcher-gui/        # GUI application entry point
│   └── apiwatcher-daemon/     # Background daemon service
├── frontend/                  # React application
│   ├── src/
│   │   ├── screens/          # Full-page components
│   │   ├── components/       # Reusable React components
│   │   ├── layouts/          # Page layouts
│   │   ├── api.js           # Backend API wrapper
│   │   └── App.jsx          # Root component
│   ├── webpack.config.js     # Webpack configuration
│   ├── tailwind.config.js    # TailwindCSS configuration
│   └── package.json          # NPM dependencies
├── internal/                  # Core Go packages
│   ├── config/               # Settings and configuration
│   ├── daemon/               # Monitoring daemon service
│   ├── monitor/              # Monitoring workers
│   ├── snapshot/             # Recording and replay logic
│   ├── remote/               # SSH remote connections
│   ├── email/                # Email notifications
│   └── models/               # Data structures
├── app.go                     # Main Wails app backend
├── main.go                    # Application entry point
├── go.mod & go.sum           # Go dependencies
└── project.json              # Wails configuration
```

## Usage

### Starting the Application

1. Run `wails dev` for development or launch the compiled executable
2. The application will start in **connection mode**
3. Choose **Local Mode** to monitor your current machine, or **SSH Mode** to connect to a remote server

### Recording a Snapshot

1. Navigate to **Snapshots** tab
2. Enter a website URL
3. Click **Start Recording**
4. A visible browser window will open
5. Perform the interactions you want to record (clicks, form fills, navigation)
6. When finished, press ENTER or click the finish button
7. Your snapshot is saved and ready for replay

### Replaying Snapshots

1. Go to **Snapshots** tab
2. Select a recorded snapshot
3. Click **Replay**
4. Watch as the browser automatically replays your recorded interactions
5. Check the logs for any API errors detected during replay

### Configuring Monitoring

1. Navigate to **Websites** tab
2. Add URLs to monitor
3. Configure monitoring interval in **Settings → General Settings**
4. Enable **Headless Browser Mode** to run recordings invisibly (Settings → General Settings)
5. Set up email alerts in **Settings → Email (SMTP)**

## Settings

### General Settings
- **Worker Sleep Time** - Minutes between monitoring cycles (1-1440, default: 10)
- **Headless Browser Mode** - Run browser windows invisibly during recording and replay (default: off)

### Email Settings
Configure SMTP to receive email alerts when websites go down:
- SMTP Host and Port
- Username and Password
- From and To email addresses

## API Methods

The Go backend exposes these methods to the React frontend:

**Monitoring**
- `StartMonitoring(websites)` - Begin monitoring URLs
- `StopMonitoring()` - Stop monitoring
- `GetDashboardData()` - Get statistics and status

**Snapshots**
- `StartRecording(url)` - Begin recording user interactions
- `FinishRecording(recordingId)` - End recording
- `ReplaySnapshot(id)` - Replay a snapshot
- `ListSnapshots(url)` - List snapshots for a URL
- `DeleteSnapshot(id)` - Delete a snapshot

**Settings**
- `SaveAppSettings(workerSleepTime, headlessBrowserMode)` - Save application settings
- `GetAppSettings()` - Get current settings

**Remote**
- `ConnectToServer(host, username, password)` - SSH connection
- `ListSSHProfiles()` - Get saved SSH profiles

## Troubleshooting

**Chrome not found:**
- Ensure Google Chrome/Chromium is installed and in PATH
- On Linux: `sudo apt-get install chromium-browser` or `google-chrome`
- On macOS: Chrome should be in `/Applications/Google Chrome.app`
- On Windows: Chrome should be in Program Files

**SMTP errors:**
- Verify credentials in Settings → Email (SMTP)
- Check SMTP host and port are correct
- Use app-specific passwords for Gmail or other providers

**Recording not appearing:**
- Check that the snapshot directory exists: `~/.apiwatcher/snapshots/`
- Verify file permissions

**Daemon not connecting:**
- Ensure daemon is running: Check `~/.apiwatcher/logs/daemon.log`
- Verify port 9876 is not blocked
- Check network connectivity for remote SSH connections

## Development

### Frontend Development
```bash
cd frontend
npm run serve    # Start dev server on port 9901
npm run build    # Build for production
npm run lint     # Run ESLint
```

### Backend Development
```bash
go build -o apiwatcher-daemon ./cmd/apiwatcher-daemon/
go test ./...    # Run tests
```

### Database
Snapshots and configurations are stored in:
- `~/.apiwatcher/snapshots/` - Snapshot recordings
- `~/.apiwatcher/saved-configs/` - Monitoring configurations
- `~/.apiwatcher/app-settings.json` - Application settings
- `~/.apiwatcher/logs/` - Daemon logs

## License

MIT License - See LICENSE file for details

## Support

For issues, feature requests, or questions, please open an issue on the repository.

---

**Made with ❤️ for website and API monitoring**
