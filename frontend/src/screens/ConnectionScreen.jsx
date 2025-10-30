import React, { useState, useEffect } from 'react'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faCheck, faTimes, faCircle } from '@fortawesome/free-solid-svg-icons'
import api from '../api'

function ConnectionScreen({ onConnect, loading, error }) {
  const [profiles, setProfiles] = useState([])
  const [selectedProfile, setSelectedProfile] = useState(null)
  const [showNewServerForm, setShowNewServerForm] = useState(false)
  const [newServer, setNewServer] = useState({
    name: '',
    host: '',
    username: '',
    password: '',
    port: 22,
  })
  const [lastServer, setLastServer] = useState('')
  const [testingConnection, setTestingConnection] = useState(false)
  const [testMessage, setTestMessage] = useState('')
  const [testSuccess, setTestSuccess] = useState(null)
  const [passwordPrompt, setPasswordPrompt] = useState('')
  const [showPasswordDialog, setShowPasswordDialog] = useState(false)
  const [deploymentStatus, setDeploymentStatus] = useState(null)
  const [daemonCheckInProgress, setDaemonCheckInProgress] = useState(false)
  const [pendingConnection, setPendingConnection] = useState(null)

  useEffect(() => {
    loadProfiles()
  }, [])

  useEffect(() => {
    if (profiles.length > 0) {
      loadLastServer()
    }
  }, [profiles])

  const loadProfiles = async () => {
    try {
      const profs = await api.listSSHProfiles()
      setProfiles(profs || [])
      if (profs && profs.length > 0) {
        setSelectedProfile(profs[0])
      }
    } catch (err) {
      console.error('Failed to load profiles:', err)
    }
  }

  const loadLastServer = async () => {
    try {
      const server = await api.getLastConnectedServer()
      setLastServer(server)
      if (server && profiles.length > 0) {
        const prof = profiles.find(p => p.name === server)
        if (prof) {
          setSelectedProfile(prof)
        }
      }
    } catch (err) {
      console.error('Failed to load last server:', err)
    }
  }

  const handleConnectToSelected = () => {
    if (!selectedProfile) return
    setShowPasswordDialog(true)
  }

  const handlePasswordConfirm = () => {
    if (!selectedProfile) return
    setShowPasswordDialog(false)
    handleConnectionAttempt(
      selectedProfile.host,
      selectedProfile.username,
      passwordPrompt
    )
    setPasswordPrompt('')
  }

  const handleAddNewServer = () => {
    if (!newServer.name || !newServer.host || !newServer.username) {
      alert('Please fill in all fields')
      return
    }
    handleConnectionAttempt(
      newServer.host,
      newServer.username,
      newServer.password
    )
  }

  const handleLocalConnect = () => {
    onConnect('local', {})
  }

  const handleTestConnection = async () => {
    if (!newServer.host || !newServer.username) {
      setTestMessage('Please fill in host and username')
      setTestSuccess(false)
      return
    }

    setTestingConnection(true)
    setTestMessage('')
    setTestSuccess(null)

    try {
      await api.testConnection(
        newServer.host,
        newServer.username,
        newServer.password
      )
      setTestMessage('Connection successful!')
      setTestSuccess(true)
    } catch (err) {
      setTestMessage(`Connection failed: ${err.message}`)
      setTestSuccess(false)
    } finally {
      setTestingConnection(false)
    }
  }

  const handleConnectionAttempt = async (host, username, password) => {
    setPendingConnection({ host, username, password })
    setDaemonCheckInProgress(true)

    try {
      // Check daemon status
      const status = await api.checkDaemonStatus(host, username, password)

      if (status.error) {
        // If there's an SSH error, just connect - error handling will happen at the connection level
        onConnect('ssh', { host, username, password })
        setPendingConnection(null)
        setDaemonCheckInProgress(false)
      } else if (status.daemon_running) {
        // Daemon is running, proceed to connect
        onConnect('ssh', { host, username, password })
        setPendingConnection(null)
        setDaemonCheckInProgress(false)
      } else if (status.daemon_installed && !status.daemon_running) {
        // Daemon is installed but not running - show error
        setDeploymentStatus({
          state: 'error',
          message: 'Daemon is installed but not running on the remote server. Please check the daemon.'
        })
        setPendingConnection(null)
        setDaemonCheckInProgress(false)
      } else {
        // Daemon not installed - offer to deploy
        setDeploymentStatus({
          state: 'prompt',
          message: 'Daemon is not installed on the remote server. Deploy it now?'
        })
        setDaemonCheckInProgress(false)
      }
    } catch (err) {
      setDeploymentStatus({
        state: 'error',
        message: `Failed to check daemon: ${err.message}`
      })
      setPendingConnection(null)
      setDaemonCheckInProgress(false)
    }
  }

  const handleDeployDaemon = async () => {
    if (!pendingConnection) return

    setDeploymentStatus({
      state: 'deploying',
      message: 'Deploying daemon to remote server...'
    })

    try {
      const result = await api.deployDaemonToServer(
        pendingConnection.host,
        pendingConnection.username,
        pendingConnection.password
      )

      if (result.error) {
        setDeploymentStatus({
          state: 'error',
          message: `Deployment failed: ${result.error}`
        })
      } else if (result.daemon_running) {
        setDeploymentStatus({
          state: 'success',
          message: 'Daemon deployed successfully! Connecting...'
        })
        // Wait a moment then connect
        setTimeout(() => {
          onConnect('ssh', {
            host: pendingConnection.host,
            username: pendingConnection.username,
            password: pendingConnection.password
          })
          setDeploymentStatus(null)
          setPendingConnection(null)
        }, 1500)
      }
    } catch (err) {
      setDeploymentStatus({
        state: 'error',
        message: `Deployment error: ${err.message}`
      })
    }
  }

  const cancelDeployment = () => {
    setDeploymentStatus(null)
    setPendingConnection(null)
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-lg max-w-md w-full p-8">
        <h1 className="text-3xl font-bold text-center text-gray-800 mb-8">
          API Watcher
        </h1>
        <p className="text-center text-gray-600 mb-6">Server Connection</p>

        {deploymentStatus && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg shadow-lg p-6 max-w-sm w-full">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">
                {deploymentStatus.state === 'deploying' && 'Deploying Daemon'}
                {deploymentStatus.state === 'prompt' && 'Deploy Daemon'}
                {deploymentStatus.state === 'success' && 'Deployment Successful'}
                {deploymentStatus.state === 'error' && 'Deployment Error'}
              </h2>
              <div className="flex items-center justify-center mb-4">
                {deploymentStatus.state === 'deploying' && (
                  <FontAwesomeIcon
                    icon={faCircle}
                    className="animate-spin text-blue-600"
                    size="lg"
                  />
                )}
                {deploymentStatus.state === 'success' && (
                  <FontAwesomeIcon
                    icon={faCheck}
                    className="text-green-600"
                    size="lg"
                  />
                )}
                {deploymentStatus.state === 'error' && (
                  <FontAwesomeIcon
                    icon={faTimes}
                    className="text-red-600"
                    size="lg"
                  />
                )}
              </div>
              <p className="text-sm text-gray-600 text-center mb-4">
                {deploymentStatus.message}
              </p>
              {deploymentStatus.state === 'prompt' && (
                <div className="flex gap-2">
                  <button
                    onClick={cancelDeployment}
                    className="flex-1 bg-gray-200 hover:bg-gray-300 text-gray-800 font-semibold py-2 px-4 rounded-lg transition"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleDeployDaemon}
                    disabled={daemonCheckInProgress}
                    className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-semibold py-2 px-4 rounded-lg transition"
                  >
                    Deploy
                  </button>
                </div>
              )}
              {deploymentStatus.state === 'error' && (
                <div className="flex gap-2">
                  <button
                    onClick={cancelDeployment}
                    className="w-full bg-gray-200 hover:bg-gray-300 text-gray-800 font-semibold py-2 px-4 rounded-lg transition"
                  >
                    Close
                  </button>
                </div>
              )}
            </div>
          </div>
        )}

        {!showNewServerForm && !showPasswordDialog && !deploymentStatus && (
          <div className="mb-6">
            {profiles.length > 0 && (
              <>
                <h2 className="text-lg font-semibold text-gray-700 mb-3">
                  Saved Servers
                </h2>
                <div className="space-y-2 mb-4">
                  {profiles.map(profile => (
                    <label
                      key={profile.name}
                      className="flex items-center p-3 border border-gray-200 rounded-lg cursor-pointer hover:bg-blue-50 transition"
                    >
                      <input
                        type="radio"
                        name="profile"
                        checked={selectedProfile?.name === profile.name}
                        onChange={() => setSelectedProfile(profile)}
                        className="w-4 h-4"
                      />
                      <span className="ml-3 text-sm text-gray-700">
                        {profile.name} ({profile.username}@{profile.host})
                      </span>
                    </label>
                  ))}
                </div>
                <button
                  onClick={handleConnectToSelected}
                  disabled={loading || !selectedProfile}
                  className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-semibold py-2 px-4 rounded-lg transition mb-2"
                >
                  {loading ? 'Connecting...' : 'Connect to Selected Server'}
                </button>
              </>
            )}
            <button
              onClick={() => setShowNewServerForm(true)}
              className="w-full bg-gray-200 hover:bg-gray-300 text-gray-800 font-semibold py-2 px-4 rounded-lg transition"
            >
              {profiles.length > 0 ? 'Add New Server' : 'Connect to SSH Server'}
            </button>
          </div>
        )}

        {showPasswordDialog && !daemonCheckInProgress && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg shadow-lg p-6 max-w-sm w-full">
              <h2 className="text-lg font-semibold text-gray-900 mb-4">
                Password Required
              </h2>
              <p className="text-sm text-gray-600 mb-4">
                Enter the SSH password for {selectedProfile?.username}@{selectedProfile?.host}
              </p>
              <input
                type="password"
                placeholder="SSH Password"
                value={passwordPrompt}
                onChange={(e) => setPasswordPrompt(e.target.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter') handlePasswordConfirm()
                }}
                autoFocus
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 mb-4"
              />
              <div className="flex gap-2">
                <button
                  onClick={() => {
                    setShowPasswordDialog(false)
                    setPasswordPrompt('')
                  }}
                  className="flex-1 bg-gray-200 hover:bg-gray-300 text-gray-800 font-semibold py-2 px-4 rounded-lg transition"
                >
                  Cancel
                </button>
                <button
                  onClick={handlePasswordConfirm}
                  disabled={loading || !passwordPrompt}
                  className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-semibold py-2 px-4 rounded-lg transition"
                >
                  {loading ? 'Connecting...' : 'Connect'}
                </button>
              </div>
            </div>
          </div>
        )}

        {showNewServerForm && (
          <div className="mb-6">
            <h2 className="text-lg font-semibold text-gray-700 mb-4">
              Add New Server
            </h2>
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Server Name
                </label>
                <input
                  type="text"
                  placeholder="My Server"
                  value={newServer.name}
                  onChange={e =>
                    setNewServer({ ...newServer, name: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Host
                </label>
                <input
                  type="text"
                  placeholder="example.com"
                  value={newServer.host}
                  onChange={e =>
                    setNewServer({ ...newServer, host: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Username
                </label>
                <input
                  type="text"
                  placeholder="user"
                  value={newServer.username}
                  onChange={e =>
                    setNewServer({ ...newServer, username: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Password
                </label>
                <input
                  type="password"
                  placeholder="password"
                  value={newServer.password}
                  onChange={e =>
                    setNewServer({ ...newServer, password: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Port
                </label>
                <input
                  type="number"
                  value={newServer.port}
                  onChange={e =>
                    setNewServer({
                      ...newServer,
                      port: parseInt(e.target.value),
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
            <div className="flex gap-2 mt-4">
              <button
                onClick={handleTestConnection}
                disabled={testingConnection || loading}
                className="flex-1 bg-gray-600 hover:bg-gray-700 disabled:bg-gray-400 text-white font-semibold py-2 px-4 rounded-lg transition"
              >
                {testingConnection ? 'Testing...' : 'Test Connection'}
              </button>
              <button
                onClick={handleAddNewServer}
                disabled={loading}
                className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-semibold py-2 px-4 rounded-lg transition"
              >
                {loading ? 'Connecting...' : 'Connect'}
              </button>
            </div>
            {testMessage && (
              <div
                className={`mt-3 p-3 rounded-lg text-sm flex items-center gap-2 ${
                  testSuccess
                    ? 'bg-green-100 text-green-800'
                    : 'bg-red-100 text-red-800'
                }`}
              >
                <FontAwesomeIcon icon={testSuccess ? faCheck : faTimes} />
                {testMessage}
              </div>
            )}
            <button
              onClick={() => setShowNewServerForm(false)}
              className="w-full mt-2 bg-gray-200 hover:bg-gray-300 text-gray-800 font-semibold py-2 px-4 rounded-lg transition"
            >
              Cancel
            </button>
          </div>
        )}

        <div className="border-t border-gray-200 pt-6">
          <h2 className="text-lg font-semibold text-gray-700 mb-2">
            Or run locally
          </h2>
          <p className="text-sm text-gray-600 mb-4">
            Connect to a daemon running on this machine
          </p>
          <button
            onClick={handleLocalConnect}
            disabled={loading}
            className="w-full bg-green-600 hover:bg-green-700 disabled:bg-gray-400 text-white font-semibold py-2 px-4 rounded-lg transition"
          >
            {loading ? 'Connecting...' : 'Run Locally'}
          </button>
        </div>
      </div>
    </div>
  )
}

export default ConnectionScreen
