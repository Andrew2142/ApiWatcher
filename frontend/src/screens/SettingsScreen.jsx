import React, { useState, useEffect } from 'react'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faCheck, faExclamationTriangle, faCog, faEnvelope, faBell } from '@fortawesome/free-solid-svg-icons'
import api from '../api'

function SettingsScreen({ onNavigate, error, setError }) {
  const [activeTab, setActiveTab] = useState('general')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  // General Settings
  const [workerSleepTime, setWorkerSleepTime] = useState(10)
  const [headlessBrowserMode, setHeadlessBrowserMode] = useState(false)
  const [generalSaved, setGeneralSaved] = useState(false)

  // SMTP Settings
  const [smtpConfig, setSMTPConfig] = useState({ host: '', port: 587, username: '', password: '', from: '', to: '' })
  const [smtpStatus, setSMTPStatus] = useState(null)
  const [smtpSaved, setSMTPSaved] = useState(false)

  // Notifications (placeholder)
  const [notificationsEnabled, setNotificationsEnabled] = useState(true)

  useEffect(() => {
    loadAllSettings()
  }, [])

  const loadAllSettings = async () => {
    try {
      setLoading(true)
      const appSettings = await api.getAppSettings()
      if (appSettings && appSettings.worker_sleep_time) {
        setWorkerSleepTime(appSettings.worker_sleep_time)
      }
      if (appSettings && appSettings.headless_browser_mode !== undefined) {
        setHeadlessBrowserMode(appSettings.headless_browser_mode)
      }
      await loadSMTPStatus()
      setGeneralSaved(false)
      setSMTPSaved(false)
    } catch (err) {
      setError(err.message || 'Failed to load settings')
    } finally {
      setLoading(false)
    }
  }

  const loadSMTPStatus = async () => {
    try {
      const [status, config] = await Promise.all([
        api.getSMTPStatus(),
        api.getSMTPConfig().catch(() => null)
      ])
      setSMTPStatus(status)
      if (config) {
        setSMTPConfig({
          host: config.host || '',
          port: parseInt(config.port) || 587,
          username: config.username || '',
          from: config.from || '',
          to: config.to || '',
          password: ''
        })
      }
    } catch (err) {
      console.error('Failed to load SMTP status:', err)
    }
  }

  const validateEmail = (email) => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    return emailRegex.test(email)
  }

  const handleSaveGeneral = async () => {
    if (workerSleepTime < 1 || workerSleepTime > 1440) {
      alert('Worker sleep time must be between 1 and 1440 minutes')
      return
    }
    try {
      setSaving(true)
      await api.saveAppSettings(workerSleepTime, headlessBrowserMode)
      setGeneralSaved(true)
      setError(null)
      setTimeout(() => setGeneralSaved(false), 2000)
    } catch (err) {
      setError(err.message || 'Failed to save general settings')
    } finally {
      setSaving(false)
    }
  }

  const handleSaveSMTP = async () => {
    if (!smtpConfig.host.trim()) {
      alert('Please enter SMTP host')
      return
    }
    if (!smtpConfig.username.trim()) {
      alert('Please enter username')
      return
    }
    if (!smtpConfig.password.trim()) {
      alert('Please enter password')
      return
    }
    if (!smtpConfig.from.trim()) {
      alert('Please enter from email address')
      return
    }
    if (!validateEmail(smtpConfig.from)) {
      alert('Please enter a valid from email address')
      return
    }
    if (!smtpConfig.to.trim()) {
      alert('Please enter alert email address')
      return
    }
    if (!validateEmail(smtpConfig.to)) {
      alert('Please enter a valid alert email address')
      return
    }
    if (smtpConfig.port < 1 || smtpConfig.port > 65535) {
      alert('Port must be between 1 and 65535')
      return
    }
    try {
      setSaving(true)
      await api.configureSMTP(smtpConfig.host, smtpConfig.port, smtpConfig.username, smtpConfig.password, smtpConfig.from, smtpConfig.to)
      setSMTPStatus({ configured: true })
      setSMTPSaved(true)
      setError(null)
      setTimeout(() => setSMTPSaved(false), 2000)
    } catch (err) {
      setError(err.message || 'Failed to save SMTP configuration')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-lg text-gray-600">Loading settings...</p>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="bg-white border-b border-gray-200 p-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Settings</h1>
          <p className="text-sm text-gray-600 mt-1">Configure your monitoring application settings</p>
        </div>
      </div>

      <div className="max-w-4xl mx-auto px-6 py-8">
        {/* Tabs */}
        <div className="flex gap-4 mb-8 border-b border-gray-200">
          <button
            onClick={() => setActiveTab('general')}
            className={`flex items-center gap-2 px-4 py-3 font-semibold border-b-2 transition ${activeTab === 'general'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-600 hover:text-gray-900'
              }`}
          >
            <FontAwesomeIcon icon={faCog} />
            General Settings
          </button>
          <button
            onClick={() => setActiveTab('smtp')}
            className={`flex items-center gap-2 px-4 py-3 font-semibold border-b-2 transition ${activeTab === 'smtp'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-600 hover:text-gray-900'
              }`}
          >
            <FontAwesomeIcon icon={faEnvelope} />
            Email (SMTP)
          </button>
          <button
            onClick={() => setActiveTab('notifications')}
            className={`flex items-center gap-2 px-4 py-3 font-semibold border-b-2 transition ${activeTab === 'notifications'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-600 hover:text-gray-900'
              }`}
          >
            <FontAwesomeIcon icon={faBell} />
            Notifications
          </button>
        </div>

        {/* General Settings Tab */}
        {activeTab === 'general' && (
          <div className="bg-white rounded-lg shadow p-8">
            <h2 className="text-2xl font-bold text-gray-900 mb-2">General Settings</h2>
            <p className="text-gray-600 mb-6">Configure general monitoring behavior</p>

            <div className="space-y-6">
              {/* Worker Sleep Time */}
              <div className="border-b border-gray-200 pb-6">
                <label className="block text-sm font-semibold text-gray-700 mb-2">
                  Worker Sleep Time (minutes)
                </label>
                <p className="text-xs text-gray-500 mb-3">
                  How long the monitoring daemon waits between cycles. Lower values = more frequent checks.
                </p>
                <div className="flex items-center gap-4">
                  <input
                    type="number"
                    min="1"
                    max="1440"
                    value={workerSleepTime}
                    onChange={(e) => {
                      setWorkerSleepTime(parseInt(e.target.value) || 10)
                      setGeneralSaved(false)
                    }}
                    className="w-32 px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <span className="text-gray-600 font-medium">
                    {workerSleepTime === 1 ? '1 minute' : `${workerSleepTime} minutes`}
                  </span>
                </div>
                <p className="text-xs text-gray-500 mt-2">
                  Default: 10 minutes | Min: 1 minute | Max: 24 hours (1440 minutes)
                </p>
              </div>

              {/* Headless Browser Mode */}
              <div className="border-b border-gray-200 pb-6">
                <label className="flex items-center gap-3 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={headlessBrowserMode}
                    onChange={(e) => {
                      setHeadlessBrowserMode(e.target.checked)
                      setGeneralSaved(false)
                    }}
                    className="w-4 h-4 rounded border-gray-300 text-blue-600 focus:ring-2 focus:ring-blue-500"
                  />
                  <div className="flex-1">
                    <p className="text-sm font-semibold text-gray-700">Enable Headless Browser Mode</p>
                    <p className="text-xs text-gray-500">
                      When enabled, browser windows will run in headless mode (invisible) during snapshot recording and replay. Disable to see browser windows.
                    </p>
                  </div>
                </label>
              </div>

              {/* Save Button */}
              <div className="flex items-center gap-4">
                <button
                  onClick={handleSaveGeneral}
                  disabled={saving}
                  className="px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded-lg font-semibold"
                >
                  {saving ? 'Saving...' : 'Save Settings'}
                </button>
                {generalSaved && (
                  <div className="flex items-center gap-2 text-green-600">
                    <FontAwesomeIcon icon={faCheck} />
                    <span className="text-sm">Settings saved</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* SMTP Settings Tab */}
        {activeTab === 'smtp' && (
          <div className="bg-white rounded-lg shadow p-8">
            <h2 className="text-2xl font-bold text-gray-900 mb-2">Email Configuration (SMTP)</h2>
            <p className="text-gray-600 mb-6">Configure email settings for alert notifications</p>

            {!loading && smtpStatus && (
              <div
                className={`mb-6 p-4 rounded-lg ${smtpStatus.configured
                    ? 'bg-green-50 border border-green-200'
                    : 'bg-yellow-50 border border-yellow-200'
                  }`}
              >
                {smtpStatus.configured ? (
                  <p className="text-green-800 flex items-center gap-2">
                    <FontAwesomeIcon icon={faCheck} /> SMTP is configured and email alerts are enabled
                  </p>
                ) : (
                  <p className="text-yellow-800 flex items-center gap-2">
                    <FontAwesomeIcon icon={faExclamationTriangle} /> SMTP is not configured. Email alerts will not work.
                  </p>
                )}
              </div>
            )}

            <div className="space-y-4 mb-8">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">SMTP Host</label>
                <input
                  type="text"
                  placeholder="smtp.gmail.com"
                  value={smtpConfig.host}
                  onChange={(e) => {
                    setSMTPConfig({ ...smtpConfig, host: e.target.value })
                    setSMTPSaved(false)
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">SMTP Port</label>
                <input
                  type="number"
                  value={smtpConfig.port}
                  onChange={(e) => {
                    setSMTPConfig({ ...smtpConfig, port: parseInt(e.target.value) })
                    setSMTPSaved(false)
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">SMTP Username</label>
                <input
                  type="text"
                  placeholder="your-api-key or username"
                  value={smtpConfig.username}
                  onChange={(e) => {
                    setSMTPConfig({ ...smtpConfig, username: e.target.value })
                    setSMTPSaved(false)
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">SMTP Password/App Password</label>
                <input
                  type="password"
                  placeholder="Password or API key"
                  value={smtpConfig.password}
                  onChange={(e) => {
                    setSMTPConfig({ ...smtpConfig, password: e.target.value })
                    setSMTPSaved(false)
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">From Email Address</label>
                <input
                  type="email"
                  placeholder="sender@example.com"
                  value={smtpConfig.from}
                  onChange={(e) => {
                    setSMTPConfig({ ...smtpConfig, from: e.target.value })
                    setSMTPSaved(false)
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Verified sender email address in SMTP provider</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">Alert Email Address</label>
                <input
                  type="email"
                  placeholder="alerts@example.com"
                  value={smtpConfig.to}
                  onChange={(e) => {
                    setSMTPConfig({ ...smtpConfig, to: e.target.value })
                    setSMTPSaved(false)
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Email address to receive error alerts</p>
              </div>
            </div>

            <div className="flex items-center gap-4 mb-8">
              <button
                onClick={handleSaveSMTP}
                disabled={saving}
                className="px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded-lg font-semibold"
              >
                {saving ? 'Saving...' : 'Save Configuration'}
              </button>
              {smtpSaved && (
                <div className="flex items-center gap-2 text-green-600">
                  <FontAwesomeIcon icon={faCheck} />
                  <span className="text-sm">Configuration saved</span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Notifications Tab */}
        {activeTab === 'notifications' && (
          <div className="bg-white rounded-lg shadow p-8">
            <h2 className="text-2xl font-bold text-gray-900 mb-2">Notifications</h2>
            <p className="text-gray-600 mb-6">Configure how and when you receive notifications</p>

            <div className="space-y-6">
              {/* Placeholder for future notification settings */}
              <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
                <FontAwesomeIcon icon={faBell} className="text-3xl text-yellow-600 mb-3" />
                <h3 className="text-lg font-semibold text-yellow-900 mb-2">Coming Soon</h3>
                <p className="text-yellow-800 text-sm">
                  Additional notification features are coming in a future release. For now, use email (SMTP) for alerts.
                </p>
              </div>

              {/* Current notification method */}
              <div className="border-t border-gray-200 pt-6">
                <h3 className="font-semibold text-gray-900 mb-3">Current Notification Method</h3>
                <div className="flex items-center gap-3 p-4 bg-gray-50 rounded-lg border border-gray-200">
                  <FontAwesomeIcon icon={faEnvelope} className="text-blue-600 text-lg" />
                  <div>
                    <p className="font-semibold text-gray-900">Email Alerts</p>
                    <p className="text-sm text-gray-600">
                      Notifications are sent via SMTP when websites go down or experience issues.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default SettingsScreen
