// API utility to call Go backend functions via Wails

export const api = {
  // Connection Management
  listSSHProfiles: () => window.backend.App.ListSSHProfiles(),
  connectToServer: (host, username, password) => window.backend.App.ConnectToServer(host, username, password),
  testConnection: (host, username, password) => window.backend.App.TestConnection(host, username, password),
  startLocalDaemon: () => window.backend.App.StartLocalDaemon(),
  getConnectionStatus: () => window.backend.App.GetConnectionStatus(),
  disconnectFromServer: () => window.backend.App.DisconnectFromServer(),
  checkDaemonStatus: (host, username, password) => window.backend.App.CheckDaemonStatus(host, username, password),
  deployDaemonToServer: (host, username, password) => window.backend.App.DeployDaemonToServer(host, username, password),
  getConnectionHealth: () => window.backend.App.GetConnectionHealth(),

  // Dashboard & Monitoring
  getDashboardData: () => window.backend.App.GetDashboardData(),
  getWebsiteStats: () => window.backend.App.GetWebsiteStats(),
  getDaemonLogs: (lines) => window.backend.App.GetDaemonLogs(lines || 100),
  clearLogs: () => window.backend.App.ClearLogs(),
  startMonitoring: (websites) => window.backend.App.StartMonitoring(websites),
  stopMonitoring: () => window.backend.App.StopMonitoring(),

  // Configuration
  listConfigs: () => window.backend.App.ListConfigs(),
  loadConfig: (name) => window.backend.App.LoadConfig(name),
  saveConfig: (name, urls) => window.backend.App.SaveConfig(name, urls),
  createNewConfig: (name, urls) => window.backend.App.CreateNewConfig(name, urls),
  deleteConfig: (name) => window.backend.App.DeleteConfig(name),

  // Snapshots
  listSnapshots: (url) => window.backend.App.ListSnapshots(url),
  createSnapshot: (url) => window.backend.App.CreateSnapshot(url),
  startRecording: (url) => window.backend.App.StartRecording(url),
  finishRecording: (recordingId) => window.backend.App.FinishRecording(recordingId),
  deleteSnapshot: (id) => window.backend.App.DeleteSnapshot(id),
  replaySnapshot: (id) => window.backend.App.ReplaySnapshot(id),

  // SMTP
  configureSMTP: (host, port, username, password, from, to) =>
    window.backend.App.ConfigureSMTP(host, port, username, password, from, to),
  getSMTPStatus: () => window.backend.App.GetSMTPStatus(),
  getSMTPConfig: () => window.backend.App.GetSMTPConfig(),

  // Settings
  getAppSettings: () => window.backend.App.GetAppSettings(),
  saveAppSettings: (workerSleepTime, headlessBrowserMode) => window.backend.App.SaveAppSettings(workerSleepTime, headlessBrowserMode),

  // Utilities
  ping: () => window.backend.App.Ping(),
  getLastConnectedServer: () => window.backend.App.GetLastConnectedServer(),
}

export default api
