import React from 'react';
import { AlertTriangle, XCircle, Info, CheckCircle, X, RefreshCw } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { cn } from '@/utils';

export type AlertType = 'error' | 'warning' | 'info' | 'success';

interface ErrorMessageProps {
  type?: AlertType;
  title?: string;
  message: string;
  onRetry?: () => void;
  onDismiss?: () => void;
  className?: string;
  showIcon?: boolean;
}

const alertConfig = {
  error: {
    icon: XCircle,
    className: 'border-red-200 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400',
    iconClassName: 'text-red-500'
  },
  warning: {
    icon: AlertTriangle,
    className: 'border-yellow-200 bg-yellow-50 text-yellow-800 dark:border-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400',
    iconClassName: 'text-yellow-500'
  },
  info: {
    icon: Info,
    className: 'border-blue-200 bg-blue-50 text-blue-800 dark:border-blue-800 dark:bg-blue-900/20 dark:text-blue-400',
    iconClassName: 'text-blue-500'
  },
  success: {
    icon: CheckCircle,
    className: 'border-green-200 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-900/20 dark:text-green-400',
    iconClassName: 'text-green-500'
  }
};

export function ErrorMessage({ 
  type = 'error',
  title,
  message, 
  onRetry, 
  onDismiss, 
  className,
  showIcon = true
}: ErrorMessageProps) {
  const config = alertConfig[type];
  const Icon = config.icon;

  return (
    <div className={cn('relative', className)}>
      <Alert className={cn(config.className)}>
        {showIcon && <Icon className={cn('h-4 w-4', config.iconClassName)} />}
        {title && <AlertTitle>{title}</AlertTitle>}
        <AlertDescription className="flex items-center justify-between">
          <span className="flex-1">{message}</span>
          <div className="flex items-center gap-2 ml-4">
            {onRetry && (
              <Button 
                variant="outline" 
                size="sm"
                onClick={onRetry}
                className="h-7 px-2 text-xs"
              >
                <RefreshCw className="h-3 w-3 mr-1" />
                Retry
              </Button>
            )}
            {onDismiss && (
              <Button 
                variant="ghost" 
                size="sm"
                onClick={onDismiss}
                className="h-7 w-7 p-0"
              >
                <X className="h-3 w-3" />
              </Button>
            )}
          </div>
        </AlertDescription>
      </Alert>
    </div>
  );
}

// Inline error component for forms
export function InlineError({ message, className }: { message: string; className?: string }) {
  return (
    <div className={cn('flex items-center gap-1 text-sm text-red-600 dark:text-red-400', className)}>
      <XCircle className="h-3 w-3" />
      <span>{message}</span>
    </div>
  );
}

// Toast-style error notification
export function ErrorToast({ 
  message, 
  onClose 
}: { 
  message: string; 
  onClose: () => void;
}) {
  return (
    <div className="fixed top-4 right-4 z-50 max-w-sm">
      <Alert className="border-red-200 bg-red-50 text-red-800 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400 shadow-lg">
        <XCircle className="h-4 w-4 text-red-500" />
        <AlertDescription className="flex items-center justify-between">
          <span className="flex-1 pr-2">{message}</span>
          <Button 
            variant="ghost" 
            size="sm"
            onClick={onClose}
            className="h-6 w-6 p-0 text-red-600 hover:text-red-700"
          >
            <X className="h-3 w-3" />
          </Button>
        </AlertDescription>
      </Alert>
    </div>
  );
}

// Network error specific component
export function NetworkError({ onRetry, className }: { onRetry?: () => void; className?: string }) {
  return (
    <ErrorMessage
      type="error"
      title="Connection Failed"
      message="Unable to connect to the server. Please check your network connection and try again."
      onRetry={onRetry}
      className={className}
    />
  );
}

// Generic API error component
export function APIError({ 
  error, 
  onRetry, 
  className 
}: { 
  error: any; 
  onRetry?: () => void;
  className?: string;
}) {
  const message = error?.message || error?.error?.message || 'An unexpected error occurred';
  const isNetworkError = message.includes('fetch') || message.includes('NetworkError');
  
  if (isNetworkError) {
    return <NetworkError onRetry={onRetry} className={className} />;
  }
  
  return (
    <ErrorMessage
      type="error"
      title="API Error"
      message={message}
      onRetry={onRetry}
      className={className}
    />
  );
}