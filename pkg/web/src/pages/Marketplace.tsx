import React, { useState } from 'react';
import { 
  Search, 
  Star, 
  Download, 
  Heart, 
  Filter,
  Grid,
  List,
  Sparkles,
  TrendingUp,
  Clock,
  User,
  Package,
  Shield,
  ExternalLink,
  Eye,
  Tag
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { cn, formatCompactNumber, formatTimeAgo } from '@/utils';
import { AgentTemplate } from '@/types';

// Mock marketplace data
const mockTemplates: AgentTemplate[] = [
  {
    id: '1',
    name: 'Document Analyzer Pro',
    description: 'Advanced document analysis with AI-powered insights, extraction, and summarization capabilities.',
    author: 'RAGO Labs',
    version: '2.1.0',
    category: 'document-processing',
    tags: ['analysis', 'extraction', 'summarization', 'ai'],
    icon: '📄',
    screenshots: ['/api/screenshots/doc-analyzer-1.png', '/api/screenshots/doc-analyzer-2.png'],
    readme: '# Document Analyzer Pro\n\nAdvanced document analysis tool...',
    workflow: {} as any, // Mock workflow object
    downloads: 12500,
    rating: 4.8,
    reviews: 89,
    createdAt: '2024-01-10T08:00:00Z',
    updatedAt: '2024-01-20T14:30:00Z',
    verified: true,
    dependencies: ['openai-gpt4', 'text-extraction'],
    license: 'MIT'
  },
  {
    id: '2',
    name: 'Smart Query Assistant',
    description: 'Intelligent query interface with natural language processing and contextual suggestions.',
    author: 'AI Collective',
    version: '1.5.2',
    category: 'query-tools',
    tags: ['nlp', 'query', 'assistant', 'suggestions'],
    icon: '🤖',
    screenshots: ['/api/screenshots/query-assistant-1.png'],
    readme: '# Smart Query Assistant\n\nEnhance your queries...',
    workflow: {} as any,
    downloads: 8900,
    rating: 4.6,
    reviews: 67,
    createdAt: '2024-01-08T12:00:00Z',
    updatedAt: '2024-01-22T09:15:00Z',
    verified: true,
    dependencies: ['nlp-processor'],
    license: 'Apache-2.0'
  },
  {
    id: '3',
    name: 'Data Pipeline Optimizer',
    description: 'Streamline your data processing workflows with automated optimization and monitoring.',
    author: 'DevFlow Studio',
    version: '3.0.1',
    category: 'automation',
    tags: ['pipeline', 'optimization', 'monitoring', 'automation'],
    icon: '⚡',
    screenshots: [],
    readme: '# Data Pipeline Optimizer\n\nOptimize your data pipelines...',
    workflow: {} as any,
    downloads: 15600,
    rating: 4.9,
    reviews: 142,
    createdAt: '2023-12-15T10:00:00Z',
    updatedAt: '2024-01-18T16:45:00Z',
    verified: true,
    dependencies: ['data-processor', 'monitoring-tools'],
    license: 'GPL-3.0'
  },
  {
    id: '4',
    name: 'Content Generator',
    description: 'Generate high-quality content using advanced AI models with customizable templates.',
    author: 'ContentAI Inc',
    version: '1.2.0',
    category: 'content-generation',
    tags: ['content', 'generation', 'templates', 'creative'],
    icon: '✍️',
    screenshots: ['/api/screenshots/content-gen-1.png', '/api/screenshots/content-gen-2.png'],
    readme: '# Content Generator\n\nCreate amazing content...',
    workflow: {} as any,
    downloads: 6700,
    rating: 4.3,
    reviews: 45,
    createdAt: '2024-01-05T14:00:00Z',
    updatedAt: '2024-01-19T11:20:00Z',
    verified: false,
    dependencies: ['openai-gpt4', 'template-engine'],
    license: 'MIT'
  },
  {
    id: '5',
    name: 'Workflow Builder',
    description: 'Visual workflow builder with drag-and-drop interface and pre-built components.',
    author: 'FlowCraft',
    version: '2.3.0',
    category: 'workflow-tools',
    tags: ['workflow', 'builder', 'visual', 'components'],
    icon: '🔧',
    screenshots: ['/api/screenshots/workflow-builder-1.png'],
    readme: '# Workflow Builder\n\nBuild complex workflows easily...',
    workflow: {} as any,
    downloads: 11200,
    rating: 4.7,
    reviews: 78,
    createdAt: '2023-12-28T09:00:00Z',
    updatedAt: '2024-01-21T13:30:00Z',
    verified: true,
    dependencies: ['flow-engine', 'ui-components'],
    license: 'BSD-3-Clause'
  }
];

const categories = [
  { id: 'all', name: 'All Categories', count: mockTemplates.length },
  { id: 'document-processing', name: 'Document Processing', count: 1 },
  { id: 'query-tools', name: 'Query Tools', count: 1 },
  { id: 'automation', name: 'Automation', count: 1 },
  { id: 'content-generation', name: 'Content Generation', count: 1 },
  { id: 'workflow-tools', name: 'Workflow Tools', count: 1 }
];

// Template Card Component
function TemplateCard({ template, viewMode }: { template: AgentTemplate; viewMode: 'grid' | 'list' }) {
  if (viewMode === 'list') {
    return (
      <Card className="hover:shadow-md transition-shadow">
        <CardContent className="p-4">
          <div className="flex items-center gap-4">
            <div className="text-4xl">{template.icon}</div>
            
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <h3 className="font-semibold text-lg truncate">{template.name}</h3>
                {template.verified && (
                  <Shield className="h-4 w-4 text-blue-500" />
                )}
                <Badge variant="outline" className="text-xs">
                  v{template.version}
                </Badge>
              </div>
              
              <p className="text-sm text-muted-foreground mb-2 line-clamp-2">
                {template.description}
              </p>
              
              <div className="flex items-center gap-4 text-xs text-muted-foreground">
                <div className="flex items-center gap-1">
                  <User className="h-3 w-3" />
                  {template.author}
                </div>
                <div className="flex items-center gap-1">
                  <Download className="h-3 w-3" />
                  {formatCompactNumber(template.downloads)}
                </div>
                <div className="flex items-center gap-1">
                  <Star className="h-3 w-3 fill-yellow-400 text-yellow-400" />
                  {template.rating} ({template.reviews})
                </div>
                <div>Updated {formatTimeAgo(template.updatedAt)}</div>
              </div>
              
              <div className="flex flex-wrap gap-1 mt-2">
                {template.tags.slice(0, 3).map((tag) => (
                  <Badge key={tag} variant="secondary" className="text-xs">
                    {tag}
                  </Badge>
                ))}
                {template.tags.length > 3 && (
                  <Badge variant="secondary" className="text-xs">
                    +{template.tags.length - 3}
                  </Badge>
                )}
              </div>
            </div>
            
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm">
                <Eye className="h-4 w-4 mr-1" />
                View
              </Button>
              <Button size="sm">
                <Download className="h-4 w-4 mr-1" />
                Install
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="hover:shadow-md transition-shadow h-full">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="text-4xl mb-2">{template.icon}</div>
          <div className="flex items-center gap-1">
            {template.verified && (
              <Shield className="h-4 w-4 text-blue-500" />
            )}
            <Button variant="ghost" size="icon" className="h-6 w-6">
              <Heart className="h-3 w-3" />
            </Button>
          </div>
        </div>
        
        <CardTitle className="text-base line-clamp-1">{template.name}</CardTitle>
        <CardDescription className="text-sm line-clamp-2">
          {template.description}
        </CardDescription>
      </CardHeader>
      
      <CardContent className="pt-0">
        <div className="space-y-3">
          {/* Author and Version */}
          <div className="flex items-center justify-between text-sm">
            <div className="flex items-center gap-1 text-muted-foreground">
              <User className="h-3 w-3" />
              {template.author}
            </div>
            <Badge variant="outline" className="text-xs">
              v{template.version}
            </Badge>
          </div>
          
          {/* Stats */}
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="flex items-center gap-1">
              <Download className="h-3 w-3 text-muted-foreground" />
              {formatCompactNumber(template.downloads)}
            </div>
            <div className="flex items-center gap-1">
              <Star className="h-3 w-3 fill-yellow-400 text-yellow-400" />
              {template.rating}
            </div>
          </div>
          
          {/* Tags */}
          <div className="flex flex-wrap gap-1">
            {template.tags.slice(0, 2).map((tag) => (
              <Badge key={tag} variant="secondary" className="text-xs">
                {tag}
              </Badge>
            ))}
            {template.tags.length > 2 && (
              <Badge variant="secondary" className="text-xs">
                +{template.tags.length - 2}
              </Badge>
            )}
          </div>
          
          {/* Updated */}
          <div className="text-xs text-muted-foreground">
            Updated {formatTimeAgo(template.updatedAt)}
          </div>
          
          {/* Actions */}
          <div className="flex gap-2 pt-2 border-t">
            <Button variant="outline" size="sm" className="flex-1">
              <Eye className="h-3 w-3 mr-1" />
              View
            </Button>
            <Button size="sm" className="flex-1">
              <Download className="h-3 w-3 mr-1" />
              Install
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// Featured Templates Section
function FeaturedTemplates({ templates }: { templates: AgentTemplate[] }) {
  const featured = templates.filter(t => t.verified && t.downloads > 10000).slice(0, 3);
  
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Sparkles className="h-5 w-5" />
          Featured Templates
        </CardTitle>
        <CardDescription>Popular and verified templates from the community</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {featured.map((template) => (
            <div key={template.id} className="text-center">
              <div className="text-3xl mb-2">{template.icon}</div>
              <h4 className="font-medium text-sm mb-1">{template.name}</h4>
              <p className="text-xs text-muted-foreground mb-2 line-clamp-2">
                {template.description}
              </p>
              <div className="flex items-center justify-center gap-4 text-xs text-muted-foreground mb-2">
                <div className="flex items-center gap-1">
                  <Download className="h-3 w-3" />
                  {formatCompactNumber(template.downloads)}
                </div>
                <div className="flex items-center gap-1">
                  <Star className="h-3 w-3 fill-yellow-400 text-yellow-400" />
                  {template.rating}
                </div>
              </div>
              <Button variant="outline" size="sm" className="w-full">
                Install Template
              </Button>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

// Stats Section
function MarketplaceStats({ templates }: { templates: AgentTemplate[] }) {
  const totalDownloads = templates.reduce((sum, t) => sum + t.downloads, 0);
  const avgRating = templates.reduce((sum, t) => sum + t.rating, 0) / templates.length;
  const verifiedCount = templates.filter(t => t.verified).length;
  const recentlyUpdated = templates.filter(t => 
    new Date(t.updatedAt) > new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)
  ).length;

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <Package className="h-4 w-4 text-blue-500" />
            <div>
              <div className="text-2xl font-bold">{templates.length}</div>
              <div className="text-xs text-muted-foreground">Total Templates</div>
            </div>
          </div>
        </CardContent>
      </Card>
      
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <Download className="h-4 w-4 text-green-500" />
            <div>
              <div className="text-2xl font-bold">{formatCompactNumber(totalDownloads)}</div>
              <div className="text-xs text-muted-foreground">Total Downloads</div>
            </div>
          </div>
        </CardContent>
      </Card>
      
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <Star className="h-4 w-4 text-yellow-500" />
            <div>
              <div className="text-2xl font-bold">{avgRating.toFixed(1)}</div>
              <div className="text-xs text-muted-foreground">Avg Rating</div>
            </div>
          </div>
        </CardContent>
      </Card>
      
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            <Shield className="h-4 w-4 text-purple-500" />
            <div>
              <div className="text-2xl font-bold">{verifiedCount}</div>
              <div className="text-xs text-muted-foreground">Verified</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

// Main Marketplace Component
export function Marketplace() {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState('all');
  const [sortBy, setSortBy] = useState<'name' | 'downloads' | 'rating' | 'updated'>('downloads');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');

  // Filter and sort templates
  const filteredTemplates = mockTemplates
    .filter(template => {
      const matchesSearch = template.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           template.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           template.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase()));
      const matchesCategory = selectedCategory === 'all' || template.category === selectedCategory;
      return matchesSearch && matchesCategory;
    })
    .sort((a, b) => {
      let aVal: any, bVal: any;
      
      switch (sortBy) {
        case 'name':
          aVal = a.name;
          bVal = b.name;
          break;
        case 'downloads':
          aVal = a.downloads;
          bVal = b.downloads;
          break;
        case 'rating':
          aVal = a.rating;
          bVal = b.rating;
          break;
        case 'updated':
          aVal = new Date(a.updatedAt);
          bVal = new Date(b.updatedAt);
          break;
        default:
          return 0;
      }
      
      if (aVal < bVal) return sortOrder === 'asc' ? -1 : 1;
      if (aVal > bVal) return sortOrder === 'asc' ? 1 : -1;
      return 0;
    });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Agent Marketplace</h1>
        <p className="text-muted-foreground">
          Discover and install community-created workflow templates and agents
        </p>
      </div>

      {/* Stats */}
      <MarketplaceStats templates={mockTemplates} />

      {/* Featured Templates */}
      <FeaturedTemplates templates={mockTemplates} />

      {/* Search and Filters */}
      <div className="flex flex-col lg:flex-row gap-4 items-center justify-between">
        <div className="flex items-center gap-4 flex-1">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search templates, tags, or authors..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-10"
            />
          </div>
          
          <select 
            className="p-2 border rounded-md bg-background min-w-[150px]"
            value={selectedCategory}
            onChange={(e) => setSelectedCategory(e.target.value)}
          >
            {categories.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name} ({category.count})
              </option>
            ))}
          </select>
          
          <select 
            className="p-2 border rounded-md bg-background"
            value={`${sortBy}-${sortOrder}`}
            onChange={(e) => {
              const [sort, order] = e.target.value.split('-');
              setSortBy(sort as any);
              setSortOrder(order as any);
            }}
          >
            <option value="downloads-desc">Most Downloaded</option>
            <option value="rating-desc">Highest Rated</option>
            <option value="updated-desc">Recently Updated</option>
            <option value="name-asc">Name (A-Z)</option>
          </select>
        </div>
        
        <div className="flex items-center gap-2">
          <Button
            variant={viewMode === 'grid' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('grid')}
          >
            <Grid className="h-4 w-4" />
          </Button>
          <Button
            variant={viewMode === 'list' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setViewMode('list')}
          >
            <List className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Templates Grid/List */}
      {filteredTemplates.length === 0 ? (
        <Card>
          <CardContent className="p-12 text-center">
            <Package className="h-12 w-12 mx-auto mb-4 text-muted-foreground opacity-50" />
            <h3 className="text-lg font-medium mb-2">No templates found</h3>
            <p className="text-muted-foreground">
              Try adjusting your search terms or filters to find more templates.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-medium">
              Templates ({filteredTemplates.length})
            </h2>
            <div className="text-sm text-muted-foreground">
              Sorted by {sortBy} ({sortOrder === 'desc' ? 'descending' : 'ascending'})
            </div>
          </div>
          
          {viewMode === 'grid' ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {filteredTemplates.map((template) => (
                <TemplateCard key={template.id} template={template} viewMode="grid" />
              ))}
            </div>
          ) : (
            <div className="space-y-3">
              {filteredTemplates.map((template) => (
                <TemplateCard key={template.id} template={template} viewMode="list" />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}