import React, { useState, useEffect } from 'react'
import api from './api'
import ConnectionScreen from './screens/ConnectionScreen'
import DashboardScreen from './screens/DashboardScreen'
import WebsitesScreen from './screens/WebsitesScreen'
import SnapshotScreen from './screens/SnapshotScreen'
import SnapshotRecordingScreen from './screens/SnapshotRecordingScreen'
import SettingsScreen from './screens/SettingsScreen'
import DaemonSetupScreen from './screens/DaemonSetupScreen'
import DashboardLayout from './layouts/DashboardLayout'

function App() {
  const [currentScreen, setCurrentScreen] = useState('connection')
  const [screenParams, setScreenParams] = useState({})
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    // Check if already connected on startup
    checkConnection()
  }, [])

  const checkConnection = async () => {
    try {
      const status = await api.getConnectionStatus()
      setIsConnected(status.connected)
      if (status.connected) {
        setCurrentScreen('dashboard')
      }
    } catch (err) {
      // Failed to check connection - this is expected if daemon isn't running
      // Only log at info level since this is normal startup behavior
      console.debug('Connection check failed (expected if daemon not running):', err.message)
      setIsConnected(false)
    }
  }

  const handleConnect = async (connectionType, connectionData) => {
    setLoading(true)
    setError(null)
    try {
      if (connectionType === 'local') {
        await api.startLocalDaemon()
      } else if (connectionType === 'ssh') {
        await api.connectToServer(
          connectionData.host,
          connectionData.username,
          connectionData.password
        )
      }
      setIsConnected(true)
      setCurrentScreen('dashboard')
    } catch (err) {
      setError(err.message || 'Connection failed')
      console.error('Connection error:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleDisconnect = async () => {
    try {
      await api.disconnectFromServer()
      setIsConnected(false)
      setCurrentScreen('connection')
    } catch (err) {
      setError(err.message || 'Disconnection failed')
    }
  }

  const handleNavigate = (screen, params = {}) => {
    setScreenParams(params)
    setCurrentScreen(screen)
  }

  const renderMainContent = () => {
    switch (currentScreen) {
      case 'dashboard':
        return (
          <DashboardScreen
            error={error}
            setError={setError}
            onNavigate={handleNavigate}
          />
        )
      case 'websites':
        return (
          <WebsitesScreen
            error={error}
            setError={setError}
            onNavigate={handleNavigate}
          />
        )
      case 'snapshots':
        return (
          <SnapshotScreen
            onNavigate={handleNavigate}
            error={error}
            setError={setError}
            screenParams={screenParams}
          />
        )
      case 'settings':
        return (
          <SettingsScreen
            onNavigate={handleNavigate}
            error={error}
            setError={setError}
          />
        )
      case 'snapshot-recording':
        return (
          <SnapshotRecordingScreen
            onBack={() => handleNavigate('snapshots')}
            url={screenParams.url}
          />
        )
      default:
        return (
          <div className="flex items-center justify-center h-screen">
            <p className="text-lg text-gray-600">Unknown screen: {currentScreen}</p>
          </div>
        )
    }
  }

  const renderScreen = () => {
    switch (currentScreen) {
      case 'connection':
        return (
          <ConnectionScreen
            onConnect={handleConnect}
            loading={loading}
            error={error}
          />
        )
      case 'daemon-setup':
        return (
          <DaemonSetupScreen
            onBack={() => handleNavigate('connection')}
            onComplete={() => handleNavigate('connection')}
          />
        )
      default:
        // All other screens use the dashboard layout with sidebar
        return (
          <DashboardLayout
            currentScreen={currentScreen}
            onNavigate={handleNavigate}
            onDisconnect={handleDisconnect}
            error={error}
            setError={setError}
            screenParams={screenParams}
          >
            {renderMainContent()}
          </DashboardLayout>
        )
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {renderScreen()}
    </div>
  )
}

export default App
