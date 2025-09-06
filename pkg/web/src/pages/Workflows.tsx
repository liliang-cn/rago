import React, { useState, useCallback } from 'react';
import ReactFlow, {
  Node,
  Edge,
  addEdge,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  Connection,
  ReactFlowProvider,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { 
  Play, 
  Save, 
  Plus, 
  Settings, 
  Download, 
  Upload,
  GitBranch,
  Zap,
  Database,
  MessageSquare,
  Calendar
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/utils';

// Node types for the workflow
const nodeTypes = [
  { type: 'input', label: 'Input', icon: Upload, color: 'bg-blue-500' },
  { type: 'query', label: 'Query Documents', icon: MessageSquare, color: 'bg-green-500' },
  { type: 'process', label: 'Process Data', icon: Zap, color: 'bg-purple-500' },
  { type: 'storage', label: 'Store Result', icon: Database, color: 'bg-orange-500' },
  { type: 'schedule', label: 'Schedule Task', icon: Calendar, color: 'bg-red-500' },
  { type: 'output', label: 'Output', icon: Download, color: 'bg-gray-500' },
];

// Initial workflow nodes
const initialNodes: Node[] = [
  {
    id: '1',
    type: 'input',
    position: { x: 100, y: 100 },
    data: { 
      label: 'Start Workflow',
      description: 'Workflow entry point',
      config: {}
    },
  },
  {
    id: '2',
    type: 'default',
    position: { x: 300, y: 200 },
    data: { 
      label: 'Query Documents',
      description: 'Search knowledge base',
      config: { query: '', limit: 5 }
    },
    style: { 
      background: '#10b981', 
      color: 'white',
      border: '1px solid #059669'
    },
  },
];

// Initial edges
const initialEdges: Edge[] = [
  { id: 'e1-2', source: '1', target: '2', animated: true },
];

// Node Palette Component
function NodePalette() {
  const onDragStart = (event: React.DragEvent, nodeType: string, label: string) => {
    event.dataTransfer.setData('application/reactflow', nodeType);
    event.dataTransfer.setData('application/nodelabel', label);
    event.dataTransfer.effectAllowed = 'move';
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Node Palette</CardTitle>
        <CardDescription>Drag nodes to the canvas</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {nodeTypes.map((node) => (
            <div
              key={node.type}
              draggable
              onDragStart={(event) => onDragStart(event, node.type, node.label)}
              className="flex items-center gap-3 p-2 border rounded-lg cursor-grab hover:bg-accent"
            >
              <div className={cn('w-3 h-3 rounded-full', node.color)} />
              <node.icon className="h-4 w-4" />
              <span className="text-sm font-medium">{node.label}</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

// Workflow Properties Panel
function PropertiesPanel({ selectedNode }: { selectedNode: Node | null }) {
  if (!selectedNode) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Properties</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Select a node to edit its properties
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Node Properties</CardTitle>
        <CardDescription>{selectedNode.data.label}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div>
            <label className="text-sm font-medium mb-2 block">Name</label>
            <input 
              className="w-full p-2 border rounded-md bg-background"
              defaultValue={selectedNode.data.label}
            />
          </div>
          
          <div>
            <label className="text-sm font-medium mb-2 block">Description</label>
            <textarea 
              className="w-full p-2 border rounded-md bg-background resize-none"
              rows={3}
              defaultValue={selectedNode.data.description}
            />
          </div>

          {selectedNode.type === 'query' && (
            <div>
              <label className="text-sm font-medium mb-2 block">Query Text</label>
              <input 
                className="w-full p-2 border rounded-md bg-background"
                placeholder="Enter query..."
              />
            </div>
          )}

          {selectedNode.type === 'process' && (
            <div>
              <label className="text-sm font-medium mb-2 block">Processing Type</label>
              <select className="w-full p-2 border rounded-md bg-background">
                <option>Transform</option>
                <option>Filter</option>
                <option>Aggregate</option>
                <option>Validate</option>
              </select>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

// Main Flow Canvas Component
function FlowCanvas() {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [isExecuting, setIsExecuting] = useState(false);

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  const onNodeClick = useCallback((event: React.MouseEvent, node: Node) => {
    setSelectedNode(node);
  }, []);

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();

      const reactFlowBounds = (event.target as Element).getBoundingClientRect();
      const type = event.dataTransfer.getData('application/reactflow');
      const label = event.dataTransfer.getData('application/nodelabel');

      if (!type) return;

      const position = {
        x: event.clientX - reactFlowBounds.left - 75,
        y: event.clientY - reactFlowBounds.top - 25,
      };

      const newNode: Node = {
        id: `${nodes.length + 1}`,
        type: 'default',
        position,
        data: { 
          label,
          description: `New ${label} node`,
          config: {}
        },
        style: getNodeStyle(type),
      };

      setNodes((nds) => nds.concat(newNode));
    },
    [nodes, setNodes]
  );

  const getNodeStyle = (type: string) => {
    const nodeType = nodeTypes.find(n => n.type === type);
    if (nodeType?.color) {
      return {
        background: nodeType.color.replace('bg-', ''),
        color: 'white',
        border: '1px solid rgba(0,0,0,0.2)'
      };
    }
    return {};
  };

  const executeWorkflow = async () => {
    setIsExecuting(true);
    
    // Simulate workflow execution
    for (let i = 0; i < nodes.length; i++) {
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      // Highlight current node
      setNodes(nds => 
        nds.map(node => ({
          ...node,
          style: node.id === nodes[i].id 
            ? { ...node.style, border: '3px solid #3b82f6', boxShadow: '0 0 10px #3b82f6' }
            : node.style
        }))
      );
    }
    
    // Reset styles
    setTimeout(() => {
      setNodes(nds => 
        nds.map(node => ({
          ...node,
          style: getNodeStyle(node.type || 'default')
        }))
      );
      setIsExecuting(false);
    }, 1000);
  };

  const saveWorkflow = () => {
    const workflow = { nodes, edges };
    const dataStr = JSON.stringify(workflow, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'workflow.json';
    link.click();
  };

  return (
    <div className="flex h-full gap-4">
      {/* Left Sidebar */}
      <div className="w-64 space-y-4">
        <NodePalette />
        <PropertiesPanel selectedNode={selectedNode} />
      </div>

      {/* Main Canvas */}
      <div className="flex-1 border rounded-lg overflow-hidden">
        <div className="h-12 border-b bg-background px-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <GitBranch className="h-4 w-4" />
            <span className="font-medium">Workflow Designer</span>
            <Badge variant="secondary">Draft</Badge>
          </div>
          
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={saveWorkflow}>
              <Save className="h-4 w-4 mr-1" />
              Save
            </Button>
            <Button 
              size="sm" 
              onClick={executeWorkflow}
              disabled={isExecuting}
            >
              {isExecuting ? (
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-1"></div>
              ) : (
                <Play className="h-4 w-4 mr-1" />
              )}
              Execute
            </Button>
          </div>
        </div>
        
        <div className="h-[calc(100%-48px)]">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onNodeClick={onNodeClick}
            onDrop={onDrop}
            onDragOver={onDragOver}
            fitView
          >
            <Background />
            <Controls />
            <MiniMap />
          </ReactFlow>
        </div>
      </div>
    </div>
  );
}

// Main Workflows Component
export function Workflows() {
  return (
    <ReactFlowProvider>
      <div className="h-full space-y-6">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold">Workflow Designer</h1>
          <p className="text-muted-foreground">
            Create and manage automated workflows with drag-and-drop interface
          </p>
        </div>

        {/* Canvas Container */}
        <div className="h-[calc(100vh-200px)]">
          <FlowCanvas />
        </div>
      </div>
    </ReactFlowProvider>
  );
}