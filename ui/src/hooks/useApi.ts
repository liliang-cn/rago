import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, QueryRequest, ChatRequest, CreateSkillRequest, AddMCPServerRequest, CallToolRequest } from '../lib/api'

// RAG Hooks
export function useQueryRAG() {
  return useMutation({
    mutationFn: (data: QueryRequest) => api.query(data),
  })
}

export function useDocuments() {
  return useQuery({
    queryKey: ['documents'],
    queryFn: api.getDocuments,
  })
}

export function useCollections() {
  return useQuery({
    queryKey: ['collections'],
    queryFn: api.getCollections,
  })
}

export function useStatus() {
  return useQuery({
    queryKey: ['status'],
    queryFn: api.getStatus,
    refetchInterval: 30000,
  })
}

export function useChat() {
  return useMutation({
    mutationFn: (data: ChatRequest) => api.chat(data),
  })
}

// Skills Hooks
export function useSkills() {
  return useQuery({
    queryKey: ['skills'],
    queryFn: api.getSkills,
  })
}

export function useSkill(id: string) {
  return useQuery({
    queryKey: ['skills', id],
    queryFn: () => api.getSkill(id),
    enabled: !!id,
  })
}

export function useCreateSkill() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateSkillRequest) => api.createSkill(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['skills'] })
    },
  })
}

export function useDeleteSkill() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.deleteSkill(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['skills'] })
    },
  })
}

// MCP Hooks
export function useMCPServers() {
  return useQuery({
    queryKey: ['mcp', 'servers'],
    queryFn: api.getMCPServers,
  })
}

export function useMCPTools() {
  return useQuery({
    queryKey: ['mcp', 'tools'],
    queryFn: api.getMCPTools,
  })
}

export function useAddMCPServer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: AddMCPServerRequest) => api.addMCPServer(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['mcp', 'servers'] })
      queryClient.invalidateQueries({ queryKey: ['mcp', 'tools'] })
    },
  })
}

export function useCallMCPTool() {
  return useMutation({
    mutationFn: (data: CallToolRequest) => api.callTool(data),
  })
}
