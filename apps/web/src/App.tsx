import { Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'sonner'
import Layout from './components/layout/AppShell'
import Login from './pages/Login'
import Register from './pages/Register'
import Inbox from './pages/Inbox'
import Contacts from './pages/Contacts'
import Kanban from './pages/Kanban'
import Campaigns from './pages/Campaigns'
import Agents from './pages/Agents'
import Instances from './pages/Instances'
import Flows from './pages/Flows'
import FlowDetail from './pages/FlowDetail'
import Analytics from './pages/Analytics'
import Settings from './pages/Settings'
import { hasAuthSession } from '@/lib/auth-storage'

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  if (!hasAuthSession()) return <Navigate to="/login" replace />
  return <>{children}</>
}

function App() {
  return (
    <>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        
        {/* Protected Routes inside AppShell Layout */}
        <Route path="/" element={<ProtectedRoute><Layout /></ProtectedRoute>}>
          <Route index element={<Navigate to="/inbox" replace />} />
          <Route path="inbox" element={<Inbox />} />
          <Route path="contacts" element={<Contacts />} />
          <Route path="kanban" element={<Kanban />} />
          <Route path="campaigns" element={<Campaigns />} />
          <Route path="agents" element={<Agents />} />
          <Route path="instances" element={<Instances />} />
          <Route path="flows" element={<Flows />} />
          <Route path="flows/:id" element={<FlowDetail />} />
          <Route path="analytics" element={<Analytics />} />
          <Route path="settings" element={<Settings />} />
        </Route>
        
        {/* Fallback 404 */}
        <Route path="*" element={
          <div className="flex flex-col items-center justify-center min-h-screen text-center bg-background">
            <h1 className="text-4xl font-bold text-primary mb-4">404</h1>
            <p className="text-text-secondary mb-8">Página não encontrada</p>
            <a href="/" className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary-hover transition-colors">Voltar ao início</a>
          </div>
        } />
      </Routes>
      <Toaster theme="dark" position="top-right" />
    </>
  )
}

export default App
