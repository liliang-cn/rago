import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { 
  BarChart3, 
  Database, 
  GitBranch, 
  Calendar, 
  Store, 
  Settings, 
  MessageSquare,
  Activity,
  FileText,
  Zap,
  ChevronLeft,
  ChevronRight
} from 'lucide-react';
import { useUIStore } from '@/store';
import { cn } from '@/utils';
import { Button } from '@/components/ui/button';

const navigationItems = [
  { 
    name: 'Dashboard', 
    href: '/', 
    icon: BarChart3,
    description: 'System overview and metrics'
  },
  { 
    name: 'Documents', 
    href: '/documents', 
    icon: FileText,
    description: 'Document management and RAG'
  },
  { 
    name: 'Query', 
    href: '/query', 
    icon: MessageSquare,
    description: 'AI-powered search and Q&A'
  },
  { 
    name: 'Workflows', 
    href: '/workflows', 
    icon: GitBranch,
    description: 'Visual workflow designer'
  },
  { 
    name: 'Scheduler', 
    href: '/scheduler', 
    icon: Calendar,
    description: 'Job scheduling and automation'
  },
  { 
    name: 'Marketplace', 
    href: '/marketplace', 
    icon: Store,
    description: 'Agent templates and tools'
  },
  { 
    name: 'Monitoring', 
    href: '/monitoring', 
    icon: Activity,
    description: 'System health and alerts'
  },
  { 
    name: 'Providers', 
    href: '/providers', 
    icon: Zap,
    description: 'LLM and service providers'
  },
  { 
    name: 'Settings', 
    href: '/settings', 
    icon: Settings,
    description: 'Configuration and preferences'
  }
];

export function Sidebar() {
  const location = useLocation();
  const { sidebarCollapsed, toggleSidebar } = useUIStore();

  return (
    <aside 
      className={cn(
        "fixed left-0 top-0 z-40 h-screen bg-card border-r border-border transition-all duration-300",
        sidebarCollapsed ? "w-16" : "w-64"
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-border">
        <div className={cn("flex items-center", sidebarCollapsed && "justify-center")}>
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
              <Database className="w-4 h-4 text-primary-foreground" />
            </div>
            {!sidebarCollapsed && (
              <div className="flex flex-col">
                <span className="font-semibold text-sm">RAGO</span>
                <span className="text-xs text-muted-foreground">AI Platform</span>
              </div>
            )}
          </div>
        </div>
        
        {!sidebarCollapsed && (
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleSidebar}
            className="h-8 w-8"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
        )}
      </div>

      {/* Collapsed Toggle Button */}
      {sidebarCollapsed && (
        <div className="p-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleSidebar}
            className="w-full h-8"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      )}

      {/* Navigation */}
      <nav className="flex-1 p-4 space-y-2">
        {navigationItems.map((item) => {
          const isActive = location.pathname === item.href || 
                          (item.href !== '/' && location.pathname.startsWith(item.href));
          
          return (
            <Link
              key={item.href}
              to={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-muted-foreground transition-all hover:text-primary hover:bg-accent group",
                isActive && "bg-accent text-primary",
                sidebarCollapsed && "justify-center px-2"
              )}
              title={sidebarCollapsed ? `${item.name} - ${item.description}` : undefined}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {!sidebarCollapsed && (
                <div className="flex flex-col">
                  <span className="text-sm font-medium">{item.name}</span>
                  <span className="text-xs text-muted-foreground/80 group-hover:text-primary/80">
                    {item.description}
                  </span>
                </div>
              )}
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      {!sidebarCollapsed && (
        <div className="p-4 border-t border-border">
          <div className="text-xs text-muted-foreground">
            <div className="flex items-center justify-between">
              <span>Version 3.0.0</span>
              <div className="flex items-center gap-1">
                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                <span>Online</span>
              </div>
            </div>
          </div>
        </div>
      )}
    </aside>
  );
}