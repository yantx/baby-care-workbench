# baby-care-workbench 爸妈育儿工作台

新手爸妈协作育儿小程序：喂养 / 睡眠 / 尿布 / 体温 / 用药记录，家庭多角色实时同步。

- **后端**：Go (Gin) + GORM + PostgreSQL + WebSocket（多角色实时同步核心）
- **小程序端**：微信小程序（测试号可直接开发调试）
- **架构文档**：见 `docs/` 与产品设计方案

## 目录结构

```
baby-care-workbench/
├── backend/                 # Go 后端
│   ├── cmd/server/          # 服务入口
│   ├── cmd/migrate/         # 数据库建库+迁移工具
│   ├── config/              # 配置文件（config.example.yaml 模板）
│   ├── migrations/          # SQL 迁移脚本
│   └── internal/            # 分层代码：config/model/repository/service/controller/middleware/router/ws
├── miniprogram/             # 微信小程序端
│   ├── app.js|json|wxss     # 全局入口
│   ├── utils/               # request 封装 + WebSocket 管理器
│   └── pages/               # 登录/今日看板/添加记录/记录时间线/家庭成员/我的
└── README.md
```

## 快速开始

### 1. 数据库初始化

后端依赖 PostgreSQL（Redis 可选）。复制配置模板并填写你的连接信息：

```bash
cp backend/config/config.example.yaml backend/config/config.yaml
# 编辑 config.yaml：database.password 等（config.yaml 已加入 .gitignore，不会提交）
```

一键建库 + 建表（自动创建数据库并执行 migrations/ 下的 SQL）：

```bash
cd backend
go run ./cmd/migrate
```

### 2. 启动后端

```bash
cd backend
go run ./cmd/server
# 默认监听 http://127.0.0.1:8080，WebSocket: ws://127.0.0.1:8080/ws
```

> 微信 appid/secret 留空时自动启用 **Mock 登录模式**：登录接口的 `code` 作为设备标识，
> 可用不同 code 模拟妈妈/爸爸/长辈等多个家庭成员，方便本地联调多角色实时同步。

### 3. 小程序端

1. 微信开发者工具 → 导入项目 → 选择 `miniprogram/` 目录，AppID 选「测试号」
2. 「详情 → 本地设置」勾选「不校验合法域名」（本地 HTTP/WS 开发必需）
3. 真机调试时把 `miniprogram/app.js` 里的 `apiBase` 改为电脑的局域网 IP

### 4. 体验多角色协作

1. 设备 A 登录 → 创建家庭（填写宝宝信息）→ 复制邀请码
2. 设备 B（开发者工具可新开一个项目目录模拟）登录 → 输入邀请码加入家庭
3. 任一端添加记录，另一端今日看板 1 秒内实时刷新（WebSocket）

## 核心接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/auth/login | 登录（Mock 模式 code=设备标识） |
| GET  | /api/v1/user/me | 当前用户信息 |
| PUT  | /api/v1/user/me | 更新昵称/头像 |
| POST | /api/v1/families | 创建家庭（可同时建宝宝档案） |
| POST | /api/v1/families/join | 邀请码加入家庭 |
| GET  | /api/v1/families/current | 当前家庭详情（成员/宝宝/邀请码） |
| GET  | /api/v1/records | 记录列表（按日期/类型过滤） |
| POST | /api/v1/records | 添加护理记录 |
| DELETE | /api/v1/records/:id | 删除记录 |
| GET  | /api/v1/stats/today | 今日看板统计 |
| WS   | /ws?token=xxx | WebSocket 实时同步（心跳 ping/pong） |

统一响应格式：`{code, message, data}`，code=0 成功；认证方式 `Authorization: Bearer <token>`。

## 环境变量覆盖（生产部署）

| 变量 | 说明 |
|------|------|
| BABY_SERVER_PORT | 服务端口 |
| BABY_DB_HOST / BABY_DB_PORT / BABY_DB_USER / BABY_DB_PASSWORD / BABY_DB_NAME | 数据库连接 |
| BABY_REDIS_HOST / BABY_REDIS_PORT / BABY_REDIS_PASSWORD | Redis 连接 |
| BABY_JWT_SECRET | JWT 密钥（生产必换） |
| BABY_WECHAT_APPID / BABY_WECHAT_SECRET | 微信正式登录配置 |

## 路线图

- [x] Phase 1 MVP：护理记录 + 家庭多角色 + WebSocket 实时同步
- [ ] Phase 2：成长曲线（WHO/中国双标准）、疫苗计划、PDF 导出
- [ ] Phase 3：AI 睡眠预测、喂养建议、内容社区
