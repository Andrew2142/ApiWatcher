import React, { useState } from 'react'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faCheck, faTimes } from '@fortawesome/free-solid-svg-icons'
import api from '../api'

function DaemonSetupScreen({ onBack, onComplete }) {
  const [step, setStep] = useState(1)
  const [host, setHost] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [checking, setChecking] = useState(false)
  const [daemonStatus, setDaemonStatus] = useState(null)
  const [error, setError] = useState('')

  const handleCheckDaemon = async () => {
    if (!host || !username) {
      setError('Please fill in host and username')
      return
    }

    setChecking(true)
    setError('')

    try {
      const status = await api.checkDaemonStatus(host, username, password)
      setDaemonStatus(status)

      if (status.error) {
        setError(status.error)
      } else if (status.daemon_installed && status.daemon_running) {
        // Daemon is ready, complete setup
        setStep(3)
      } else if (status.daemon_installed && !status.daemon_running) {
        setStep(2)
        setError('Daemon is installed but not running. Please start it on the server.')
      } else {
        setStep(2)
        setError('Daemon is not installed. Please install it first.')
      }
    } catch (err) {
      setError(`Failed to check daemon status: ${err.message}`)
    } finally {
      setChecking(false)
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-lg max-w-md w-full p-8">
        <h1 className="text-2xl font-bold text-gray-800 mb-2">
          Daemon Setup Wizard
        </h1>
        <p className="text-gray-600 mb-6">Step {step} of 3</p>

        {/* Step 1: Server Connection */}
        {step === 1 && (
          <div>
            <div className="mb-6 p-4 bg-blue-50 rounded-lg">
              <h2 className="text-lg font-semibold text-blue-900 mb-2">
                Step 1: Connect to Server
              </h2>
              <p className="text-sm text-blue-800">
                Enter the server details where you want to check/install the daemon
              </p>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Host / IP Address
                </label>
                <input
                  type="text"
                  placeholder="example.com"
                  value={host}
                  onChange={(e) => setHost(e.target.value)}
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
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
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
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              {error && (
                <div className="p-3 bg-red-100 text-red-800 rounded-lg text-sm">
                  {error}
                </div>
              )}
            </div>

            <div className="flex gap-2 mt-6">
              <button
                onClick={() => {
                  setStep(1)
                  setHost('')
                  setUsername('')
                  setPassword('')
                  setError('')
                  setDaemonStatus(null)
                  onBack()
                }}
                className="flex-1 bg-gray-200 hover:bg-gray-300 text-gray-800 font-semibold py-2 px-4 rounded-lg transition"
              >
                Back
              </button>
              <button
                onClick={handleCheckDaemon}
                disabled={checking || !host || !username}
                className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-semibold py-2 px-4 rounded-lg transition"
              >
                {checking ? 'Checking...' : 'Check Daemon'}
              </button>
            </div>
          </div>
        )}

        {/* Step 2: Daemon Status */}
        {step === 2 && daemonStatus && (
          <div>
            <div className="mb-6 p-4 bg-yellow-50 rounded-lg">
              <h2 className="text-lg font-semibold text-yellow-900 mb-2">
                Step 2: Daemon Status
              </h2>
              <p className="text-sm text-yellow-800">
                {daemonStatus.daemon_installed
                  ? 'Daemon is installed but needs to be running'
                  : 'Daemon needs to be installed on this server'}
              </p>
            </div>

            <div className="space-y-4 mb-6">
              <div className="p-4 bg-gray-50 rounded-lg">
                <p className="text-sm text-gray-600 mb-1">Daemon Installed:</p>
                <p className="text-lg font-semibold text-gray-900 flex items-center gap-2">
                  <FontAwesomeIcon icon={daemonStatus.daemon_installed ? faCheck : faTimes} style={{color: daemonStatus.daemon_installed ? '#16a34a' : '#dc2626'}} />
                  {daemonStatus.daemon_installed ? 'Yes' : 'No'}
                </p>
              </div>

              <div className="p-4 bg-gray-50 rounded-lg">
                <p className="text-sm text-gray-600 mb-1">Daemon Running:</p>
                <p className="text-lg font-semibold text-gray-900 flex items-center gap-2">
                  <FontAwesomeIcon icon={daemonStatus.daemon_running ? faCheck : faTimes} style={{color: daemonStatus.daemon_running ? '#16a34a' : '#dc2626'}} />
                  {daemonStatus.daemon_running ? 'Yes' : 'No'}
                </p>
              </div>

              {daemonStatus.error && (
                <div className="p-3 bg-red-100 text-red-800 rounded-lg text-sm">
                  {daemonStatus.error}
                </div>
              )}
            </div>

            <div className="mb-6 p-4 bg-blue-50 rounded-lg">
              <p className="text-sm text-blue-800 mb-3">
                <strong>To install/start daemon:</strong>
              </p>
              <ol className="text-sm text-blue-800 list-decimal list-inside space-y-1">
                <li>SSH into your server</li>
                <li>Run: <code className="bg-blue-100 px-1 rounded">go build -o apiwatcher-daemon ./cmd/apiwatcher-daemon</code></li>
                <li>Run: <code className="bg-blue-100 px-1 rounded">mkdir -p ~/.apiwatcher/bin && mv apiwatcher-daemon ~/.apiwatcher/bin/</code></li>
                <li>Run: <code className="bg-blue-100 px-1 rounded">~/.apiwatcher/bin/apiwatcher-daemon &</code></li>
              </ol>
            </div>

            <div className="flex gap-2">
              <button
                onClick={() => {
                  setStep(1)
                  setError('')
                  setDaemonStatus(null)
                }}
                className="flex-1 bg-gray-200 hover:bg-gray-300 text-gray-800 font-semibold py-2 px-4 rounded-lg transition"
              >
                Back
              </button>
              <button
                onClick={handleCheckDaemon}
                disabled={checking}
                className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-semibold py-2 px-4 rounded-lg transition"
              >
                {checking ? 'Checking...' : 'Check Again'}
              </button>
            </div>
          </div>
        )}

        {/* Step 3: Complete */}
        {step === 3 && (
          <div>
            <div className="mb-6 p-4 bg-green-50 rounded-lg">
              <h2 className="text-lg font-semibold text-green-900 mb-2 flex items-center gap-2">
                <FontAwesomeIcon icon={faCheck} />
                Setup Complete!
              </h2>
              <p className="text-sm text-green-800">
                Your daemon is installed and running successfully
              </p>
            </div>

            <div className="p-4 bg-green-50 rounded-lg mb-6">
              <div className="text-center">
                <p className="text-4xl mb-2"><FontAwesomeIcon icon={faCheck} style={{color: '#16a34a'}} /></p>
                <p className="text-green-900 font-semibold">Ready to Monitor</p>
              </div>
            </div>

            <button
              onClick={onComplete}
              className="w-full bg-green-600 hover:bg-green-700 text-white font-semibold py-2 px-4 rounded-lg transition"
            >
              Continue to Dashboard
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

export default DaemonSetupScreen
