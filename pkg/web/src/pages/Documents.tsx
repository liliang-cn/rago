import React, { useState, useCallback } from 'react';
import { useDropzone } from 'react-dropzone';
import { 
  Upload, 
  FileText, 
  Search, 
  Filter, 
  MoreHorizontal,
  Download,
  Trash2,
  Eye,
  RefreshCw,
  AlertCircle,
  CheckCircle,
  Clock,
  Grid,
  List
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { useDocumentsStore } from '@/store';
import { cn, formatFileSize, formatTimeAgo, getStatusColor } from '@/utils';
import { Document } from '@/types';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import * as Dialog from '@radix-ui/react-dialog';

// File Upload Component
function FileUpload() {
  const { addDocument, setLoading } = useDocumentsStore();
  
  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    setLoading('upload', true);
    
    for (const file of acceptedFiles) {
      try {
        // Simulate file upload - replace with actual API call
        const mockDocument: Document = {
          id: crypto.randomUUID(),
          name: file.name,
          type: file.type,
          size: file.size,
          uploadedAt: new Date().toISOString(),
          status: 'processing',
          chunks: 0
        };
        
        addDocument(mockDocument);
        
        // Simulate processing delay
        setTimeout(() => {
          useDocumentsStore.getState().updateDocument(mockDocument.id, {
            status: 'ready',
            chunks: Math.floor(file.size / 1000) + 1
          });
        }, 2000);
        
      } catch (error) {
        console.error('Upload failed:', error);
      }
    }
    
    setLoading('upload', false);
  }, [addDocument, setLoading]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'application/pdf': ['.pdf'],
      'text/plain': ['.txt'],
      'application/msword': ['.doc'],
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document': ['.docx'],
      'text/markdown': ['.md'],
    },
    multiple: true
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Upload Documents</CardTitle>
        <CardDescription>
          Add documents to your knowledge base. Supports PDF, DOCX, TXT, and MD files.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div
          {...getRootProps()}
          className={cn(
            "border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors",
            isDragActive 
              ? "border-primary bg-primary/5" 
              : "border-muted-foreground/25 hover:border-muted-foreground/50"
          )}
        >
          <input {...getInputProps()} />
          <Upload className="h-8 w-8 mx-auto mb-4 text-muted-foreground" />
          {isDragActive ? (
            <p className="text-lg">Drop the files here...</p>
          ) : (
            <div>
              <p className="text-lg mb-2">Drag & drop files here, or click to select</p>
              <p className="text-sm text-muted-foreground">
                Maximum file size: 10MB per file
              </p>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

// Document Card Component
function DocumentCard({ document, onSelect }: { document: Document; onSelect: (doc: Document) => void }) {
  const { removeDocument, updateDocument } = useDocumentsStore();

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'ready': return CheckCircle;
      case 'processing': return Clock;
      case 'error': return AlertCircle;
      default: return FileText;
    }
  };

  const StatusIcon = getStatusIcon(document.status);

  return (
    <Card className="hover:shadow-md transition-shadow cursor-pointer" onClick={() => onSelect(document)}>
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <FileText className="h-4 w-4 text-muted-foreground" />
            <div className="flex-1 min-w-0">
              <CardTitle className="text-sm truncate">{document.name}</CardTitle>
              <CardDescription className="text-xs">
                {formatFileSize(document.size)} • {formatTimeAgo(document.uploadedAt)}
              </CardDescription>
            </div>
          </div>
          
          <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild>
              <Button variant="ghost" size="icon" className="h-6 w-6">
                <MoreHorizontal className="h-3 w-3" />
              </Button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content className="w-48 bg-popover rounded-md border shadow-lg p-1">
                <DropdownMenu.Item className="flex items-center gap-2 px-2 py-1.5 text-sm cursor-pointer rounded-sm hover:bg-accent">
                  <Eye className="h-3 w-3" />
                  View Details
                </DropdownMenu.Item>
                <DropdownMenu.Item className="flex items-center gap-2 px-2 py-1.5 text-sm cursor-pointer rounded-sm hover:bg-accent">
                  <Download className="h-3 w-3" />
                  Download
                </DropdownMenu.Item>
                <DropdownMenu.Item 
                  className="flex items-center gap-2 px-2 py-1.5 text-sm cursor-pointer rounded-sm hover:bg-accent"
                  onClick={() => updateDocument(document.id, { status: 'processing' })}
                >
                  <RefreshCw className="h-3 w-3" />
                  Reprocess
                </DropdownMenu.Item>
                <DropdownMenu.Separator className="my-1 h-px bg-border" />
                <DropdownMenu.Item 
                  className="flex items-center gap-2 px-2 py-1.5 text-sm cursor-pointer rounded-sm hover:bg-accent text-destructive"
                  onClick={() => removeDocument(document.id)}
                >
                  <Trash2 className="h-3 w-3" />
                  Delete
                </DropdownMenu.Item>
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        </div>
      </CardHeader>
      
      <CardContent className="pt-0">
        <div className="flex items-center justify-between">
          <Badge 
            variant={document.status === 'ready' ? 'success' : 
                    document.status === 'processing' ? 'warning' : 'destructive'}
            className="text-xs"
          >
            <StatusIcon className="h-3 w-3 mr-1" />
            {document.status}
          </Badge>
          
          {document.chunks > 0 && (
            <span className="text-xs text-muted-foreground">
              {document.chunks} chunks
            </span>
          )}
        </div>
        
        {document.metadata && (
          <div className="mt-2 text-xs text-muted-foreground">
            Type: {document.type}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// Document List Component
function DocumentList({ documents, onSelect }: { documents: Document[]; onSelect: (doc: Document) => void }) {
  return (
    <div className="space-y-2">
      {documents.map((document) => (
        <Card key={document.id} className="hover:bg-accent/50 cursor-pointer" onClick={() => onSelect(document)}>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3 flex-1 min-w-0">
                <FileText className="h-4 w-4 text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="font-medium truncate">{document.name}</p>
                  <div className="flex items-center gap-4 text-xs text-muted-foreground">
                    <span>{formatFileSize(document.size)}</span>
                    <span>{formatTimeAgo(document.uploadedAt)}</span>
                    {document.chunks > 0 && <span>{document.chunks} chunks</span>}
                  </div>
                </div>
              </div>
              
              <div className="flex items-center gap-2">
                <Badge 
                  variant={document.status === 'ready' ? 'success' : 
                          document.status === 'processing' ? 'warning' : 'destructive'}
                  className="text-xs"
                >
                  {document.status}
                </Badge>
                
                <DropdownMenu.Root>
                  <DropdownMenu.Trigger asChild>
                    <Button variant="ghost" size="icon" className="h-6 w-6">
                      <MoreHorizontal className="h-3 w-3" />
                    </Button>
                  </DropdownMenu.Trigger>
                  <DropdownMenu.Portal>
                    <DropdownMenu.Content className="w-32 bg-popover rounded-md border shadow-lg p-1">
                      <DropdownMenu.Item className="flex items-center gap-2 px-2 py-1.5 text-xs cursor-pointer rounded-sm hover:bg-accent">
                        <Eye className="h-3 w-3" />
                        View
                      </DropdownMenu.Item>
                      <DropdownMenu.Item className="flex items-center gap-2 px-2 py-1.5 text-xs cursor-pointer rounded-sm hover:bg-accent">
                        <Download className="h-3 w-3" />
                        Download
                      </DropdownMenu.Item>
                      <DropdownMenu.Item className="flex items-center gap-2 px-2 py-1.5 text-xs cursor-pointer rounded-sm hover:bg-accent text-destructive">
                        <Trash2 className="h-3 w-3" />
                        Delete
                      </DropdownMenu.Item>
                    </DropdownMenu.Content>
                  </DropdownMenu.Portal>
                </DropdownMenu.Root>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

// Main Documents Component
export function Documents() {
  const { documents, selectedDocument, setSelectedDocument, loading } = useDocumentsStore();
  const [searchTerm, setSearchTerm] = useState('');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const [statusFilter, setStatusFilter] = useState<'all' | 'ready' | 'processing' | 'error'>('all');

  // Filter documents
  const filteredDocuments = documents.filter(doc => {
    const matchesSearch = doc.name.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = statusFilter === 'all' || doc.status === statusFilter;
    return matchesSearch && matchesStatus;
  });

  const statusCounts = {
    all: documents.length,
    ready: documents.filter(d => d.status === 'ready').length,
    processing: documents.filter(d => d.status === 'processing').length,
    error: documents.filter(d => d.status === 'error').length,
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">Documents</h1>
        <p className="text-muted-foreground">
          Manage your knowledge base and document processing
        </p>
      </div>

      {/* Upload Section */}
      <FileUpload />

      {/* Search and Filter Controls */}
      <div className="flex flex-col sm:flex-row gap-4 items-center justify-between">
        <div className="flex items-center gap-2 flex-1">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search documents..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="pl-10"
            />
          </div>
          
          <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild>
              <Button variant="outline" size="sm">
                <Filter className="h-4 w-4 mr-2" />
                {statusFilter === 'all' ? 'All Status' : `Status: ${statusFilter}`}
              </Button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content className="w-40 bg-popover rounded-md border shadow-lg p-1">
                {Object.entries(statusCounts).map(([status, count]) => (
                  <DropdownMenu.Item
                    key={status}
                    className="flex items-center justify-between px-2 py-1.5 text-sm cursor-pointer rounded-sm hover:bg-accent"
                    onClick={() => setStatusFilter(status as any)}
                  >
                    <span className="capitalize">{status}</span>
                    <Badge variant="secondary" className="text-xs">
                      {count}
                    </Badge>
                  </DropdownMenu.Item>
                ))}
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
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

      {/* Documents Grid/List */}
      <Card>
        <CardHeader>
          <CardTitle>
            Documents ({filteredDocuments.length})
          </CardTitle>
          <CardDescription>
            {statusCounts.ready} ready • {statusCounts.processing} processing • {statusCounts.error} errors
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading.documents ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
          ) : filteredDocuments.length === 0 ? (
            <div className="text-center py-8">
              <FileText className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
              <p className="text-lg font-medium mb-2">No documents found</p>
              <p className="text-muted-foreground">
                {searchTerm || statusFilter !== 'all' 
                  ? 'Try adjusting your search or filters'
                  : 'Upload your first document to get started'
                }
              </p>
            </div>
          ) : viewMode === 'grid' ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {filteredDocuments.map((document) => (
                <DocumentCard
                  key={document.id}
                  document={document}
                  onSelect={setSelectedDocument}
                />
              ))}
            </div>
          ) : (
            <DocumentList 
              documents={filteredDocuments} 
              onSelect={setSelectedDocument} 
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}