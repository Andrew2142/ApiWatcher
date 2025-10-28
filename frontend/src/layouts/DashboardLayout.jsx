import React, { useState } from 'react'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faChartLine, faGlobe, faCamera, faCog, faSignOutAlt, faChevronLeft, faChevronRight, faTimes } from '@fortawesome/free-solid-svg-icons'

function DashboardLayout({
  currentScreen,
  onNavigate,
  onDisconnect,
  error,
  setError,
  screenParams,
  children
}) {
  const [sidebarOpen, setSidebarOpen] = useState(true)

  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: faChartLine },
    { id: 'websites', label: 'Websites', icon: faGlobe },
    { id: 'snapshots', label: 'Snapshots', icon: faCamera },
    { id: 'settings', label: 'Settings', icon: faCog },
  ]

  return (
    <div className="min-h-screen bg-gray-50 flex">
      {/* Sidebar */}
      <div className={`bg-white border-r border-gray-200 transition-all duration-300 ${sidebarOpen ? 'w-64' : 'w-20'}`}>
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between">
            {sidebarOpen && (
              <h1 className="text-xl font-bold text-gray-900">API Watcher</h1>
            )}
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-2 hover:bg-gray-100 rounded-lg text-gray-600"
              title={sidebarOpen ? 'Collapse sidebar' : 'Expand sidebar'}
            >
              <FontAwesomeIcon icon={sidebarOpen ? faChevronLeft : faChevronRight} />
            </button>
          </div>
        </div>

        {/* Navigation */}
        <nav className="p-4 space-y-2">
          {navItems.map(item => (
            <button
              key={item.id}
              onClick={() => onNavigate(item.id)}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition ${
                currentScreen === item.id
                  ? 'bg-blue-50 text-blue-600 font-semibold'
                  : 'text-gray-700 hover:bg-gray-50'
              }`}
              title={!sidebarOpen ? item.label : ''}
            >
              <FontAwesomeIcon icon={item.icon} className="text-lg" />
              {sidebarOpen && <span>{item.label}</span>}
            </button>
          ))}
        </nav>

        {/* Bottom section */}
        <div className="absolute bottom-0 left-0 right-0 border-t border-gray-200 p-4" style={{ width: sidebarOpen ? '16rem' : '5rem' }}>
          <button
            onClick={onDisconnect}
            className={`w-full flex items-center gap-3 px-4 py-2 rounded-lg bg-red-50 text-red-600 hover:bg-red-100 transition ${!sidebarOpen && 'justify-center'}`}
            title={!sidebarOpen ? 'Disconnect' : ''}
          >
            <FontAwesomeIcon icon={faSignOutAlt} className="text-lg" />
            {sidebarOpen && <span className="font-semibold text-sm">Disconnect</span>}
          </button>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col">
        {/* Error banner */}
        {error && (
          <div className="bg-red-50 border-b border-red-200 p-4">
            <div className="flex justify-between items-center max-w-7xl mx-auto">
              <span className="text-red-700">{error}</span>
              <button
                onClick={() => setError(null)}
                className="text-red-700 hover:text-red-900"
              >
                <FontAwesomeIcon icon={faTimes} />
              </button>
            </div>
          </div>
        )}

        {/* Content area */}
        <div className="flex-1 overflow-auto">
          {children}
        </div>
      </div>
    </div>
  )
}

export default DashboardLayout
