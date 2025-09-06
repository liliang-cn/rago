# RAGO Web Frontend

A comprehensive, enterprise-grade web interface for the RAGO AI Platform built with React, TypeScript, and modern web technologies.

## 🚀 Features

### ✅ Implemented
- **Real-time Dashboard** - System monitoring with live metrics and health status
- **Document Management** - Drag-and-drop upload, processing status, and RAG interface
- **AI Query Interface** - Natural language search with source references and streaming responses
- **Visual Workflow Designer** - Drag-and-drop workflow builder with React Flow
- **Responsive Design** - Mobile-first approach with adaptive layouts
- **Dark/Light Themes** - User preference with system detection
- **Real-time Updates** - WebSocket integration for live data
- **Type Safety** - Full TypeScript implementation with strict types

### 🚧 In Progress
- **Agent Marketplace** - Template browser and management system
- **Job Scheduler** - Cron-based task automation interface
- **Provider Management** - LLM/AI service configuration and monitoring
- **Settings & Configuration** - System-wide configuration management
- **Advanced Monitoring** - Distributed tracing and performance analytics

## 🏗️ Architecture

### Technology Stack
- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite for fast development and optimized builds
- **Styling**: Tailwind CSS with custom design system
- **State Management**: Zustand with persistence and middleware
- **Data Fetching**: TanStack Query for server state management
- **UI Components**: Radix UI primitives with custom styling
- **Charts**: Recharts for data visualization
- **Workflow Designer**: React Flow for visual DAG editing
- **File Uploads**: React Dropzone for drag-and-drop uploads
- **Code Editor**: Monaco Editor integration
- **Real-time**: WebSocket client with auto-reconnection

### Project Structure
```
src/
├── components/
│   ├── ui/           # Reusable UI components (Shadcn/ui)
│   ├── dashboard/    # Dashboard-specific components
│   ├── marketplace/  # Marketplace components
│   ├── workflow/     # Workflow designer components
│   ├── scheduler/    # Job scheduler UI
│   └── common/       # Shared components (Layout, Header, Sidebar)
├── pages/            # Route-level components
├── hooks/            # Custom React hooks
├── services/         # API clients and WebSocket
├── store/            # Zustand state management
├── types/            # TypeScript type definitions
└── utils/            # Utility functions and helpers
```

### State Management
- **UI Store**: Theme, sidebar, notifications, active view
- **System Store**: Providers, metrics, component status, alerts, MCP servers
- **Documents Store**: Document management and query history
- **Workflows Store**: Workflow definitions and execution history
- **Jobs Store**: Scheduled jobs and execution tracking

## 🛠️ Development

### Prerequisites
- Node.js 18+ and npm/yarn
- RAGO backend server running on port 7127

### Installation
```bash
cd pkg/web
npm install
```

### Development Server
```bash
npm run dev
```
Starts development server on http://localhost:3000 with:
- Hot module replacement
- Proxy to backend API on port 7127
- WebSocket connection for real-time updates

### Build for Production
```bash
npm run build
```
Creates optimized production build in `dist/` directory with:
- Code splitting and tree shaking
- Asset optimization and compression
- Source maps for debugging

### Type Checking
```bash
npm run type-check
```

### Linting
```bash
npm run lint
```

## 🎨 Design System

### Color Scheme
- **Primary**: Main brand color for actions and highlights
- **Secondary**: Supporting colors for less prominent elements
- **Accent**: Interactive states and emphasis
- **Muted**: Subtle backgrounds and disabled states
- **Destructive**: Error states and dangerous actions

### Typography
- **Headings**: Clear hierarchy with responsive scaling
- **Body Text**: Optimized for readability across devices
- **Code**: Monospace font for technical content

### Components
- **Consistent API**: All components follow the same prop patterns
- **Accessibility**: WCAG 2.1 compliant with proper ARIA labels
- **Responsive**: Mobile-first design with adaptive breakpoints
- **Themeable**: Full dark/light mode support

## 📱 Responsive Design

### Breakpoints
- **Mobile**: < 768px - Single column layout with collapsed sidebar
- **Tablet**: 768px - 1024px - Adaptive grid with mobile navigation
- **Desktop**: > 1024px - Full multi-column layout with persistent sidebar

### Mobile Optimizations
- Touch-friendly interactive elements
- Swipe gestures for navigation
- Optimized viewport and font scaling
- Progressive enhancement

## 🔄 Real-time Features

### WebSocket Integration
- **Auto-reconnection**: Automatic reconnection with exponential backoff
- **Event Subscription**: Type-safe event handling system
- **Real-time Updates**: Live metrics, status changes, and notifications

### Supported Events
- **Metrics Updates**: System performance data
- **Status Changes**: Component health notifications  
- **Workflow Events**: Execution progress and completion
- **Document Events**: Processing status updates

## 🚦 API Integration

### HTTP Client
- **Type-safe**: Full TypeScript support with response typing
- **Error Handling**: Centralized error management with user-friendly messages
- **Authentication**: Token-based auth with automatic refresh
- **Caching**: Intelligent caching with TanStack Query

### Endpoints
- `/api/documents` - Document management and RAG operations
- `/api/workflows` - Workflow CRUD and execution
- `/api/jobs` - Job scheduling and management
- `/api/providers` - Provider configuration and monitoring
- `/api/marketplace` - Agent template marketplace
- `/api/monitoring` - System metrics and health
- `/api/settings` - Configuration management

## 🎯 Performance

### Optimization Strategies
- **Code Splitting**: Route-based and component-based splitting
- **Lazy Loading**: Dynamic imports for heavy components
- **Bundle Analysis**: Webpack bundle analyzer integration
- **Asset Optimization**: Image compression and CDN integration

### Core Web Vitals
- **Largest Contentful Paint (LCP)**: < 2.5s
- **First Input Delay (FID)**: < 100ms
- **Cumulative Layout Shift (CLS)**: < 0.1

### Caching Strategy
- **Static Assets**: Long-term caching with content hashing
- **API Responses**: Smart caching with TTL and invalidation
- **Local Storage**: Persistent user preferences and session data

## 🧪 Testing Strategy

### Unit Testing
- **Jest**: JavaScript testing framework
- **Testing Library**: Component testing utilities
- **MSW**: API mocking for isolated tests

### Integration Testing
- **Cypress**: End-to-end testing for critical user flows
- **Playwright**: Cross-browser testing automation

### Accessibility Testing
- **axe-core**: Automated accessibility testing
- **Screen Reader**: Manual testing with assistive technologies

## 🚀 Deployment

### Production Build
```bash
npm run build
npm run preview  # Preview production build locally
```

### Environment Variables
```env
VITE_API_BASE_URL=http://localhost:7127
VITE_WS_BASE_URL=ws://localhost:7127
VITE_APP_TITLE=RAGO Platform
```

### Docker Deployment
```dockerfile
FROM node:18-alpine as build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

## 📊 Monitoring & Analytics

### Error Tracking
- **Sentry**: Production error monitoring and performance tracking
- **Console Logging**: Structured logging in development

### Performance Monitoring
- **Web Vitals**: Automatic Core Web Vitals reporting
- **Custom Metrics**: Application-specific performance tracking

### User Analytics
- **Usage Patterns**: Feature adoption and user flows
- **Performance Impact**: Real-user monitoring (RUM)

## 🔐 Security

### Client-side Security
- **Content Security Policy**: XSS prevention
- **HTTPS Only**: Secure communication enforced
- **Input Validation**: Client-side validation with server verification
- **Dependency Auditing**: Regular security vulnerability scans

### Data Protection
- **Local Storage**: Encrypted sensitive data storage
- **Session Management**: Secure token handling
- **CORS**: Proper cross-origin request configuration

## 📚 Contributing

### Development Guidelines
1. Follow TypeScript strict mode
2. Use semantic commit messages
3. Add tests for new features
4. Document complex logic
5. Maintain accessibility standards

### Code Style
- **Prettier**: Automatic code formatting
- **ESLint**: Code quality and consistency
- **Husky**: Pre-commit hooks for quality gates

## 📝 License

This project is part of the RAGO AI Platform. See the main project LICENSE file for details.

---

## 🎉 Getting Started

1. **Start the RAGO backend server**
2. **Install dependencies**: `npm install`
3. **Start development server**: `npm run dev`
4. **Open browser**: http://localhost:3000
5. **Begin exploring** the comprehensive AI platform interface!

The web interface provides an intuitive way to interact with all RAGO platform features, from document management and AI queries to workflow automation and system monitoring.