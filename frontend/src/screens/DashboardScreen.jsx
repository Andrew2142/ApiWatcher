import React, { useState, useEffect, useMemo, useCallback } from 'react'
import { AgGridReact } from 'ag-grid-react'
import { themeQuartz } from 'ag-grid-community'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faSyncAlt, faExclamationTriangle, faTrash } from '@fortawesome/free-solid-svg-icons'
import api from '../api'

function DashboardScreen({ error, setError, onNavigate }) {
  const [dashboardData, setDashboardData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [logs, setLogs] = useState([])
  const [logsLoading, setLogsLoading] = useState(false)
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(true)

  // Monitoring selection state
  const [showStartMonitor, setShowStartMonitor] = useState(true)
  const [allAvailableWebsites, setAllAvailableWebsites] = useState([])
  const [selectedWebsites, setSelectedWebsites] = useState({}) // { url: { enabled: bool, snapshotMonitoring: bool } }
  const [monitoredWebsites, setMonitoredWebsites] = useState([]) // URLs currently being monitored (for display after refresh)
  const [isMonitoringActive, setIsMonitoringActive] = useState(false)
  const [isStopping, setIsStopping] = useState(false)

  useEffect(() => {
    let isMounted = true

    const initLoad = async () => {
      if (!isMounted) return
      await loadAvailableWebsites()
      if (!isMounted) return
      await loadDashboard()
      if (!isMounted) return

      // Check if monitoring is actually active in the daemon
      try {
        const data = await api.getDashboardData()
        if (data && data.daemon_status && (data.daemon_status.state === 'running' || data.daemon_status.state === 'paused')) {
          // Daemon is actively monitoring
          if (isMounted) {
            setIsMonitoringActive(true)
            setShowStartMonitor(false)
            // Restore the list of monitored websites from the dashboard stats
            if (data.website_stats && data.website_stats.length > 0) {
              const monitoredUrls = data.website_stats.map(site => site.url)
              setMonitoredWebsites(monitoredUrls)
            }
          }
        }
      } catch (err) {
        console.error('Failed to check daemon state:', err)
      }

      if (!isMounted) return
      await loadLogs()
    }

    initLoad()

    let dashboardInterval

    if (autoRefreshEnabled) {
      dashboardInterval = setInterval(() => {
        if (isMounted) loadDashboard()
      }, 60000)
    }

    return () => {
      isMounted = false
      if (dashboardInterval) clearInterval(dashboardInterval)
    }
  }, [autoRefreshEnabled])

  const loadDashboard = async () => {
    try {
      if (!refreshing) {
        setLoading(true)
      }
      const data = await api.getDashboardData()
      setDashboardData(data)
      setError(null)
    } catch (err) {
      setError(err.message || 'Failed to load dashboard')
      console.error('Dashboard error:', err)
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  const loadLogs = async () => {
    try {
      setLogsLoading(true)
      const daemonLogs = await api.getDaemonLogs(50)
      setLogs(daemonLogs || [])
    } catch (err) {
      console.error('Failed to load logs:', err)
    } finally {
      setLogsLoading(false)
    }
  }

  const handleRefresh = async () => {
    setRefreshing(true)
    await Promise.all([loadDashboard(), loadLogs()])
  }

  const handleClearLogs = async () => {
    try {
      await api.clearLogs()
      setLogs([])
      setError(null)
    } catch (err) {
      setError(err.message || 'Failed to clear logs')
    }
  }

  const loadAvailableWebsites = async () => {
    try {
      const configs = await api.listConfigs()
      const websites = []
      if (configs && configs.length > 0) {
        configs.forEach(cfg => {
          cfg.urls.forEach((url) => {
            websites.push(url)
          })
        })
      }
      setAllAvailableWebsites(websites)
    } catch (err) {
      console.error('Failed to load available websites:', err)
    }
  }

  const handleStartMonitoring = async () => {
    // Check if any websites are selected
    const selected = Object.entries(selectedWebsites).filter(([_, val]) => val.enabled)
    if (selected.length === 0) {
      setError('Please select at least one website to monitor')
      return
    }

    try {
      // Store the URLs of monitored websites for display after page refresh
      const monitoredUrls = selected.map(([url]) => url)
      setMonitoredWebsites(monitoredUrls)

      // Create a map of URLs to their snapshot monitoring preference
      const monitoringConfig = {}
      selected.forEach(([url, config]) => {
        monitoringConfig[url] = {
          enableSnapshots: config.snapshotMonitoring || false
        }
      })

      // Tell the daemon to start monitoring these websites with their snapshot preferences
      await api.startMonitoring(monitoringConfig)

      // Hide the start monitor grid and show the dashboard data
      setShowStartMonitor(false)
      setIsMonitoringActive(true)
      setError(null)

      // Refresh dashboard data to show the monitored websites
      await loadDashboard()
    } catch (err) {
      setError(err.message || 'Failed to start monitoring')
      console.error('Start monitoring error:', err)
    }
  }

  const handleStopMonitoring = async () => {
    try {
      setIsStopping(true)
      await api.stopMonitoring()

      // Poll daemon status until it's actually stopped
      let maxAttempts = 60 // 30 seconds max (500ms * 60)
      let attempts = 0
      while (attempts < maxAttempts) {
        await new Promise(resolve => setTimeout(resolve, 500))
        const data = await api.getDashboardData()
        if (data && data.daemon_status && data.daemon_status.state === 'stopped') {
          break
        }
        attempts++
      }

      // Daemon has stopped, now update UI
      setShowStartMonitor(true)
      setIsMonitoringActive(false)
      setSelectedWebsites({})
      setMonitoredWebsites([])
      setError(null)
      setIsStopping(false)
    } catch (err) {
      setError(err.message || 'Failed to stop monitoring')
      setIsStopping(false)
    }
  }

 

  const getStartMonitorColumnDefs = useCallback(() => [
    {
      headerName: 'Select',
      width: 80,
      cellRenderer: (props) => (
        <div className="flex items-center h-full">
          <input
            type="checkbox"
            checked={selectedWebsites[props.data] ? selectedWebsites[props.data].enabled : false}
            onChange={(e) =>
              setSelectedWebsites({
                ...selectedWebsites,
                [props.data]: {
                  ...selectedWebsites[props.data],
                  enabled: e.target.checked,
                },
              })
            }
            className="w-4 h-4 cursor-pointer"
          />
        </div>
      ),
      sortable: false,
      filter: false,
    },
    {
      headerName: 'Website URL',
      field: 'value',
  
      minWidth: 250,
      cellRenderer: (props) => (
        <a
          href={props.data}
          target="_blank"
          rel="noopener noreferrer"
          className="text-blue-600 hover:underline break-all"
        >
          {props.data}
        </a>
      ),
    },
    {
      headerName: 'Enable Snapshot Monitoring',
      width:250,
      cellRenderer: (props) => (
        <div className="flex items-center h-full justify-between">
          <input
            type="checkbox"
            checked={selectedWebsites[props.data] ? selectedWebsites[props.data].snapshotMonitoring : false}
            onChange={(e) =>
              setSelectedWebsites({
                ...selectedWebsites,
                [props.data]: {
                  ...selectedWebsites[props.data],
                  snapshotMonitoring: e.target.checked,
                },
              })
            }
            className="w-4 h-4 cursor-pointer"
          />
          <h1 class="cursor-pointer" onClick={() => onNavigate && onNavigate('snapshots', { url: props.data })}>
             View Snapshots
          </h1>
        </div>
      ),
      sortable: false,
      filter: false,
    },
 
  ], [selectedWebsites, onNavigate])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-lg text-gray-600">Loading dashboard...</p>
      </div>
    )
  }

  if (!dashboardData) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-lg text-red-600">Failed to load dashboard data</p>
      </div>
    )
  }

  const { connection_status = {}, daemon_status = {}, website_stats = [] } = dashboardData

  // Verify required fields exist before rendering
  if (!connection_status || !daemon_status) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-lg text-red-600">Dashboard data is incomplete</p>
      </div>
    )
  }

  const statusColor =
    daemon_status.state === 'running' ? 'text-green-600' : 'text-red-600'

  return (
    <div className="min-h-screen bg-gray-50">
      {isStopping && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl p-8 text-center">
            <div className="mb-4">
              <FontAwesomeIcon icon={faSyncAlt} spin size="3x" className="text-blue-600" />
            </div>
            <h2 className="text-2xl font-bold text-gray-900 mb-2">Stopping Monitoring</h2>
            <p className="text-gray-600">Waiting for running jobs to complete...</p>
          </div>
        </div>
      )}

      <div className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex justify-between items-start">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">
                Monitoring Dashboard
              </h1>
              <p className="text-sm text-gray-600 mt-1">
                Connected to:{' '}
                {connection_status.is_local
                  ? 'Local Daemon (localhost:9876)'
                  : connection_status.host}
              </p>
            </div>
            <div className="flex gap-2">
              <button
                onClick={handleRefresh}
                disabled={refreshing}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded-lg font-semibold text-sm flex items-center gap-2"
              >
                <FontAwesomeIcon icon={faSyncAlt} spin={refreshing} />
                {refreshing ? 'Refreshing...' : 'Refresh'}
              </button>
              <label className="flex items-center gap-2 px-3 py-2 bg-gray-200 hover:bg-gray-300 rounded-lg cursor-pointer">
                <input
                  type="checkbox"
                  checked={autoRefreshEnabled}
                  onChange={(e) => setAutoRefreshEnabled(e.target.checked)}
                  className="w-4 h-4"
                />
                <span className="text-sm font-semibold text-gray-800">Auto (1m)</span>
              </label>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-6 py-6">
        <div className="mb-8 bg-white rounded-lg shadow p-6">
          <h2 className="text-xl font-bold text-gray-900 mb-4">Daemon Status</h2>
          <div className="flex items-center gap-4">
            <span className="text-gray-700 font-medium">State:</span>
            <span className={`text-lg font-semibold ${statusColor}`}>
              {daemon_status.state}
            </span>
          </div>
          {!daemon_status.has_smtp && (
            <div className="mt-4 p-4 bg-yellow-50 border-l-4 border-yellow-400 rounded">
              <div className="flex justify-between items-center">
                <span className="text-yellow-800 flex items-center gap-2">
                  <FontAwesomeIcon icon={faExclamationTriangle} />
                  SMTP not configured - email alerts disabled
                </span>
              </div>
            </div>
          )}
        </div>

        {showStartMonitor && (
          <div className="mt-8 mb-8 bg-white rounded-lg shadow overflow-hidden flex flex-col">
            <div className="p-6 border-b border-gray-200 flex justify-between items-center">
              <h2 className="text-xl font-bold text-gray-900">
                Start Monitor
              </h2>
              <button
                onClick={handleStartMonitoring}
                disabled={!Object.values(selectedWebsites).some(w => w?.enabled)}
                className="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-400 text-white rounded-lg font-semibold"
              >
                Start Monitoring
              </button>
            </div>
            {allAvailableWebsites && allAvailableWebsites.length > 0 ? (
              <div style={{ height: '400px', width: '100%' }}>
                <AgGridReact
                  theme={themeQuartz}
                  rowData={allAvailableWebsites}
                  columnDefs={getStartMonitorColumnDefs()}
                  defaultColDef={{
                    resizable: true,
                    sortable: true,
                    filter: true,
                  }}
                  pagination={true}
                  paginationPageSize={10}
                  paginationPageSizeSelector={[10, 20, 50, 100]}
                  style={{ width: '100%', height: '100%' }}
                />
              </div>
            ) : (
              <div className="p-8 text-center text-gray-500">
                No websites available. Please add websites in the Websites section first.
              </div>
            )}
          </div>
        )}

        {isMonitoringActive && (
        <div className="mt-8 grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <div className="bg-white rounded-lg shadow overflow-hidden flex flex-col">
              <div className="p-6 border-b border-gray-200 flex justify-between items-center">
                <h2 className="text-xl font-bold text-gray-900">
                  Website Monitoring
                </h2>
                {isMonitoringActive && (
                  <button
                    onClick={handleStopMonitoring}
                    disabled={isStopping}
                    className="px-3 py-1 bg-red-600 hover:bg-red-700 disabled:bg-gray-400 text-white rounded text-sm font-semibold"
                  >
                    {isStopping ? 'Stopping...' : 'Stop Monitoring'}
                  </button>
                )}
              </div>
              {isMonitoringActive && website_stats && website_stats.length > 0 ? (
                <div className="p-6 grid grid-cols-1 md:grid-cols-2 gap-4 overflow-y-auto" style={{ maxHeight: '500px' }}>
                  {website_stats.filter(site => monitoredWebsites.includes(site.url)).map((site, idx) => (
                    <div key={idx} className="border border-gray-200 rounded-lg p-4 hover:shadow-lg transition-shadow">
                      {/* URL */}
                      <div className="mb-3">
                        <a
                          href={site.url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-blue-600 hover:underline font-semibold truncate block"
                          title={site.url}
                        >
                          {site.url}
                        </a>
                      </div>

                      {/* Status Badge */}
                      <div className="flex items-center gap-2 mb-3">
                        <div
                          className={`w-3 h-3 rounded-full ${
                            site.current_status === 'Up'
                              ? 'bg-green-500 animate-pulse'
                              : 'bg-red-500 animate-pulse'
                          }`}
                        />
                        <span
                          className={`px-2 py-1 rounded text-sm font-semibold ${
                            site.current_status === 'Up'
                              ? 'bg-green-100 text-green-800'
                              : 'bg-red-100 text-red-800'
                          }`}
                        >
                          {site.current_status === 'Up' ? '✓ UP' : '✗ DOWN'}
                        </span>
                      </div>

                      {/* Stats Grid */}
                      <div className="space-y-2 text-sm">
                        {/* Uptime */}
                        <div className="flex justify-between items-center">
                          <span className="text-gray-600">Uptime (7d)</span>
                          <div className="flex items-center gap-2">
                            <span className="font-semibold">{(site.uptime_last_7_days || 0).toFixed(1)}%</span>
                            <div className="w-16 bg-gray-200 rounded-full h-2">
                              <div
                                className={`h-2 rounded-full ${
                                  (site.uptime_last_7_days || 0) >= 99
                                    ? 'bg-green-500'
                                    : (site.uptime_last_7_days || 0) >= 95
                                    ? 'bg-yellow-500'
                                    : 'bg-red-500'
                                }`}
                                style={{
                                  width: `${Math.min((site.uptime_last_7_days || 0), 100)}%`,
                                }}
                              />
                            </div>
                          </div>
                        </div>

                        {/* Response Time */}
                        <div className="flex justify-between">
                          <span className="text-gray-600">Avg Response</span>
                          <span className="font-semibold">{site.average_response_time || 'N/A'}</span>
                        </div>

                        {/* Checks */}
                        <div className="flex justify-between">
                          <span className="text-gray-600">Total Checks</span>
                          <span className="font-semibold">{site.total_checks}</span>
                        </div>

                        {/* Failed Checks */}
                        <div className="flex justify-between">
                          <span className="text-gray-600">Failed</span>
                          <span className={`font-semibold ${site.failed_checks > 0 ? 'text-red-600' : 'text-green-600'}`}>
                            {site.failed_checks}
                          </span>
                        </div>

                        {/* Last Check */}
                        <div className="flex justify-between text-xs text-gray-500">
                          <span>Last Check</span>
                          <span>{site.last_check_time || 'Never'}</span>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="p-8 text-center text-gray-500">
                  {isMonitoringActive ? 'No websites being monitored' : 'Select websites below to start monitoring'}
                </div>
              )}
            </div>
          </div>

          <div>
            <div className="bg-white rounded-lg shadow h-full max-h-96 overflow-hidden flex flex-col">
              <div className="p-6 border-b border-gray-200 flex justify-between items-center">
                <h2 className="text-xl font-bold text-gray-900">Activity Log</h2>
                <div className="flex gap-2">
                  <button
                    onClick={loadLogs}
                    disabled={logsLoading}
                    className="text-blue-600 hover:text-blue-800 disabled:text-gray-400"
                    title="Refresh logs"
                  >
                    <FontAwesomeIcon icon={faSyncAlt} spin={logsLoading} size="lg" />
                  </button>
                  <button
                    onClick={handleClearLogs}
                    disabled={logs.length === 0}
                    className="text-red-600 hover:text-red-800 disabled:text-gray-400"
                    title="Clear logs"
                  >
                    <FontAwesomeIcon icon={faTrash} size="lg" />
                  </button>
                </div>
              </div>
              <div className="flex-1 overflow-y-auto">
                {logs && logs.length > 0 ? (
                  <div className="divide-y divide-gray-200">
                    {[...logs].reverse().map((log, idx) => {
                      // Parse timestamp from format: [YYYY/MM/DD HH:MM:SS] message
                      const timestampMatch = log.match(/^\[\d{4}\/\d{2}\/\d{2}\s\d{2}:\d{2}:\d{2}\]/)
                      let timestamp = ''
                      let logText = log
                      if (timestampMatch) {
                        // Extract timestamp without brackets
                        timestamp = timestampMatch[0].slice(1, -1) // Remove [ and ]
                        logText = log.substring(timestampMatch[0].length).trim()
                      }
                      return (
                        <div key={idx} className="p-3 text-xs text-gray-600 font-mono hover:bg-gray-50">
                          {timestamp && <p className="text-gray-400 text-xs mb-1">{timestamp}</p>}
                          <p className="break-words text-gray-700">{logText}</p>
                        </div>
                      )
                    })}
                  </div>
                ) : (
                  <div className="p-4 text-center text-gray-400 text-sm">
                    {logsLoading ? 'Loading logs...' : 'No activity logs'}
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
        )}
      </div>
    </div>
  )
}

export default DashboardScreen
