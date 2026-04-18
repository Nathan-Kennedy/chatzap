import { Outlet, NavLink, useLocation, useNavigate } from 'react-router-dom'
import {
  Inbox,
  Users,
  LayoutDashboard,
  MessageSquare,
  Bot,
  Smartphone,
  GitBranch,
  BarChart3,
  Settings,
  Bell,
  CircleUser,
  Menu,
  LogOut,
} from 'lucide-react'
import { useState } from 'react'
import { cn } from '@/lib/utils'
import type { NavKey } from '@/lib/permissions'
import { canAccessNav } from '@/lib/permissions'
import { useAuthProfile, useLogoutCleanup, useUserRole } from '@/hooks/useAuthProfile'
import { useInboxUnreadTotal } from '@/hooks/useInboxUnreadTotal'
import { getAccessToken, isAuthMockEnabled } from '@/lib/auth-storage'
import type { UserRole } from '@/types/user'
import { useRealtime } from '@/hooks/useRealtime'
import { logoutRequest } from '@/lib/auth-api'
import { initialsFromName } from '@/utils/initials'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

type NavItem = {
  navKey: NavKey
  icon: typeof Inbox
  label: string
  path: string
}

const allNavItems: NavItem[] = [
  { navKey: 'inbox', icon: Inbox, label: 'Caixa de Entrada', path: '/inbox' },
  { navKey: 'contacts', icon: Users, label: 'Contatos', path: '/contacts' },
  { navKey: 'kanban', icon: LayoutDashboard, label: 'Kanban', path: '/kanban' },
  { navKey: 'campaigns', icon: MessageSquare, label: 'Campanhas', path: '/campaigns' },
  { navKey: 'agents', icon: Bot, label: 'Agentes IA', path: '/agents' },
  { navKey: 'instances', icon: Smartphone, label: 'Instâncias', path: '/instances' },
  { navKey: 'flows', icon: GitBranch, label: 'Fluxos', path: '/flows' },
  { navKey: 'analytics', icon: BarChart3, label: 'Analytics', path: '/analytics' },
]

export default function AppShell() {
  const [collapsed, setCollapsed] = useState(false)
  const location = useLocation()
  const navigate = useNavigate()
  const role = useUserRole()
  const profile = useAuthProfile()
  const clearSession = useLogoutCleanup()

  useRealtime({ enabled: !!import.meta.env.VITE_WS_URL })

  const roleResolved: UserRole | undefined =
    role ??
    (getAccessToken() || isAuthMockEnabled() ? 'agent' : undefined)

  const inboxUnreadTotal = useInboxUnreadTotal()

  const navItems = allNavItems.filter((item) =>
    canAccessNav(roleResolved, item.navKey)
  )
  const canSettings = canAccessNav(roleResolved, 'settings')

  const displayName = profile?.user_name ?? profile?.user_email ?? 'Usuário'
  const userInitials = initialsFromName(displayName)
  const workspaceLabel =
    profile?.workspace_name ?? profile?.workspace_id?.slice(0, 8) ?? 'Workspace'

  async function handleLogout() {
    await logoutRequest()
    clearSession()
    navigate('/login', { replace: true })
  }

  return (
    <TooltipProvider>
      <div className="flex h-screen overflow-hidden bg-background text-text-primary">
        <aside
          className={cn(
            'bg-sidebar border-r border-border flex flex-col transition-all duration-300',
            collapsed ? 'w-16' : 'w-60',
            'hidden md:flex'
          )}
        >
          <div className="h-14 flex items-center justify-between px-4 border-b border-border shrink-0">
            {!collapsed && (
              <div className="flex flex-col min-w-0 gap-0.5">
                <div className="flex items-center gap-2">
                  <div className="w-6 h-6 rounded bg-primary flex items-center justify-center shrink-0">
                    <MessageSquare size={14} className="text-white" />
                  </div>
                  <span className="font-bold text-lg text-text-primary tracking-tight truncate">
                    WhatsSaaS
                  </span>
                </div>
                <span className="text-[10px] text-text-muted truncate pl-8">
                  {workspaceLabel}
                </span>
              </div>
            )}
            <button
              type="button"
              onClick={() => setCollapsed(!collapsed)}
              className="p-1 hover:bg-white/5 rounded text-text-muted shrink-0"
            >
              <Menu size={20} />
            </button>
          </div>

          <nav className="flex-1 py-4 flex flex-col gap-1 px-2 overflow-y-auto">
            {navItems.map((item) => {
              const inboxNavBadge =
                item.navKey === 'inbox' && inboxUnreadTotal > 0
                  ? inboxUnreadTotal > 99
                    ? '99+'
                    : String(inboxUnreadTotal)
                  : null
              return (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors group relative',
                    isActive
                      ? 'bg-primary/10 text-primary'
                      : 'text-text-secondary hover:bg-card-hover hover:text-text-primary'
                  )
                }
                title={collapsed ? item.label : undefined}
              >
                <div className="relative">
                  <item.icon
                    size={20}
                    className={cn(
                      'shrink-0',
                      location.pathname.startsWith(item.path)
                        ? 'text-primary'
                        : 'text-text-muted group-hover:text-text-secondary'
                    )}
                  />
                  {collapsed && inboxNavBadge ? (
                    <span className="absolute -top-1.5 -right-1.5 w-3 h-3 bg-primary rounded-full border border-sidebar text-[0px]">
                      .
                    </span>
                  ) : null}
                </div>
                {!collapsed && (
                  <span className="flex-1 truncate text-sm font-medium">{item.label}</span>
                )}
                {!collapsed && inboxNavBadge ? (
                  <span className="bg-primary text-white text-[10px] font-bold px-1.5 py-0.5 rounded-full leading-none min-w-[1.25rem] text-center">
                    {inboxNavBadge}
                  </span>
                ) : null}
                {location.pathname.startsWith(item.path) && (
                  <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-primary rounded-r" />
                )}
              </NavLink>
              )
            })}
          </nav>

          <div className="p-2 border-t border-border mt-auto">
            {canSettings && (
              <NavLink
                to="/settings"
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors relative group',
                    isActive
                      ? 'bg-primary/10 text-primary'
                      : 'text-text-secondary hover:bg-card-hover hover:text-text-primary'
                  )
                }
                title={collapsed ? 'Configurações' : undefined}
              >
                <Settings
                  size={20}
                  className={cn(
                    location.pathname.startsWith('/settings')
                      ? 'text-primary'
                      : 'text-text-muted group-hover:text-text-secondary'
                  )}
                />
                {!collapsed && <span className="text-sm font-medium">Configurações</span>}
                {location.pathname.startsWith('/settings') && (
                  <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-primary rounded-r" />
                )}
              </NavLink>
            )}
            <div className="mt-2 flex items-center gap-3 px-3 py-2">
              <div className="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center shrink-0 border border-primary/20">
                <span className="text-xs font-bold text-primary">{userInitials}</span>
              </div>
              {!collapsed && (
                <div className="flex flex-col overflow-hidden flex-1 min-w-0">
                  <span className="text-sm font-medium text-text-primary truncate">
                    {displayName}
                  </span>
                  <span className="text-xs text-text-muted truncate capitalize">
                    {roleResolved ?? '—'} · Plano Pro
                  </span>
                </div>
              )}
              {!collapsed && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0">
                      <LogOut size={16} />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="bg-card border-border">
                    <DropdownMenuItem onClick={() => void handleLogout()}>
                      Sair
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </div>
          </div>
        </aside>

        <div className="flex-1 flex flex-col min-w-0">
          <header className="h-14 bg-background border-b border-border flex items-center justify-between px-6 shrink-0">
            <div className="flex flex-col min-w-0">
              <span className="text-sm font-medium text-text-secondary capitalize">
                {location.pathname.split('/')[1] || 'Dashboard'}
              </span>
              <span className="text-[11px] text-text-muted truncate md:hidden">
                {workspaceLabel}
              </span>
            </div>
            <div className="flex items-center gap-4">
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    className="relative p-1.5 text-text-secondary hover:text-text-primary hover:bg-white/5 rounded-md transition-colors"
                  >
                    <Bell size={18} />
                    <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-primary rounded-full ring-2 ring-background" />
                  </button>
                </TooltipTrigger>
                <TooltipContent>Notificações</TooltipContent>
              </Tooltip>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button
                    type="button"
                    className="w-8 h-8 rounded-full overflow-hidden bg-white/10 hover:ring-2 hover:ring-primary/50 transition-all focus:outline-none flex items-center justify-center"
                  >
                    <CircleUser size={32} className="text-text-muted" />
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="bg-card border-border">
                  <DropdownMenuItem onClick={() => void handleLogout()}>
                    Sair
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </header>

          <main className="flex-1 overflow-hidden bg-background relative flex flex-col min-h-0">
            <Outlet />
          </main>
        </div>
      </div>
    </TooltipProvider>
  )
}
