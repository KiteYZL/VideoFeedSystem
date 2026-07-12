# FeedSystemDemo 项目 Postman 接口测试学习指导手册

> 适用对象：第一次接触 Postman、希望通过接口验证 Go 后端功能的初学者。
>
> 本手册根据当前 `D:\FeedSystemDemo\backend` 项目的实际代码编写。当前项目还没有完成 HTTP 服务启动和路由注册，因此文中的接口地址是根据已有 Handler 推导出的“建议接口约定”，不是项目当前已经可以访问的地址。

## 1. 先理解当前项目能做什么

当前代码的主要功能集中在账户模块：

| 模块 | 当前代码情况 | 作用 |
|---|---|---|
| 账户实体 | 已完成 | 保存用户名、密码哈希、访问 Token、刷新 Token 等字段 |
| 注册 | Handler 和 Service 已有 | 接收 `username`、`passwd`，使用 bcrypt 加密密码后写入 MySQL |
| 登录 | Handler 和 Service 已有 | 校验用户名和密码，返回 access token 和 refresh token |
| 刷新 Token | Handler 和 Service 已有 | 根据 refresh token 生成新的 access token |
| 退出登录 | Handler、Repository 已有 | 清空数据库中的 Token，使旧 Token 失效 |
| JWT 鉴权 | 中间件已写 | 检查 `Authorization: Bearer <token>` |
| MySQL 连接 | 连接和自动建表函数已写 | 需要在启动入口中调用 |
| HTTP 服务 | 尚未完成 | `cmd/main.go` 目前只有 `package main` |
| 路由 | 尚未完成 | `internal/http/router.go` 目前只有包声明 |
| 自动化测试 | 尚未添加 | 当前 `go test ./...` 只能说明各包可以编译，没有接口测试 |

### 当前最重要的结论

现在不能直接在 Postman 中访问这个项目，因为：

1. `cmd/main.go` 没有创建 Gin 服务，也没有调用 `Run`。
2. `internal/http/router.go` 没有注册任何 URL。
3. 没有把配置、MySQL、Repository、Service、Handler 组装起来。
4. 还没有明确服务端口和数据库配置的读取方式。

因此，Postman 的“连接失败”并不一定是 Postman 用错了，可能只是后端还没有启动。

## 2. 你需要先掌握的接口概念

一次 HTTP 请求可以先简单理解为：

```text
请求 = 方法 + URL + 请求头 + 请求体
响应 = 状态码 + 响应头 + 响应体
```

### 2.1 方法

本项目主要使用：

| 方法 | 初学者理解 | 本项目示例 |
|---|---|---|
| `GET` | 查询数据 | 当前暂无已注册的查询接口 |
| `POST` | 提交数据、执行动作 | 注册、登录、刷新 Token、退出登录 |
| `PUT/PATCH` | 修改数据 | 当前暂无已注册的修改接口 |
| `DELETE` | 删除数据 | 当前暂无已注册的删除接口 |

### 2.2 URL

本手册使用以下建议基础地址：

```text
http://127.0.0.1:8080
```

如果后端实际监听的是其他端口，例如 `9000`，需要统一修改为：

```text
http://127.0.0.1:9000
```

### 2.3 请求头

请求头用于说明请求格式或身份信息。

发送 JSON 时通常需要：

```text
Content-Type: application/json
```

访问受保护接口时还需要：

```text
Authorization: Bearer <access_token>
```

注意 `Bearer` 和 Token 之间必须有一个空格，不能写成：

```text
Authorization: <access_token>
```

也不能把 refresh token 当成 access token 使用。

### 2.4 请求体

请求体是你提交给服务器的数据。注册接口的 JSON 请求体例如：

```json
{
  "username": "postman_user_001",
  "passwd": "Passw0rd!"
}
```

JSON 中的字段名必须和 Go 结构体标签一致。例如代码使用的是 `passwd`，不是 `password`。

## 3. 开始前的环境准备

### 3.1 Postman

使用 Postman Desktop 离线版本即可。初学阶段只需要会用：

- 新建 Request。
- 选择 HTTP 方法。
- 填写 URL。
- 设置 Headers。
- 选择 Body 的 raw/JSON 格式。
- 点击 Send 查看响应。
- 如果版本提供 Tests，可以添加简单断言；没有 Tests 也不影响手工测试。

不同版本的界面名称可能略有差异。本文不依赖云端同步、Mock Server 或在线监控功能。

### 3.2 Go 后端

在 PowerShell 中进入后端目录：

```powershell
cd D:\FeedSystemDemo\backend
```

当前项目可以执行：

```powershell
go test ./...
```

但这个命令不会启动 HTTP 服务。只有在 `main.go` 和路由补齐之后，才可以尝试：

```powershell
go run ./cmd
```

### 3.3 MySQL

当前项目使用 GORM 的 MySQL 驱动。后端真正启动前，至少需要准备：

- MySQL 服务正在运行。
- 数据库已经创建。
- 数据库地址、端口、用户名、密码正确。
- 启动代码调用了 `db.NewDB` 和 `db.AutoMigrate`。

例如可以准备一个名为 `feed_demo` 的数据库。具体账号和密码以你的本机配置为准，不要照抄示例密码。

### 3.4 固定 JWT_SECRET

启动时应设置固定的 JWT 密钥，例如在 PowerShell 中：

```powershell
$env:JWT_SECRET = "local-development-secret-change-me"
```

当前 `internal/auth/jwt.go` 中的 `jwtSecret()` 在环境变量为空时会生成随机密钥，但它没有把随机密钥缓存起来。这样生成 Token 和解析 Token 时可能使用不同的密钥，导致刚登录拿到的 Token 立刻无法通过鉴权。

开发阶段应设置 `JWT_SECRET`，并计划后续修改代码，使随机密钥只生成一次或直接要求环境变量必须存在。

## 4. 建议的 Postman 工作区结构

建议在 Postman 中创建一个 Collection，名称可以是：

```text
FeedSystemDemo API
```

在 Collection 下建立这些文件夹：

```text
00 - 服务检查
01 - 账户注册
02 - 账户登录
03 - Token 刷新
04 - 退出登录
05 - 异常场景
```

如果你的离线版本支持 Environment，可以创建环境：

| 变量名 | 初始值示例 | 用途 |
|---|---|---|
| `base_url` | `http://127.0.0.1:8080` | 后端基础地址 |
| `test_username` | `postman_user_001` | 测试用户名 |
| `test_passwd` | `Passw0rd!` | 测试密码 |
| `account_id` | 留空 | 保存登录响应中的账户 ID |
| `access_token` | 留空 | 保存访问 Token |
| `refresh_token` | 留空 | 保存刷新 Token |

请求中使用变量的写法：

```text
{{base_url}}/api/accounts/register
```

如果当前 Postman 版本没有环境变量功能，可以先把 `{{base_url}}` 替换成完整地址，把 Token 手工复制到请求头中。

## 5. 先确定后端路由约定

因为项目的路由文件尚未实现，下面采用一套清晰的建议路由：

| 功能 | 方法 | 建议 URL | 是否需要 access token |
|---|---|---|---|
| 注册 | `POST` | `/api/accounts/register` | 否 |
| 登录 | `POST` | `/api/accounts/login` | 否 |
| 刷新 access token | `POST` | `/api/accounts/refresh` | 否，使用 refresh token |
| 退出登录 | `POST` | `/api/accounts/logout` | 是 |

这只是测试手册中的约定。你在路由代码中如果采用了 `/register`、`/login` 等其他路径，Postman 的 URL 必须跟着实际代码修改。

## 6. 第一个请求：注册账户

### 6.1 创建请求

1. 在 Collection 中点击 New Request。
2. 名称填写 `Register - success`。
3. 方法选择 `POST`。
4. URL 填写：

```text
{{base_url}}/api/accounts/register
```

5. 打开 `Body`。
6. 选择 `raw`。
7. 右侧格式选择 `JSON`。
8. 填写：

```json
{
  "username": "postman_user_001",
  "passwd": "Passw0rd!"
}
```

选择 JSON 后，Postman 通常会自动添加 `Content-Type: application/json`。如果没有自动添加，就在 Headers 中手工添加。

### 6.2 预期结果

注册成功时，当前 Handler 设计的是：

- HTTP 状态码：`200`
- 响应体：

```json
{
  "message": "account is successfully created"
}
```

密码不会以明文返回。Service 会先使用 bcrypt 加密后再写入数据库。

### 6.3 注册失败测试

将用户名改成已经注册过的用户名，重新发送：

```json
{
  "username": "postman_user_001",
  "passwd": "Passw0rd!"
}
```

由于数据库中存在唯一索引，预期是 `400`。当前 Handler 会把具体数据库错误统一转换为“注册失败，请稍后重试”。

再测试空密码：

```json
{
  "username": "postman_user_002",
  "passwd": ""
}
```

Service 明确拒绝空密码，预期状态码为 `400`。

再测试错误 JSON：

```json
{
  "username": "postman_user_003",
  "passwd":
}
```

预期是 `400`，响应中包含 JSON 解析错误信息。

## 7. 第二个请求：登录

### 7.1 创建登录请求

请求名称填写 `Login - success`。

```text
POST {{base_url}}/api/accounts/login
```

请求体：

```json
{
  "username": "postman_user_001",
  "passwd": "Passw0rd!"
}
```

### 7.2 预期成功响应

当前代码设计的响应结构是：

```json
{
  "token": "eyJ...",
  "refresh_token": "a-long-random-string",
  "account_id": 1,
  "username": "postman_user_001"
}
```

其中：

- `token` 是 access token，默认有效期约 15 分钟。
- `refresh_token` 用于换取新的 access token。
- `account_id` 是数据库账户 ID。
- `username` 是登录用户名。

登录成功后，先复制并保存：

1. `token` 到环境变量 `access_token`。
2. `refresh_token` 到环境变量 `refresh_token`。
3. `account_id` 到环境变量 `account_id`。

如果 Postman 版本支持 Tests，可以在登录请求的 Tests 中尝试：

```javascript
const body = pm.response.json();

pm.test("状态码是 200", function () {
  pm.response.to.have.status(200);
});

pm.test("返回 access token", function () {
  pm.expect(body.token).to.be.a("string").and.not.empty;
});

pm.test("返回 refresh token", function () {
  pm.expect(body.refresh_token).to.be.a("string").and.not.empty;
});

pm.environment.set("access_token", body.token);
pm.environment.set("refresh_token", body.refresh_token);
pm.environment.set("account_id", body.account_id);
```

如果离线版本没有 Tests 或脚本执行功能，跳过这段，手工复制字段即可。

### 7.3 登录失败测试

错误密码：

```json
{
  "username": "postman_user_001",
  "passwd": "wrong-password"
}
```

预期：`400`，业务错误是用户名或密码错误。

不存在的用户名：

```json
{
  "username": "not-exist-user",
  "passwd": "Passw0rd!"
}
```

预期：`400`。

注意：当前 `Login` Handler 在 JSON 解析失败或登录 Service 出错后，部分分支缺少 `return`。如果你故意发送错误请求却看到多个响应、状态码异常或响应体混杂，这属于后端代码问题，不是 Postman 操作问题。

## 8. 第三个请求：访问需要鉴权的接口

当前项目没有定义账户详情或个人资料路由，因此暂时可以把退出登录接口作为第一个受保护接口。

请求名称填写 `Logout - with token`。

```text
POST {{base_url}}/api/accounts/logout
```

在 Headers 中添加：

| Key | Value |
|---|---|
| `Authorization` | `Bearer {{access_token}}` |

也可以在 Postman 的 Authorization 标签页选择：

- Type：`Bearer Token`
- Token：`{{access_token}}`

两种方式不要重复配置，以免实际发送的 Header 难以判断。

### 8.1 预期结果

如果后端路由使用了 `JWTAuth` 中间件并且代码问题已修复，预期是：

- HTTP 状态码：`200`
- 响应体：

```json
{
  "message": "logout successfully"
}
```

退出登录后，Repository 会把数据库中的 `token` 和 `refresh_token` 清空。此时再次使用原 access token，预期会得到 `401`，错误含义是 Token 已被撤销。

### 8.2 当前代码的已知问题

JWT 中间件写入的上下文键是：

```text
account_id
```

但账户 Handler 的 `getAccountID` 读取的是：

```text
accountID
```

大小写和下划线不一致。因此，即使 Postman 正确发送了 Token，当前 `/logout` 逻辑也可能返回 `400` 和 `accountID no found`。修复前应把它记录为已知缺陷。

## 9. 第四个请求：刷新 access token

刷新接口通常不要求旧 access token，而是要求 refresh token。

请求名称填写 `Refresh - access token`。

```text
POST {{base_url}}/api/accounts/refresh
```

请求头：

```text
Content-Type: application/json
```

请求体：

```json
{
  "refresh_token": "{{refresh_token}}"
}
```

### 9.1 预期结果

成功响应仍然使用登录响应结构：

```json
{
  "token": "new-access-token",
  "refresh_token": "old-refresh-token",
  "account_id": 1,
  "username": "postman_user_001"
}
```

刷新成功后要用新的 `token` 覆盖 `access_token`。当前代码会更新数据库中的 access token，因此旧 access token 会因为不再等于数据库中的 Token 而失效。

如果支持 Tests，可以使用：

```javascript
const body = pm.response.json();

pm.test("状态码是 200", function () {
  pm.response.to.have.status(200);
});

pm.test("返回新的 access token", function () {
  pm.expect(body.token).to.be.a("string").and.not.empty;
});

pm.environment.set("access_token", body.token);
```

### 9.2 无效 refresh token

发送：

```json
{
  "refresh_token": "invalid-refresh-token"
}
```

预期：`400`，表示 refresh token 无效或已经过期。

## 10. 推荐的完整测试顺序

每次验证新代码时，按照下面顺序执行：

1. 确认 MySQL 正在运行。
2. 启动 Go 服务，确认控制台没有启动错误。
3. 注册一个全新的用户名。
4. 用相同用户名和密码登录。
5. 保存登录返回的 access token 和 refresh token。
6. 带 access token 调用受保护接口。
7. 用 refresh token 刷新 access token，并替换旧 access token。
8. 用旧 access token 调用受保护接口，确认它已经失效。
9. 用新 access token 调用退出登录。
10. 再次使用新 access token，确认返回 `401`。

这个顺序体现了项目的核心状态变化：

```text
未注册
  -> 注册成功
  -> 登录成功，数据库保存 Token
  -> 刷新 Token，旧 access token 失效
  -> 退出登录，当前 Token 失效
```

## 11. Postman 中如何判断问题在哪一层

| 现象 | 常见原因 | 排查顺序 |
|---|---|---|
| `Could not send request` 或连接被拒绝 | Go 服务未启动、端口错误 | 看后端控制台、确认端口、确认 `base_url` |
| `404 Not Found` | URL 路径没有注册 | 查看路由代码，确认方法和路径 |
| `405 Method Not Allowed` | URL 对了但 HTTP 方法错 | 检查 `POST/GET` 是否选对 |
| `400` 且提示 JSON 错误 | Body 不是合法 JSON 或字段格式不对 | 检查 raw、JSON、逗号、引号和字段名 |
| `400` 注册失败 | 用户名重复、空密码或数据库写入失败 | 查看服务端日志和数据库 |
| `400` 登录失败 | 用户名或密码错误 | 先重新注册一组确定的测试数据 |
| `401` 缺少认证信息 | 没有 `Authorization` Header | 添加 `Bearer {{access_token}}` |
| `401` 认证格式错误 | 写成了错误格式 | 必须是 `Bearer` 加空格加 Token |
| `401` Token 已撤销 | 退出登录或刷新后仍在使用旧 Token | 替换为最新 access token |
| `401` 刚登录就失败 | JWT_SECRET 不稳定或 Token 未正确保存 | 设置固定 `JWT_SECRET`，检查服务端实现 |
| `500` | 服务端依赖未初始化或数据库异常 | 先看 Go 控制台日志 |

### 一个重要的排查原则

先看 HTTP 状态码，再看响应体，最后看 Go 服务端日志：

```text
状态码 -> 接口响应 -> 后端日志 -> 数据库状态
```

不要只根据 Postman 的红色失败提示判断原因。Postman 只是把服务器的结果显示出来。

## 12. 建议添加到项目中的最小启动检查

为了让 Postman 初学练习更顺畅，建议后端增加一个简单健康检查接口：

```text
GET /health
```

成功时返回：

```json
{
  "status": "ok"
}
```

这样每次测试账户接口前，可以先发送：

```text
GET {{base_url}}/health
```

如果健康检查都访问不了，就先不要排查注册或登录 Body。

## 13. 当前项目建议优先修复的代码问题

这些问题会直接影响接口测试结果，建议按顺序处理：

### 第一优先级：让服务真正启动

- 在 `cmd/main.go` 中读取配置。
- 创建 MySQL 连接。
- 执行自动迁移。
- 创建 Repository、Service、Handler。
- 创建 Gin Engine。
- 注册账户路由。
- 调用 `Run` 监听端口。

### 第二优先级：修复鉴权正确性

- 让 JWT 密钥在一个进程内保持稳定。
- 统一中间件和 Handler 使用的上下文键，例如统一为 `account_id`。
- 确保登录成功后 Token 确实保存到数据库。

### 第三优先级：修复 Handler 的错误流程

- `Login` 中 `ShouldBindJSON` 出错后应立即 `return`。
- `Login` 中 Service 登录失败后应立即 `return`。
- 对空用户名、空密码增加明确校验。
- 统一成功和失败的 HTTP 状态码约定。

### 第四优先级：增加可观察性

- 增加 `/health`。
- 在启动失败时输出清晰的数据库错误。
- 为每个路由写一条最小集成测试。

## 14. 从零基础的学习路线

### 第 1 阶段：只学会发送请求

目标：能发送一个请求并读懂响应。

- 新建 Request。
- 选择 POST。
- 填写 URL。
- 在 Body 中选择 raw/JSON。
- 点击 Send。
- 观察状态码、响应体和响应时间。

练习：完成注册请求。

### 第 2 阶段：学会按接口要求准备数据

目标：知道请求头和 JSON 字段为什么必须正确。

- 修改用户名。
- 修改密码。
- 删除一个字段。
- 发送非法 JSON。
- 对比成功和失败响应。

练习：完成注册失败和登录失败测试。

### 第 3 阶段：学会 Token 鉴权

目标：理解登录和受保护接口的关系。

- 从登录响应中找到 `token`。
- 添加 `Authorization` Header。
- 使用 `Bearer` 格式调用退出登录。
- 删除 Header，观察 `401`。
- 修改 Token 一个字符，观察 `401`。

### 第 4 阶段：学会验证状态变化

目标：不仅验证“这次响应成功”，还验证前后状态是否正确。

- 登录后 Token 可用。
- 刷新后新 Token 可用。
- 刷新后旧 Token 失效。
- 退出后 Token 失效。

### 第 5 阶段：学会简单自动断言

目标：让 Postman 自动告诉你是否通过。

优先断言：

- 状态码正确。
- 响应是 JSON。
- 必需字段存在。
- Token 是非空字符串。
- 错误请求返回预期状态码。

离线版本如果不支持脚本，继续使用手工检查即可；手工检查本身也是接口测试的一部分。

## 15. 最小验收清单

当你认为账户模块完成时，至少应验证：

- [ ] 服务可以稳定启动。
- [ ] `GET /health` 返回成功。
- [ ] 正常注册返回 `200`。
- [ ] 重复用户名注册返回 `400`。
- [ ] 空密码注册返回 `400`。
- [ ] 正常登录返回 `token`、`refresh_token`、`account_id`、`username`。
- [ ] 错误密码登录返回 `400`。
- [ ] 无 Token 访问受保护接口返回 `401`。
- [ ] 错误格式 Token 返回 `401`。
- [ ] 正常刷新返回新的 access token。
- [ ] 刷新后旧 access token 不能继续使用。
- [ ] 正常退出登录返回 `200`。
- [ ] 退出后原 Token 返回 `401`。
- [ ] 服务重启后，使用固定 JWT_SECRET 时，数据库中的有效 Token 行为符合设计。

## 16. 最后记住三件事

1. Postman 不能替代后端启动；连接失败时先确认服务是否真的监听端口。
2. 请求成功不只看 `200`，还要检查响应体和数据库状态是否符合预期。
3. 当前项目的路由和启动入口尚未完成，所以先把手册中的建议路由落实到 Go 代码，再按注册、登录、刷新、退出的顺序测试。

