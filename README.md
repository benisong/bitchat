# bitchat

FARP (Decentralized Asynchronous Routing & Mutual Ledger Protocol) 参考实现。

全局可达的去中心化 P2P 聊天节点，顶端有限流进华、弱网自愈、功劳记账。

---

## 快速开始

```bash
# 1. 启动节点守护进程
./bin/bitchat-daemon --data-dir ~/.bitchat --relay --dht-server
# 输出会包含本场 API 地址，如 http://127.0.0.1:49281

# 2. 在另一个终端使用 CLI 客户端
./bin/bitchat-cli --api-addr http://127.0.0.1:49281 status
./bin/bitchat-cli --api-addr http://127.0.0.1:49281 contacts
./bin/bitchat-cli --api-addr http://127.0.0.1:49281 credits
```

---

## 参考鑑定

See [FARP_高级设计约定][link to be added]

## 小事项

- 项目以傳递訓 [MIT](LICENSE); 需要小心車鿛推動
- 默认 CLI 客户端; 可自行字非春落页面程式连接 daemon HTTP API

### API 接口

所有路径均是 localhost-only（默认绑定 `127.0.0.1:自动端口`）

#### 只读（安全）
- `GET /status` — 节点状态、筺调余量、身份简介
- `GET /contacts` — 联系人列表
- `GET /messages/:pubkey ?limit=50` — 对话记录分页
- `GET /credits` — 积分余额、冻结状态、贡献创
- `GET /routes` — 本地路由缓存
- `GET /halloffame` — 荣誉殿堂
- `GET /outbox` — 待发封邮

#### 轻量写（有业务检验）
- `POST /contacts` — 添加联系人
- `POST /outbox` — 发送消息（前端明文 → daemon 加密、路由、透送）
- `POST /ui-migration-listen` — 请求正向迁移监听
