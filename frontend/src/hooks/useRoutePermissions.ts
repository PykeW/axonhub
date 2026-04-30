import { useCallback, useMemo } from 'react';
import { routeConfigs, type RouteConfig, type RouteGroup, type ScopeLevel } from '@/config/route-permission';
import { useAuthStore } from '@/stores/authStore';
import { useSelectedProjectId } from '@/stores/projectStore';
import { type NavGroup, type NavItem } from '@/components/layout/types';
import { useMe } from '@/features/auth/data/auth';

export function useRoutePermissions() {
  const { user: authUser } = useAuthStore((state) => state.auth);
  const { data: meData } = useMe();
  const selectedProjectId = useSelectedProjectId();

  // Use data from me query if available, otherwise fall back to auth store
  const user = meData || authUser;
  const systemScopes = useMemo(() => user?.scopes || [], [user?.scopes]);
  const isOwner = user?.isOwner || false;

  // Get project-level scopes for the selected project
  const projectScopes = useMemo(() => {
    if (!selectedProjectId || !user?.projects) {
      return [];
    }
    const project = user.projects.find((p) => p.projectID === selectedProjectId);
    return project?.scopes || [];
  }, [selectedProjectId, user?.projects]);

  const hasAllScopes = useCallback(
    (scopes: string[], requiredScopes: string[]) => scopes.includes('*') || requiredScopes.every((scope) => scopes.includes(scope)),
    []
  );

  const canAccessScopes = useCallback((requiredScopes: string[] = [], scopeLevel: ScopeLevel = 'any'): boolean => {
    if (requiredScopes.length === 0) {
      return true;
    }

    // Owner 拥有所有权限
    if (isOwner) {
      return true;
    }

    if (scopeLevel === 'system') {
      return hasAllScopes(systemScopes, requiredScopes);
    }

    if (scopeLevel === 'project') {
      return hasAllScopes(projectScopes, requiredScopes);
    }

    // Project REST 后端允许“全量系统权限”或“全量项目权限”，不能跨级别拼接 scope。
    return hasAllScopes(systemScopes, requiredScopes) || hasAllScopes(projectScopes, requiredScopes);
  }, [hasAllScopes, isOwner, projectScopes, systemScopes]);

  // 检查路由权限（根据 scopeLevel 决定检查哪个级别的权限）
  const hasRouteAccess = useCallback((routeConfig: RouteConfig, groupScopeLevel?: ScopeLevel): boolean => {
    const scopeLevel = routeConfig.scopeLevel || groupScopeLevel || 'any';
    return canAccessScopes(routeConfig.requiredScopes, scopeLevel);
  }, [canAccessScopes]);

  // 检查路由组权限
  const hasGroupAccess = useCallback((group: RouteGroup): boolean => {
    return group.routes.some((route) => hasRouteAccess(route, group.scopeLevel) || route.children?.some((child) => hasRouteAccess(child, group.scopeLevel)));
  }, [hasRouteAccess]);
  // 检查单个路由权限
  const checkRouteAccess = useMemo(() => {
    return (path: string): { hasAccess: boolean; mode?: 'hidden' | 'disabled' } => {
      const { routeConfig, groupScopeLevel } = getRouteConfigByPathWithGroup(path);
      if (!routeConfig) {
        return { hasAccess: true };
      }

      const access = hasRouteAccess(routeConfig, groupScopeLevel);
      return {
        hasAccess: access,
        mode: routeConfig.mode,
      };
    };
  }, [hasRouteAccess]);

  // 检查路由组权限
  const checkGroupAccess = useMemo(() => {
    return (group: RouteGroup): boolean => {
      return hasGroupAccess(group);
    };
  }, [hasGroupAccess]);

  // 过滤导航项
  const filterNavItems = useMemo(() => {
    return (items: NavItem[]): NavItem[] => {
      return items
        .filter((item) => {
          if ('url' in item) {
            const access = checkRouteAccess(item.url as string);

            // 如果是隐藏模式且没有权限，则过滤掉
            if (!access.hasAccess && access.mode === 'hidden') {
              return false;
            }
          }

          return true;
        })
        .map((item) => {
          if ('url' in item) {
            const access = checkRouteAccess(item.url as string);

            return {
              ...item,
              isDisabled: !access.hasAccess && access.mode === 'disabled',
            };
          }

          return item;
        });
    };
  }, [checkRouteAccess]);

  // 过滤导航组
  const filterNavGroups = useMemo(() => {
    return (groups: NavGroup[]): NavGroup[] => {
      return groups
        .filter((group) => {
          // 找到对应的路由组配置
          const configTitle = group.routeConfigTitle ?? group.title;
          const routeGroup = routeConfigs.find((rg) => rg.title === configTitle);
          if (!routeGroup) {
            return true; // 如果没有配置，默认显示
          }

          // 检查组是否有可访问的路由
          return checkGroupAccess(routeGroup);
        })
        .map((group) => ({
          ...group,
          items: filterNavItems(group.items),
        }));
    };
  }, [checkGroupAccess, filterNavItems]);

  return {
    systemScopes,
    projectScopes,
    userScopes: [...systemScopes, ...projectScopes],
    isOwner,
    canAccessScopes,
    checkRouteAccess,
    checkGroupAccess,
    filterNavItems,
    filterNavGroups,
  };
}

// 辅助函数：根据路径查找路由配置及其所属组的 scopeLevel
function getRouteConfigByPathWithGroup(path: string): {
  routeConfig?: RouteConfig;
  groupScopeLevel?: ScopeLevel;
} {
  for (const group of routeConfigs) {
    for (const route of group.routes) {
      if (routePathMatches(route.path, path)) {
        return { routeConfig: route, groupScopeLevel: group.scopeLevel };
      }
      if (route.children) {
        const childConfig = route.children.find((child) => routePathMatches(child.path, path));
        if (childConfig) {
          return { routeConfig: childConfig, groupScopeLevel: group.scopeLevel };
        }
      }
    }
  }
  return {};
}

function routePathMatches(pattern: string, path: string): boolean {
  if (pattern === path) return true;
  const patternParts = pattern.split('/');
  const pathParts = path.split('/');
  if (patternParts.length !== pathParts.length) return false;

  return patternParts.every((part, index) => part.startsWith('$') || part === pathParts[index]);
}
