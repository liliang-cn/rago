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
    `nav-pill ${
      isActive
        ? 'nav-pill-active'
        : 'nav-pill-idle'
    }`

  return (
    <nav className="flex flex-wrap gap-2">
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
    <div className="min-h-screen app-shell">
      <div className="app-backdrop" />
      <header className="relative z-10 border-b border-slate-200/80">
        <div className="mx-auto flex max-w-[1440px] flex-col gap-5 px-5 py-6 lg:px-8">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.38em] text-sky-700">AgentGo Control Surface</p>
              <h1 className="mt-3 text-3xl font-semibold tracking-tight text-slate-900 sm:text-4xl">
                Multi-agent orchestration with a clean control plane
              </h1>
            </div>
            <div className="max-w-md text-sm leading-7 text-slate-600">
              Observe provider health, govern specialists, delegate tasks, and inspect supporting services from one UI.
            </div>
          </div>
          <Nav />
        </div>
      </header>
      <main className="relative z-10 mx-auto max-w-[1440px] px-5 py-8 lg:px-8">
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
