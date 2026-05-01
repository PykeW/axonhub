// 路由权限配置
export type ScopeLevel = 'system' | 'project' | 'any';

export interface RouteConfig {
  path: string;
  requiredScopes?: string[];
  scopeLevel?: ScopeLevel; // 权限级别：system 只检查系统级权限，project 只检查项目级权限，any 检查两者
  mode?: 'hidden' | 'disabled'; // 当没有权限时的处理方式
  children?: RouteConfig[];
}

export interface RouteGroup {
  title: string;
  scopeLevel?: ScopeLevel; // 路由组的默认权限级别
  routes: RouteConfig[];
}

// 定义所有路由的权限配置
export const routeConfigs: RouteGroup[] = [
  {
    title: 'Admin',
    scopeLevel: 'system', // Admin 路由组只能通过 system-level 权限访问
    routes: [
      {
        path: '/',
        requiredScopes: ['read_dashboard'],
        mode: 'hidden',
      },
      {
        path: '/projects',
        requiredScopes: ['read_projects'],
        mode: 'hidden',
      },
      {
        path: '/users',
        requiredScopes: ['read_users'],
        mode: 'hidden',
      },
      {
        path: '/roles',
        requiredScopes: ['read_roles'],
        mode: 'hidden',
      },
      {
        path: '/channels',
        requiredScopes: ['read_channels'],
        mode: 'hidden',
      },
      {
        path: '/share',
        requiredScopes: ['read_channels'],
        mode: 'hidden',
      },
      {
        path: '/relay-subkeys',
        requiredScopes: ['read_channels', 'read_api_keys', 'read_requests'],
        mode: 'hidden',
        children: [
          { path: '/relay-subkeys/products', requiredScopes: ['read_channels'], mode: 'hidden' },
          { path: '/relay-subkeys/products/create', requiredScopes: ['write_channels'], mode: 'hidden' },
          { path: '/relay-subkeys/products/$productId', requiredScopes: ['read_channels'], mode: 'hidden' },
          { path: '/relay-subkeys/keys', requiredScopes: ['read_api_keys'], mode: 'hidden' },
          { path: '/relay-subkeys/keys/create', requiredScopes: ['write_api_keys'], mode: 'hidden' },
          { path: '/relay-subkeys/keys/$keyId', requiredScopes: ['read_api_keys'], mode: 'hidden' },
          { path: '/relay-subkeys/keys/$keyId/billing', requiredScopes: ['read_api_keys'], mode: 'hidden' },
          { path: '/relay-subkeys/requests', requiredScopes: ['read_requests'], mode: 'hidden' },
          { path: '/relay-subkeys/channel-pool-health', requiredScopes: ['read_channels'], mode: 'hidden' },
        ],
      },
      {
        path: '/models',
        requiredScopes: ['read_channels'],
        mode: 'hidden',
      },
      {
        path: '/prompt-protection-rules',
        requiredScopes: ['read_channels'],
        mode: 'hidden',
      },
      {
        path: '/data-storages',
        requiredScopes: ['read_data_storages'],
        mode: 'hidden',
      },
      {
        path: '/system',
        requiredScopes: ['read_system'],
        mode: 'hidden',
      },
      {
        path: '/permission-demo',
        // 权限演示页面所有用户都可以访问
      },
    ],
  },
  {
    title: 'Project',
    scopeLevel: 'any', // Project 路由组可以通过 system-level 或 project-level 权限访问
    routes: [
      {
        path: '/project/api-keys',
        requiredScopes: ['read_api_keys'],
        mode: 'hidden',
      },
      {
        path: '/use',
        requiredScopes: ['read_api_keys'],
        mode: 'hidden',
      },
      {
        path: '/project/relay-subkeys',
        requiredScopes: ['read_api_keys', 'read_requests'],
        mode: 'hidden',
        children: [
          { path: '/project/relay-subkeys/products', requiredScopes: ['read_api_keys', 'read_requests'], mode: 'hidden' },
          { path: '/project/relay-subkeys/keys', requiredScopes: ['read_api_keys', 'read_requests'], mode: 'hidden' },
          { path: '/project/relay-subkeys/keys/$keyId', requiredScopes: ['read_api_keys', 'read_requests'], mode: 'hidden' },
          { path: '/project/relay-subkeys/usage', requiredScopes: ['read_api_keys', 'read_requests'], mode: 'hidden' },
          { path: '/project/relay-subkeys/get-started', requiredScopes: ['read_api_keys', 'read_requests'], mode: 'hidden' },
          { path: '/project/relay-subkeys/verify', requiredScopes: ['read_api_keys', 'read_requests'], mode: 'hidden' },
        ],
      },
      {
        path: '/project/prompts',
        requiredScopes: ['read_prompts'],
        mode: 'hidden',
      },
      {
        path: '/project/requests',
        requiredScopes: ['read_requests'],
        mode: 'hidden',
      },
      {
        path: '/project/usage-logs',
        requiredScopes: ['read_requests'],
        mode: 'hidden',
      },
      {
        path: '/project/traces',
        requiredScopes: ['read_requests'],
        mode: 'hidden',
      },
      {
        path: '/project/threads',
        requiredScopes: ['read_requests'],
        mode: 'hidden',
      },
      {
        path: '/project/users',
        requiredScopes: ['read_users'],
        mode: 'hidden',
      },
      {
        path: '/project/roles',
        requiredScopes: ['read_roles'],
        mode: 'hidden',
      },
      {
        path: '/project/playground',
        // Playground is accessible to all users
      },
    ],
  },
  {
    title: 'Settings',
    routes: [
      {
        path: '/settings',
        // Profile 设置所有用户都可以访问
      },
      {
        path: '/settings/profile',
        // Profile 设置所有用户都可以访问
      },
      {
        path: '/settings/appearance',
        // Appearance 设置所有用户都可以访问
      },
      {
        path: '/settings/notifications',
        // Notifications 设置所有用户都可以访问
      },
    ],
  },
];

// 获取路由配置的辅助函数
export function getRouteConfig(path: string): RouteConfig | undefined {
  for (const group of routeConfigs) {
    for (const route of group.routes) {
      if (route.path === path) {
        return route;
      }
      if (route.children) {
        const childConfig = route.children.find((child) => child.path === path);
        if (childConfig) return childConfig;
      }
    }
  }
  return undefined;
}

// 检查用户是否有访问路由的权限
export function hasRouteAccess(userScopes: string[], routeConfig: RouteConfig): boolean {
  if (!routeConfig.requiredScopes || routeConfig.requiredScopes.length === 0) {
    return true;
  }

  // 如果用户有通配符权限，则拥有所有权限
  if (userScopes.includes('*')) {
    return true;
  }

  // 路由配置和后端中间件保持一致：列出的 scope 需要全部满足。
  return routeConfig.requiredScopes.every((scope) => userScopes.includes(scope));
}

// 检查用户是否有访问路由组的权限
export function hasGroupAccess(userScopes: string[], group: RouteGroup): boolean {
  return group.routes.some((route) => hasRouteAccess(userScopes, route));
}
