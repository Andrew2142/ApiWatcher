import 'core-js/stable'
import 'regenerator-runtime/runtime'
import 'tailwindcss/tailwind.css'
import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.jsx'

// Register AG Grid modules
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community'
ModuleRegistry.registerModules([AllCommunityModule])

// Wait for Wails runtime to be ready
const runtime = require('@wailsapp/runtime')

function start() {
	const container = document.getElementById('app')
	container.style.width = '100%'
	container.style.height = '100%'
	const root = createRoot(container)
	root.render(<App />)
}

runtime.Init(start)