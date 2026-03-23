import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import './index.css'

// Setup offline sync if connected
const token = localStorage.getItem('khayal_token')
const host = localStorage.getItem('khayal_host')
if (token && host) {
  import('./lib/offline').then(({ setupOfflineSync }) => {
    setupOfflineSync(host, token)
  })
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
