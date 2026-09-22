# FiberScope OTDR 光纤故障定位复核台

FiberScope 是面向光纤线路维护分析人员的离线 OTDR 轨迹分析 Web 应用。它保留原始采样点和全部处理参数，识别事件、与基线轨迹比较，并由复核人员形成定位结论。系统不连接真实 OTDR 或网管，不派工、不切换链路、不控制设备；算法结果只作分析参考。

## Docker 快速启动

```bash
cp .env.example .env
docker compose up -d --build
docker compose ps
```

所有容器为 `healthy` 后访问 [http://localhost:18522](http://localhost:18522)。后端健康检查为 `http://localhost:19522/healthz`，前端代理健康检查为 `http://localhost:18522/api/healthz`。

测试账号的统一密码为 `DemoPass123!`：

| 账号 | 角色 | 主要权限 |
| --- | --- | --- |
| `analyst` | analyst | 线路维护、轨迹导入和检测、案例分析 |
| `reviewer` | reviewer | 基线设置、事件复核、案例确认/关闭、审计检索 |
| `admin` | admin | 上述全部权限 |

停止并移除本项目数据卷：

```bash
docker compose down -v --remove-orphans
```

## 功能

- 线路档案：校验线路长度和折射率，查看历史轨迹，由 reviewer/admin 设置基线。
  - 更换复核基线时若仍有未关闭案例（`draft/analyzing/pending_review/confirmed`）引用当前基线，整次请求以 `409 BASELINE_REPLACEMENT_BLOCKED` 拒绝并在错误体 `blocking_cases` 中列出阻塞案例；历史（已关闭）案例继续保留各自创建时的原基线，不随基线更换改写。
  - 目标轨迹不属于该线路（`trace_not_on_route`）、与当前基线相同（`same_baseline`，重复操作只成功一次）、或请求进入事务时基线已被并发改写（`baseline_concurrent_change`）均拒绝且不改写线路。
  - 检查、守卫式更新、审计 `route.baseline_changed` 与拒绝记录落在同一个数据库事务；拒绝原因持久化，线路详情抽屉提供“可切换轨迹 / 待处理案例 / 基线拒绝原因”三个视图，刷新后仍可回读。
- 轨迹分析：导入离线采样，记录去噪窗口、检测阈值和合并窗口，缩放真实 API 曲线。
- 事件复核：按线路、类型和复核状态筛选，保留算法原值并单独保存人工修订。
- 定位案例：执行基线差异比较，按 `draft -> analyzing -> pending_review -> confirmed -> closed` 流转。
- 不可变审计：记录轨迹导入、基线变更、算法参数、事件修订、案例确认和关闭，携带 request ID 与前后值摘要。

## 技术栈与目录

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.22、Gin、GORM、validator/v10、JWT、slog |
| 正式数据库 | PostgreSQL 16 |
| 自包含验证 | GORM SQLite（仅 runtime smoke/测试） |
| 前端 | Vue 3、TypeScript、Vite、Element Plus、Pinia、ECharts |
| 部署 | Docker Compose、Nginx |

```text
backend/cmd/server                 组装、启动与优雅停机
backend/internal/algorithm         去噪、距离、峰值检测、基线比对
backend/internal/{model,dto}       数据库实体与 API 边界
backend/internal/repository        GORM 数据访问与事务
backend/internal/service           业务规则与状态机
backend/internal/{handler,router}  HTTP 处理与权限路由
frontend/src/{api,stores,types}     前端数据层
frontend/src/components/common     轨迹、事件标识、复核对话框
frontend/src/pages                 五个业务页与登录页
```

## 环境变量与端口

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `COMPOSE_PROJECT_NAME` | `fiber-otdr-fault-localization` | Compose 资源隔离名 |
| `FRONTEND_PORT` | `18522` | Nginx 宿主端口 |
| `BACKEND_PORT` | `19522` | Gin 宿主端口 |
| `DB_PORT` | `57522` | PostgreSQL 宿主端口 |
| `POSTGRES_DB/USER/PASSWORD` | 见 `.env.example` | 正式 Compose 数据库 |
| `JWT_SECRET` | 必须不少于 32 字符 | JWT HS256 密钥 |
| `CORS_ORIGINS` | 前端本地地址 | 逗号分隔白名单 |
| `MAX_TRACE_POINTS` | `20000` | 单条轨迹采样点上限 |
| `LOG_LEVEL` | `info` | 结构化日志级别 |

## API

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/login` | 登录 |
| `GET/POST` | `/api/v1/routes` | 线路列表/新建 |
| `GET/PATCH` | `/api/v1/routes/:id` | 线路详情/编辑 |
| `POST` | `/api/v1/routes/:id/baseline` | 设置基线 |
| `GET` | `/api/v1/traces` | 轨迹列表 |
| `POST` | `/api/v1/traces/import` | 导入采样点 |
| `GET` | `/api/v1/traces/:id` | 轨迹、处理点和事件 |
| `POST` | `/api/v1/traces/:id/detect` | 执行事件检测 |
| `GET` | `/api/v1/events` | 事件筛选 |
| `PATCH` | `/api/v1/events/:id/review` | 人工复核 |
| `GET/POST` | `/api/v1/cases` | 案例列表/新建 |
| `GET` | `/api/v1/cases/:id` | 案例与差异 |
| `POST` | `/api/v1/cases/:id/analyze` | 基线比对 |
| `POST` | `/api/v1/cases/:id/confirm` | reviewer 确认 |
| `POST` | `/api/v1/cases/:id/close` | 关闭案例 |
| `GET` | `/api/v1/audit` | 审计检索 |

## 共享枚举出现位置

`EventType = connector | splice | bend | break | end | unknown`：

- 数据库与 model：`backend/internal/model/event_marker.go`
- constants：`backend/internal/constants/event.go`
- dto/service/handler/router：`backend/internal/dto/event.go`、`service/event_service.go`、`handler/event_handler.go`、`router/event_router.go`
- 前端 type/api/store/component/page：`frontend/src/types/event.ts`、`api/domain.ts`、`stores/events.ts`、`components/common/EventTypeBadge.vue`、`pages/EventsPage.vue`、`pages/TracesPage.vue`

`CaseStatus = draft | analyzing | pending_review | confirmed | closed`：

- 数据库与 model：`backend/internal/model/localization_case.go`
- constants：`backend/internal/constants/case.go`
- dto/service/handler/router：`backend/internal/dto/case.go`、`service/case_service.go`、`handler/case_handler.go`、`router/case_audit_router.go`
- 前端 type/api/store/component/page：`frontend/src/types/case.ts`、`api/domain.ts`、`stores/cases.ts`、`components/common/ReviewDialog.vue`、`pages/CasesPage.vue`

## 算法、约束与安全边界

1. 移动中值去噪：对每个采样使用最多 31 点的奇数窗口，边界处截断窗口。
2. 噪声底：取轨迹尾部 20% 样本的中位数。
3. 事件检测：一阶差分绝对值超过阈值的点为峰值，连续峰按窗口合并为幅度最大的一点。
4. 距离公式：`distance = c * sample_index * sample_interval_ns * 1e-9 / (2 * refractive_index)`，其中 `c = 299792458 m/s`。超过线路长度的候选事件被拒绝。
5. 基线比对：在距离容差内一对一最近匹配，输出新增、消失和损耗增大三类差异与置信度。

状态迁移使用条件更新和 `version` 乐观锁。分析失败回到 `draft` 并保存错误；只有 reviewer/admin 能确认；关闭后不可修改。登录、轨迹导入和分析使用本地内存限流。访问日志不记录 JWT、密码、请求体或完整采样数组。

## 本地开发与验证

```bash
go work sync
go build ./backend/...
go vet ./backend/...
go test ./backend/...
npm --prefix frontend ci
npm --prefix frontend run build


```

`runtime_smoke.json` 使用 `20522` 端口和内存 SQLite，用于脱离 Compose 的真实启动检查，不代替正式 PostgreSQL 部署。

## 排错

- 容器未健康：执行 `docker compose logs postgres backend frontend`，首先检查端口占用和 `JWT_SECRET` 长度。
- 前端 API 返回 502：确认 backend 为 `healthy`，Nginx 通过 Compose 服务名 `backend:8080` 连接。
- 轨迹导入被拒绝：确认至少 16 点、无 NaN/Inf，采样范围覆盖线路至少 5%，且不超过 `MAX_TRACE_POINTS`。
- 案例无法分析：基线和当前轨迹均需先执行事件检测。
- 确认返回 `STATE_CONFLICT`：刷新案例取得最新 `version`，并确认状态为 `pending_review`。

## License

MIT，见 [LICENSE](./LICENSE)。
