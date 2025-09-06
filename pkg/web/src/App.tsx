import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { Toaster } from 'react-hot-toast';
import { Layout } from '@/components/common/Layout';
import { ErrorBoundary } from '@/components/common/ErrorBoundary';
import { Dashboard } from '@/pages/Dashboard';
import { Documents } from '@/pages/Documents';
import { Query } from '@/pages/Query';
import { Workflows } from '@/pages/Workflows';
import { Settings } from '@/pages/Settings';
import { Providers } from '@/pages/Providers';
import { Monitoring } from '@/pages/Monitoring';
import { Scheduler } from '@/pages/Scheduler';
import { Marketplace } from '@/pages/Marketplace';
import { wsClient } from '@/services/api';
import './App.css';

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      refetchOnWindowFocus: false,
    },
  },
});


function App() {
  React.useEffect(() => {
    // Connect WebSocket for real-time updates
    wsClient.connect();
    
    return () => {
      wsClient.disconnect();
    };
  }, []);

  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <Router>
          <div className="App">
            <Routes>
              <Route path="/" element={<Layout />}>
                <Route index element={<Dashboard />} />
                <Route path="documents" element={<Documents />} />
                <Route path="query" element={<Query />} />
                <Route path="workflows" element={<Workflows />} />
                <Route path="scheduler" element={<Scheduler />} />
                <Route path="marketplace" element={<Marketplace />} />
                <Route path="monitoring" element={<Monitoring />} />
                <Route path="providers" element={<Providers />} />
                <Route path="settings" element={<Settings />} />
              </Route>
            </Routes>
          </div>
        </Router>

        {/* Global Toaster for notifications */}
        <Toaster
          position="top-right"
          toastOptions={{
            duration: 4000,
            style: {
              background: 'hsl(var(--background))',
              color: 'hsl(var(--foreground))',
              border: '1px solid hsl(var(--border))',
            },
          }}
        />

        {/* React Query DevTools */}
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </ErrorBoundary>
  );
}

export default App;