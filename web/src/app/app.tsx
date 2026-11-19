import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Activity } from 'lucide-react'
import { BrowserRouter, NavLink, Navigate, Route, Routes } from 'react-router-dom'
import { MonitorPage } from '../routes/monitor-page'
import { MonitorDetailPage } from '../routes/monitor-detail-page'
import { OverviewPage } from '../routes/overview-page'

const navClass = ({ isActive }: { isActive: boolean }) => `inline-flex h-full items-center border-b-2 px-3 text-[13px] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent ${isActive ? 'border-accent font-medium text-foreground' : 'border-transparent text-muted-foreground hover:border-border-strong hover:text-foreground'}`

export function App({ queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: 10_000 } } }) }: { queryClient?: QueryClient }) {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <div className="min-h-screen bg-background text-foreground">
          <header className="app-header border-b border-border bg-sidebar">
            <div className="mx-auto flex h-14 max-w-[1440px] items-center gap-5 px-4 sm:gap-9 sm:px-8 lg:px-12">
              <LinkBrand />
              <nav aria-label="Primary navigation" className="flex h-full items-center gap-2">
                <NavLink to="/overview" className={navClass}>Overview</NavLink>
                <NavLink to="/monitors" className={navClass}>Monitors</NavLink>
              </nav>
              <span className="ml-auto hidden text-xs text-muted-foreground sm:block">Uptime monitoring</span>
            </div>
          </header>
          <div className="min-w-0">
            <main className="mx-auto max-w-[1440px] px-4 py-8 sm:px-8 lg:px-12 lg:py-11">
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

function LinkBrand() {
  return <NavLink to="/overview" aria-label="Horus home" className="flex shrink-0 items-center gap-2.5 rounded-sm text-[15px] font-semibold tracking-[-0.03em] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"><Activity size={18} className="text-accent" strokeWidth={2.2} /> Horus</NavLink>
}
