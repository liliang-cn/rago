import React from 'react';
import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { SkipLink } from './SkipLink';
import { useUIStore } from '@/store';
import { cn } from '@/utils';

export function Layout() {
  const { sidebarCollapsed } = useUIStore();

  return (
    <div className="flex h-screen bg-background">
      {/* Skip Link for Accessibility */}
      <SkipLink />
      
      {/* Sidebar */}
      <Sidebar />
      
      {/* Main Content Area */}
      <div 
        className={cn(
          "flex flex-col flex-1 transition-all duration-300",
          sidebarCollapsed ? "ml-16" : "ml-64"
        )}
      >
        {/* Header */}
        <Header />
        
        {/* Page Content */}
        <main id="main-content" className="flex-1 overflow-auto p-6 bg-muted/5" role="main" tabIndex={-1}>
          <Outlet />
        </main>
      </div>
    </div>
  );
}