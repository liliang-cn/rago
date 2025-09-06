import React, { useState, useRef, useEffect } from 'react';
import { 
  Send, 
  Search, 
  FileText, 
  Clock, 
  Sparkles, 
  Copy, 
  ThumbsUp, 
  ThumbsDown,
  Settings,
  History,
  Trash2,
  Download
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { useDocumentsStore } from '@/store';
import { cn, formatTimeAgo } from '@/utils';
import { QueryResult, DocumentChunk } from '@/types';
import * as Dialog from '@radix-ui/react-dialog';
import * as Collapsible from '@radix-ui/react-collapsible';

// Mock query history - replace with real data from store
const mockQueryHistory = [
  {
    id: '1',
    query: 'What are the main features of the platform?',
    response: 'The platform offers several key features including document processing, AI-powered search, workflow automation, and real-time monitoring.',
    timestamp: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
    processingTime: 1250,
    sources: []
  },
  {
    id: '2',
    query: 'How to configure providers?',
    response: 'Provider configuration involves setting up endpoints, API keys, and model preferences in the settings panel.',
    timestamp: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
    processingTime: 980,
    sources: []
  }
];

// Query Settings Component
function QuerySettings({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  return (
    <Dialog.Root open={isOpen} onOpenChange={onClose}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50" />
        <Dialog.Content className="fixed top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 bg-background border rounded-lg shadow-lg p-6 w-96 z-50">
          <Dialog.Title className="text-lg font-semibold mb-4">Query Settings</Dialog.Title>
          
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium mb-2 block">Model</label>
              <select className="w-full p-2 border rounded-md bg-background">
                <option>GPT-4</option>
                <option>GPT-3.5</option>
                <option>Claude-3</option>
              </select>
            </div>
            
            <div>
              <label className="text-sm font-medium mb-2 block">Max Results</label>
              <Input type="number" defaultValue="5" min="1" max="20" />
            </div>
            
            <div>
              <label className="text-sm font-medium mb-2 block">Relevance Threshold</label>
              <Input type="number" defaultValue="0.7" min="0" max="1" step="0.1" />
            </div>
            
            <div className="flex items-center gap-2">
              <input type="checkbox" id="show-sources" defaultChecked />
              <label htmlFor="show-sources" className="text-sm">Show source documents</label>
            </div>
          </div>
          
          <div className="flex justify-end gap-2 mt-6">
            <Button variant="outline" onClick={onClose}>Cancel</Button>
            <Button onClick={onClose}>Save Settings</Button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

// Source Document Component
function SourceDocument({ source }: { source: DocumentChunk }) {
  return (
    <Card className="border-l-4 border-l-primary">
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm">Document Source</CardTitle>
          <Badge variant="secondary" className="text-xs">
            Score: {source.score.toFixed(2)}
          </Badge>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          <div className="text-sm bg-muted p-3 rounded-md">
            {source.content}
          </div>
          <div className="text-xs text-muted-foreground">
            Position: {source.startOffset}-{source.endOffset}
            {source.metadata && (
              <span className="ml-2">
                • Page: {source.metadata.page || 'N/A'}
              </span>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// Query Result Component
function QueryResultCard({ result }: { result: QueryResult }) {
  const [showSources, setShowSources] = useState(false);

  return (
    <Card className="mb-4">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <CardTitle className="text-base">{result.query}</CardTitle>
            <CardDescription className="flex items-center gap-2 mt-1">
              <Clock className="h-3 w-3" />
              {formatTimeAgo(result.timestamp)} • {result.processingTime}ms
            </CardDescription>
          </div>
          
          <div className="flex items-center gap-1">
            <Button variant="ghost" size="icon" className="h-6 w-6">
              <Copy className="h-3 w-3" />
            </Button>
            <Button variant="ghost" size="icon" className="h-6 w-6">
              <ThumbsUp className="h-3 w-3" />
            </Button>
            <Button variant="ghost" size="icon" className="h-6 w-6">
              <ThumbsDown className="h-3 w-3" />
            </Button>
          </div>
        </div>
      </CardHeader>
      
      <CardContent>
        <div className="space-y-4">
          <div className="prose prose-sm max-w-none dark:prose-invert">
            <p className="whitespace-pre-wrap">{result.response}</p>
          </div>
          
          {result.sources && result.sources.length > 0 && (
            <Collapsible.Root open={showSources} onOpenChange={setShowSources}>
              <Collapsible.Trigger asChild>
                <Button variant="outline" size="sm" className="w-full">
                  <FileText className="h-4 w-4 mr-2" />
                  {showSources ? 'Hide' : 'Show'} Sources ({result.sources.length})
                </Button>
              </Collapsible.Trigger>
              
              <Collapsible.Content className="space-y-2 mt-4">
                {result.sources.map((source, index) => (
                  <SourceDocument key={index} source={source} />
                ))}
              </Collapsible.Content>
            </Collapsible.Root>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

// Query History Sidebar
function QueryHistory({ 
  history, 
  onSelectQuery,
  onClearHistory 
}: { 
  history: any[]; 
  onSelectQuery: (query: string) => void;
  onClearHistory: () => void;
}) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">Query History</CardTitle>
          <Button variant="ghost" size="icon" onClick={onClearHistory}>
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div className="max-h-96 overflow-y-auto">
          {history.length === 0 ? (
            <div className="p-4 text-center text-muted-foreground">
              <History className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p className="text-sm">No queries yet</p>
            </div>
          ) : (
            <div className="space-y-1 p-2">
              {history.map((item) => (
                <button
                  key={item.id}
                  onClick={() => onSelectQuery(item.query)}
                  className="w-full text-left p-2 rounded-md hover:bg-accent text-sm border-none bg-transparent"
                >
                  <p className="font-medium truncate">{item.query}</p>
                  <p className="text-xs text-muted-foreground">
                    {formatTimeAgo(item.timestamp)}
                  </p>
                </button>
              ))}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

// Main Query Component
export function Query() {
  const [query, setQuery] = useState('');
  const [isQuerying, setIsQuerying] = useState(false);
  const [results, setResults] = useState<QueryResult[]>([]);
  const [showSettings, setShowSettings] = useState(false);
  const [queryHistory] = useState(mockQueryHistory);
  const inputRef = useRef<HTMLInputElement>(null);
  
  const { loading } = useDocumentsStore();

  const handleSubmitQuery = async (queryText?: string) => {
    const searchQuery = queryText || query.trim();
    if (!searchQuery || isQuerying) return;

    setIsQuerying(true);
    
    try {
      // Simulate API call - replace with real API
      const mockResult: QueryResult = {
        id: crypto.randomUUID(),
        query: searchQuery,
        response: `This is a simulated response for the query: "${searchQuery}". In a real implementation, this would be the AI-generated answer based on your document knowledge base.`,
        sources: [
          {
            id: '1',
            documentId: 'doc1',
            content: 'This is a sample chunk of text from a document that was used to generate the response.',
            metadata: { page: 5, section: 'Introduction' },
            score: 0.89,
            startOffset: 100,
            endOffset: 200
          }
        ],
        timestamp: new Date().toISOString(),
        model: 'gpt-4',
        processingTime: Math.floor(Math.random() * 2000) + 500,
        relevanceScore: 0.85
      };

      // Simulate processing delay
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      setResults(prev => [mockResult, ...prev]);
      setQuery('');
    } catch (error) {
      console.error('Query failed:', error);
    } finally {
      setIsQuerying(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmitQuery();
    }
  };

  const clearHistory = () => {
    // Implement clear history logic
  };

  const exportResults = () => {
    const dataStr = JSON.stringify(results, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'query-results.json';
    link.click();
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
      {/* Main Query Interface */}
      <div className="lg:col-span-3 space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold">AI Query</h1>
          <p className="text-muted-foreground">
            Search and ask questions about your documents using AI
          </p>
        </div>

        {/* Query Input */}
        <Card>
          <CardContent className="p-6">
            <div className="flex gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  ref={inputRef}
                  placeholder="Ask me anything about your documents..."
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyPress={handleKeyPress}
                  className="pl-10 pr-4"
                  disabled={isQuerying}
                />
              </div>
              
              <Button 
                onClick={() => handleSubmitQuery()}
                disabled={!query.trim() || isQuerying}
                className="px-6"
              >
                {isQuerying ? (
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
                ) : (
                  <>
                    <Send className="h-4 w-4 mr-2" />
                    Query
                  </>
                )}
              </Button>
              
              <Button 
                variant="outline"
                onClick={() => setShowSettings(true)}
              >
                <Settings className="h-4 w-4" />
              </Button>
            </div>

            {/* Quick Actions */}
            <div className="flex items-center justify-between mt-4">
              <div className="flex gap-2">
                <Button 
                  variant="outline" 
                  size="sm"
                  onClick={() => setQuery('Summarize the main points from all documents')}
                >
                  <Sparkles className="h-4 w-4 mr-1" />
                  Summarize
                </Button>
                <Button 
                  variant="outline" 
                  size="sm"
                  onClick={() => setQuery('What are the key technical specifications?')}
                >
                  Technical Specs
                </Button>
                <Button 
                  variant="outline" 
                  size="sm"
                  onClick={() => setQuery('List the most important requirements')}
                >
                  Requirements
                </Button>
              </div>
              
              {results.length > 0 && (
                <Button variant="outline" size="sm" onClick={exportResults}>
                  <Download className="h-4 w-4 mr-1" />
                  Export Results
                </Button>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Results */}
        <div className="space-y-4">
          {isQuerying && (
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center gap-3">
                  <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-primary"></div>
                  <span className="text-sm">Processing your query...</span>
                </div>
              </CardContent>
            </Card>
          )}
          
          {results.map((result) => (
            <QueryResultCard key={result.id} result={result} />
          ))}
          
          {results.length === 0 && !isQuerying && (
            <Card>
              <CardContent className="p-12 text-center">
                <Search className="h-12 w-12 mx-auto mb-4 text-muted-foreground opacity-50" />
                <h3 className="text-lg font-medium mb-2">Ready to Search</h3>
                <p className="text-muted-foreground">
                  Ask questions about your documents and get AI-powered answers with source references.
                </p>
                <div className="mt-4 text-sm text-muted-foreground">
                  <p>Try asking:</p>
                  <ul className="list-disc list-inside mt-2 space-y-1">
                    <li>"What are the main topics covered?"</li>
                    <li>"Summarize the key findings"</li>
                    <li>"What are the technical requirements?"</li>
                  </ul>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Sidebar */}
      <div className="space-y-6">
        <QueryHistory 
          history={queryHistory}
          onSelectQuery={setQuery}
          onClearHistory={clearHistory}
        />
        
        {/* Query Stats */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Query Statistics</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Total Queries</span>
                <span className="font-medium">{results.length + queryHistory.length}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Avg Response Time</span>
                <span className="font-medium">1.2s</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Success Rate</span>
                <span className="font-medium text-green-600">98.5%</span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Settings Modal */}
      <QuerySettings isOpen={showSettings} onClose={() => setShowSettings(false)} />
    </div>
  );
}