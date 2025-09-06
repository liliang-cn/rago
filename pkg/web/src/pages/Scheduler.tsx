import React, { useState } from 'react';
import { 
  Plus, 
  Calendar, 
  Clock, 
  Play, 
  Pause, 
  Trash2, 
  Settings,
  History,
  CheckCircle,
  AlertTriangle,
  XCircle,
  RotateCcw,
  Edit,
  Copy
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { cn, formatTimeAgo } from '@/utils';
import { Job, JobExecution } from '@/types';

// Mock job data
const mockJobs: Job[] = [
  {
    id: '1',
    name: 'Daily Document Sync',
    description: 'Synchronize documents from external sources every day at 3 AM',
    workflowId: 'wf-1',
    schedule: {
      type: 'cron',
      expression: '0 3 * * *',
      timezone: 'UTC'
    },
    enabled: true,
    inputs: { source: 'external-api', destination: 'local-db' },
    retryPolicy: {
      maxRetries: 3,
      backoff: 'exponential',
      delay: 1000
    },
    notifications: {
      onSuccess: true,
      onFailure: true,
      channels: ['email', 'slack']
    },
    tags: ['sync', 'daily', 'documents'],
    createdAt: '2024-01-15T10:00:00Z',
    updatedAt: '2024-01-20T15:30:00Z',
    lastRun: '2024-01-25T03:00:00Z',
    nextRun: '2024-01-26T03:00:00Z'
  },
  {
    id: '2',
    name: 'Weekly Analytics Report',
    description: 'Generate comprehensive analytics report every Monday morning',
    workflowId: 'wf-2',
    schedule: {
      type: 'cron',
      expression: '0 9 * * 1',
      timezone: 'America/New_York'
    },
    enabled: true,
    inputs: { reportType: 'weekly', format: 'pdf' },
    retryPolicy: {
      maxRetries: 2,
      backoff: 'fixed',
      delay: 5000
    },
    notifications: {
      onSuccess: false,
      onFailure: true,
      channels: ['email']
    },
    tags: ['analytics', 'weekly', 'report'],
    createdAt: '2024-01-10T08:00:00Z',
    updatedAt: '2024-01-22T12:15:00Z',
    lastRun: '2024-01-22T09:00:00Z',
    nextRun: '2024-01-29T09:00:00Z'
  },
  {
    id: '3',
    name: 'Cleanup Old Logs',
    description: 'Remove log files older than 30 days to save disk space',
    workflowId: 'wf-3',
    schedule: {
      type: 'interval',
      expression: '86400000',
      timezone: 'UTC'
    },
    enabled: false,
    inputs: { retention: '30d', path: '/var/logs' },
    retryPolicy: {
      maxRetries: 1,
      backoff: 'fixed',
      delay: 0
    },
    notifications: {
      onSuccess: false,
      onFailure: true,
      channels: ['email']
    },
    tags: ['cleanup', 'maintenance'],
    createdAt: '2024-01-08T14:00:00Z',
    updatedAt: '2024-01-20T10:45:00Z'
  }
];

// Mock execution history
const mockExecutions: JobExecution[] = [
  {
    id: '1',
    jobId: '1',
    workflowExecutionId: 'we-1',
    status: 'completed',
    startTime: '2024-01-25T03:00:00Z',
    endTime: '2024-01-25T03:05:32Z',
    duration: 332000,
    retryAttempt: 0,
    output: { processed: 125, errors: 0 },
    logs: []
  },
  {
    id: '2',
    jobId: '2',
    workflowExecutionId: 'we-2',
    status: 'completed',
    startTime: '2024-01-22T09:00:00Z',
    endTime: '2024-01-22T09:15:45Z',
    duration: 945000,
    retryAttempt: 0,
    output: { reportGenerated: true, fileSize: '2.4MB' },
    logs: []
  },
  {
    id: '3',
    jobId: '1',
    workflowExecutionId: 'we-3',
    status: 'failed',
    startTime: '2024-01-24T03:00:00Z',
    endTime: '2024-01-24T03:02:15Z',
    duration: 135000,
    retryAttempt: 2,
    error: 'Connection timeout to external API',
    logs: []
  }
];

// Job Status Badge Component
function JobStatusBadge({ status }: { status: JobExecution['status'] }) {
  const config = {
    completed: { variant: 'success' as const, icon: CheckCircle },
    failed: { variant: 'destructive' as const, icon: XCircle },
    running: { variant: 'warning' as const, icon: Clock },
    pending: { variant: 'secondary' as const, icon: Clock },
    cancelled: { variant: 'secondary' as const, icon: XCircle }
  };

  const { variant, icon: Icon } = config[status] || config.pending;

  return (
    <Badge variant={variant} className="text-xs">
      <Icon className="h-3 w-3 mr-1" />
      {status}
    </Badge>
  );
}

// Schedule Display Component
function ScheduleDisplay({ schedule }: { schedule: Job['schedule'] }) {
  const getScheduleDescription = () => {
    switch (schedule.type) {
      case 'cron':
        // Simple cron description - in real app, use a cron parser library
        if (schedule.expression === '0 3 * * *') return 'Daily at 3:00 AM';
        if (schedule.expression === '0 9 * * 1') return 'Weekly on Monday at 9:00 AM';
        return `Cron: ${schedule.expression}`;
      case 'interval':
        const hours = Math.floor(parseInt(schedule.expression) / 3600000);
        if (hours >= 24) return `Every ${Math.floor(hours / 24)} day(s)`;
        if (hours >= 1) return `Every ${hours} hour(s)`;
        return `Every ${Math.floor(parseInt(schedule.expression) / 60000)} minute(s)`;
      case 'once':
        return 'Run once';
      default:
        return schedule.expression;
    }
  };

  return (
    <div className="text-sm">
      <div className="font-medium">{getScheduleDescription()}</div>
      {schedule.timezone && schedule.timezone !== 'UTC' && (
        <div className="text-xs text-muted-foreground">{schedule.timezone}</div>
      )}
    </div>
  );
}

// Job Card Component
function JobCard({ job, executions }: { job: Job; executions: JobExecution[] }) {
  const jobExecutions = executions.filter(e => e.jobId === job.id);
  const lastExecution = jobExecutions[0];

  return (
    <Card className={cn('transition-all duration-200', !job.enabled && 'opacity-60')}>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-1">
              <CardTitle className="text-base">{job.name}</CardTitle>
              {!job.enabled && <Badge variant="secondary" className="text-xs">Disabled</Badge>}
            </div>
            <CardDescription className="text-sm">{job.description}</CardDescription>
          </div>
          
          <div className="flex items-center gap-1">
            <Button variant="ghost" size="icon" className="h-6 w-6">
              <Edit className="h-3 w-3" />
            </Button>
            <Button variant="ghost" size="icon" className="h-6 w-6">
              <Copy className="h-3 w-3" />
            </Button>
            <Button 
              variant="ghost" 
              size="icon" 
              className="h-6 w-6"
              onClick={() => {/* Toggle job enabled state */}}
            >
              {job.enabled ? <Pause className="h-3 w-3" /> : <Play className="h-3 w-3" />}
            </Button>
            <Button variant="ghost" size="icon" className="h-6 w-6 text-destructive">
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
        </div>
      </CardHeader>
      
      <CardContent>
        <div className="space-y-4">
          {/* Schedule Info */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <ScheduleDisplay schedule={job.schedule} />
            </div>
          </div>
          
          {/* Next/Last Run */}
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <div className="text-muted-foreground text-xs mb-1">Next Run</div>
              <div className="font-medium">
                {job.nextRun ? formatTimeAgo(job.nextRun) : 'Not scheduled'}
              </div>
            </div>
            <div>
              <div className="text-muted-foreground text-xs mb-1">Last Run</div>
              <div className="font-medium">
                {job.lastRun ? formatTimeAgo(job.lastRun) : 'Never'}
              </div>
            </div>
          </div>
          
          {/* Last Execution Status */}
          {lastExecution && (
            <div className="flex items-center justify-between p-2 bg-muted/50 rounded-md">
              <div className="flex items-center gap-2">
                <JobStatusBadge status={lastExecution.status} />
                <span className="text-sm">
                  {lastExecution.duration ? `${(lastExecution.duration / 1000).toFixed(1)}s` : ''}
                </span>
              </div>
              <Button variant="ghost" size="sm" className="h-6 text-xs">
                View Details
              </Button>
            </div>
          )}
          
          {/* Tags */}
          {job.tags.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {job.tags.map((tag) => (
                <Badge key={tag} variant="outline" className="text-xs">
                  {tag}
                </Badge>
              ))}
            </div>
          )}
          
          {/* Quick Actions */}
          <div className="flex items-center gap-2 pt-2 border-t">
            <Button variant="outline" size="sm" disabled={!job.enabled}>
              <Play className="h-3 w-3 mr-1" />
              Run Now
            </Button>
            <Button variant="outline" size="sm">
              <History className="h-3 w-3 mr-1" />
              History
            </Button>
            <Button variant="outline" size="sm">
              <Settings className="h-3 w-3 mr-1" />
              Configure
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// Recent Executions Component
function RecentExecutions({ executions }: { executions: JobExecution[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Recent Executions</CardTitle>
        <CardDescription>Latest job execution results</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {executions.slice(0, 10).map((execution) => {
            const job = mockJobs.find(j => j.id === execution.jobId);
            return (
              <div key={execution.id} className="flex items-center justify-between p-3 border rounded-lg">
                <div className="flex-1">
                  <div className="font-medium text-sm">{job?.name || 'Unknown Job'}</div>
                  <div className="text-xs text-muted-foreground">
                    Started {formatTimeAgo(execution.startTime)}
                    {execution.duration && ` • ${(execution.duration / 1000).toFixed(1)}s`}
                    {execution.retryAttempt > 0 && ` • Retry ${execution.retryAttempt}`}
                  </div>
                  {execution.error && (
                    <div className="text-xs text-red-600 mt-1">{execution.error}</div>
                  )}
                </div>
                
                <div className="flex items-center gap-2">
                  <JobStatusBadge status={execution.status} />
                  <Button variant="ghost" size="sm" className="h-6 text-xs">
                    Details
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

// Job Stats Component
function JobStats({ jobs, executions }: { jobs: Job[]; executions: JobExecution[] }) {
  const enabledJobs = jobs.filter(j => j.enabled).length;
  const totalExecutions = executions.length;
  const successRate = executions.length > 0 
    ? (executions.filter(e => e.status === 'completed').length / executions.length) * 100 
    : 0;
  const avgDuration = executions.filter(e => e.duration).length > 0
    ? executions.filter(e => e.duration).reduce((sum, e) => sum + e.duration!, 0) / executions.filter(e => e.duration).length / 1000
    : 0;

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <Calendar className="h-4 w-4 text-blue-500" />
            <div>
              <div className="text-2xl font-bold">{enabledJobs}</div>
              <div className="text-xs text-muted-foreground">Active Jobs</div>
            </div>
          </div>
        </CardContent>
      </Card>
      
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <History className="h-4 w-4 text-green-500" />
            <div>
              <div className="text-2xl font-bold">{totalExecutions}</div>
              <div className="text-xs text-muted-foreground">Total Executions</div>
            </div>
          </div>
        </CardContent>
      </Card>
      
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <CheckCircle className="h-4 w-4 text-purple-500" />
            <div>
              <div className="text-2xl font-bold">{successRate.toFixed(1)}%</div>
              <div className="text-xs text-muted-foreground">Success Rate</div>
            </div>
          </div>
        </CardContent>
      </Card>
      
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-orange-500" />
            <div>
              <div className="text-2xl font-bold">{avgDuration.toFixed(1)}s</div>
              <div className="text-xs text-muted-foreground">Avg Duration</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Add Job Dialog Component
function AddJobDialog({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border rounded-lg shadow-lg p-6 w-96 max-h-[80vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Create New Job</h2>
          <Button variant="ghost" size="sm" onClick={onClose}>×</Button>
        </div>
        
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium mb-2 block">Job Name</label>
            <Input placeholder="Enter job name" />
          </div>
          
          <div>
            <label className="text-sm font-medium mb-2 block">Description</label>
            <Input placeholder="Job description" />
          </div>
          
          <div>
            <label className="text-sm font-medium mb-2 block">Workflow</label>
            <select className="w-full p-2 border rounded-md bg-background">
              <option value="">Select workflow</option>
              <option value="wf-1">Document Processing</option>
              <option value="wf-2">Analytics Report</option>
              <option value="wf-3">Data Cleanup</option>
            </select>
          </div>
          
          <div>
            <label className="text-sm font-medium mb-2 block">Schedule Type</label>
            <select className="w-full p-2 border rounded-md bg-background">
              <option value="cron">Cron Expression</option>
              <option value="interval">Interval</option>
              <option value="once">Run Once</option>
            </select>
          </div>
          
          <div>
            <label className="text-sm font-medium mb-2 block">Schedule</label>
            <Input placeholder="0 3 * * * (daily at 3 AM)" />
          </div>
          
          <div>
            <label className="text-sm font-medium mb-2 block">Timezone</label>
            <select className="w-full p-2 border rounded-md bg-background">
              <option value="UTC">UTC</option>
              <option value="America/New_York">Eastern Time</option>
              <option value="America/Los_Angeles">Pacific Time</option>
              <option value="Europe/London">London</option>
            </select>
          </div>
          
          <div className="flex items-center gap-2">
            <input type="checkbox" id="enabled" defaultChecked />
            <label htmlFor="enabled" className="text-sm">Enable job immediately</label>
          </div>
          
          <div className="flex justify-end gap-2 pt-4 border-t">
            <Button variant="outline" onClick={onClose}>Cancel</Button>
            <Button onClick={onClose}>Create Job</Button>
          </div>
        </div>
      </div>
    </div>
  );
}

// Main Scheduler Component
export function Scheduler() {
  const [jobs] = useState(mockJobs);
  const [executions] = useState(mockExecutions);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'enabled' | 'disabled'>('all');

  // Filter jobs
  const filteredJobs = jobs.filter(job => {
    const matchesSearch = job.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         job.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = statusFilter === 'all' || 
                         (statusFilter === 'enabled' && job.enabled) ||
                         (statusFilter === 'disabled' && !job.enabled);
    return matchesSearch && matchesStatus;
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Job Scheduler</h1>
          <p className="text-muted-foreground">
            Schedule and manage automated workflows
          </p>
        </div>
        
        <Button onClick={() => setShowAddDialog(true)}>
          <Plus className="h-4 w-4 mr-2" />
          New Job
        </Button>
      </div>

      {/* Stats */}
      <JobStats jobs={jobs} executions={executions} />

      {/* Filters */}
      <div className="flex items-center gap-4">
        <Input
          placeholder="Search jobs..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="max-w-md"
        />
        
        <select 
          className="p-2 border rounded-md bg-background"
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as any)}
        >
          <option value="all">All Jobs</option>
          <option value="enabled">Enabled Only</option>
          <option value="disabled">Disabled Only</option>
        </select>
      </div>

      {/* Jobs Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="space-y-4">
          <h2 className="text-lg font-medium">Scheduled Jobs ({filteredJobs.length})</h2>
          {filteredJobs.length === 0 ? (
            <Card>
              <CardContent className="p-12 text-center">
                <Calendar className="h-12 w-12 mx-auto mb-4 text-muted-foreground opacity-50" />
                <h3 className="text-lg font-medium mb-2">No jobs found</h3>
                <p className="text-muted-foreground mb-4">
                  {searchTerm || statusFilter !== 'all' 
                    ? 'Try adjusting your search or filters'
                    : 'Create your first scheduled job to automate workflows'
                  }
                </p>
                <Button onClick={() => setShowAddDialog(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Job
                </Button>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-4">
              {filteredJobs.map((job) => (
                <JobCard key={job.id} job={job} executions={executions} />
              ))}
            </div>
          )}
        </div>
        
        <div>
          <RecentExecutions executions={executions} />
        </div>
      </div>

      {/* Add Job Dialog */}
      <AddJobDialog isOpen={showAddDialog} onClose={() => setShowAddDialog(false)} />
    </div>
  );
}