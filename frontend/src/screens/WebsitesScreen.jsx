import React, { useState, useEffect, useCallback, useMemo } from 'react'
import { AgGridReact } from 'ag-grid-react'
import { themeQuartz } from 'ag-grid-community'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faPlus } from '@fortawesome/free-solid-svg-icons'
import api from '../api'
import { normalizeUrl } from '../utils/urlUtils'

function WebsitesScreen({ error, setError, onNavigate }) {
  const [websites, setWebsites] = useState([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({ name: '', url: '' })
  const [deleting, setDeleting] = useState(null)
  const [saving, setSaving] = useState(false)
  const [gridApi, setGridApi] = useState(null)

  useEffect(() => {
    loadWebsites()
  }, [])

  const loadWebsites = async () => {
    try {
      setLoading(true)
      const configs = await api.listConfigs()
      const websitesList = []
      if (configs && configs.length > 0) {
        configs.forEach(cfg => {
          cfg.urls.forEach((url, idx) => {
            websitesList.push({
              id: `${cfg.name}-${idx}`,
              name: cfg.name || url,
              url: url,
              configName: cfg.name,
              snapshotId: null
            })
          })
        })
      }
      setWebsites(websitesList)
      setError(null)
    } catch (err) {
      setError(err.message || 'Failed to load websites')
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    if (!formData.name.trim() || !formData.url.trim()) {
      alert('Please enter website name and URL')
      return
    }

    const normalizedUrl = normalizeUrl(formData.url)

    try {
      setSaving(true)
      await api.saveConfig(formData.name, [normalizedUrl])
      setFormData({ name: '', url: '' })
      setShowForm(false)
      await loadWebsites()
    } catch (err) {
      setError(err.message || 'Failed to save website')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (websiteId) => {
    if (!window.confirm('Delete this website?')) return

    try {
      setDeleting(websiteId)
      const website = websites.find(w => w.id === websiteId)
      if (website) {
        await api.deleteConfig(website.configName)
        await loadWebsites()
      }
    } catch (err) {
      setError(err.message || 'Failed to delete website')
    } finally {
      setDeleting(null)
    }
  }

  const onGridReady = useCallback((params) => {
    setGridApi(params.api)
  }, [])

  // Action column renderer
  const ActionCellRenderer = (props) => {
    const website = props.data
    return (
      <div className="flex gap-2 h-full items-center justify-between">
        <h1
          onClick={() => onNavigate && onNavigate('snapshots', { url: website.url })}
          className="text-blue-600 hover:text-blue-800 cursor-pointer font-semibold text-sm"
        >
          View Snapshots
        </h1>
        <button
          onClick={() => handleDelete(website.id)}
          disabled={deleting === website.id}
          className="px-2 py-1 bg-red-600 hover:bg-red-700 disabled:bg-gray-400 text-white rounded text-xs font-semibold"
        >
          {deleting === website.id ? '...' : 'Delete'}
        </button>
      </div>
    )
  }

  const columnDefs = useMemo(() => [
    {
      field: 'name',
      headerName: 'Name',
      width: 200,
      sortable: true,
      filter: true
    },
    {
      field: 'url',
      headerName: 'URL',
      width: 300,
      sortable: true,
      filter: true,
      resizable: true
    },
    {
      headerName: 'Actions',
      width: 250,
      cellRenderer: ActionCellRenderer,
      sortable: false,
      filter: false
    }
  ], [deleting, onNavigate])

  const defaultColDef = useMemo(() => ({
    resizable: true,
    sortable: true,
    filter: true
  }), [])

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="bg-white border-b border-gray-200">
        <div className="max-w-full px-6 py-6">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Websites</h1>
              <p className="text-sm text-gray-600 mt-1">Manage your monitored websites</p>
            </div>
            {!showForm && (
              <button
                onClick={() => setShowForm(true)}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-semibold flex items-center gap-2"
              >
                <FontAwesomeIcon icon={faPlus} />
                Add Website
              </button>
            )}
          </div>
        </div>
      </div>

      <div className="px-6 py-8">
        {/* Add/Edit Form */}
        {showForm && (
          <div className="bg-white rounded-lg shadow p-6 mb-8">
            <h2 className="text-xl font-bold text-gray-900 mb-6">Add New Website</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Website Name
                </label>
                <input
                  type="text"
                  placeholder="e.g., Production API"
                  value={formData.name}
                  onChange={e => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Website URL
                </label>
                <input
                  type="text"
                  placeholder="example.com or api.example.com"
                  value={formData.url}
                  onChange={e => setFormData({ ...formData, url: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">https:// will be added automatically</p>
              </div>
            </div>
            <div className="flex gap-2 mt-6">
              <button
                onClick={handleSave}
                disabled={saving}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded-lg font-semibold"
              >
                {saving ? 'Saving...' : 'Save Website'}
              </button>
              <button
                onClick={() => {
                  setShowForm(false)
                  setFormData({ name: '', url: '' })
                }}
                disabled={saving}
                className="px-4 py-2 bg-gray-200 hover:bg-gray-300 disabled:bg-gray-400 text-gray-800 rounded-lg font-semibold"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {/* Websites Grid */}
        {loading ? (
          <div className="text-center text-gray-600 py-12">
            <p>Loading websites...</p>
          </div>
        ) : websites.length === 0 ? (
          <div className="bg-white rounded-lg shadow p-12 text-center">
            <p className="text-gray-500 mb-4">No websites added yet</p>
            {!showForm && (
              <button
                onClick={() => setShowForm(true)}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-semibold flex items-center gap-2"
              >
                <FontAwesomeIcon icon={faPlus} />
                Add Your First Website
              </button>
            )}
          </div>
        ) : (
          <div className="bg-white rounded-lg shadow overflow-hidden" style={{ display: 'flex', flexDirection: 'column' }}>
            <div className="p-6 border-b border-gray-200">
              <h2 className="text-xl font-bold text-gray-900">
                Websites ({websites.length})
              </h2>
            </div>
            <div style={{ height: '600px', width: '100%' }}>
              <AgGridReact
                theme={themeQuartz}
                rowData={websites}
                columnDefs={columnDefs}
                defaultColDef={defaultColDef}
                onGridReady={onGridReady}
                pagination={true}
                paginationPageSize={10}
                paginationPageSizeSelector={[10, 20, 50, 100]}
                suppressHorizontalScroll={false}
                style={{ width: '100%', height: '100%' }}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default WebsitesScreen
