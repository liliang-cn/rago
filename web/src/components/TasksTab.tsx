import { useState, useEffect } from 'react'
import { Card, Button, Input, Space, Typography, Statistic, Row, Col, Empty } from 'antd'
import { 
  CheckCircleOutlined, 
  ClockCircleOutlined, 
  PlayCircleOutlined, 
  PlusOutlined, 
  DeleteOutlined, 
  EditOutlined,
  UnorderedListOutlined,
  DashboardOutlined
} from '@ant-design/icons'
import { TaskItem } from '@/lib/api'

const { Title, Text } = Typography

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
        return <CheckCircleOutlined style={{ color: '#52c41a' }} />
      case 'in_progress':
        return <DashboardOutlined style={{ color: '#1890ff' }} />
      case 'pending':
        return <ClockCircleOutlined style={{ color: '#bfbfbf' }} />
    }
  }

  const getStatusColor = (status: TaskItem['status']) => {
    switch (status) {
      case 'completed':
        return { borderColor: '#52c41a', backgroundColor: '#f6ffed' }
      case 'in_progress':
        return { borderColor: '#1890ff', backgroundColor: '#f0f9ff' }
      case 'pending':
        return { borderColor: '#d9d9d9', backgroundColor: '#fafafa' }
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
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <Title level={2} style={{ margin: 0 }}>Task Management</Title>
            <Text type="secondary">Organize and track your work progress</Text>
          </div>
          <Space>
            <UnorderedListOutlined />
            <Text type="secondary">{tasks.length} total tasks</Text>
          </Space>
        </div>

        <Card
          title={
            <Space>
              <PlusOutlined />
              <span>Add New Task</span>
            </Space>
          }
        >
          <Text type="secondary" style={{ marginBottom: 16, display: 'block' }}>
            Create a new task to track your work progress
          </Text>
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <div>
              <Text strong style={{ marginBottom: 8, display: 'block' }}>Task Description</Text>
              <Input
                placeholder="e.g., Implement user authentication"
                value={newTaskContent}
                onChange={(e) => setNewTaskContent(e.target.value)}
              />
            </div>
            <div>
              <Text strong style={{ marginBottom: 8, display: 'block' }}>Active Form (optional)</Text>
              <Input
                placeholder="e.g., Implementing user authentication"
                value={newTaskActiveForm}
                onChange={(e) => setNewTaskActiveForm(e.target.value)}
              />
            </div>
            <Button 
              type="primary"
              icon={<PlusOutlined />}
              onClick={addTask} 
              disabled={!newTaskContent.trim()}
            >
              Add Task
            </Button>
          </Space>
        </Card>

        <Row gutter={[16, 16]}>
          <Col xs={24} sm={8}>
            <Card size="small" style={{ borderColor: '#d9d9d9', backgroundColor: '#fafafa' }}>
              <Statistic
                title="Pending"
                value={pendingTasks.length}
                prefix={<ClockCircleOutlined style={{ color: '#bfbfbf' }} />}
                valueStyle={{ color: '#8c8c8c' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card size="small" style={{ borderColor: '#1890ff', backgroundColor: '#f0f9ff' }}>
              <Statistic
                title="In Progress"
                value={inProgressTasks.length}
                prefix={<DashboardOutlined style={{ color: '#1890ff' }} />}
                valueStyle={{ color: '#1890ff' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card size="small" style={{ borderColor: '#52c41a', backgroundColor: '#f6ffed' }}>
              <Statistic
                title="Completed"
                value={completedTasks.length}
                prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
                valueStyle={{ color: '#52c41a' }}
              />
            </Card>
          </Col>
        </Row>

        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          {tasks.length === 0 ? (
            <Empty
              image={<UnorderedListOutlined style={{ fontSize: 48, color: '#bfbfbf' }} />}
              description={
                <div style={{ textAlign: 'center' }}>
                  <Title level={4} style={{ color: '#8c8c8c' }}>No tasks yet</Title>
                  <Text type="secondary">Add your first task to get started with tracking your progress.</Text>
                </div>
              }
            />
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
                <Card 
                  key={task.id} 
                  style={getStatusColor(task.status)}
                  size="small"
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 16 }}>
                    <div style={{ flex: 1 }}>
                      <Space style={{ marginBottom: 8 }}>
                        {getStatusIcon(task.status)}
                        <Text strong style={{ fontSize: 12 }}>
                          {getStatusText(task.status)}
                        </Text>
                        <Text type="secondary" style={{ fontSize: 10 }}>
                          {task.createdAt.toLocaleString()}
                        </Text>
                      </Space>
                      
                      {editingTask === task.id ? (
                        <Space direction="vertical" style={{ width: '100%' }} size="small">
                          <Input
                            defaultValue={task.content}
                            onBlur={(e) => updateTask(task.id, { content: e.target.value })}
                            size="small"
                          />
                          <Input
                            defaultValue={task.activeForm}
                            onBlur={(e) => updateTask(task.id, { activeForm: e.target.value })}
                            placeholder="Active form"
                            size="small"
                          />
                        </Space>
                      ) : (
                        <div>
                          <Text strong style={{ display: 'block' }}>{task.content}</Text>
                          <Text type="secondary" style={{ fontSize: 12, fontStyle: 'italic' }}>{task.activeForm}</Text>
                        </div>
                      )}
                    </div>

                    <Space>
                      <Button
                        size="small"
                        type="text"
                        icon={<EditOutlined />}
                        onClick={() => setEditingTask(editingTask === task.id ? null : task.id)}
                      />

                      {task.status === 'pending' && (
                        <Button
                          size="small"
                          type="primary"
                          icon={<PlayCircleOutlined />}
                          onClick={() => updateTaskStatus(task.id, 'in_progress')}
                        >
                          Start
                        </Button>
                      )}

                      {task.status === 'in_progress' && (
                        <Button
                          size="small"
                          type="primary"
                          style={{ backgroundColor: '#52c41a', borderColor: '#52c41a' }}
                          icon={<CheckCircleOutlined />}
                          onClick={() => updateTaskStatus(task.id, 'completed')}
                        >
                          Complete
                        </Button>
                      )}

                      {task.status === 'completed' && (
                        <Button
                          size="small"
                          icon={<ClockCircleOutlined />}
                          onClick={() => updateTaskStatus(task.id, 'pending')}
                        >
                          Reopen
                        </Button>
                      )}

                      <Button
                        size="small"
                        type="text"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => deleteTask(task.id)}
                      />
                    </Space>
                  </div>
                </Card>
              ))
          )}
        </Space>
      </Space>
    </div>
  )
}