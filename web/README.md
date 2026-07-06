# 前端开发指南

> CodeAtlas Web 前端开发文档

## 技术栈

- **框架**: Svelte 4.0
- **构建工具**: Rsbuild
- **包管理器**: pnpm（必须使用 pnpm，不要用 npm 或 yarn）
- **开发服务器**: 端口 3000

## 快速开始

### 安装依赖

```bash
cd web
pnpm install
```

### 开发模式

启动开发服务器（支持热重载）：

```bash
pnpm dev
```

访问 http://localhost:3000

### 生产构建

```bash
pnpm build
```

构建文件将输出到 `dist` 目录。

### 预览生产构建

```bash
pnpm preview
```

## 项目结构

```
web/
├── src/                    # 源代码
│   ├── components/        # Svelte 组件
│   ├── routes/            # 路由页面
│   ├── lib/               # 工具函数和共享代码
│   ├── stores/            # 状态管理
│   └── App.svelte         # 根组件
├── public/                # 静态资源
├── dist/                  # 构建输出（生成）
├── node_modules/          # 依赖（生成）
├── package.json           # 项目配置
├── rsbuild.config.js      # Rsbuild 配置
└── README.md              # 前端文档
```

## 开发规范

### 代码风格

- 使用现代 CSS 实践
- 实现响应式设计
- 遵循 Svelte 最佳实践
- 避免使用 `any` 类型

### 组件开发

```svelte
<script>
  // 组件逻辑
  export let title = 'Default Title';
  
  let count = 0;
  
  function increment() {
    count += 1;
  }
</script>

<div class="component">
  <h1>{title}</h1>
  <button on:click={increment}>
    Count: {count}
  </button>
</div>

<style>
  .component {
    padding: 1rem;
  }
  
  button {
    padding: 0.5rem 1rem;
    cursor: pointer;
  }
</style>
```

### 状态管理

使用 Svelte stores 进行状态管理：

```javascript
// stores/user.js
import { writable } from 'svelte/store';

export const user = writable(null);

export function setUser(userData) {
  user.set(userData);
}

export function clearUser() {
  user.set(null);
}
```

在组件中使用：

```svelte
<script>
  import { user } from './stores/user';
</script>

{#if $user}
  <p>Welcome, {$user.name}!</p>
{:else}
  <p>Please log in</p>
{/if}
```

## API 集成

### 调用后端 API

```javascript
// lib/api.js
const API_BASE = 'http://localhost:8080/api/v1';

export async function searchCode(query, filters) {
  const response = await fetch(`${API_BASE}/search`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ query, filters }),
  });
  
  if (!response.ok) {
    throw new Error(`API error: ${response.statusText}`);
  }
  
  return response.json();
}
```

在组件中使用：

```svelte
<script>
  import { searchCode } from '../lib/api';
  
  let query = '';
  let results = [];
  let loading = false;
  
  async function handleSearch() {
    loading = true;
    try {
      results = await searchCode(query, {});
    } catch (error) {
      console.error('Search failed:', error);
    } finally {
      loading = false;
    }
  }
</script>

<input bind:value={query} placeholder="Search code..." />
<button on:click={handleSearch} disabled={loading}>
  {loading ? 'Searching...' : 'Search'}
</button>

{#each results as result}
  <div class="result">
    <h3>{result.name}</h3>
    <p>{result.description}</p>
  </div>
{/each}
```

## 构建配置

### Rsbuild 配置

`rsbuild.config.js` 示例：

```javascript
import { defineConfig } from '@rsbuild/core';
import { pluginSvelte } from '@rsbuild/plugin-svelte';

export default defineConfig({
  plugins: [pluginSvelte()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  output: {
    distPath: {
      root: 'dist',
    },
  },
});
```

## 测试

### 组件测试

```javascript
import { render } from '@testing-library/svelte';
import Component from './Component.svelte';

test('renders component', () => {
  const { getByText } = render(Component, {
    props: { title: 'Test Title' }
  });
  
  expect(getByText('Test Title')).toBeInTheDocument();
});
```

### 运行测试

```bash
pnpm test
```

## 性能优化

### 代码分割

Rsbuild 自动处理代码分割，但可以手动优化：

```javascript
// 动态导入
const HeavyComponent = () => import('./HeavyComponent.svelte');
```

### 懒加载

```svelte
<script>
  import { onMount } from 'svelte';
  
  let Component;
  
  onMount(async () => {
    const module = await import('./LazyComponent.svelte');
    Component = module.default;
  });
</script>

{#if Component}
  <svelte:component this={Component} />
{/if}
```

## 部署

### 静态部署

构建后的 `dist` 目录可以部署到任何静态托管服务：

- Vercel
- Netlify
- GitHub Pages
- AWS S3 + CloudFront

### Docker 部署

```dockerfile
FROM node:18-alpine as builder

WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install
COPY . .
RUN pnpm build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

## 常见问题

### pnpm 安装失败

```bash
# 清理缓存
pnpm store prune

# 重新安装
rm -rf node_modules pnpm-lock.yaml
pnpm install
```

### 开发服务器端口冲突

修改 `rsbuild.config.js` 中的端口：

```javascript
server: {
  port: 3001,
}
```

### 构建失败

```bash
# 清理构建缓存
rm -rf dist node_modules/.cache

# 重新构建
pnpm build
```

## 开发工作流

1. **启动后端服务**
   ```bash
   make run-api
   ```

2. **启动前端开发服务器**
   ```bash
   cd web
   pnpm dev
   ```

3. **开发功能**
   - 修改组件
   - 热重载自动更新
   - 测试功能

4. **构建和测试**
   ```bash
   pnpm build
   pnpm preview
   ```

## 参考资料

- [Svelte 文档](https://svelte.dev/docs)
- [Rsbuild 文档](https://rsbuild.dev/)
- [pnpm 文档](https://pnpm.io/)
- [API 文档](../docs/api.md)
