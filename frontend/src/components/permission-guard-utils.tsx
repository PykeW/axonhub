import React from 'react';
import { usePermissions } from '@/hooks/usePermissions';
import { PermissionGuard, type PermissionGuardProps } from './permission-guard';

/**
 * Higher-order component version of PermissionGuard
 */
export function withPermissionGuard<P extends object>(Component: React.ComponentType<P>, guardProps: Omit<PermissionGuardProps, 'children'>) {
  return function PermissionWrappedComponent(props: P) {
    return (
      <PermissionGuard {...guardProps}>
        <Component {...props} />
      </PermissionGuard>
    );
  };
}

/**
 * Hook version for conditional rendering based on permissions
 */
export function usePermissionCheck(
  requiredScopes?: string[],
  requiredAllScopes?: string[],
  requiredScope?: string,
  requiredSystemScopes?: string[],
  requiredAllSystemScopes?: string[],
  requiredSystemScope?: string,
  requiredProjectScopes?: string[],
  requiredAllProjectScopes?: string[],
  requiredProjectScope?: string
) {
  const { hasAnyScope, hasAllScopes, hasSystemScope, hasProjectScope } = usePermissions();

  let finalRequiredScopes: string[] = requiredScopes || [];
  if (requiredScope) {
    finalRequiredScopes = [requiredScope];
  }

  let finalRequiredSystemScopes: string[] = requiredSystemScopes || [];
  if (requiredSystemScope) {
    finalRequiredSystemScopes = [requiredSystemScope];
  }

  let finalRequiredProjectScopes: string[] = requiredProjectScopes || [];
  if (requiredProjectScope) {
    finalRequiredProjectScopes = [requiredProjectScope];
  }

  let hasPermission = true;

  // Check system-level scopes
  if (requiredAllSystemScopes && requiredAllSystemScopes.length > 0) {
    hasPermission = hasPermission && requiredAllSystemScopes.every((scope) => hasSystemScope(scope));
  }
  if (finalRequiredSystemScopes.length > 0) {
    hasPermission = hasPermission && finalRequiredSystemScopes.some((scope) => hasSystemScope(scope));
  }

  // Check project-level scopes
  if (requiredAllProjectScopes && requiredAllProjectScopes.length > 0) {
    hasPermission = hasPermission && requiredAllProjectScopes.every((scope) => hasProjectScope(scope));
  }
  if (finalRequiredProjectScopes.length > 0) {
    hasPermission = hasPermission && finalRequiredProjectScopes.some((scope) => hasProjectScope(scope));
  }

  // Check any-level scopes
  if (requiredAllScopes && requiredAllScopes.length > 0) {
    hasPermission = hasPermission && hasAllScopes(requiredAllScopes);
  }
  if (finalRequiredScopes.length > 0) {
    hasPermission = hasPermission && hasAnyScope(finalRequiredScopes);
  }

  return { hasPermission };
}
