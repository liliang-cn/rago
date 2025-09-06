import React, { useState } from 'react';
import { 
  Plus, 
  Key, 
  TestTube, 
  Activity, 
  TrendingUp,
  TrendingDown,
  AlertCircle,
  CheckCircle,
  Settings,
  Trash2,
  RefreshCw,
  Zap,
  Brain,
  Globe,
  Database,
  BarChart3,
  DollarSign
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { useSystemStore } from '@/store';
import { cn, formatCompactNumber, formatTimeAgo } from '@/utils';
import { Provider } from '@/types';

// Mock provider data - replace with real data from store
const mockProviders: Provider[] = [
  {
    id: '1',
    name: 'OpenAI GPT-4',
    type: 'llm',
    endpoint: 'https://api.openai.com/v1',
    status: 'active',
    models: ['gpt-4', 'gpt-4-turbo', 'gpt-3.5-turbo'],
    cost: { input: 0.03, output: 0.06, unit: 'per 1K tokens' },
    performance: { latency: 1200, throughput: 50, reliability: 99.8 },
    usage: { requests: 1250, tokens: 89000, cost: 2.45 }
  },
  {
    id: '2',
    name: 'OpenAI Embeddings',
    type: 'embedder',
    endpoint: 'https://api.openai.com/v1',
    status: 'active',
    models: ['text-embedding-ada-002', 'text-embedding-3-small'],
    cost: { input: 0.0001, output: 0, unit: 'per 1K tokens' },
    performance: { latency: 800, throughput: 100, reliability: 99.9 },
    usage: { requests: 2100, tokens: 450000, cost: 0.45 }
  },
  {
    id: '3',
    name: 'Anthropic Claude',
    type: 'llm',
    endpoint: 'https://api.anthropic.com',
    status: 'active',
    models: ['claude-3-sonnet', 'claude-3-haiku'],
    cost: { input: 0.003, output: 0.015, unit: 'per 1K tokens' },
    performance: { latency: 1500, throughput: 40, reliability: 99.5 },
    usage: { requests: 890, tokens: 67000, cost: 1.20 }
  },
  {
    id: '4',
    name: 'Local Ollama',
    type: 'both',
    endpoint: 'http://localhost:11434',
    status: 'inactive',
    models: ['llama2', 'codellama', 'mistral'],
    performance: { latency: 3000, throughput: 20, reliability: 95.0 },
    usage: { requests: 0, tokens: 0, cost: 0 }
  },
  {
    id: '5',
    name: 'LM Studio',
    type: 'llm',
    endpoint: 'http://localhost:1234',
    status: 'error',
    models: ['local-model'],
    performance: { latency: 0, throughput: 0, reliability: 0 },
    usage: { requests: 0, tokens: 0, cost: 0 }
  }
];

// Provider Status Icon Component
function ProviderStatusIcon({ status }: { status: Provider['status'] }) {
  switch (status) {
    case 'active':
      return <CheckCircle className="h-4 w-4 text-green-500" />;
    case 'inactive':
      return <AlertCircle className="h-4 w-4 text-yellow-500" />;
    case 'error':
      return <AlertCircle className="h-4 w-4 text-red-500" />;
    default:
      return <AlertCircle className="h-4 w-4 text-gray-500" />;
  }
}

// Provider Type Icon Component
function ProviderTypeIcon({ type }: { type: Provider['type'] }) {
  switch (type) {
    case 'llm':
      return <Brain className="h-4 w-4" />;
    case 'embedder':
      return <Database className="h-4 w-4" />;
    case 'both':
      return <Zap className="h-4 w-4" />;
    default:
      return <Globe className="h-4 w-4" />;
  }
}

// Provider Metrics Card Component
function ProviderMetrics({ provider }: { provider: Provider }) {
  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
      <div className="text-center">
        <div className="text-2xl font-bold">{provider.performance?.latency || 0}ms</div>
        <div className="text-xs text-muted-foreground">Avg Latency</div>
      </div>
      
      <div className="text-center">
        <div className="text-2xl font-bold">{provider.performance?.throughput || 0}/min</div>
        <div className="text-xs text-muted-foreground">Throughput</div>
      </div>
      
      <div className="text-center">
        <div className="text-2xl font-bold">{provider.performance?.reliability || 0}%</div>
        <div className="text-xs text-muted-foreground">Reliability</div>
      </div>
      
      <div className="text-center">
        <div className="text-2xl font-bold">
          ${provider.cost ? provider.usage?.cost?.toFixed(2) || '0.00' : 'Free'}
        </div>
        <div className="text-xs text-muted-foreground">Total Cost</div>
      </div>
    </div>
  );
}

// Provider Card Component
function ProviderCard({ provider }: { provider: Provider }) {
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <Card className={cn(
      'transition-all duration-200',
      isExpanded && 'ring-2 ring-primary/20'
    )}>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <ProviderTypeIcon type={provider.type} />
            <div>
              <CardTitle className="text-base">{provider.name}</CardTitle>
              <CardDescription className="text-xs">
                {provider.endpoint}
              </CardDescription>
            </div>
          </div>
          
          <div className="flex items-center gap-2">
            <ProviderStatusIcon status={provider.status} />
            <Badge 
              variant={provider.status === 'active' ? 'success' : 
                      provider.status === 'inactive' ? 'warning' : 'destructive'}
              className="text-xs"
            >
              {provider.status}
            </Badge>
          </div>
        </div>
      </CardHeader>
      
      <CardContent>
        <div className="space-y-4">
          {/* Provider Info */}
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Type:</span>
            <Badge variant="outline" className="text-xs">
              {provider.type.toUpperCase()}
            </Badge>
          </div>
          
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Models:</span>
            <span className="font-medium">{provider.models.length} available</span>
          </div>

          {provider.usage && (
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Requests:</span>
              <span className="font-medium">{formatCompactNumber(provider.usage.requests)}</span>
            </div>
          )}

          {/* Metrics (collapsed) */}
          {!isExpanded && provider.performance && (
            <div className="grid grid-cols-3 gap-2 pt-2 border-t">
              <div className="text-center">
                <div className="text-sm font-medium">{provider.performance.latency}ms</div>
                <div className="text-xs text-muted-foreground">Latency</div>
              </div>
              <div className="text-center">
                <div className="text-sm font-medium">{provider.performance.reliability}%</div>
                <div className="text-xs text-muted-foreground">Uptime</div>
              </div>
              <div className="text-center">
                <div className="text-sm font-medium">
                  {provider.cost && provider.usage ? `$${provider.usage.cost.toFixed(2)}` : 'Free'}
                </div>
                <div className="text-xs text-muted-foreground">Cost</div>
              </div>
            </div>
          )}

          {/* Expanded Metrics */}
          {isExpanded && (
            <div className="pt-4 border-t space-y-4">
              <ProviderMetrics provider={provider} />
              
              {/* Models List */}
              <div>
                <h4 className="text-sm font-medium mb-2">Available Models</h4>
                <div className="flex flex-wrap gap-1">
                  {provider.models.map((model) => (
                    <Badge key={model} variant="outline" className="text-xs">
                      {model}
                    </Badge>
                  ))}
                </div>
              </div>
              
              {/* Cost Information */}
              {provider.cost && (
                <div>
                  <h4 className="text-sm font-medium mb-2">Pricing</h4>
                  <div className="text-xs text-muted-foreground space-y-1">
                    <div>Input: ${provider.cost.input} {provider.cost.unit}</div>
                    {provider.cost.output > 0 && (
                      <div>Output: ${provider.cost.output} {provider.cost.unit}</div>
                    )}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center justify-between pt-4 border-t">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setIsExpanded(!isExpanded)}
            >
              {isExpanded ? 'Show Less' : 'Show More'}
            </Button>
            
            <div className="flex items-center gap-1">
              <Button variant="outline" size="sm">
                <TestTube className="h-3 w-3" />
              </Button>
              <Button variant="outline" size="sm">
                <Settings className="h-3 w-3" />
              </Button>
              <Button variant="outline" size="sm">
                <RefreshCw className="h-3 w-3" />
              </Button>
              <Button variant="outline" size="sm">
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// Provider Stats Component
function ProviderStats({ providers }: { providers: Provider[] }) {
  const activeProviders = providers.filter(p => p.status === 'active').length;
  const totalRequests = providers.reduce((sum, p) => sum + (p.usage?.requests || 0), 0);
  const totalCost = providers.reduce((sum, p) => sum + (p.usage?.cost || 0), 0);
  const avgLatency = providers
    .filter(p => p.status === 'active' && p.performance?.latency)
    .reduce((sum, p, _, arr) => sum + (p.performance!.latency / arr.length), 0);

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <CheckCircle className="h-4 w-4 text-green-500" />
            <div>
              <div className="text-2xl font-bold">{activeProviders}</div>
              <div className="text-xs text-muted-foreground">Active Providers</div>
            </div>
          </div>
        </CardContent>
      </Card>
      
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-blue-500" />
            <div>
              <div className="text-2xl font-bold">{formatCompactNumber(totalRequests)}</div>
              <div className="text-xs text-muted-foreground">Total Requests</div>
            </div>
          </div>
        </CardContent>
      </Card>
      
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <BarChart3 className="h-4 w-4 text-purple-500" />
            <div>
              <div className="text-2xl font-bold">{Math.round(avgLatency)}ms</div>
              <div className="text-xs text-muted-foreground">Avg Latency</div>
            </div>
          </div>
        </CardContent>
      </Card>
      
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <DollarSign className="h-4 w-4 text-green-500" />
            <div>
              <div className="text-2xl font-bold">${totalCost.toFixed(2)}</div>
              <div className="text-xs text-muted-foreground">Total Cost</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Add Provider Modal Component
function AddProviderDialog({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border rounded-lg shadow-lg p-6 w-96 max-h-[80vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Add New Provider</h2>
          <Button variant="ghost" size="sm" onClick={onClose}>×</Button>
        </div>
        
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium mb-2 block">Provider Type</label>
            <select className="w-full p-2 border rounded-md bg-background">
              <option value="llm">LLM Provider</option>
              <option value="embedder">Embedding Provider</option>
              <option value="both">Both</option>
            </select>
          </div>
          
          <div>
            <label className="text-sm font-medium mb-2 block">Name</label>
            <Input placeholder="Provider name" />
          </div>
          
          <div>
            <label className="text-sm font-medium mb-2 block">Endpoint URL</label>
            <Input placeholder="https://api.example.com/v1" />
          </div>
          
          <div>
            <label className="text-sm font-medium mb-2 block">API Key</label>
            <Input type="password" placeholder="Your API key" />
          </div>
          
          <div>
            <label className="text-sm font-medium mb-2 block">Models (comma separated)</label>
            <Input placeholder="gpt-4, gpt-3.5-turbo" />
          </div>
          
          <div className="flex justify-end gap-2 pt-4 border-t">
            <Button variant="outline" onClick={onClose}>Cancel</Button>
            <Button onClick={onClose}>Add Provider</Button>
          </div>
        </div>
      </div>
    </div>
  );
}

// Main Providers Component
export function Providers() {
  const [searchTerm, setSearchTerm] = useState('');
  const [typeFilter, setTypeFilter] = useState<'all' | 'llm' | 'embedder' | 'both'>('all');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive' | 'error'>('all');
  const [showAddDialog, setShowAddDialog] = useState(false);
  
  const providers = mockProviders;

  // Filter providers
  const filteredProviders = providers.filter(provider => {
    const matchesSearch = provider.name.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesType = typeFilter === 'all' || provider.type === typeFilter;
    const matchesStatus = statusFilter === 'all' || provider.status === statusFilter;
    return matchesSearch && matchesType && matchesStatus;
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">AI Providers</h1>
          <p className="text-muted-foreground">
            Manage your LLM and embedding service providers
          </p>
        </div>
        
        <Button onClick={() => setShowAddDialog(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Add Provider
        </Button>
      </div>

      {/* Stats */}
      <ProviderStats providers={providers} />

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-4 items-center justify-between">
        <div className="flex items-center gap-2 flex-1">
          <Input
            placeholder="Search providers..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="max-w-md"
          />
          
          <select 
            className="p-2 border rounded-md bg-background"
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value as any)}
          >
            <option value="all">All Types</option>
            <option value="llm">LLM</option>
            <option value="embedder">Embedder</option>
            <option value="both">Both</option>
          </select>
          
          <select 
            className="p-2 border rounded-md bg-background"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as any)}
          >
            <option value="all">All Status</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
            <option value="error">Error</option>
          </select>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm">
            <TestTube className="h-4 w-4 mr-1" />
            Test All
          </Button>
          <Button variant="outline" size="sm">
            <RefreshCw className="h-4 w-4 mr-1" />
            Refresh
          </Button>
        </div>
      </div>

      {/* Providers Grid */}
      {filteredProviders.length === 0 ? (
        <Card>
          <CardContent className="p-12 text-center">
            <Key className="h-12 w-12 mx-auto mb-4 text-muted-foreground opacity-50" />
            <h3 className="text-lg font-medium mb-2">No providers found</h3>
            <p className="text-muted-foreground mb-4">
              {searchTerm || typeFilter !== 'all' || statusFilter !== 'all' 
                ? 'Try adjusting your search or filters'
                : 'Add your first AI provider to get started'
              }
            </p>
            <Button onClick={() => setShowAddDialog(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Add Provider
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {filteredProviders.map((provider) => (
            <ProviderCard key={provider.id} provider={provider} />
          ))}
        </div>
      )}

      {/* Add Provider Dialog */}
      <AddProviderDialog 
        isOpen={showAddDialog} 
        onClose={() => setShowAddDialog(false)} 
      />
    </div>
  );
}