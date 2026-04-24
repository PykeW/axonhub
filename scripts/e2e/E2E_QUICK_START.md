# E2E Testing Quick Start

## 🚀 Quick Start

### One-Command Test
```bash
cd frontend
pnpm test:e2e
```

**That's it!** The script will automatically:
1. Remove old E2E database
2. Start backend server (port 8099)
3. Start frontend server (port 5173)
4. Run initialization tests
5. Run all tests in parallel
6. Automatically stop backend server after tests

## Test Execution Flow

1. ✅ **Remove old database** - Delete `axonhub-e2e.db`
2. ✅ **Start backend server** - Start on port 8099 with `axonhub-e2e.db`
3. ✅ **Start frontend server** - Start on port 5173
4. ✅ **Initialize system** - Run `setup.spec.ts`, create random owner account
5. ✅ **Parallel tests** - All other tests run in parallel
6. ✅ **Auto cleanup** - Stop backend server after tests

## Database Support

The E2E test suite defaults to SQLite. MySQL and PostgreSQL require an externally managed database and a DSN.

**Supported database paths:**
- **SQLite** (default) - Fast, file-based database for development
- **MySQL/PostgreSQL** - External database only; provide `AXONHUB_E2E_DB_DSN`

**Using different databases:**

```bash
# SQLite (default)
./scripts/e2e/e2e-test.sh

# MySQL with an external DSN
AXONHUB_E2E_DB_DSN="user:pass@tcp(host:3306)/axonhub_e2e?charset=utf8mb4&parseTime=True&loc=Local" \
  ./scripts/e2e/e2e-test.sh -d mysql

# PostgreSQL with an external DSN
AXONHUB_E2E_DB_DSN="host=localhost port=5432 user=axonhub password=secret dbname=axonhub_e2e sslmode=disable" \
  ./scripts/e2e/e2e-test.sh --dbtype postgres
```

**Database requirements:**
- **SQLite**: No additional requirements
- **MySQL/PostgreSQL**: Provide an existing database DSN; the E2E scripts do not create database services

**Database configuration:**
- SQLite uses `scripts/e2e/axonhub-e2e.db`
- External databases use `AXONHUB_E2E_DB_DSN`

## Output Example

```
🚀 Starting E2E Test Suite...

📦 Starting E2E backend server...
Removing old E2E database: axonhub-e2e.db
Building backend...
Starting backend on port 8099 with database axonhub-e2e.db...
E2E backend server started (PID: 12345)
Waiting for server to be ready...
E2E backend server is ready!

✅ Backend server ready

🧪 Running Playwright tests...

Running 15 tests using 4 workers

  ✓  [setup] › setup.spec.ts:12:3 › System Setup › initialize system with owner account (5.2s)
  ✓  [chromium] › api-keys.spec.ts:5:3 › Admin API Keys Management › can create and delete API key (3.1s)
  ✓  [chromium] › channels.spec.ts:5:3 › Admin Channels Management › can create channel (2.8s)
  ✓  [chromium] › users.spec.ts:5:3 › Admin Users Management › can create user (2.5s)
  ...

  15 passed (45.3s)

✅ All tests passed!

🧹 Cleaning up...
Stopping E2E backend server...
```

## Common Scenarios

### Debug Failed Tests

```bash
# 1. Run tests (browser visible)
pnpm test:e2e:headed

# 2. Or use debug mode
pnpm test:e2e:debug

# 3. View test report
pnpm test:e2e:report
```

### Run Specific Tests

```bash
# Run specific file
pnpm test:e2e -- tests/users.spec.ts

# Run matching tests
pnpm test:e2e -- --grep "create user"
```

### UI Mode (Recommended for Development)

```bash
pnpm test:e2e:ui
```

This opens an interactive interface where you can:
- Select tests to run
- View test steps
- Time-travel debugging
- View network requests

## Manual Backend Management

**Note:** Usually no need to manually manage backend, `pnpm test:e2e` handles it automatically!

If manual control is needed:
```bash
cd ../..  # Go back to project root

# Start backend
./scripts/e2e-backend.sh start

# Stop backend
./scripts/e2e-backend.sh stop

# Check status
./scripts/e2e-backend.sh status

# Restart backend
./scripts/e2e-backend.sh restart

# Clean all E2E files
./scripts/e2e-backend.sh clean
```

## Debugging After Test Failure

When tests fail, the database is preserved for debugging:

```bash
# View backend logs
cat ../../scripts/e2e-backend.log

# Check SQLite database
sqlite3 ../../scripts/axonhub-e2e.db ".tables"

# View users (example)
sqlite3 ../../scripts/axonhub-e2e.db "SELECT * FROM users;"
```

## Important Files

- `../../scripts/axonhub-e2e.db` - E2E test database (preserved after tests for debugging)
- `../../scripts/e2e-backend.log` - Backend service logs
- `../../scripts/axonhub-e2e` - E2E backend executable
- `../../scripts/.e2e-backend.pid` - Backend process ID
- `playwright-report/` - Test report directory

## Environment Variables

```bash
# Defaults
AXONHUB_ADMIN_PASSWORD=pwd123456  # Owner password
AXONHUB_API_URL=http://localhost:8099  # Backend API URL
```

## Configuration

**Backend configuration:**
- Port: 8099
- Database: SQLite `axonhub-e2e.db` by default, or external MySQL/PostgreSQL via `AXONHUB_E2E_DB_DSN`
- Logs: `e2e-backend.log`

**Frontend configuration:**
- Port: 5173
- API URL: `http://localhost:8099`

**Test configuration:**
- Setup tests: `setup.spec.ts` (run serially)
- Other tests: run in parallel
- Retry on failure: 2 times in CI, 0 times locally

## Troubleshooting

### Backend Start Failed
```bash
# View backend logs
cat ../../scripts/e2e-backend.log

# Check port usage
lsof -i :8099

# Manually stop and restart
../../scripts/e2e-backend.sh stop
../../scripts/e2e-backend.sh start
```

### Test Failed
```bash
# View test report
pnpm test:e2e:report

# Run in debug mode
pnpm test:e2e:debug

# Check SQLite database
sqlite3 ../../scripts/axonhub-e2e.db ".tables"
sqlite3 ../../scripts/axonhub-e2e.db "SELECT * FROM users;"
```

### Test Stuck

1. Check if backend is running: `../../scripts/e2e-backend.sh status`
2. View backend logs: `cat ../../scripts/e2e-backend.log`
3. Restart backend: `../../scripts/e2e-backend.sh restart`

### Port 8099 Already in Use

```bash
# View process using the port
lsof -i :8099

# Stop E2E backend
cd ../..
./scripts/e2e-backend.sh stop
```

### Keep Database for Manual Testing

Database is preserved by default! You can:

```bash
# 1. Run tests
pnpm test:e2e

# 2. Manually start backend (using same database)
cd ../..
./scripts/e2e-backend.sh start

# 3. Now you can access http://localhost:8099 in browser
# Login with accounts created in tests

# 4. Stop when done
./scripts/e2e-backend.sh stop
```

## Cleanup

```bash
# Completely clean E2E environment (including database, logs, executables)
../../scripts/e2e-backend.sh clean

# Delete test reports
rm -rf playwright-report test-results
```

## Best Practices

1. ✅ Use `pw-test-` prefix to identify test data
2. ✅ Use timestamps or random strings to ensure uniqueness
3. ✅ Each test should be independent, not dependent on other tests
4. ✅ Use `waitForGraphQLOperation()` to wait for async operations
5. ✅ Use flexible selectors (support both English and Chinese)

## Performance Tips

- **Parallel execution**: Tests run in parallel automatically, utilizing multi-core CPU
- **Reuse server**: Reuse running frontend server during development
- **Fast feedback**: Other tests start immediately after setup tests complete

## CI/CD Integration

In CI environment, tests will:
- Run serially (more stable)
- Retry 2 times on failure
- Not reuse server
- Generate HTML reports

### GitHub Actions Example
```yaml
# GitHub Actions example
- name: Run E2E tests
  run: |
    cd frontend
    pnpm test:e2e
```

---

# E2E 测试快速开始

## 🚀 快速开始

### 一键运行所有测试
```bash
cd frontend
pnpm test:e2e
```

**就这么简单！** 脚本会自动：
1. 删除旧的 E2E 数据库
2. 启动后端服务（端口 8099）
3. 启动前端服务（端口 5173）
4. 运行初始化测试
5. 并行运行所有测试
6. 测试结束后自动停止后端服务

## 测试执行流程

1. ✅ **删除旧数据库** - 删除 `axonhub-e2e.db`
2. ✅ **启动后端服务** - 在端口 8099 上启动，使用 `axonhub-e2e.db`
3. ✅ **启动前端服务** - 在端口 5173 上启动
4. ✅ **初始化系统** - 运行 `setup.spec.ts`，创建随机 owner 账户
5. ✅ **并行测试** - 所有其他测试并行运行
6. ✅ **自动清理** - 测试结束后停止后端服务

## 数据库支持

E2E 测试套件默认使用 SQLite。MySQL 和 PostgreSQL 需要外部维护的数据库并提供 DSN。

**支持的数据库路径：**
- **SQLite** (默认) - 快速、基于文件的数据库，适合开发
- **MySQL/PostgreSQL** - 仅支持外部数据库；提供 `AXONHUB_E2E_DB_DSN`

**使用不同的数据库：**

```bash
# SQLite (默认)
./scripts/e2e-test.sh

# MySQL（外部 DSN）
AXONHUB_E2E_DB_DSN="user:pass@tcp(host:3306)/axonhub_e2e?charset=utf8mb4&parseTime=True&loc=Local" \
  ./scripts/e2e-test.sh -d mysql

# PostgreSQL（外部 DSN）
AXONHUB_E2E_DB_DSN="host=localhost port=5432 user=axonhub password=secret dbname=axonhub_e2e sslmode=disable" \
  ./scripts/e2e-test.sh --dbtype postgres
```

**数据库要求：**
- **SQLite**: 无额外要求
- **MySQL/PostgreSQL**: 提供已有数据库 DSN；E2E 脚本不会创建数据库服务

**数据库配置：**
- SQLite 使用 `scripts/axonhub-e2e.db`
- 外部数据库使用 `AXONHUB_E2E_DB_DSN`

## 输出示例

```
🚀 Starting E2E Test Suite...

📦 Starting E2E backend server...
Removing old E2E database: axonhub-e2e.db
Building backend...
Starting backend on port 8099 with database axonhub-e2e.db...
E2E backend server started (PID: 12345)
Waiting for server to be ready...
E2E backend server is ready!

✅ Backend server ready

🧪 Running Playwright tests...

Running 15 tests using 4 workers

  ✓  [setup] › setup.spec.ts:12:3 › System Setup › initialize system with owner account (5.2s)
  ✓  [chromium] › api-keys.spec.ts:5:3 › Admin API Keys Management › can create and delete API key (3.1s)
  ✓  [chromium] › channels.spec.ts:5:3 › Admin Channels Management › can create channel (2.8s)
  ✓  [chromium] › users.spec.ts:5:3 › Admin Users Management › can create user (2.5s)
  ...

  15 passed (45.3s)

✅ All tests passed!

🧹 Cleaning up...
Stopping E2E backend server...
```

## 其他常用场景

### 调试失败的测试

```bash
# 1. 运行测试（会显示浏览器）
pnpm test:e2e:headed

# 2. 或者使用调试模式
pnpm test:e2e:debug

# 3. 查看测试报告
pnpm test:e2e:report
```

### 只运行特定测试

```bash
# 运行特定文件
pnpm test:e2e -- tests/users.spec.ts

# 运行匹配的测试
pnpm test:e2e -- --grep "create user"
```

### 使用 UI 模式（推荐用于开发）

```bash
pnpm test:e2e:ui
```

这会打开一个交互式界面，可以：
- 选择要运行的测试
- 查看测试步骤
- 时间旅行调试
- 查看网络请求

## 手动管理后端

**注意：** 通常不需要手动管理后端，`pnpm test:e2e` 会自动处理！

如果需要手动控制：
```bash
cd ../..  # 回到项目根目录

# 启动后端
./scripts/e2e-backend.sh start

# 停止后端
./scripts/e2e-backend.sh stop

# 查看状态
./scripts/e2e-backend.sh status

# 重启后端
./scripts/e2e-backend.sh restart

# 清理所有 E2E 文件
./scripts/e2e-backend.sh clean
```

## 测试失败后的调试

测试失败时，数据库会保留，方便调试：

```bash
# 查看后端日志
cat ../../scripts/e2e-backend.log

# 检查 SQLite 数据库
sqlite3 ../../scripts/axonhub-e2e.db ".tables"

# 查看用户（示例）
sqlite3 ../../scripts/axonhub-e2e.db "SELECT * FROM users;"
```

## 重要文件

- `../../scripts/axonhub-e2e.db` - E2E 测试数据库（测试后保留，用于复现问题）
- `../../scripts/e2e-backend.log` - 后端服务日志
- `../../scripts/axonhub-e2e` - E2E 后端可执行文件
- `../../scripts/.e2e-backend.pid` - 后端进程 ID
- `playwright-report/` - 测试报告目录

## 环境变量

```bash
# 默认值
AXONHUB_ADMIN_PASSWORD=pwd123456  # Owner 密码
AXONHUB_API_URL=http://localhost:8099  # 后端 API 地址
```

## 配置说明

**后端配置:**
- 端口: 8099
- 数据库: 默认 SQLite `axonhub-e2e.db`，或通过 `AXONHUB_E2E_DB_DSN` 使用外部 MySQL/PostgreSQL
- 日志: `e2e-backend.log`

**前端配置:**
- 端口: 5173
- API 地址: `http://localhost:8099`

**测试配置:**
- 初始化测试: `setup.spec.ts` (串行运行)
- 其他测试: 并行运行
- 失败重试: CI 环境 2 次，本地 0 次

## 故障排查

### 后端启动失败
```bash
# 查看后端日志
cat ../../scripts/e2e-backend.log

# 检查端口占用
lsof -i :8099

# 手动停止并重启
../../scripts/e2e-backend.sh stop
../../scripts/e2e-backend.sh start
```

### 测试失败
```bash
# 查看测试报告
pnpm test:e2e:report

# 调试模式运行
pnpm test:e2e:debug

# 检查 SQLite 数据库
sqlite3 ../../scripts/axonhub-e2e.db ".tables"
sqlite3 ../../scripts/axonhub-e2e.db "SELECT * FROM users;"
```

### 测试卡住不动

1. 检查后端是否正常运行：`../../scripts/e2e-backend.sh status`
2. 查看后端日志：`cat ../../scripts/e2e-backend.log`
3. 重启后端：`../../scripts/e2e-backend.sh restart`

### 端口 8099 被占用

```bash
# 查看占用端口的进程
lsof -i :8099

# 停止 E2E 后端
cd ../..
./scripts/e2e-backend.sh stop
```

### 想保留数据库进行手动测试

数据库默认会保留！你可以：

```bash
# 1. 运行测试
pnpm test:e2e

# 2. 手动启动后端（使用同一个数据库）
cd ../..
./scripts/e2e-backend.sh start

# 3. 现在可以在浏览器中访问 http://localhost:8099
# 使用测试中创建的账户登录

# 4. 完成后停止
./scripts/e2e-backend.sh stop
```

## 清理环境

```bash
# 完全清理 E2E 环境（包括数据库、日志、可执行文件）
../../scripts/e2e-backend.sh clean

# 删除测试报告
rm -rf playwright-report test-results
```

## 测试最佳实践

1. ✅ 使用 `pw-test-` 前缀标识测试数据
2. ✅ 使用时间戳或随机字符串保证唯一性
3. ✅ 每个测试应该独立，不依赖其他测试
4. ✅ 使用 `waitForGraphQLOperation()` 等待异步操作
5. ✅ 使用灵活的选择器（支持中英文）

## 性能提示

- **并行执行**：测试会自动并行运行，充分利用多核 CPU
- **复用服务器**：开发时会复用已运行的前端服务器
- **快速反馈**：setup 测试完成后，其他测试立即开始

## CI/CD 集成

在 CI 环境中，测试会：
- 串行运行（更稳定）
- 失败时重试 2 次
- 不复用服务器
- 生成 HTML 报告

### GitHub Actions 示例
```yaml
# GitHub Actions 示例
- name: Run E2E tests
  run: |
    cd frontend
    pnpm test:e2e
```
