import React, { useState } from 'react';
import { 
  Settings2, 
  User, 
  Palette, 
  Database, 
  Shield, 
  Bell,
  Globe,
  Save,
  RotateCcw,
  Download,
  Upload,
  Trash2,
  Key,
  Monitor,
  Sun,
  Moon
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { useUIStore } from '@/store';
import { cn } from '@/utils';

// Settings Section Interface
interface SettingsSection {
  id: string;
  title: string;
  description: string;
  icon: React.ElementType;
}

const settingsSections: SettingsSection[] = [
  {
    id: 'general',
    title: 'General',
    description: 'Basic application settings and preferences',
    icon: Settings2,
  },
  {
    id: 'appearance',
    title: 'Appearance',
    description: 'Theme, layout, and visual preferences',
    icon: Palette,
  },
  {
    id: 'providers',
    title: 'AI Providers',
    description: 'Configure LLM and embedding providers',
    icon: Key,
  },
  {
    id: 'storage',
    title: 'Storage',
    description: 'Database and file storage settings',
    icon: Database,
  },
  {
    id: 'security',
    title: 'Security',
    description: 'Authentication and access control',
    icon: Shield,
  },
  {
    id: 'notifications',
    title: 'Notifications',
    description: 'Alert and notification preferences',
    icon: Bell,
  },
  {
    id: 'api',
    title: 'API',
    description: 'API endpoints and integration settings',
    icon: Globe,
  },
];

// General Settings Component
function GeneralSettings() {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">General Settings</h3>
        <p className="text-sm text-muted-foreground">
          Configure basic application behavior and preferences
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="text-sm font-medium mb-2 block">Application Name</label>
          <Input defaultValue="RAGO AI Platform" />
          <p className="text-xs text-muted-foreground mt-1">
            Display name shown in the interface
          </p>
        </div>

        <div>
          <label className="text-sm font-medium mb-2 block">Default Language</label>
          <select className="w-full p-2 border rounded-md bg-background">
            <option value="en">English</option>
            <option value="es">Spanish</option>
            <option value="fr">French</option>
            <option value="de">German</option>
          </select>
        </div>

        <div>
          <label className="text-sm font-medium mb-2 block">Session Timeout (minutes)</label>
          <Input type="number" defaultValue="60" min="5" max="1440" />
        </div>

        <div className="flex items-center gap-2">
          <input type="checkbox" id="auto-save" defaultChecked />
          <label htmlFor="auto-save" className="text-sm">Enable auto-save</label>
        </div>

        <div className="flex items-center gap-2">
          <input type="checkbox" id="analytics" defaultChecked />
          <label htmlFor="analytics" className="text-sm">Share anonymous usage analytics</label>
        </div>
      </div>
    </div>
  );
}

// Appearance Settings Component
function AppearanceSettings() {
  const { theme, setTheme } = useUIStore();

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">Appearance</h3>
        <p className="text-sm text-muted-foreground">
          Customize the look and feel of the application
        </p>
      </div>

      <div className="space-y-4">
        <div>
          <label className="text-sm font-medium mb-3 block">Theme</label>
          <div className="grid grid-cols-3 gap-3">
            {[
              { value: 'light', label: 'Light', icon: Sun },
              { value: 'dark', label: 'Dark', icon: Moon },
              { value: 'system', label: 'System', icon: Monitor },
            ].map(({ value, label, icon: Icon }) => (
              <button
                key={value}
                onClick={() => setTheme(value as any)}
                className={cn(
                  'flex flex-col items-center gap-2 p-4 border rounded-lg transition-colors',
                  theme === value 
                    ? 'border-primary bg-primary/5' 
                    : 'border-border hover:bg-accent'
                )}
              >
                <Icon className="h-5 w-5" />
                <span className="text-sm font-medium">{label}</span>
              </button>
            ))}
          </div>
        </div>

        <div>
          <label className="text-sm font-medium mb-2 block">Primary Color</label>
          <div className="grid grid-cols-8 gap-2">
            {[
              '#3b82f6', '#ef4444', '#10b981', '#f59e0b',
              '#8b5cf6', '#ec4899', '#06b6d4', '#84cc16'
            ].map((color) => (
              <button
                key={color}
                className="w-8 h-8 rounded-md border border-border"
                style={{ backgroundColor: color }}
                onClick={() => {
                  document.documentElement.style.setProperty('--primary', color);
                }}
              />
            ))}
          </div>
        </div>

        <div>
          <label className="text-sm font-medium mb-2 block">Font Size</label>
          <select className="w-full p-2 border rounded-md bg-background">
            <option value="small">Small</option>
            <option value="medium" selected>Medium</option>
            <option value="large">Large</option>
          </select>
        </div>

        <div className="flex items-center gap-2">
          <input type="checkbox" id="compact-mode" />
          <label htmlFor="compact-mode" className="text-sm">Compact interface mode</label>
        </div>

        <div className="flex items-center gap-2">
          <input type="checkbox" id="animations" defaultChecked />
          <label htmlFor="animations" className="text-sm">Enable animations</label>
        </div>
      </div>
    </div>
  );
}

// Provider Settings Component
function ProviderSettings() {
  const [providers, setProviders] = useState([
    { id: '1', name: 'OpenAI GPT-4', type: 'LLM', status: 'Active', endpoint: 'https://api.openai.com/v1' },
    { id: '2', name: 'OpenAI Embeddings', type: 'Embeddings', status: 'Active', endpoint: 'https://api.openai.com/v1' },
    { id: '3', name: 'Local Ollama', type: 'LLM', status: 'Inactive', endpoint: 'http://localhost:11434' },
  ]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium">AI Providers</h3>
          <p className="text-sm text-muted-foreground">
            Configure your LLM and embedding providers
          </p>
        </div>
        <Button>
          <Key className="h-4 w-4 mr-2" />
          Add Provider
        </Button>
      </div>

      <div className="space-y-3">
        {providers.map((provider) => (
          <Card key={provider.id}>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <h4 className="font-medium">{provider.name}</h4>
                    <Badge variant={provider.status === 'Active' ? 'success' : 'secondary'}>
                      {provider.status}
                    </Badge>
                    <Badge variant="outline">{provider.type}</Badge>
                  </div>
                  <p className="text-sm text-muted-foreground mt-1">{provider.endpoint}</p>
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm">Test</Button>
                  <Button variant="outline" size="sm">Configure</Button>
                  <Button variant="outline" size="sm">
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

// Main Settings Component
export function Settings() {
  const [activeSection, setActiveSection] = useState('general');

  const renderSettingsContent = () => {
    switch (activeSection) {
      case 'general':
        return <GeneralSettings />;
      case 'appearance':
        return <AppearanceSettings />;
      case 'providers':
        return <ProviderSettings />;
      case 'storage':
        return (
          <div className="space-y-4">
            <h3 className="text-lg font-medium">Storage Settings</h3>
            <p className="text-muted-foreground">Database and file storage configuration</p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Database</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-sm">Type:</span>
                      <span className="text-sm font-medium">SQLite</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-sm">Size:</span>
                      <span className="text-sm font-medium">245 MB</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-sm">Documents:</span>
                      <span className="text-sm font-medium">1,247</span>
                    </div>
                  </div>
                  <Button className="w-full mt-4" variant="outline">
                    <Database className="h-4 w-4 mr-2" />
                    Optimize Database
                  </Button>
                </CardContent>
              </Card>
              
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">File Storage</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex justify-between">
                      <span className="text-sm">Location:</span>
                      <span className="text-sm font-medium">~/.rago/</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-sm">Used:</span>
                      <span className="text-sm font-medium">1.2 GB</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-sm">Available:</span>
                      <span className="text-sm font-medium">50 GB</span>
                    </div>
                  </div>
                  <Button className="w-full mt-4" variant="outline">
                    <Trash2 className="h-4 w-4 mr-2" />
                    Clean Temp Files
                  </Button>
                </CardContent>
              </Card>
            </div>
          </div>
        );
      default:
        return (
          <div className="space-y-4">
            <h3 className="text-lg font-medium">{settingsSections.find(s => s.id === activeSection)?.title}</h3>
            <p className="text-muted-foreground">
              This settings section is not yet implemented. It will be available in a future update.
            </p>
            <div className="p-8 border-2 border-dashed border-muted-foreground/25 rounded-lg text-center">
              <Settings2 className="h-12 w-12 mx-auto mb-4 text-muted-foreground opacity-50" />
              <p className="text-sm text-muted-foreground">Settings panel coming soon</p>
            </div>
          </div>
        );
    }
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
      {/* Settings Navigation */}
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Settings</h1>
          <p className="text-muted-foreground">
            Configure your RAGO platform
          </p>
        </div>

        <Card>
          <CardContent className="p-0">
            <nav className="space-y-1 p-2">
              {settingsSections.map((section) => {
                const Icon = section.icon;
                return (
                  <button
                    key={section.id}
                    onClick={() => setActiveSection(section.id)}
                    className={cn(
                      'w-full flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md transition-colors text-left',
                      activeSection === section.id
                        ? 'bg-primary text-primary-foreground'
                        : 'hover:bg-accent'
                    )}
                  >
                    <Icon className="h-4 w-4" />
                    <div className="flex-1">
                      <div>{section.title}</div>
                    </div>
                  </button>
                );
              })}
            </nav>
          </CardContent>
        </Card>

        {/* Quick Actions */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Quick Actions</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <Button variant="outline" className="w-full justify-start">
              <Download className="h-4 w-4 mr-2" />
              Export Settings
            </Button>
            <Button variant="outline" className="w-full justify-start">
              <Upload className="h-4 w-4 mr-2" />
              Import Settings
            </Button>
            <Button variant="outline" className="w-full justify-start">
              <RotateCcw className="h-4 w-4 mr-2" />
              Reset to Defaults
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Settings Content */}
      <div className="lg:col-span-3">
        <Card>
          <CardContent className="p-6">
            {renderSettingsContent()}
            
            {/* Save Actions */}
            <div className="flex justify-end gap-2 mt-8 pt-6 border-t">
              <Button variant="outline">Cancel</Button>
              <Button>
                <Save className="h-4 w-4 mr-2" />
                Save Changes
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}