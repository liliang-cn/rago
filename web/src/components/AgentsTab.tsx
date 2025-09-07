import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { 
  Bot, 
  Play, 
  GitBranch,
  Clock,
  CheckCircle,
  Sparkles,
  Calendar
} from 'lucide-react'
import { api, WorkflowDefinition, useAsyncOperation } from '@/lib/api'

export function AgentsTab() {
  const [taskDescription, setTaskDescription] = useState('')
  const [workflow, setWorkflow] = useState<WorkflowDefinition | null>(null)
  const [executionResult, setExecutionResult] = useState<any>(null)
  const [customWorkflow, setCustomWorkflow] = useState<string>(
    JSON.stringify({
      id: 'custom-workflow',
      name: 'Custom Workflow',
      description: 'A custom workflow',
      steps: [
        {
          id: 'step1',
          type: 'llm_generate',
          description: 'Generate content',
          parameters: {
            prompt: 'Hello, world!',
            max_tokens: 100,
          },
        },
      ],
    }, null, 2)
  )

  const generateOp = useAsyncOperation<WorkflowDefinition>()
  const executeOp = useAsyncOperation<any>()
  const taskOp = useAsyncOperation<any>()

  const handleGenerateWorkflow = async () => {
    if (!taskDescription.trim()) return

    const result = await generateOp.execute(() =>
      api.generateWorkflow(taskDescription)
    )

    if (result.data) {
      setWorkflow(result.data)
      setCustomWorkflow(JSON.stringify(result.data, null, 2))
    }
  }

  const handleExecuteWorkflow = async () => {
    if (!workflow) return

    const result = await executeOp.execute(() =>
      api.executeWorkflow(workflow)
    )

    if (result.data) {
      setExecutionResult(result.data)
    }
  }

  const handleExecuteCustomWorkflow = async () => {
    try {
      const parsedWorkflow = JSON.parse(customWorkflow)
      const result = await executeOp.execute(() =>
        api.executeWorkflow(parsedWorkflow)
      )

      if (result.data) {
        setExecutionResult(result.data)
      }
    } catch (error) {
      alert('Invalid workflow JSON')
    }
  }

  const handleExecuteTask = async () => {
    if (!taskDescription.trim()) return

    const result = await taskOp.execute(() =>
      api.executeTask({
        goal: taskDescription,
      })
    )

    if (result.data) {
      setExecutionResult(result.data)
    }
  }

  const getStepTypeIcon = (type: string) => {
    switch (type) {
      case 'llm_generate':
        return '🧠'
      case 'rag_query':
        return '📚'
      case 'mcp_tool':
        return '🔧'
      case 'condition':
        return '🔀'
      default:
        return '📦'
    }
  }

  const getStepTypeBadge = (type: string) => {
    const colors: Record<string, string> = {
      llm_generate: 'bg-blue-500',
      rag_query: 'bg-green-500',
      mcp_tool: 'bg-purple-500',
      condition: 'bg-orange-500',
    }
    return colors[type] || 'bg-gray-500'
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bot className="h-5 w-5" />
            Agents Pillar
          </CardTitle>
          <CardDescription>
            Autonomous workflow orchestration combining LLM, RAG, and MCP capabilities
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-4 gap-4">
            <div className="flex items-center gap-2">
              <GitBranch className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium">Workflows</p>
                <p className="text-xs text-gray-500">Multi-step</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Sparkles className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium">NL→Workflow</p>
                <p className="text-xs text-gray-500">Generation</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium">Scheduling</p>
                <p className="text-xs text-gray-500">Time-based</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Bot className="h-4 w-4 text-gray-500" />
              <div>
                <p className="text-sm font-medium">Reasoning</p>
                <p className="text-xs text-gray-500">Multi-step</p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="natural" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="natural">Natural Language</TabsTrigger>
          <TabsTrigger value="workflow">Workflow Builder</TabsTrigger>
          <TabsTrigger value="scheduled">Scheduled Tasks</TabsTrigger>
        </TabsList>

        <TabsContent value="natural" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Natural Language Task</CardTitle>
              <CardDescription>
                Describe a task and let the agent figure out how to accomplish it
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>Task Description</Label>
                <Textarea
                  value={taskDescription}
                  onChange={(e) => setTaskDescription(e.target.value)}
                  placeholder="E.g., Analyze the latest documents about AI and create a summary report..."
                  className="min-h-[100px] mt-2"
                />
              </div>

              <div className="flex gap-2">
                <Button
                  onClick={handleGenerateWorkflow}
                  disabled={generateOp.loading || !taskDescription.trim()}
                  variant="outline"
                >
                  <Sparkles className="h-4 w-4 mr-2" />
                  {generateOp.loading ? 'Generating...' : 'Generate Workflow'}
                </Button>
                <Button
                  onClick={handleExecuteTask}
                  disabled={taskOp.loading || !taskDescription.trim()}
                >
                  <Play className="h-4 w-4 mr-2" />
                  {taskOp.loading ? 'Executing...' : 'Execute Directly'}
                </Button>
              </div>

              {generateOp.error && (
                <Alert variant="destructive">
                  <AlertDescription>{generateOp.error}</AlertDescription>
                </Alert>
              )}

              {workflow && (
                <Card className="bg-gray-50">
                  <CardHeader>
                    <CardTitle className="text-sm">Generated Workflow</CardTitle>
                    <CardDescription>{workflow.description}</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-2">
                      {workflow.steps.map((step) => (
                        <div key={step.id} className="flex items-start gap-2">
                          <span className="text-lg">{getStepTypeIcon(step.type)}</span>
                          <div className="flex-1">
                            <div className="flex items-center gap-2">
                              <Badge className={`${getStepTypeBadge(step.type)} text-white`}>
                                {step.type}
                              </Badge>
                              <span className="text-sm font-medium">{step.id}</span>
                            </div>
                            <p className="text-xs text-gray-600 mt-1">{step.description}</p>
                            {step.depends_on && step.depends_on.length > 0 && (
                              <p className="text-xs text-gray-500 mt-1">
                                Depends on: {step.depends_on.join(', ')}
                              </p>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                    <Button
                      onClick={handleExecuteWorkflow}
                      disabled={executeOp.loading}
                      className="w-full mt-4"
                    >
                      <Play className="h-4 w-4 mr-2" />
                      {executeOp.loading ? 'Executing Workflow...' : 'Execute Workflow'}
                    </Button>
                  </CardContent>
                </Card>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="workflow" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Custom Workflow</CardTitle>
              <CardDescription>
                Define a custom workflow in JSON format
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>Workflow Definition (JSON)</Label>
                <Textarea
                  value={customWorkflow}
                  onChange={(e) => setCustomWorkflow(e.target.value)}
                  className="font-mono text-sm min-h-[300px] mt-2"
                />
              </div>

              <Button
                onClick={handleExecuteCustomWorkflow}
                disabled={executeOp.loading}
                className="w-full"
              >
                <Play className="h-4 w-4 mr-2" />
                {executeOp.loading ? 'Executing...' : 'Execute Custom Workflow'}
              </Button>

              {executeOp.error && (
                <Alert variant="destructive">
                  <AlertDescription>{executeOp.error}</AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="scheduled" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Scheduled Tasks</CardTitle>
              <CardDescription>
                Manage time-based and recurring agent tasks
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-center py-8 text-gray-500">
                <Calendar className="h-12 w-12 mx-auto mb-4 text-gray-300" />
                <p className="text-sm">Scheduled tasks feature coming soon</p>
                <p className="text-xs mt-2">
                  Schedule workflows to run at specific times or intervals
                </p>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Execution Results */}
      {executionResult && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <CheckCircle className="h-5 w-5 text-green-500" />
              Execution Result
            </CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="bg-gray-50 p-4 rounded-lg overflow-x-auto text-sm">
              {JSON.stringify(executionResult, null, 2)}
            </pre>
          </CardContent>
        </Card>
      )}
    </div>
  )
}