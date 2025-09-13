import { useState } from 'react'
import { Search, Filter, Zap } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs'
import { Slider } from './ui/slider'
import { Switch } from './ui/switch'
import { Badge } from './ui/badge'
import { ScrollArea } from './ui/scroll-area'
import { apiClient, SearchRequest, SearchResult } from '../lib/api'
import { Alert, AlertDescription } from './ui/alert'

export function SearchTab() {
  const [searchRequest, setSearchRequest] = useState<SearchRequest>({
    query: '',
    top_k: 10,
    score_threshold: 0,
    hybrid_search: false,
    vector_weight: 0.7,
    include_content: true,
  })
  
  const [searchResults, setSearchResults] = useState<{
    semantic: SearchResult[]
    hybrid: SearchResult[]
    filtered: SearchResult[]
  }>({
    semantic: [],
    hybrid: [],
    filtered: [],
  })
  
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState('semantic')
  const [filters, setFilters] = useState<Record<string, string>>({})

  const handleSemanticSearch = async () => {
    if (!searchRequest.query) return
    
    setLoading(true)
    setError(null)
    
    try {
      const response = await apiClient.semanticSearch(searchRequest)
      if (response.error) {
        setError(response.error)
      } else if (response.data) {
        setSearchResults(prev => ({
          ...prev,
          semantic: response.data!.results || []
        }))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setLoading(false)
    }
  }

  const handleHybridSearch = async () => {
    if (!searchRequest.query) return
    
    setLoading(true)
    setError(null)
    
    try {
      const response = await apiClient.hybridSearch({
        ...searchRequest,
        hybrid_search: true,
      })
      if (response.error) {
        setError(response.error)
      } else if (response.data) {
        setSearchResults(prev => ({
          ...prev,
          hybrid: response.data!.results || []
        }))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setLoading(false)
    }
  }

  const handleFilteredSearch = async () => {
    if (!searchRequest.query) return
    
    setLoading(true)
    setError(null)
    
    try {
      const response = await apiClient.filteredSearch({
        ...searchRequest,
        filters: Object.keys(filters).length > 0 ? filters : undefined,
      })
      if (response.error) {
        setError(response.error)
      } else if (response.data) {
        setSearchResults(prev => ({
          ...prev,
          filtered: response.data!.results || []
        }))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setLoading(false)
    }
  }

  const renderSearchResults = (results: SearchResult[]) => {
    if (results.length === 0) {
      return (
        <div className="text-center text-muted-foreground py-8">
          No results found. Try adjusting your search parameters.
        </div>
      )
    }

    return (
      <ScrollArea className="h-[400px] pr-4">
        <div className="space-y-4">
          {results.map((result, index) => (
            <Card key={result.id || index} className="border-l-4 border-l-primary/50">
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <CardTitle className="text-sm font-medium">
                    Document: {result.id}
                  </CardTitle>
                  <Badge variant="secondary">
                    Score: {result.score.toFixed(3)}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent>
                {searchRequest.include_content && (
                  <p className="text-sm text-muted-foreground line-clamp-3">
                    {result.content}
                  </p>
                )}
                {result.metadata && Object.keys(result.metadata).length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1">
                    {Object.entries(result.metadata).map(([key, value]) => (
                      <Badge key={key} variant="outline" className="text-xs">
                        {key}: {String(value)}
                      </Badge>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      </ScrollArea>
    )
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Advanced Search</CardTitle>
          <CardDescription>
            Explore different search strategies with fine-tuned parameters
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Search Query Input */}
          <div className="space-y-2">
            <Label htmlFor="query">Search Query</Label>
            <div className="flex gap-2">
              <Input
                id="query"
                value={searchRequest.query}
                onChange={(e) => setSearchRequest(prev => ({ ...prev, query: e.target.value }))}
                placeholder="Enter your search query..."
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    if (activeTab === 'semantic') handleSemanticSearch()
                    else if (activeTab === 'hybrid') handleHybridSearch()
                    else if (activeTab === 'filtered') handleFilteredSearch()
                  }
                }}
              />
            </div>
          </div>

          {/* Search Parameters */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="top_k">Top K Results: {searchRequest.top_k}</Label>
              <Slider
                id="top_k"
                min={1}
                max={50}
                step={1}
                value={[searchRequest.top_k || 10]}
                onValueChange={(value: number[]) => setSearchRequest(prev => ({ ...prev, top_k: value[0] }))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="score_threshold">
                Score Threshold: {searchRequest.score_threshold?.toFixed(2) || '0.00'}
              </Label>
              <Slider
                id="score_threshold"
                min={0}
                max={1}
                step={0.05}
                value={[searchRequest.score_threshold || 0]}
                onValueChange={(value: number[]) => setSearchRequest(prev => ({ ...prev, score_threshold: value[0] }))}
              />
            </div>
          </div>

          {/* Hybrid Search Settings */}
          <div className="space-y-2 p-4 border rounded-lg">
            <div className="flex items-center justify-between">
              <Label htmlFor="hybrid">Enable Hybrid Search</Label>
              <Switch
                id="hybrid"
                checked={searchRequest.hybrid_search}
                onCheckedChange={(checked: boolean) => setSearchRequest(prev => ({ ...prev, hybrid_search: checked }))}
              />
            </div>
            
            {searchRequest.hybrid_search && (
              <div className="space-y-2 mt-4">
                <Label htmlFor="vector_weight">
                  Vector Weight: {((searchRequest.vector_weight || 0.7) * 100).toFixed(0)}%
                </Label>
                <Slider
                  id="vector_weight"
                  min={0}
                  max={1}
                  step={0.05}
                  value={[searchRequest.vector_weight || 0.7]}
                  onValueChange={(value: number[]) => setSearchRequest(prev => ({ ...prev, vector_weight: value[0] }))}
                />
                <p className="text-xs text-muted-foreground">
                  Keyword Weight: {((1 - (searchRequest.vector_weight || 0.7)) * 100).toFixed(0)}%
                </p>
              </div>
            )}
          </div>

          {/* Include Content Toggle */}
          <div className="flex items-center justify-between p-4 border rounded-lg">
            <Label htmlFor="include_content">Include Full Content in Results</Label>
            <Switch
              id="include_content"
              checked={searchRequest.include_content}
              onCheckedChange={(checked: boolean) => setSearchRequest(prev => ({ ...prev, include_content: checked }))}
            />
          </div>
        </CardContent>
      </Card>

      {/* Search Results Tabs */}
      <Card>
        <CardHeader>
          <CardTitle>Search Results</CardTitle>
        </CardHeader>
        <CardContent>
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="semantic" className="flex items-center gap-2">
                <Search className="h-4 w-4" />
                Semantic
              </TabsTrigger>
              <TabsTrigger value="hybrid" className="flex items-center gap-2">
                <Zap className="h-4 w-4" />
                Hybrid
              </TabsTrigger>
              <TabsTrigger value="filtered" className="flex items-center gap-2">
                <Filter className="h-4 w-4" />
                Filtered
              </TabsTrigger>
            </TabsList>

            <TabsContent value="semantic" className="space-y-4">
              <div className="flex justify-between items-center">
                <p className="text-sm text-muted-foreground">
                  Pure vector similarity search
                </p>
                <Button
                  onClick={handleSemanticSearch}
                  disabled={loading || !searchRequest.query}
                  size="sm"
                >
                  <Search className="h-4 w-4 mr-2" />
                  Search
                </Button>
              </div>
              {error && (
                <Alert variant="destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              {renderSearchResults(searchResults.semantic)}
            </TabsContent>

            <TabsContent value="hybrid" className="space-y-4">
              <div className="flex justify-between items-center">
                <p className="text-sm text-muted-foreground">
                  Combined vector and keyword search
                </p>
                <Button
                  onClick={handleHybridSearch}
                  disabled={loading || !searchRequest.query}
                  size="sm"
                >
                  <Zap className="h-4 w-4 mr-2" />
                  Search
                </Button>
              </div>
              {error && (
                <Alert variant="destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              {renderSearchResults(searchResults.hybrid)}
            </TabsContent>

            <TabsContent value="filtered" className="space-y-4">
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>Metadata Filters</Label>
                  <div className="flex gap-2">
                    <Input
                      placeholder="Key"
                      id="filter-key"
                    />
                    <Input
                      placeholder="Value"
                      id="filter-value"
                    />
                    <Button
                      size="sm"
                      onClick={() => {
                        const key = (document.getElementById('filter-key') as HTMLInputElement)?.value
                        const value = (document.getElementById('filter-value') as HTMLInputElement)?.value
                        if (key && value) {
                          setFilters(prev => ({ ...prev, [key]: value }))
                        }
                      }}
                    >
                      Add Filter
                    </Button>
                  </div>
                  <div className="flex flex-wrap gap-1">
                    {Object.entries(filters).map(([key, value]) => (
                      <Badge
                        key={key}
                        variant="secondary"
                        className="cursor-pointer"
                        onClick={() => setFilters(prev => {
                          const newFilters = { ...prev }
                          delete newFilters[key]
                          return newFilters
                        })}
                      >
                        {key}: {value} ✕
                      </Badge>
                    ))}
                  </div>
                </div>
                
                <div className="flex justify-between items-center">
                  <p className="text-sm text-muted-foreground">
                    Search with metadata filtering
                  </p>
                  <Button
                    onClick={handleFilteredSearch}
                    disabled={loading || !searchRequest.query}
                    size="sm"
                  >
                    <Filter className="h-4 w-4 mr-2" />
                    Search
                  </Button>
                </div>
              </div>
              {error && (
                <Alert variant="destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              {renderSearchResults(searchResults.filtered)}
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  )
}