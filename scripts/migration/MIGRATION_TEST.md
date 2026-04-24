# AxonHub 迁移测试脚本

自动化测试数据库版本升级迁移的脚本。

## 概述

`migration-test.sh` 脚本用于验证 AxonHub 的数据库迁移功能。它会自动下载指定发布标签的二进制文件，初始化数据库，然后使用当前分支的代码执行迁移，并可选地运行 E2E 测试来验证数据的完整性。

## 快速开始

```bash
# SQLite（默认）
./scripts/migration/migration-test.sh v0.1.0
```

## 前置条件

- **Go 环境**: 脚本需要编译当前分支代码，确保已安装 Go。
- **SQLite (默认)**: 无需额外依赖。
- **工具**: 需要 `unzip` 用于解压下载的二进制文件。

## 功能特性

1. **自动下载和缓存二进制文件** - 从 GitHub Releases 下载指定 tag 的可执行文件，并缓存到本地。
2. **测试版本升级** - 支持从任意 tag 版本迁移到当前分支最新代码。
3. **SQLite 支持** - 支持默认 SQLite 数据库的迁移测试。
4. **生成迁移计划** - 自动生成包含初始化和迁移步骤的 JSON 计划。
5. **E2E 测试验证** - 迁移完成后自动运行 E2E 测试验证数据完整性。
6. **配置一致性** - 使用与 `e2e-test.sh` 相同的配置，确保测试环境一致。

## 命令行参数

| 参数 | 说明 |
|------|------|
| `from-tag` | **(必需)** 要测试迁移的起始 Git tag（例如：v0.1.0） |
| `--skip-download` | 如果缓存中已存在二进制文件，跳过下载直接使用 |
| `--skip-e2e` | 迁移后跳过运行 E2E 测试 |
| `--keep-artifacts` | 测试完成后保留工作目录（日志、数据库文件等） |
| `-h, --help` | 显示帮助信息 |

## 常用示例

- **跳过 E2E 测试仅验证迁移过程**:
  ```bash
  ./scripts/migration-test.sh v0.1.0 --skip-e2e
  ```
- **使用缓存的二进制文件并保留测试产物**:
  ```bash
  ./scripts/migration-test.sh v0.1.0 --skip-download --keep-artifacts
  ```

## 数据库配置与连接

### SQLite
- **数据库文件**: `scripts/migration/migration-test/work/migration-test.db`
- **查看命令**: `sqlite3 scripts/migration/migration-test/work/migration-test.db`

## 批量测试

`migration-test-all.sh` 脚本可以测试多个版本在 SQLite 上的迁移：

```bash
# 测试指定版本在 SQLite 上的迁移
./scripts/migration/migration-test-all.sh --tags v0.1.0,v0.2.0
```

## 工作流程

1. **检测系统架构** - 自动检测操作系统和 CPU 架构（linux/darwin, amd64/arm64）。
2. **设置数据库环境** - 设置 SQLite 文件。
3. **下载/缓存旧版本二进制** - 从 GitHub 下载指定版本的可执行文件。
4. **构建当前版本** - 编译当前分支的最新代码。
5. **生成迁移计划** - 创建包含版本信息的 `migration-plan.json`。
6. **初始化数据库** - 使用旧版本二进制初始化数据库结构。
7. **执行迁移** - 使用当前分支版本运行数据库迁移。
8. **运行 E2E 测试** - 验证迁移后的数据库功能是否正常（可选）。
9. **清理** - 清理临时文件（可选保留）。

## 目录结构

```
scripts/
├── migration-test.sh           # 主脚本
├── migration-test/             # 测试工作根目录
│   ├── cache/                  # 二进制文件缓存
│   │   └── v0.1.0/
│   │       └── axonhub         # 缓存的 v0.1.0 二进制
│   └── work/                   # 工作目录（测试后默认清理）
│       ├── axonhub-current     # 当前分支编译的二进制
│       ├── migration-test.db   # 测试数据库（SQLite）
│       ├── migration-test.log  # 测试详细日志
│       └── migration-plan.json # 迁移步骤计划
```

## 故障排查

- **端口占用**: 确保 8099 (Server) 端口未被占用。
- **下载失败/限流**: 设置 `GITHUB_TOKEN` 环境变量以避免 GitHub API 速率限制。
- **查看详细日志**: 实时查看日志：`tail -f scripts/migration-test/work/migration-test.log`。

## CI/CD 集成

### GitHub Actions 示例

```yaml
jobs:
  migration-test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        from-version: [v0.1.0, v0.2.0]
    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run migration test
        run: |
          ./scripts/migration-test.sh ${{ matrix.from-version }}
```

## 与其他脚本的关系

- `e2e-test.sh` - 运行完整的 E2E 测试套件。
- `e2e-backend.sh` - 管理 E2E 测试后端服务器。
- `migration-test.sh` - 测试单个版本的数据库迁移。
- `migration-test-all.sh` - 批量测试多个版本的迁移。
