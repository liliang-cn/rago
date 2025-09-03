import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { TaskItem } from '@/lib/api'
import { 
  CheckCircle2, 
  Clock, 
  Play, 
  Plus, 
  Trash2, 
  Edit3,
  ListTodo,
  Activity
} from 'lucide-react'

interface TaskWithId extends TaskItem {
  id: string
  createdAt: Date
}

export function TasksTab() {
  const [tasks, setTasks] = useState<TaskWithId[]>([])
  const [newTaskContent, setNewTaskContent] = useState('')
  const [newTaskActiveForm, setNewTaskActiveForm] = useState('')
  const [editingTask, setEditingTask] = useState<string | null>(null)

  // Load tasks from localStorage
  useEffect(() => {
    const savedTasks = localStorage.getItem('rago-tasks')
    if (savedTasks) {
      try {
        const parsed = JSON.parse(savedTasks)
        setTasks(parsed.map((task: any) => ({
          ...task,
          createdAt: new Date(task.createdAt)
        })))
      } catch (err) {
        console.error('Failed to parse saved tasks:', err)
      }
    }
  }, [])

  // Save tasks to localStorage
  const saveTasks = (updatedTasks: TaskWithId[]) => {
    localStorage.setItem('rago-tasks', JSON.stringify(updatedTasks))
    setTasks(updatedTasks)
  }

  const addTask = () => {
    if (!newTaskContent.trim()) return

    const newTask: TaskWithId = {
      id: Date.now().toString(),
      content: newTaskContent.trim(),
      status: 'pending',
      activeForm: newTaskActiveForm.trim() || `Working on ${newTaskContent.trim()}`,
      createdAt: new Date()
    }

    saveTasks([...tasks, newTask])
    setNewTaskContent('')
    setNewTaskActiveForm('')
  }

  const updateTaskStatus = (taskId: string, status: TaskItem['status']) => {
    const updatedTasks = tasks.map(task =>
      task.id === taskId ? { ...task, status } : task
    )
    saveTasks(updatedTasks)
  }

  const deleteTask = (taskId: string) => {
    saveTasks(tasks.filter(task => task.id !== taskId))
  }

  const updateTask = (taskId: string, updates: Partial<TaskItem>) => {
    const updatedTasks = tasks.map(task =>
      task.id === taskId ? { ...task, ...updates } : task
    )
    saveTasks(updatedTasks)
    setEditingTask(null)
  }

  const getStatusIcon = (status: TaskItem['status']) => {
    switch (status) {
      case 'completed':
        return <CheckCircle2 className="h-4 w-4 text-green-600" />
      case 'in_progress':
        return <Activity className="h-4 w-4 text-blue-600 animate-pulse" />
      case 'pending':
        return <Clock className="h-4 w-4 text-gray-400" />
    }
  }

  const getStatusColor = (status: TaskItem['status']) => {
    switch (status) {
      case 'completed':
        return 'border-green-200 bg-green-50'
      case 'in_progress':
        return 'border-blue-200 bg-blue-50'
      case 'pending':
        return 'border-gray-200 bg-gray-50'
    }
  }

  const getStatusText = (status: TaskItem['status']) => {
    switch (status) {
      case 'completed':
        return 'Completed'
      case 'in_progress':
        return 'In Progress'
      case 'pending':
        return 'Pending'
    }
  }

  const pendingTasks = tasks.filter(task => task.status === 'pending')
  const inProgressTasks = tasks.filter(task => task.status === 'in_progress')
  const completedTasks = tasks.filter(task => task.status === 'completed')

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Task Management</h2>
          <p className="text-gray-600">Organize and track your work progress</p>
        </div>
        <div className="flex items-center gap-2 text-sm text-gray-500">
          <ListTodo className="h-4 w-4" />
          <span>{tasks.length} total tasks</span>
        </div>
      </div>

      {/* Add New Task */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Add New Task</CardTitle>
          <CardDescription>
            Create a new task to track your work progress
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="text-sm font-medium">Task Description</label>
            <Input
              placeholder="e.g., Implement user authentication"
              value={newTaskContent}
              onChange={(e) => setNewTaskContent(e.target.value)}
              className="mt-1"
            />
          </div>
          <div>
            <label className="text-sm font-medium">Active Form (optional)</label>
            <Input
              placeholder="e.g., Implementing user authentication"
              value={newTaskActiveForm}
              onChange={(e) => setNewTaskActiveForm(e.target.value)}
              className="mt-1"
            />
          </div>
          <Button onClick={addTask} disabled={!newTaskContent.trim()}>
            <Plus className="h-4 w-4 mr-2" />
            Add Task
          </Button>
        </CardContent>
      </Card>

      {/* Task Statistics */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card className="border-gray-200 bg-gray-50">
          <CardContent className="pt-4">
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-gray-400" />
              <div>
                <div className="text-2xl font-bold">{pendingTasks.length}</div>
                <div className="text-sm text-gray-600">Pending</div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <Card className="border-blue-200 bg-blue-50">
          <CardContent className="pt-4">
            <div className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-blue-600" />
              <div>
                <div className="text-2xl font-bold">{inProgressTasks.length}</div>
                <div className="text-sm text-gray-600">In Progress</div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <Card className="border-green-200 bg-green-50">
          <CardContent className="pt-4">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-green-600" />
              <div>
                <div className="text-2xl font-bold">{completedTasks.length}</div>
                <div className="text-sm text-gray-600">Completed</div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Tasks List */}
      <div className="space-y-4">
        {tasks.length === 0 ? (
          <Card className="text-center py-8">
            <CardContent>
              <ListTodo className="h-12 w-12 mx-auto mb-4 text-gray-400" />
              <h3 className="text-lg font-medium text-gray-900 mb-2">No tasks yet</h3>
              <p className="text-gray-600">Add your first task to get started with tracking your progress.</p>
            </CardContent>
          </Card>
        ) : (
          tasks
            .sort((a, b) => {
              // Sort by status (in_progress, pending, completed), then by creation date
              const statusOrder = { 'in_progress': 0, 'pending': 1, 'completed': 2 }
              if (statusOrder[a.status] !== statusOrder[b.status]) {
                return statusOrder[a.status] - statusOrder[b.status]
              }
              return b.createdAt.getTime() - a.createdAt.getTime()
            })
            .map((task) => (
              <Card key={task.id} className={`transition-all ${getStatusColor(task.status)}`}>
                <CardContent className="pt-4">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 space-y-2">
                      <div className="flex items-center gap-2">
                        {getStatusIcon(task.status)}
                        <span className="text-sm font-medium text-gray-600">
                          {getStatusText(task.status)}
                        </span>
                        <span className="text-xs text-gray-400">
                          {task.createdAt.toLocaleString()}
                        </span>
                      </div>
                      
                      {editingTask === task.id ? (
                        <div className="space-y-2">
                          <Input
                            defaultValue={task.content}
                            onBlur={(e) => updateTask(task.id, { content: e.target.value })}
                            className="text-sm"
                          />
                          <Input
                            defaultValue={task.activeForm}
                            onBlur={(e) => updateTask(task.id, { activeForm: e.target.value })}
                            placeholder="Active form"
                            className="text-xs"
                          />
                        </div>
                      ) : (
                        <div>
                          <p className="font-medium">{task.content}</p>
                          <p className="text-sm text-gray-600 italic">{task.activeForm}</p>
                        </div>
                      )}
                    </div>

                    <div className="flex items-center gap-2">
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => setEditingTask(editingTask === task.id ? null : task.id)}
                      >
                        <Edit3 className="h-3 w-3" />
                      </Button>

                      {task.status === 'pending' && (
                        <Button
                          size="sm"
                          onClick={() => updateTaskStatus(task.id, 'in_progress')}
                        >
                          <Play className="h-3 w-3 mr-1" />
                          Start
                        </Button>
                      )}

                      {task.status === 'in_progress' && (
                        <Button
                          size="sm"
                          onClick={() => updateTaskStatus(task.id, 'completed')}
                          className="bg-green-600 hover:bg-green-700"
                        >
                          <CheckCircle2 className="h-3 w-3 mr-1" />
                          Complete
                        </Button>
                      )}

                      {task.status === 'completed' && (
                        <Button
                          size="sm"
                          variant="secondary"
                          onClick={() => updateTaskStatus(task.id, 'pending')}
                        >
                          <Clock className="h-3 w-3 mr-1" />
                          Reopen
                        </Button>
                      )}

                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => deleteTask(task.id)}
                        className="text-red-600 hover:text-red-700 hover:bg-red-50"
                      >
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))
        )}
      </div>
    </div>
  )
}