import React from 'react';
import { Button, ButtonProps } from '@/components/ui/button';
import { cn } from '@/utils';

interface AccessibleButtonProps extends ButtonProps {
  'aria-label'?: string;
  'aria-describedby'?: string;
  loadingText?: string;
  isLoading?: boolean;
}

export function AccessibleButton({
  children,
  'aria-label': ariaLabel,
  'aria-describedby': ariaDescribedby,
  loadingText = 'Loading...',
  isLoading = false,
  disabled,
  className,
  ...props
}: AccessibleButtonProps) {
  return (
    <Button
      {...props}
      className={cn(className)}
      disabled={disabled || isLoading}
      aria-label={isLoading ? loadingText : ariaLabel}
      aria-describedby={ariaDescribedby}
      aria-busy={isLoading}
    >
      {isLoading ? loadingText : children}
    </Button>
  );
}