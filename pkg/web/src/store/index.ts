import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { subscribeWithSelector } from 'zustand/middleware';
import { 
  UIState, 
  Notification, 
  Provider, 
  Document, 
  SystemMetrics, 
  ComponentStatus, 
  Alert,
  Workflow,
  WorkflowExecution,
  Job,
  JobExecution,
  MCPServer,
  Config
} from '@/types';
import { getSystemTheme, applyTheme } from '@/utils';

// UI Store
interface UIStore extends UIState {
  // Actions
  setTheme: (theme: 'light' | 'dark' | 'system') => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp' | 'read'>) => void;
  removeNotification: (id: string) => void;
  markNotificationRead: (id: string) => void;
  clearNotifications: () => void;
  setActiveView: (view: string) => void;
}

export const useUIStore = create<UIStore>()(
  persist(
    subscribeWithSelector((set, get) => ({
      // State
      theme: 'system',
      sidebarCollapsed: false,
      notifications: [],
      activeView: 'dashboard',

      // Actions
      setTheme: (theme) => {
        set({ theme });
        applyTheme(theme);
      },
      
      toggleSidebar: () => {
        const { sidebarCollapsed } = get();
        set({ sidebarCollapsed: !sidebarCollapsed });
      },
      
      setSidebarCollapsed: (collapsed) => {
        set({ sidebarCollapsed: collapsed });
      },
      
      addNotification: (notification) => {
        const newNotification: Notification = {
          ...notification,
          id: crypto.randomUUID(),
          timestamp: new Date().toISOString(),
          read: false,
        };
        
        set((state) => ({
          notifications: [newNotification, ...state.notifications.slice(0, 49)], // Keep max 50
        }));
      },
      
      removeNotification: (id) => {
        set((state) => ({
          notifications: state.notifications.filter(n => n.id !== id),
        }));
      },
      
      markNotificationRead: (id) => {
        set((state) => ({
          notifications: state.notifications.map(n =>
            n.id === id ? { ...n, read: true } : n
          ),
        }));
      },
      
      clearNotifications: () => {
        set({ notifications: [] });
      },
      
      setActiveView: (view) => {
        set({ activeView: view });
      },
    })),
    {
      name: 'rago-ui-store',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({ 
        theme: state.theme, 
        sidebarCollapsed: state.sidebarCollapsed 
      }),
    }
  )
);

// System Store - for real-time system data
interface SystemStore {
  // State
  providers: Provider[];
  systemMetrics: SystemMetrics;
  componentStatuses: ComponentStatus[];
  alerts: Alert[];
  mcpServers: MCPServer[];
  config: Config | null;
  
  // Loading states
  loading: {
    providers: boolean;
    metrics: boolean;
    statuses: boolean;
    alerts: boolean;
    mcpServers: boolean;
    config: boolean;
  };
  
  // Actions
  setProviders: (providers: Provider[]) => void;
  updateProvider: (id: string, updates: Partial<Provider>) => void;
  setSystemMetrics: (metrics: SystemMetrics) => void;
  updateMetrics: (component: string, metrics: any) => void;
  setComponentStatuses: (statuses: ComponentStatus[]) => void;
  updateComponentStatus: (name: string, status: ComponentStatus) => void;
  setAlerts: (alerts: Alert[]) => void;
  addAlert: (alert: Omit<Alert, 'id' | 'timestamp'>) => void;
  acknowledgeAlert: (id: string) => void;
  resolveAlert: (id: string) => void;
  setMCPServers: (servers: MCPServer[]) => void;
  updateMCPServer: (name: string, updates: Partial<MCPServer>) => void;
  setConfig: (config: Config) => void;
  setLoading: (key: keyof SystemStore['loading'], loading: boolean) => void;
}

export const useSystemStore = create<SystemStore>()(
  subscribeWithSelector((set, get) => ({
    // State
    providers: [],
    systemMetrics: { cpu: [], memory: [], disk: [], network: [] },
    componentStatuses: [],
    alerts: [],
    mcpServers: [],
    config: null,
    loading: {
      providers: false,
      metrics: false,
      statuses: false,
      alerts: false,
      mcpServers: false,
      config: false,
    },
    
    // Actions
    setProviders: (providers) => set({ providers }),
    
    updateProvider: (id, updates) => {
      set((state) => ({
        providers: state.providers.map(p => 
          p.id === id ? { ...p, ...updates } : p
        ),
      }));
    },
    
    setSystemMetrics: (metrics) => set({ systemMetrics: metrics }),
    
    updateMetrics: (component, metrics) => {
      set((state) => ({
        systemMetrics: {
          ...state.systemMetrics,
          [component]: metrics,
        },
      }));
    },
    
    setComponentStatuses: (statuses) => set({ componentStatuses: statuses }),
    
    updateComponentStatus: (name, status) => {
      set((state) => ({
        componentStatuses: state.componentStatuses.map(s =>
          s.name === name ? status : s
        ),
      }));
    },
    
    setAlerts: (alerts) => set({ alerts }),
    
    addAlert: (alert) => {
      const newAlert: Alert = {
        ...alert,
        id: crypto.randomUUID(),
        timestamp: new Date().toISOString(),
      };
      
      set((state) => ({
        alerts: [newAlert, ...state.alerts],
      }));
    },
    
    acknowledgeAlert: (id) => {
      set((state) => ({
        alerts: state.alerts.map(a =>
          a.id === id ? { ...a, acknowledged: true } : a
        ),
      }));
    },
    
    resolveAlert: (id) => {
      set((state) => ({
        alerts: state.alerts.map(a =>
          a.id === id ? { ...a, resolved: true } : a
        ),
      }));
    },
    
    setMCPServers: (servers) => set({ mcpServers: servers }),
    
    updateMCPServer: (name, updates) => {
      set((state) => ({
        mcpServers: state.mcpServers.map(s =>
          s.name === name ? { ...s, ...updates } : s
        ),
      }));
    },
    
    setConfig: (config) => set({ config }),
    
    setLoading: (key, loading) => {
      set((state) => ({
        loading: { ...state.loading, [key]: loading },
      }));
    },
  }))
);

// Documents Store
interface DocumentsStore {
  // State
  documents: Document[];
  selectedDocument: Document | null;
  queryHistory: any[];
  
  // Loading states
  loading: {
    documents: boolean;
    upload: boolean;
    query: boolean;
  };
  
  // Actions
  setDocuments: (documents: Document[]) => void;
  addDocument: (document: Document) => void;
  updateDocument: (id: string, updates: Partial<Document>) => void;
  removeDocument: (id: string) => void;
  setSelectedDocument: (document: Document | null) => void;
  addQueryResult: (result: any) => void;
  setLoading: (key: keyof DocumentsStore['loading'], loading: boolean) => void;
}

export const useDocumentsStore = create<DocumentsStore>()(
  (set) => ({
    // State
    documents: [],
    selectedDocument: null,
    queryHistory: [],
    loading: {
      documents: false,
      upload: false,
      query: false,
    },
    
    // Actions
    setDocuments: (documents) => set({ documents }),
    
    addDocument: (document) => {
      set((state) => ({
        documents: [document, ...state.documents],
      }));
    },
    
    updateDocument: (id, updates) => {
      set((state) => ({
        documents: state.documents.map(d =>
          d.id === id ? { ...d, ...updates } : d
        ),
      }));
    },
    
    removeDocument: (id) => {
      set((state) => ({
        documents: state.documents.filter(d => d.id !== id),
      }));
    },
    
    setSelectedDocument: (document) => set({ selectedDocument: document }),
    
    addQueryResult: (result) => {
      set((state) => ({
        queryHistory: [result, ...state.queryHistory.slice(0, 99)], // Keep max 100
      }));
    },
    
    setLoading: (key, loading) => {
      set((state) => ({
        loading: { ...state.loading, [key]: loading },
      }));
    },
  })
);

// Workflows Store
interface WorkflowsStore {
  // State
  workflows: Workflow[];
  selectedWorkflow: Workflow | null;
  executions: WorkflowExecution[];
  selectedExecution: WorkflowExecution | null;
  
  // Loading states
  loading: {
    workflows: boolean;
    executions: boolean;
    save: boolean;
    execute: boolean;
  };
  
  // Actions
  setWorkflows: (workflows: Workflow[]) => void;
  addWorkflow: (workflow: Workflow) => void;
  updateWorkflow: (id: string, updates: Partial<Workflow>) => void;
  removeWorkflow: (id: string) => void;
  setSelectedWorkflow: (workflow: Workflow | null) => void;
  setExecutions: (executions: WorkflowExecution[]) => void;
  addExecution: (execution: WorkflowExecution) => void;
  updateExecution: (id: string, updates: Partial<WorkflowExecution>) => void;
  setSelectedExecution: (execution: WorkflowExecution | null) => void;
  setLoading: (key: keyof WorkflowsStore['loading'], loading: boolean) => void;
}

export const useWorkflowsStore = create<WorkflowsStore>()(
  (set) => ({
    // State
    workflows: [],
    selectedWorkflow: null,
    executions: [],
    selectedExecution: null,
    loading: {
      workflows: false,
      executions: false,
      save: false,
      execute: false,
    },
    
    // Actions
    setWorkflows: (workflows) => set({ workflows }),
    
    addWorkflow: (workflow) => {
      set((state) => ({
        workflows: [workflow, ...state.workflows],
      }));
    },
    
    updateWorkflow: (id, updates) => {
      set((state) => ({
        workflows: state.workflows.map(w =>
          w.id === id ? { ...w, ...updates } : w
        ),
      }));
    },
    
    removeWorkflow: (id) => {
      set((state) => ({
        workflows: state.workflows.filter(w => w.id !== id),
      }));
    },
    
    setSelectedWorkflow: (workflow) => set({ selectedWorkflow: workflow }),
    
    setExecutions: (executions) => set({ executions }),
    
    addExecution: (execution) => {
      set((state) => ({
        executions: [execution, ...state.executions],
      }));
    },
    
    updateExecution: (id, updates) => {
      set((state) => ({
        executions: state.executions.map(e =>
          e.id === id ? { ...e, ...updates } : e
        ),
      }));
    },
    
    setSelectedExecution: (execution) => set({ selectedExecution: execution }),
    
    setLoading: (key, loading) => {
      set((state) => ({
        loading: { ...state.loading, [key]: loading },
      }));
    },
  })
);

// Jobs Store
interface JobsStore {
  // State
  jobs: Job[];
  selectedJob: Job | null;
  executions: JobExecution[];
  
  // Loading states
  loading: {
    jobs: boolean;
    executions: boolean;
    save: boolean;
    execute: boolean;
  };
  
  // Actions
  setJobs: (jobs: Job[]) => void;
  addJob: (job: Job) => void;
  updateJob: (id: string, updates: Partial<Job>) => void;
  removeJob: (id: string) => void;
  setSelectedJob: (job: Job | null) => void;
  setExecutions: (executions: JobExecution[]) => void;
  addExecution: (execution: JobExecution) => void;
  updateExecution: (id: string, updates: Partial<JobExecution>) => void;
  setLoading: (key: keyof JobsStore['loading'], loading: boolean) => void;
}

export const useJobsStore = create<JobsStore>()(
  (set) => ({
    // State
    jobs: [],
    selectedJob: null,
    executions: [],
    loading: {
      jobs: false,
      executions: false,
      save: false,
      execute: false,
    },
    
    // Actions
    setJobs: (jobs) => set({ jobs }),
    
    addJob: (job) => {
      set((state) => ({
        jobs: [job, ...state.jobs],
      }));
    },
    
    updateJob: (id, updates) => {
      set((state) => ({
        jobs: state.jobs.map(j =>
          j.id === id ? { ...j, ...updates } : j
        ),
      }));
    },
    
    removeJob: (id) => {
      set((state) => ({
        jobs: state.jobs.filter(j => j.id !== id),
      }));
    },
    
    setSelectedJob: (job) => set({ selectedJob: job }),
    
    setExecutions: (executions) => set({ executions }),
    
    addExecution: (execution) => {
      set((state) => ({
        executions: [execution, ...state.executions],
      }));
    },
    
    updateExecution: (id, updates) => {
      set((state) => ({
        executions: state.executions.map(e =>
          e.id === id ? { ...e, ...updates } : e
        ),
      }));
    },
    
    setLoading: (key, loading) => {
      set((state) => ({
        loading: { ...state.loading, [key]: loading },
      }));
    },
  })
);

// Initialize theme on app start
if (typeof window !== 'undefined') {
  const { theme } = useUIStore.getState();
  applyTheme(theme);
}