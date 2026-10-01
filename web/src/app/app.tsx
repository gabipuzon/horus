import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Activity, LayoutDashboard, ListChecks } from 'lucide-react'
import { BrowserRouter, NavLink, Navigate, Route, Routes } from 'react-router-dom'
import { MonitorPage } from '../routes/monitor-page'
import { MonitorDetailPage } from '../routes/monitor-detail-page'
import { OverviewPage } from '../routes/overview-page'

const navClass = ({ isActive }: { isActive: boolean }) => `flex items-center gap-2 rounded-md px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent ${isActive ? 'bg-accent/10 font-medium text-accent' : 'text-muted-foreground hover:bg-surface-raised hover:text-foreground'}`

export function App({ queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 10_000 } } }) }: { queryClient?: QueryClient }) {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <div className="min-h-screen bg-background text-foreground md:flex">
          <aside className="hidden w-56 shrink-0 border-r border-border bg-surface px-4 py-6 md:flex md:flex-col">
            <div className="flex items-center gap-2.5 px-3 text-base font-semibold tracking-tight"><Activity size={18} className="text-accent" /> Horus</div>
            <nav aria-label="Primary navigation" className="mt-10 space-y-1">
              <NavLink to="/overview" className={navClass}><LayoutDashboard size={16} /> Overview</NavLink>
              <NavLink to="/monitors" className={navClass}><ListChecks size={16} /> Monitors</NavLink>
            </nav>
            <div className="mt-auto px-3 text-xs text-muted-foreground">Uptime monitoring</div>
          </aside>
          <div className="min-w-0 flex-1">
            <header className="flex h-14 items-center justify-between border-b border-border bg-surface px-4 md:hidden">
              <div className="flex shrink-0 items-center gap-2 text-sm font-semibold"><Activity size={18} className="text-accent" /> Horus</div>
              <nav aria-label="Mobile navigation" className="flex gap-1"><NavLink to="/overview" className={navClass}>Overview</NavLink><NavLink to="/monitors" className={navClass}>Monitors</NavLink></nav>
            </header>
            <main className="mx-auto max-w-[1400px] px-4 py-8 sm:px-6 lg:px-10">
              <Routes>
                <Route path="/" element={<Navigate to="/overview" replace />} />
                <Route path="/overview" element={<OverviewPage />} />
                <Route path="/monitors" element={<MonitorPage />} />
                <Route path="/monitors/:id" element={<MonitorDetailPage />} />
                <Route path="*" element={<Navigate to="/overview" replace />} />
              </Routes>
            </main>
          </div>
        </div>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
