import React, { useState, useEffect } from 'react'
import api from '../api'
import { normalizeUrl } from '../utils/urlUtils'

function SnapshotRecordingScreen({ onBack, url }) {
  const [recording, setRecording] = useState(false)
  const [progress, setProgress] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [normalizedUrl, setNormalizedUrl] = useState('')
  const [recordingId, setRecordingId] = useState(null)

  useEffect(() => {
    if (url) {
      const normalized = normalizeUrl(url)
      setNormalizedUrl(normalized)
      // Auto-start recording
      handleStartRecording(normalized)
    }
  }, [url])

  const handleStartRecording = async (recordingUrl) => {
    const urlToRecord = recordingUrl || normalizedUrl

    if (!urlToRecord) {
      setError('No URL specified')
      return
    }

    setRecording(true)
    setProgress('Opening browser and navigating to site...')
    setError('')
    setSuccess('')

    try {
      const id = await api.startRecording(urlToRecord)
      setRecordingId(id)
      setProgress('Browser opened. Record your interactions and click "Finish Recording" when done.')
    } catch (err) {
      setError(`Failed to start recording: ${err.message}`)
      setProgress('')
      setRecording(false)
    }
  }

  const handleFinishRecording = async () => {
    if (!recordingId) {
      setError('No active recording')
      return
    }

    try {
      setProgress('Saving snapshot...')
      await api.finishRecording(recordingId)
      setSuccess('Snapshot recorded and saved successfully!')
      setProgress('')
      setRecordingId(null)
    } catch (err) {
      setError(`Failed to finish recording: ${err.message}`)
      setProgress('')
    } finally {
      setRecording(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="bg-white border-b border-gray-200">
        <div className="max-w-4xl mx-auto px-6 py-4">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Record Snapshot</h1>
              <p className="text-sm text-gray-600 mt-1">URL: {normalizedUrl}</p>
            </div>
            <button
              onClick={onBack}
              className="px-4 py-2 bg-gray-200 hover:bg-gray-300 text-gray-800 rounded-lg font-semibold"
            >
              Back
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-2xl mx-auto px-6 py-8">
        <div className="bg-white rounded-lg shadow p-6">
          {progress && (
            <div className="p-4 bg-blue-50 border border-blue-200 rounded-lg mb-4">
              <p className="text-sm text-blue-800">
                {progress}
              </p>
              {recordingId && (
                <button
                  onClick={handleFinishRecording}
                  disabled={!recordingId}
                  className="mt-4 w-full px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-400 text-white rounded-lg font-semibold"
                >
                  Finish Recording
                </button>
              )}
            </div>
          )}

          {error && (
            <div className="p-4 bg-red-50 border border-red-200 rounded-lg mb-4">
              <p className="text-sm text-red-800">{error}</p>
              <button
                onClick={() => handleStartRecording()}
                className="mt-3 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg font-semibold"
              >
                Retry
              </button>
            </div>
          )}

          {success && (
            <div className="p-4 bg-green-50 border border-green-200 rounded-lg mb-4">
              <p className="text-sm text-green-800">{success}</p>
              <button
                onClick={onBack}
                className="mt-3 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg font-semibold"
              >
                Back to Snapshots
              </button>
            </div>
          )}

          {!success && !error && !progress && !recording && (
            <button
              onClick={() => handleStartRecording()}
              disabled={recording}
              className="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white font-semibold py-3 px-4 rounded-lg transition"
            >
              {recording ? 'Recording...' : 'Start Recording'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

export default SnapshotRecordingScreen
