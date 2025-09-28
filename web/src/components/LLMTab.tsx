import { useState } from 'react'
import { Brain, MessageSquare, Code, Sparkles } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './ui/card'
import { Button } from './ui/button'
import { Textarea } from './ui/textarea'
import { Input } from './ui/input'
import { Label } from './ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs'
import { Slider } from './ui/slider'
import { ScrollArea } from './ui/scroll-area'
import { apiClient, LLMGenerateRequest } from '../lib/api'
import { Alert, AlertDescription } from './ui/alert'

export function LLMTab() {
  const [generateRequest, setGenerateRequest] = useState<LLMGenerateRequest>({
    prompt: '',
    temperature: 0.7,
    max_tokens: 500,
    stream: false,
  })

  const [chatMessages, setChatMessages] = useState<Array<{role: string; content: string}>>([])
  const [chatInput, setChatInput] = useState('')
  const [chatSettings, setChatSettings] = useState({
    temperature: 0.7,
    max_tokens: 500,
    stream: false,
  })

  const [structuredPrompt, setStructuredPrompt] = useState('')
  const [structuredSchema, setStructuredSchema] = useState('{\n  "type": "object",\n  "properties": {\n    "name": {"type": "string"},\n    "value": {"type": "number"}\n  }\n}')
  
  const [responses, setResponses] = useState<{
    generate: string
    chat: string
    structured: any
    streaming: string
  }>({
    generate: '',
    chat: '',
    structured: null,
    streaming: '',
  })

  const [loading, setLoading] = useState<{
    generate: boolean
    chat: boolean
    structured: boolean
    streaming: boolean
  }>({
    generate: false,
    chat: false,
    structured: false,
    streaming: false,
  })

  const [error, setError] = useState<string | null>(null)

  const handleGenerate = async () => {
    if (!generateRequest.prompt) return

    setLoading(prev => ({ ...prev, generate: true }))
    setError(null)

    try {
      const response = await apiClient.llmGenerate(generateRequest)
      if (response.error) {
        setError(response.error)
      } else if (response.data) {
        setResponses(prev => ({ ...prev, generate: response.data!.content }))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Generation failed')
    } finally {
      setLoading(prev => ({ ...prev, generate: false }))
    }
  }

  const handleStreamGenerate = async () => {
    if (!generateRequest.prompt) return

    setLoading(prev => ({ ...prev, streaming: true }))
    setError(null)
    setResponses(prev => ({ ...prev, streaming: '' }))

    try {
      await apiClient.llmGenerateStream(
        { ...generateRequest, stream: true },
        (chunk) => {
          setResponses(prev => ({ ...prev, streaming: prev.streaming + chunk }))
        }
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Streaming failed')
    } finally {
      setLoading(prev => ({ ...prev, streaming: false }))
    }
  }

  const handleChat = async () => {
    if (!chatInput) return

    const newMessage = { role: 'user', content: chatInput }
    const updatedMessages = [...chatMessages, newMessage]
    setChatMessages(updatedMessages)
    setChatInput('')

    setLoading(prev => ({ ...prev, chat: true }))
    setError(null)

    try {
      const response = await apiClient.llmChat({
        messages: updatedMessages,
        ...chatSettings,
      })
      
      if (response.error) {
        setError(response.error)
      } else if (response.data) {
        setChatMessages(response.data!.messages)
        setResponses(prev => ({ ...prev, chat: response.data!.response }))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Chat failed')
    } finally {
      setLoading(prev => ({ ...prev, chat: false }))
    }
  }

  const handleStructured = async () => {
    if (!structuredPrompt) return

    setLoading(prev => ({ ...prev, structured: true }))
    setError(null)

    try {
      let schema
      try {
        schema = JSON.parse(structuredSchema)
      } catch {
        setError('Invalid JSON schema')
        return
      }

      const response = await apiClient.llmStructured(structuredPrompt, schema)
      if (response.error) {
        setError(response.error)
      } else if (response.data) {
        setResponses(prev => ({ ...prev, structured: response.data }))
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Structured generation failed')
    } finally {
      setLoading(prev => ({ ...prev, structured: false }))
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-left">LLM Operations</CardTitle>
          <CardDescription className="text-left">
            Direct LLM operations without RAG context
          </CardDescription>
        </CardHeader>
      </Card>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Tabs defaultValue="generate">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="generate">
            <Brain className="h-4 w-4 mr-2" />
            Generate
          </TabsTrigger>
          <TabsTrigger value="chat">
            <MessageSquare className="h-4 w-4 mr-2" />
            Chat
          </TabsTrigger>
          <TabsTrigger value="structured">
            <Code className="h-4 w-4 mr-2" />
            Structured
          </TabsTrigger>
          <TabsTrigger value="streaming">
            <Sparkles className="h-4 w-4 mr-2" />
            Streaming
          </TabsTrigger>
        </TabsList>

        <TabsContent value="generate" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-left">Text Generation</CardTitle>
              <CardDescription className="text-left">Generate text from a prompt</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="prompt" className="text-left">Prompt</Label>
                <Textarea
                  id="prompt"
                  value={generateRequest.prompt}
                  onChange={(e) => setGenerateRequest(prev => ({ ...prev, prompt: e.target.value }))}
                  placeholder="Enter your prompt..."
                  rows={4}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="temperature" className="text-left">
                    Temperature: {generateRequest.temperature?.toFixed(2)}
                  </Label>
                  <Slider
                    id="temperature"
                    min={0}
                    max={1}
                    step={0.05}
                    value={[generateRequest.temperature || 0.7]}
                    onValueChange={(value: number[]) => setGenerateRequest(prev => ({ ...prev, temperature: value[0] }))}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="max_tokens" className="text-left">
                    Max Tokens: {generateRequest.max_tokens}
                  </Label>
                  <Slider
                    id="max_tokens"
                    min={50}
                    max={2000}
                    step={50}
                    value={[generateRequest.max_tokens || 500]}
                    onValueChange={(value: number[]) => setGenerateRequest(prev => ({ ...prev, max_tokens: value[0] }))}
                  />
                </div>
              </div>

              <Button
                onClick={handleGenerate}
                disabled={loading.generate || !generateRequest.prompt}
                className="w-full"
              >
                {loading.generate ? 'Generating...' : 'Generate'}
              </Button>

              {responses.generate && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm text-left">Response</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm whitespace-pre-wrap">{responses.generate}</p>
                  </CardContent>
                </Card>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="chat" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-left">Chat with History</CardTitle>
              <CardDescription className="text-left">Multi-turn conversation with context</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <ScrollArea className="h-[300px] border rounded p-4">
                <div className="space-y-4">
                  {chatMessages.map((msg, index) => (
                    <div
                      key={index}
                      className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
                    >
                      <div
                        className={`max-w-[80%] rounded-lg p-3 ${
                          msg.role === 'user'
                            ? 'bg-primary text-primary-foreground'
                            : 'bg-muted'
                        }`}
                      >
                        <p className="text-sm">{msg.content}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </ScrollArea>

              <div className="flex gap-2">
                <Input
                  value={chatInput}
                  onChange={(e) => setChatInput(e.target.value)}
                  placeholder="Type your message..."
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault()
                      handleChat()
                    }
                  }}
                />
                <Button
                  onClick={handleChat}
                  disabled={loading.chat || !chatInput}
                >
                  Send
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setChatMessages([])}
                >
                  Clear
                </Button>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label className="text-left">Temperature: {chatSettings.temperature.toFixed(2)}</Label>
                  <Slider
                    min={0}
                    max={1}
                    step={0.05}
                    value={[chatSettings.temperature]}
                    onValueChange={(value: number[]) => setChatSettings(prev => ({ ...prev, temperature: value[0] }))}
                  />
                </div>
                <div className="space-y-2">
                  <Label className="text-left">Max Tokens: {chatSettings.max_tokens}</Label>
                  <Slider
                    min={50}
                    max={2000}
                    step={50}
                    value={[chatSettings.max_tokens]}
                    onValueChange={(value: number[]) => setChatSettings(prev => ({ ...prev, max_tokens: value[0] }))}
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="structured" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-left">Structured Generation</CardTitle>
              <CardDescription className="text-left">Generate JSON matching a schema</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="structured-prompt" className="text-left">Prompt</Label>
                <Textarea
                  id="structured-prompt"
                  value={structuredPrompt}
                  onChange={(e) => setStructuredPrompt(e.target.value)}
                  placeholder="Describe what you want to generate..."
                  rows={3}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="schema" className="text-left">JSON Schema</Label>
                <Textarea
                  id="schema"
                  value={structuredSchema}
                  onChange={(e) => setStructuredSchema(e.target.value)}
                  placeholder="Enter JSON schema..."
                  rows={6}
                  className="font-mono text-sm"
                />
              </div>

              <Button
                onClick={handleStructured}
                disabled={loading.structured || !structuredPrompt}
                className="w-full"
              >
                {loading.structured ? 'Generating...' : 'Generate Structured Output'}
              </Button>

              {responses.structured && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm text-left">
                      Structured Response {responses.structured.valid && '✓ Valid'}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <pre className="text-sm bg-muted p-3 rounded overflow-x-auto">
                      {JSON.stringify(responses.structured.data || responses.structured.raw, null, 2)}
                    </pre>
                  </CardContent>
                </Card>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="streaming" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-left">Streaming Generation</CardTitle>
              <CardDescription className="text-left">Real-time token streaming</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="stream-prompt" className="text-left">Prompt</Label>
                <Textarea
                  id="stream-prompt"
                  value={generateRequest.prompt}
                  onChange={(e) => setGenerateRequest(prev => ({ ...prev, prompt: e.target.value }))}
                  placeholder="Enter your prompt for streaming..."
                  rows={4}
                />
              </div>

              <Button
                onClick={handleStreamGenerate}
                disabled={loading.streaming || !generateRequest.prompt}
                className="w-full"
              >
                {loading.streaming ? 'Streaming...' : 'Start Streaming'}
              </Button>

              {(responses.streaming || loading.streaming) && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm text-left">
                      Streaming Response {loading.streaming && '(in progress...)'}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <ScrollArea className="h-[200px]">
                      <p className="text-sm whitespace-pre-wrap">
                        {responses.streaming}
                        {loading.streaming && '▊'}
                      </p>
                    </ScrollArea>
                  </CardContent>
                </Card>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}