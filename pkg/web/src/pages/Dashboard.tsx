import React, { useEffect, useState } from 'react';
import { 
  Activity, 
  Database, 
  FileText, 
  Zap, 
  TrendingUp, 
  TrendingDown, 
  AlertTriangle,
  CheckCircle,
  Clock,
  Users
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useSystemStore, useDocumentsStore, useWorkflowsStore, useJobsStore } from '@/store';
import { formatCompactNumber, formatTimeAgo, getStatusColor } from '@/utils';
import { LineChart, Line, AreaChart, Area, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

// Metric Card Component
function MetricCard({ 
  title, 
  value, 
  change, 
  icon: Icon, 
  trend = 'neutral',
  description 
}: {
  title: string;
  value: string | number;
  change?: string;
  icon: React.ElementType;
  trend?: 'up' | 'down' | 'neutral';
  description?: string;
}) {
  const TrendIcon = trend === 'up' ? TrendingUp : trend === 'down' ? TrendingDown : null;

  return (
    <Card className="metric-card">
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold metric-value">{value}</div>
        {change && (
          <div className="flex items-center gap-1 text-xs text-muted-foreground">
            {TrendIcon && <TrendIcon className="h-3 w-3" />}
            <span className={trend === 'up' ? 'text-green-600' : trend === 'down' ? 'text-red-600' : ''}>
              {change}
            </span>
            <span>from last hour</span>
          </div>
        )}
        {description && <p className="text-xs text-muted-foreground mt-1">{description}</p>}
      </CardContent>
    </Card>
  );
}

// System Status Component
function SystemStatus() {
  const { componentStatuses } = useSystemStore();

  return (
    <Card>
      <CardHeader>
        <CardTitle>System Status</CardTitle>
        <CardDescription>Real-time status of all components</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {componentStatuses.map((status) => (
            <div key={status.name} className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`w-3 h-3 rounded-full ${
                  status.status === 'healthy' ? 'bg-green-500' :
                  status.status === 'warning' ? 'bg-yellow-500' : 'bg-red-500'
                }`} />
                <div>
                  <p className="text-sm font-medium">{status.name}</p>
                  <p className="text-xs text-muted-foreground">
                    Last check: {formatTimeAgo(status.lastCheck)}
                  </p>
                </div>
              </div>
              <Badge 
                variant={status.status === 'healthy' ? 'success' : 
                        status.status === 'warning' ? 'warning' : 'destructive'}
              >
                {status.status}
              </Badge>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

// Recent Activity Component
function RecentActivity() {
  const activities = [
    { id: '1', type: 'document', message: 'New document uploaded: technical-specs.pdf', timestamp: '2 minutes ago' },
    { id: '2', type: 'workflow', message: 'Workflow "Data Processing" completed successfully', timestamp: '5 minutes ago' },
    { id: '3', type: 'query', message: 'AI query processed: "Explain the main features"', timestamp: '8 minutes ago' },
    { id: '4', type: 'job', message: 'Scheduled job "Daily Report" started', timestamp: '12 minutes ago' },
    { id: '5', type: 'alert', message: 'System alert resolved: High memory usage', timestamp: '15 minutes ago' },
  ];

  const getActivityIcon = (type: string) => {
    switch (type) {
      case 'document': return FileText;
      case 'workflow': return Activity;
      case 'query': return Database;
      case 'job': return Clock;
      case 'alert': return AlertTriangle;
      default: return Activity;
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Recent Activity</CardTitle>
        <CardDescription>Latest system events and actions</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {activities.map((activity) => {
            const Icon = getActivityIcon(activity.type);
            return (
              <div key={activity.id} className="flex items-start gap-3">
                <Icon className="h-4 w-4 mt-0.5 text-muted-foreground" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm">{activity.message}</p>
                  <p className="text-xs text-muted-foreground">{activity.timestamp}</p>
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

// Performance Chart Component
function PerformanceChart() {
  const [timeRange, setTimeRange] = useState('1h');
  
  // Mock data - replace with real data
  const data = Array.from({ length: 24 }, (_, i) => ({
    time: `${23 - i}:00`,
    queries: Math.floor(Math.random() * 100) + 20,
    documents: Math.floor(Math.random() * 50) + 10,
    workflows: Math.floor(Math.random() * 30) + 5,
  })).reverse();

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle>Performance Overview</CardTitle>
          <CardDescription>System activity over time</CardDescription>
        </div>
        <div className="flex gap-2">
          {['1h', '6h', '24h', '7d'].map((range) => (
            <Button
              key={range}
              variant={timeRange === range ? 'default' : 'outline'}
              size="sm"
              onClick={() => setTimeRange(range)}
            >
              {range}
            </Button>
          ))}
        </div>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <AreaChart data={data}>
            <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
            <XAxis dataKey="time" className="text-xs" />
            <YAxis className="text-xs" />
            <Tooltip 
              contentStyle={{ 
                backgroundColor: 'hsl(var(--popover))',
                border: '1px solid hsl(var(--border))',
                borderRadius: '6px'
              }}
            />
            <Area 
              type="monotone" 
              dataKey="queries" 
              stackId="1" 
              stroke="hsl(var(--primary))" 
              fill="hsl(var(--primary))" 
              fillOpacity={0.1}
            />
            <Area 
              type="monotone" 
              dataKey="documents" 
              stackId="1" 
              stroke="hsl(var(--secondary))" 
              fill="hsl(var(--secondary))" 
              fillOpacity={0.1}
            />
            <Area 
              type="monotone" 
              dataKey="workflows" 
              stackId="1" 
              stroke="hsl(var(--accent))" 
              fill="hsl(var(--accent))" 
              fillOpacity={0.1}
            />
          </AreaChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}

// Main Dashboard Component
export function Dashboard() {
  const { providers, componentStatuses, alerts, loading } = useSystemStore();
  const { documents } = useDocumentsStore();
  const { workflows, executions } = useWorkflowsStore();
  const { jobs } = useJobsStore();

  // Calculate metrics
  const totalDocuments = documents.length;
  const activeProviders = providers.filter(p => p.status === 'active').length;
  const runningWorkflows = executions.filter(e => e.status === 'running').length;
  const activeJobs = jobs.filter(j => j.enabled).length;
  const unacknowledgedAlerts = alerts.filter(a => !a.acknowledged).length;
  const healthyComponents = componentStatuses.filter(c => c.status === 'healthy').length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">
          Monitor your RAGO AI platform's performance and activity
        </p>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          title="Documents"
          value={formatCompactNumber(totalDocuments)}
          change="+12%"
          trend="up"
          icon={FileText}
          description="Total documents in knowledge base"
        />
        <MetricCard
          title="Active Providers"
          value={`${activeProviders}/${providers.length}`}
          change="+2"
          trend="up"
          icon={Zap}
          description="LLM and embedding providers online"
        />
        <MetricCard
          title="Running Workflows"
          value={runningWorkflows}
          change="-3"
          trend="down"
          icon={Activity}
          description="Currently executing workflows"
        />
        <MetricCard
          title="Scheduled Jobs"
          value={activeJobs}
          change="+1"
          trend="up"
          icon={Clock}
          description="Active scheduled tasks"
        />
      </div>

      {/* System Health */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CheckCircle className="h-5 w-5 text-green-500" />
                System Health
              </CardTitle>
              <CardDescription>
                {healthyComponents}/{componentStatuses.length} components healthy
                {unacknowledgedAlerts > 0 && (
                  <span className="ml-2 text-destructive">
                    • {unacknowledgedAlerts} active alerts
                  </span>
                )}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {/* Health Summary */}
                <div className="flex items-center gap-4">
                  <div className="flex-1">
                    <div className="h-2 bg-muted rounded-full overflow-hidden">
                      <div 
                        className="h-full bg-green-500 transition-all duration-300"
                        style={{ 
                          width: `${(healthyComponents / Math.max(componentStatuses.length, 1)) * 100}%` 
                        }}
                      />
                    </div>
                  </div>
                  <span className="text-sm font-medium">
                    {Math.round((healthyComponents / Math.max(componentStatuses.length, 1)) * 100)}%
                  </span>
                </div>

                {/* Component Status Grid */}
                <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                  {componentStatuses.map((status) => (
                    <div key={status.name} className="flex items-center gap-2">
                      <div className={`w-2 h-2 rounded-full ${getStatusColor(status.status).split(' ')[0]}`} />
                      <span className="text-xs">{status.name}</span>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        <div className="space-y-6">
          <SystemStatus />
        </div>
      </div>

      {/* Performance and Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <PerformanceChart />
        <RecentActivity />
      </div>

      {/* Alerts */}
      {unacknowledgedAlerts > 0 && (
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-destructive">
              <AlertTriangle className="h-5 w-5" />
              Active Alerts ({unacknowledgedAlerts})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {alerts.filter(a => !a.acknowledged).slice(0, 5).map((alert) => (
                <div key={alert.id} className="flex items-center justify-between p-3 bg-destructive/5 rounded-lg">
                  <div>
                    <p className="font-medium">{alert.title}</p>
                    <p className="text-sm text-muted-foreground">{alert.message}</p>
                    <p className="text-xs text-muted-foreground">{formatTimeAgo(alert.timestamp)}</p>
                  </div>
                  <Badge variant="destructive">{alert.severity}</Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}