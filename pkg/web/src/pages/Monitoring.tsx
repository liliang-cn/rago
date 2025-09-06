import React, { useState } from 'react';
import { 
  Activity, 
  Cpu, 
  HardDrive, 
  MemoryStick, 
  Network, 
  AlertTriangle,
  CheckCircle,
  TrendingUp,
  TrendingDown,
  RefreshCw,
  Download,
  Settings,
  Eye,
  Clock,
  Zap,
  Database
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { formatCompactNumber, formatTimeAgo, getStatusColor } from '@/utils';
import { LineChart, Line, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';

// Mock system metrics data
const generateTimeSeriesData = (points: number, baseValue: number, variance: number) => {
  return Array.from({ length: points }, (_, i) => ({
    time: new Date(Date.now() - (points - i) * 60000).toISOString(),
    timestamp: `${i}m ago`,
    value: Math.max(0, Math.min(100, baseValue + (Math.random() - 0.5) * variance))
  }));
};

const cpuData = generateTimeSeriesData(30, 45, 30);
const memoryData = generateTimeSeriesData(30, 60, 20);
const diskData = generateTimeSeriesData(30, 25, 15);
const networkData = generateTimeSeriesData(30, 30, 40);

// System Services Status
const systemServices = [
  { name: 'RAGO Core', status: 'healthy', uptime: '5d 12h', cpu: 15.2, memory: 245, description: 'Main application service' },
  { name: 'Vector Database', status: 'healthy', uptime: '5d 12h', cpu: 8.7, memory: 1200, description: 'Document embeddings storage' },
  { name: 'Text Search', status: 'healthy', uptime: '5d 11h', cpu: 3.4, memory: 180, description: 'Full-text search engine' },
  { name: 'Background Jobs', status: 'warning', uptime: '2h 15m', cpu: 2.1, memory: 95, description: 'Document processing queue' },
  { name: 'API Server', status: 'healthy', uptime: '5d 12h', cpu: 6.8, memory: 156, description: 'REST API endpoints' },
  { name: 'WebSocket Server', status: 'healthy', uptime: '5d 12h', cpu: 1.2, memory: 45, description: 'Real-time communications' },
];

const alerts = [
  { id: '1', severity: 'warning', title: 'High CPU Usage', message: 'Background Jobs service showing elevated CPU usage', timestamp: new Date(Date.now() - 5 * 60000).toISOString() },
  { id: '2', severity: 'info', title: 'Database Optimization Complete', message: 'Vector database optimization finished successfully', timestamp: new Date(Date.now() - 15 * 60000).toISOString() },
  { id: '3', severity: 'error', title: 'Memory Leak Detected', message: 'Potential memory leak in document processing', timestamp: new Date(Date.now() - 45 * 60000).toISOString(), resolved: true },
];

// System Metrics Chart Component
function MetricsChart({ title, data, color, unit = '%', icon: Icon }: {
  title: string;
  data: any[];
  color: string;
  unit?: string;
  icon: React.ElementType;
}) {
  const currentValue = data[data.length - 1]?.value || 0;
  const previousValue = data[data.length - 2]?.value || 0;
  const trend = currentValue > previousValue ? 'up' : currentValue < previousValue ? 'down' : 'neutral';
  const TrendIcon = trend === 'up' ? TrendingUp : trend === 'down' ? TrendingDown : null;

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Icon className="h-4 w-4 text-muted-foreground" />
            <CardTitle className="text-sm font-medium">{title}</CardTitle>
          </div>
          <div className="flex items-center gap-1 text-xs">
            {TrendIcon && <TrendIcon className="h-3 w-3" />}
            <span className={trend === 'up' ? 'text-red-600' : trend === 'down' ? 'text-green-600' : ''}>
              {Math.abs(currentValue - previousValue).toFixed(1)}{unit}
            </span>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold mb-1">
          {currentValue.toFixed(1)}{unit}
        </div>
        <div className="h-20 mb-2">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data}>
              <Area 
                type="monotone" 
                dataKey="value" 
                stroke={color} 
                fill={color} 
                fillOpacity={0.2}
                strokeWidth={2}
              />
              <Tooltip 
                formatter={(value: number) => [`${value.toFixed(1)}${unit}`, title]}
                labelFormatter={(label) => `Time: ${label}`}
                contentStyle={{
                  backgroundColor: 'hsl(var(--popover))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '6px',
                  fontSize: '12px'
                }}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
        <div className="text-xs text-muted-foreground">Last 30 minutes</div>
      </CardContent>
    </Card>
  );
}

// Service Status Component
function ServiceStatus() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>System Services</CardTitle>
        <CardDescription>Status of all RAGO components</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {systemServices.map((service) => (
            <div key={service.name} className="flex items-center justify-between p-3 rounded-lg border">
              <div className="flex items-center gap-3">
                <div className={`w-3 h-3 rounded-full ${
                  service.status === 'healthy' ? 'bg-green-500' :
                  service.status === 'warning' ? 'bg-yellow-500' : 'bg-red-500'
                }`} />
                <div className="flex-1">
                  <div className="font-medium text-sm">{service.name}</div>
                  <div className="text-xs text-muted-foreground">{service.description}</div>
                </div>
              </div>
              
              <div className="flex items-center gap-4 text-xs">
                <div className="text-right">
                  <div className="font-medium">{service.uptime}</div>
                  <div className="text-muted-foreground">Uptime</div>
                </div>
                <div className="text-right">
                  <div className="font-medium">{service.cpu}%</div>
                  <div className="text-muted-foreground">CPU</div>
                </div>
                <div className="text-right">
                  <div className="font-medium">{service.memory}MB</div>
                  <div className="text-muted-foreground">Memory</div>
                </div>
                <Badge 
                  variant={service.status === 'healthy' ? 'success' : 
                          service.status === 'warning' ? 'warning' : 'destructive'}
                  className="text-xs"
                >
                  {service.status}
                </Badge>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

// Performance Overview Chart
function PerformanceOverview() {
  const performanceData = Array.from({ length: 24 }, (_, i) => ({
    time: `${23 - i}:00`,
    requests: Math.floor(Math.random() * 1000) + 200,
    latency: Math.floor(Math.random() * 500) + 100,
    errors: Math.floor(Math.random() * 20),
  })).reverse();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Performance Overview</CardTitle>
        <CardDescription>System performance metrics over the last 24 hours</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="h-64">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={performanceData}>
              <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
              <XAxis dataKey="time" className="text-xs" />
              <YAxis yAxisId="left" className="text-xs" />
              <YAxis yAxisId="right" orientation="right" className="text-xs" />
              <Tooltip 
                contentStyle={{ 
                  backgroundColor: 'hsl(var(--popover))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '6px'
                }}
              />
              <Line 
                yAxisId="left"
                type="monotone" 
                dataKey="requests" 
                stroke="hsl(var(--primary))" 
                strokeWidth={2}
                name="Requests/hour"
              />
              <Line 
                yAxisId="right"
                type="monotone" 
                dataKey="latency" 
                stroke="hsl(var(--secondary))" 
                strokeWidth={2}
                name="Avg Latency (ms)"
              />
              <Line 
                yAxisId="left"
                type="monotone" 
                dataKey="errors" 
                stroke="#ef4444" 
                strokeWidth={2}
                name="Errors"
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
        
        <div className="grid grid-cols-3 gap-4 mt-4 pt-4 border-t">
          <div className="text-center">
            <div className="text-sm text-muted-foreground">Avg Requests/hour</div>
            <div className="text-lg font-bold">
              {formatCompactNumber(performanceData.reduce((sum, d) => sum + d.requests, 0) / performanceData.length)}
            </div>
          </div>
          <div className="text-center">
            <div className="text-sm text-muted-foreground">Avg Latency</div>
            <div className="text-lg font-bold">
              {Math.round(performanceData.reduce((sum, d) => sum + d.latency, 0) / performanceData.length)}ms
            </div>
          </div>
          <div className="text-center">
            <div className="text-sm text-muted-foreground">Error Rate</div>
            <div className="text-lg font-bold">
              {((performanceData.reduce((sum, d) => sum + d.errors, 0) / 
                performanceData.reduce((sum, d) => sum + d.requests, 0)) * 100).toFixed(2)}%
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// Resource Usage Pie Chart
function ResourceUsage() {
  const storageData = [
    { name: 'Documents', value: 45, color: '#3b82f6' },
    { name: 'Vectors', value: 30, color: '#10b981' },
    { name: 'Indexes', value: 15, color: '#f59e0b' },
    { name: 'Logs', value: 7, color: '#ef4444' },
    { name: 'Cache', value: 3, color: '#8b5cf6' },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Storage Usage</CardTitle>
        <CardDescription>Breakdown of disk space usage</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-6">
          <div className="w-32 h-32">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={storageData}
                  dataKey="value"
                  nameKey="name"
                  cx="50%"
                  cy="50%"
                  outerRadius={60}
                  innerRadius={30}
                >
                  {storageData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip 
                  formatter={(value: number) => [`${value}%`, 'Usage']}
                  contentStyle={{
                    backgroundColor: 'hsl(var(--popover))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '6px',
                    fontSize: '12px'
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>
          
          <div className="flex-1 space-y-2">
            {storageData.map((item) => (
              <div key={item.name} className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <div 
                    className="w-3 h-3 rounded-full" 
                    style={{ backgroundColor: item.color }}
                  />
                  <span>{item.name}</span>
                </div>
                <span className="font-medium">{item.value}%</span>
              </div>
            ))}
          </div>
        </div>
        
        <div className="mt-4 pt-4 border-t text-center">
          <div className="text-sm text-muted-foreground">Total Used</div>
          <div className="text-lg font-bold">2.4 GB / 50 GB</div>
          <div className="text-xs text-muted-foreground">95.2% available</div>
        </div>
      </CardContent>
    </Card>
  );
}

// System Alerts Component
function SystemAlerts() {
  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'error':
        return <AlertTriangle className="h-4 w-4 text-red-500" />;
      case 'warning':
        return <AlertTriangle className="h-4 w-4 text-yellow-500" />;
      case 'info':
        return <CheckCircle className="h-4 w-4 text-blue-500" />;
      default:
        return <AlertTriangle className="h-4 w-4 text-gray-500" />;
    }
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>System Alerts</CardTitle>
            <CardDescription>Recent system events and notifications</CardDescription>
          </div>
          <Button variant="outline" size="sm">
            <Eye className="h-4 w-4 mr-1" />
            View All
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {alerts.map((alert) => (
            <div 
              key={alert.id} 
              className={`flex items-start gap-3 p-3 rounded-lg border ${
                alert.resolved ? 'opacity-60' : ''
              }`}
            >
              {getSeverityIcon(alert.severity)}
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <h4 className="font-medium text-sm">{alert.title}</h4>
                  {alert.resolved && (
                    <Badge variant="outline" className="text-xs">
                      Resolved
                    </Badge>
                  )}
                </div>
                <p className="text-xs text-muted-foreground mt-1">{alert.message}</p>
                <div className="flex items-center gap-1 text-xs text-muted-foreground mt-2">
                  <Clock className="h-3 w-3" />
                  {formatTimeAgo(alert.timestamp)}
                </div>
              </div>
              <Badge 
                variant={alert.severity === 'error' ? 'destructive' : 
                        alert.severity === 'warning' ? 'warning' : 'secondary'}
                className="text-xs"
              >
                {alert.severity}
              </Badge>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

// Main Monitoring Component
export function Monitoring() {
  const [timeRange, setTimeRange] = useState('1h');
  
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">System Monitoring</h1>
          <p className="text-muted-foreground">
            Real-time system health and performance metrics
          </p>
        </div>
        
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm">
            <Download className="h-4 w-4 mr-1" />
            Export Report
          </Button>
          <Button variant="outline" size="sm">
            <Settings className="h-4 w-4 mr-1" />
            Configure
          </Button>
          <Button variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-1" />
            Refresh
          </Button>
        </div>
      </div>

      {/* System Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricsChart 
          title="CPU Usage" 
          data={cpuData} 
          color="#3b82f6" 
          icon={Cpu}
        />
        <MetricsChart 
          title="Memory Usage" 
          data={memoryData} 
          color="#10b981" 
          icon={MemoryStick}
        />
        <MetricsChart 
          title="Disk Usage" 
          data={diskData} 
          color="#f59e0b" 
          icon={HardDrive}
        />
        <MetricsChart 
          title="Network I/O" 
          data={networkData} 
          color="#8b5cf6" 
          unit="MB/s"
          icon={Network}
        />
      </div>

      {/* Performance and Services */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <PerformanceOverview />
        <ResourceUsage />
      </div>

      {/* Service Status */}
      <ServiceStatus />

      {/* Alerts */}
      <SystemAlerts />
    </div>
  );
}