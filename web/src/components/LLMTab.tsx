import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Brain, Send, RefreshCw, Zap, MessageSquare } from 'lucide-react'
import { api, Provider, ChatMessage, useAsyncOperation } from '@/lib/api'
import { Alert, AlertDescription } from '@/components/ui/alert'

export function LLMTab() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [selectedProvider, setSelectedProvider] = useState<string>('')
  const [prompt, setPrompt] = useState('')
  const [chatMessages, setChatMessages] = useState<ChatMessage[]>([])
  const [chatInput, setChatInput] = useState('')
  const [temperature, setTemperature] = useState(0.7)
  const [maxTokens, setMaxTokens] = useState(500)
  const [streaming, setStreaming] = useState(false)
  const [response, setResponse] = useState('')
  
  const providersOp = useAsyncOperation<Provider[]>()
  const generateOp = useAsyncOperation<{ content: string; provider: string }>()
  const chatOp = useAsyncOperation<{ content: string; provider: string }>()

  useEffect(() => {
    loadProviders()
  }, [])

  const loadProviders = async () => {
    const result = await providersOp.execute(() => api.listProviders())
    if (result.data) {
      setProviders(result.data)
      if (result.data.length > 0 && !selectedProvider) {
        setSelectedProvider(result.data[0].name)
      }
    }
  }

  const handleGenerate = async () => {
    if (!prompt.trim()) return

    setResponse('')
    
    if (streaming) {
      try {
        await api.streamGenerate(
          {
            prompt,
            provider: selectedProvider === 'auto' ? undefined : selectedProvider,
            temperature,
            max_tokens: maxTokens,
            stream: true,
          },
          (chunk) => {
            setResponse((prev) => prev + chunk)
          }
        )
      } catch (error) {
        setResponse(`Error: ${error}`)
      }
    } else {
      const result = await generateOp.execute(() =>
        api.generate({
          prompt,
          provider: selectedProvider === 'auto' ? undefined : selectedProvider,
          temperature,
          max_tokens: maxTokens,
        })
      )
      
      if (result.data) {
        setResponse(result.data.content)
      }
    }
  }

  const handleChat = async () => {
    if (!chatInput.trim()) return

    const newMessage: ChatMessage = { role: 'user', content: chatInput }
    const updatedMessages = [...chatMessages, newMessage]
    setChatMessages(updatedMessages)
    setChatInput('')

    const result = await chatOp.execute(() =>
      api.chat({
        messages: updatedMessages,
        provider: selectedProvider === 'auto' ? undefined : selectedProvider,
        temperature,
        max_tokens: maxTokens,
      })
    )

    if (result.data) {
      setChatMessages([
        ...updatedMessages,
        { role: 'assistant', content: result.data.content },
      ])
    }
  }

  const getProviderBadgeColor = (status?: string) => {
    switch (status) {
      case 'healthy':
        return 'bg-green-500'
      case 'unhealthy':
        return 'bg-red-500'
      default:
        return 'bg-gray-500'
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Brain className="h-5 w-5" />
            LLM Pillar
          </CardTitle>
          <CardDescription>
            Multi-provider language model interface with pooling and load balancing
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {/* Provider Status */}
            <div>
              <Label>Active Providers</Label>
              <div className="flex flex-wrap gap-2 mt-2">
                {providers.map((provider) => (
                  <Badge
                    key={provider.name}
                    variant={provider.enabled ? 'default' : 'secondary'}
                    className="flex items-center gap-1"
                  >
                    <div className={`w-2 h-2 rounded-full ${getProviderBadgeColor(provider.status)}`} />
                    {provider.name} ({provider.type})
                    {provider.weight > 0 && (
                      <span className="text-xs opacity-60">w:{provider.weight}</span>
                    )}
                  </Badge>
                ))}
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={loadProviders}
                  className="h-6 px-2"
                >
                  <RefreshCw className="h-3 w-3" />
                </Button>
              </div>
            </div>

            {/* Provider Selection */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>Provider</Label>
                <Select value={selectedProvider} onValueChange={setSelectedProvider}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select provider" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="auto">Auto (Load Balanced)</SelectItem>
                    {providers
                      .filter((p) => p.enabled)
                      .map((provider) => (
                        <SelectItem key={provider.name} value={provider.name}>
                          {provider.name} - {provider.model}
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="flex items-end gap-4">
                <div className="flex items-center gap-2">
                  <Switch
                    id="streaming"
                    checked={streaming}
                    onCheckedChange={setStreaming}
                  />
                  <Label htmlFor="streaming" className="cursor-pointer">
                    <Zap className="h-4 w-4 inline mr-1" />
                    Streaming
                  </Label>
                </div>
              </div>
            </div>

            {/* Parameters */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <Label>Temperature: {temperature}</Label>
                <Slider
                  value={[temperature]}
                  onValueChange={([v]: number[]) => setTemperature(v)}
                  min={0}
                  max={2}
                  step={0.1}
                  className="mt-2"
                />
              </div>
              <div>
                <Label>Max Tokens: {maxTokens}</Label>
                <Slider
                  value={[maxTokens]}
                  onValueChange={([v]: number[]) => setMaxTokens(v)}
                  min={50}
                  max={4000}
                  step={50}
                  className="mt-2"
                />
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Generation and Chat Interface */}
      <Tabs defaultValue="generate" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="generate">Generation</TabsTrigger>
          <TabsTrigger value="chat">Chat</TabsTrigger>
        </TabsList>

        <TabsContent value="generate" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Text Generation</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>Prompt</Label>
                <Textarea
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  placeholder="Enter your prompt here..."
                  className="min-h-[100px] mt-2"
                />
              </div>

              <Button
                onClick={handleGenerate}
                disabled={generateOp.loading || !prompt.trim()}
                className="w-full"
              >
                <Send className="h-4 w-4 mr-2" />
                {generateOp.loading ? 'Generating...' : 'Generate'}
              </Button>

              {generateOp.error && (
                <Alert variant="destructive">
                  <AlertDescription>{generateOp.error}</AlertDescription>
                </Alert>
              )}

              {response && (
                <Card className="bg-gray-50">
                  <CardHeader>
                    <CardTitle className="text-sm">Response</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="whitespace-pre-wrap">{response}</p>
                  </CardContent>
                </Card>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="chat" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Chat Interface</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="border rounded-lg p-4 h-[400px] overflow-y-auto space-y-4 bg-gray-50">
                {chatMessages.length === 0 ? (
                  <p className="text-gray-500 text-center">Start a conversation...</p>
                ) : (
                  chatMessages.map((msg, idx) => (
                    <div
                      key={idx}
                      className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
                    >
                      <div
                        className={`max-w-[70%] rounded-lg p-3 ${
                          msg.role === 'user'
                            ? 'bg-blue-500 text-white'
                            : 'bg-white border border-gray-200'
                        }`}
                      >
                        <p className="text-sm whitespace-pre-wrap">{msg.content}</p>
                      </div>
                    </div>
                  ))
                )}
              </div>

              <div className="flex gap-2">
                <Input
                  value={chatInput}
                  onChange={(e) => setChatInput(e.target.value)}
                  placeholder="Type your message..."
                  onKeyPress={(e) => e.key === 'Enter' && !e.shiftKey && handleChat()}
                />
                <Button onClick={handleChat} disabled={chatOp.loading || !chatInput.trim()}>
                  <MessageSquare className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setChatMessages([])}
                  disabled={chatMessages.length === 0}
                >
                  Clear
                </Button>
              </div>

              {chatOp.error && (
                <Alert variant="destructive">
                  <AlertDescription>{chatOp.error}</AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}