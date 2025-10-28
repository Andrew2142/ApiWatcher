import React, { useState, useCallback, useMemo } from 'react'
import { AgGridReact } from 'ag-grid-react'
import { themeQuartz } from 'ag-grid-community'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faPlus, faVideo } from '@fortawesome/free-solid-svg-icons'
import api from '../api'
import { normalizeUrl } from '../utils/urlUtils'

function SnapshotScreen({ onNavigate, error, setError, screenParams }) {
  const [snapshots, setSnapshots] = useState([])
  const [selectedURL, setSelectedURL] = useState(screenParams?.url || '')
  const [websites, setWebsites] = useState([])
  const [loadingWebsites, setLoadingWebsites] = useState(true)
  const [loading, setLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState(null)
  const [gridApi, setGridApi] = useState(null)
  const [showCreateMenu, setShowCreateMenu] = useState(false)

  // Load websites on mount
  const loadWebsites = async () => {
    try {
      setLoadingWebsites(true)
      const configs = await api.listConfigs()
      const websitesList = []
      if (configs && configs.length > 0) {
        configs.forEach(cfg => {
          cfg.urls.forEach((url) => {
            websitesList.push(url)
          })
        })
      }
      setWebsites(websitesList)
    } catch (err) {
      console.error('Failed to load websites:', err)
    } finally {
      setLoadingWebsites(false)
    }
  }

  React.useEffect(() => {
    loadWebsites()
  }, [])

  // Auto-load snapshots when URL is selected
  React.useEffect(() => {
    if (selectedURL.trim()) {
      handleListSnapshots()
    } else {
      setSnapshots([])
    }
  }, [selectedURL])

  const handleListSnapshots = async () => {
    if (!selectedURL.trim()) {
      alert('Please enter a URL')
      return
    }
    try {
      setLoading(true)
      const normalizedUrl = normalizeUrl(selectedURL)
      const snaps = await api.listSnapshots(normalizedUrl)
      setSnapshots(snaps || [])
      setError(null)
    } catch (err) {
      setError(err.message || 'Failed to list snapshots')
    } finally {
      setLoading(false)
    }
  }

  const handleCreateSnapshot = async () => {
    if (!selectedURL.trim()) {
      alert('Please enter a URL')
      return
    }
    try {
      setCreating(true)
      const normalizedUrl = normalizeUrl(selectedURL)
      const snap = await api.createSnapshot(normalizedUrl)
      setSnapshots([snap, ...snapshots])
      setError(null)
    } catch (err) {
      setError(err.message || 'Failed to create snapshot')
    } finally {
      setCreating(false)
    }
  }

  const [replaying, setReplaying] = useState(null)

  const handleDeleteSnapshot = async (snapshotID) => {
    if (!window.confirm('Delete this snapshot?')) return
    try {
      setDeleting(snapshotID)
      await api.deleteSnapshot(snapshotID)
      setError(null)
      // Reload snapshots after deletion
      handleListSnapshots();
    } catch (err) {
      setError(err.message || 'Failed to delete snapshot')
    } finally {
      setDeleting(null)
      setLoading(false)
    }
  }

  const handleReplaySnapshot = async (snapshotID) => {
    try {
      setReplaying(snapshotID)
      setError(null)
      await api.replaySnapshot(snapshotID)
    } catch (err) {
      setError(err.message || 'Failed to replay snapshot')
    } finally {
      setReplaying(null)
    }
  }

  const handleAdvancedRecording = () => {
    if (!selectedURL.trim()) {
      alert('Please enter a URL first')
      return
    }
    onNavigate('snapshot-recording', { url: selectedURL })
  }

  const onGridReady = useCallback((params) => {
    setGridApi(params.api)
  }, [])

  // Action column renderer
  const ActionCellRenderer = (props) => {
    const snap = props.data
    return (
      <div className="flex gap-2 h-full items-center justify-between">
        <h1
          onClick={() => handleReplaySnapshot(snap.id)}
          disabled={replaying === snap.id}
          className="text-blue-600 hover:text-blue-800 cursor-pointer font-semibold text-sm"
        >
          {replaying === snap.id ? 'Replaying...' : 'Replay'}
        </h1>
        <button
          onClick={() => handleDeleteSnapshot(snap.id)}
          disabled={deleting === snap.id}
          className="px-3 py-1 bg-red-600 hover:bg-red-700 disabled:bg-gray-400 text-white rounded text-sm font-semibold"
        >
          {deleting === snap.id ? 'Deleting...' : 'Delete'}
        </button>
      </div>
    )
  }

  const columnDefs = useMemo(() => [
    {
      field: 'created_at',
      headerName: 'Created',
      width: 200,
      sortable: true,
      filter: true
    },
    {
      field: 'id',
      headerName: 'Snapshot ID',
      width: 200,
      sortable: true,
      filter: true,
      resizable: true
    },
    {
      field: 'actions',
      headerName: 'Actions Recorded',
      width: 150,
      sortable: true,
      filter: true,
      resizable: true
    },
    {
      headerName: 'Actions',
      width: 150,
      cellRenderer: ActionCellRenderer,
      sortable: false,
      filter: false
    }
  ], [deleting, replaying, handleDeleteSnapshot, handleReplaySnapshot])

  const defaultColDef = useMemo(() => ({
    resizable: true,
    sortable: true,
    filter: true
  }), [])

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="bg-white border-b border-gray-200 p-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Snapshot Management</h1>
          <p className="text-sm text-gray-600 mt-1">Create and manage browser snapshots for API testing</p>
        </div>
      </div>

      <div className="px-6 py-8">
        <div className="bg-white rounded-lg shadow p-6 mb-6">
          <h2 className="text-xl font-bold text-gray-900 mb-4">Select URL</h2>
          <select
            value={selectedURL}
            onChange={e => setSelectedURL(e.target.value)}
            disabled={loadingWebsites || websites.length === 0}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
          >
            <option value="">
              {loadingWebsites ? 'Loading websites...' : websites.length === 0 ? 'No websites available' : 'Select a website...'}
            </option>
            {websites.map((url) => (
              <option key={url} value={url}>
                {url}
              </option>
            ))}
          </select>
        </div>

        {selectedURL && (
          <div className="bg-white rounded-lg shadow p-6 mb-6">
            <div className="relative inline-block">
              <button
                onClick={() => setShowCreateMenu(!showCreateMenu)}
                disabled={creating}
                className="px-6 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-400 text-white rounded-lg font-semibold flex items-center gap-2"
              >
                <FontAwesomeIcon icon={faPlus} />
                {creating ? 'Creating...' : 'Create Snapshot'}
              </button>

              {showCreateMenu && (
                <div className="absolute top-full left-0 mt-2 w-48 bg-white border border-gray-300 rounded-lg shadow-lg z-10">
                  <button
                    onClick={() => {
                      handleCreateSnapshot()
                      setShowCreateMenu(false)
                    }}
                    className="w-full text-left px-4 py-3 hover:bg-gray-50 border-b border-gray-200 font-medium text-gray-900"
                  >
                    Instant Snapshot
                  </button>
                  <button
                    onClick={() => {
                      handleAdvancedRecording()
                      setShowCreateMenu(false)
                    }}
                    className="w-full text-left px-4 py-3 hover:bg-gray-50 font-medium text-gray-900 flex items-center gap-2"
                  >
                    <FontAwesomeIcon icon={faVideo} />
                    Record Snapshot
                  </button>
                </div>
              )}
            </div>
          </div>
        )}

        {snapshots.length > 0 && selectedURL && (
          <div className="bg-white rounded-lg shadow overflow-hidden" style={{ display: 'flex', flexDirection: 'column' }}>
            <div className="p-6 border-b border-gray-200">
              <h2 className="text-xl font-bold text-gray-900">
                Snapshots ({snapshots.length})
              </h2>
            </div>
            <div style={{ height: '600px', width: '100%' }}>
              <AgGridReact
                theme={themeQuartz}
                rowData={snapshots}
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

        {snapshots.length === 0 && selectedURL && !loading && (
          <div className="bg-white rounded-lg shadow p-12 text-center text-gray-500">
            <p>No snapshots found for this URL. Please create a snapshot above.</p>
          </div>
        )}
      </div>
    </div>
  )
}

export default SnapshotScreen
