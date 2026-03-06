import { Routes, Route, NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { QueryTest } from './pages/QueryTest'
import { Documents } from './pages/Documents'
import { Chat } from './pages/Chat'
import { Status } from './pages/Status'
import { Skills } from './pages/Skills'
import { MCP } from './pages/MCP'
import { Memory } from './pages/Memory'
import { Agent } from './pages/Agent'
import { Settings } from './pages/Settings'

function Nav() {
  const { t } = useTranslation()
  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `px-4 py-2 rounded-lg transition-colors ${
      isActive
        ? 'bg-blue-600 text-white'
        : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800'
    }`

  return (
    <nav className="max-w-7xl mx-auto px-4 flex gap-2 py-4 overflow-x-auto">
      <NavLink to="/" className={linkClass} end>
        {t('agent')}
      </NavLink>
      <NavLink to="/chat" className={linkClass}>
        {t('chat')}
      </NavLink>
      <NavLink to="/skills" className={linkClass}>
        {t('skills')}
      </NavLink>
      <NavLink to="/mcp" className={linkClass}>
        {t('mcp')}
      </NavLink>
      <NavLink to="/memory" className={linkClass}>
        {t('memory')}
      </NavLink>
      <NavLink to="/status" className={linkClass}>
        {t('status')}
      </NavLink>
      <NavLink to="/query" className={linkClass}>
        {t('query')}
      </NavLink>
      <NavLink to="/documents" className={linkClass}>
        {t('documents')}
      </NavLink>
      <NavLink to="/settings" className={linkClass}>
        {t('settings')}
      </NavLink>
    </nav>
  )
}

function App() {
  return (
    <div className="min-h-screen bg-white dark:bg-gray-900">
      <header className="border-b border-gray-200 dark:border-gray-700">
        <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
            AgentGo UI
          </h1>
          <span className="text-sm text-gray-500 dark:text-gray-400">
            AI Agent SDK
          </span>
        </div>
        <Nav />
      </header>
      <main className="max-w-7xl mx-auto px-4 py-6">
        <Routes>
          <Route path="/" element={<Agent />} />
          <Route path="/chat" element={<Chat />} />
          <Route path="/skills" element={<Skills />} />
          <Route path="/mcp" element={<MCP />} />
          <Route path="/memory" element={<Memory />} />
          <Route path="/status" element={<Status />} />
          <Route path="/query" element={<QueryTest />} />
          <Route path="/documents" element={<Documents />} />
          <Route path="/settings" element={<Settings />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
