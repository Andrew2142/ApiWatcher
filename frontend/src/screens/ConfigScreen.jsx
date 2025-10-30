import React, { useState, useEffect } from 'react'
import api from '../api'

function ConfigScreen({ onNavigate, error, setError }) {
  const [configs, setConfigs] = useState([])
  const [selectedConfig, setSelectedConfig] = useState(null)
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({ name: '', urls: '' })
  const [loading, setLoading] = useState(true)
  const [deleting, setDeleting] = useState(null)
  const [saving, setSaving] = useState(false)
  const [loadingConfig, setLoadingConfig] = useState(null)

  useEffect(() => {
    loadConfigs()
  }, [])

  const loadConfigs = async () => {
    try {
      setLoading(true)
      const cfgs = await api.listConfigs()
      setConfigs(cfgs || [])
      setError(null)
    } catch (err) {
      setError(err.message || 'Failed to load configurations')
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    if (!formData.name.trim()) {
      alert('Please enter a configuration name')
      return
    }
    const urls = formData.urls
      .split('\n')
      .map(url => url.trim())
      .filter(url => url)
    if (urls.length === 0) {
      alert('Please enter at least one URL')
      return
    }
    try {
      setSaving(true)
      await api.saveConfig(formData.name, urls)
      setFormData({ name: '', urls: '' })
      setShowForm(false)
      await loadConfigs()
    } catch (err) {
      setError(err.message || 'Failed to save configuration')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (name) => {
    if (!window.confirm(`Delete configuration "${name}"?`)) return
    try {
      setDeleting(name)
      await api.deleteConfig(name)
      await loadConfigs()
      setError(null)
    } catch (err) {
      setError(err.message || 'Failed to delete configuration')
    } finally {
      setDeleting(null)
    }
  }

  const handleLoadConfig = async (name) => {
    try {
      setLoadingConfig(name)
      const cfg = await api.loadConfig(name)
      setSelectedConfig(cfg)
      setError(null)
    } catch (err) {
      setError(err.message || 'Failed to load configuration')
    } finally {
      setLoadingConfig(null)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="bg-white border-b border-gray-200 p-6">
        <div className="flex justify-between items-center">
          <h1 className="text-3xl font-bold text-gray-900">Configuration Management</h1>
          <button
            onClick={() => onNavigate('dashboard')}
            className="px-4 py-2 bg-gray-200 hover:bg-gray-300 text-gray-800 rounded-lg font-semibold"
          >
            Back to Dashboard
          </button>
        </div>
      </div>

      <div className="max-w-6xl mx-auto px-6 py-8">
        {loading ? (
          <p className="text-center text-gray-600">Loading configurations...</p>
        ) : !showForm ? (
          <>
            <div className="bg-white rounded-lg shadow">
              <div className="p-6 border-b border-gray-200">
                <h2 className="text-xl font-bold text-gray-900">Saved Configurations</h2>
              </div>
              {configs.length > 0 ? (
                <div className="divide-y divide-gray-200">
                  {configs.map(cfg => (
                    <div key={cfg.name} className="p-4 hover:bg-gray-50">
                      <div className="flex justify-between items-start">
                        <div className="flex-1">
                          <h3 className="font-bold text-gray-900">{cfg.name}</h3>
                          <p className="text-sm text-gray-600 mt-1">{cfg.urls.length} URLs</p>
                          <div className="mt-2 space-y-1">
                            {cfg.urls.slice(0, 3).map((url, idx) => (
                              <p key={idx} className="text-sm text-gray-700">{url}</p>
                            ))}
                            {cfg.urls.length > 3 && (
                              <p className="text-sm text-gray-600">+{cfg.urls.length - 3} more</p>
                            )}
                          </div>
                        </div>
                        <div className="flex gap-2">
                          <button
                            onClick={() => handleLoadConfig(cfg.name)}
                            disabled={loadingConfig === cfg.name || loading}
                            className="px-3 py-1 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded text-sm font-semibold"
                          >
                            {loadingConfig === cfg.name ? 'Loading...' : 'Load'}
                          </button>
                          <button
                            onClick={() => handleDelete(cfg.name)}
                            disabled={deleting === cfg.name || loading}
                            className="px-3 py-1 bg-red-600 hover:bg-red-700 disabled:bg-gray-400 text-white rounded text-sm font-semibold"
                          >
                            {deleting === cfg.name ? 'Deleting...' : 'Delete'}
                          </button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="p-8 text-center text-gray-500">No configurations saved yet</div>
              )}
            </div>
            <button
              onClick={() => setShowForm(true)}
              className="mt-6 px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-semibold"
            >
              + Create New Configuration
            </button>
          </>
        ) : (
          <div className="bg-white rounded-lg shadow p-6 max-w-2xl">
            <h2 className="text-xl font-bold text-gray-900 mb-6">Create New Configuration</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Configuration Name
                </label>
                <input
                  type="text"
                  placeholder="e.g., Production Servers"
                  value={formData.name}
                  onChange={e => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  URLs (one per line)
                </label>
                <textarea
                  placeholder="https://example.com&#10;https://api.example.com"
                  value={formData.urls}
                  onChange={e => setFormData({ ...formData, urls: e.target.value })}
                  rows="10"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="flex gap-2">
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded-lg font-semibold"
                >
                  {saving ? 'Saving...' : 'Save Configuration'}
                </button>
                <button
                  onClick={() => setShowForm(false)}
                  disabled={saving}
                  className="flex-1 px-4 py-2 bg-gray-200 hover:bg-gray-300 disabled:bg-gray-400 text-gray-800 rounded-lg font-semibold"
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        )}

        {selectedConfig && (
          <div className="mt-8 bg-white rounded-lg shadow p-6">
            <h2 className="text-xl font-bold text-gray-900 mb-4">Loaded: {selectedConfig.name}</h2>
            <div className="space-y-2">
              {selectedConfig.urls.map((url, idx) => (
                <div key={idx} className="p-2 bg-gray-50 rounded">{url}</div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default ConfigScreen
