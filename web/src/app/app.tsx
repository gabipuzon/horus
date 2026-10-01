import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Activity, ListChecks } from 'lucide-react'
import { BrowserRouter, NavLink, Navigate, Route, Routes } from 'react-router-dom'
import { MonitorPage } from '../routes/monitor-page'

export function App({ queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 10_000 } } }) }: { queryClient?: QueryClient }) {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <div className="min-h-screen bg-background text-foreground md:flex">
          <aside className="hidden w-56 shrink-0 border-r border-border bg-surface px-4 py-6 md:flex md:flex-col">
            <div className="flex items-center gap-2.5 px-3 text-base font-semibold tracking-tight"><Activity size={18} className="text-accent" /> Horus</div>
            <div className="mt-10 px-3 text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">Workspace</div>
            <nav aria-label="Primary navigation" className="mt-3">
              <NavLink to="/monitors" className={({ isActive }) => `flex items-center gap-2 rounded-md px-3 py-2 text-sm ${isActive ? 'bg-accent/10 font-medium text-accent' : 'text-muted-foreground hover:bg-surface-raised hover:text-foreground'}`}><ListChecks size={16} /> Monitors</NavLink>
            </nav>
            <div className="mt-auto px-3 text-xs text-muted-foreground">Uptime monitoring</div>
          </aside>
          <div className="min-w-0 flex-1">
            <header className="flex h-14 items-center justify-between border-b border-border bg-surface px-4 md:hidden">
              <div className="flex items-center gap-2 text-sm font-semibold"><Activity size={18} className="text-accent" /> Horus</div>
              <nav aria-label="Mobile navigation"><NavLink to="/monitors" className="rounded px-2 py-1.5 text-sm text-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent">Monitors</NavLink></nav>
            </header>
            <main className="mx-auto max-w-[1400px] px-4 py-8 sm:px-6 lg:px-10">
              <Routes>
                <Route path="/" element={<Navigate to="/monitors" replace />} />
                <Route path="/monitors" element={<MonitorPage />} />
                <Route path="*" element={<Navigate to="/monitors" replace />} />
              </Routes>
            </main>
          </div>
        </div>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
