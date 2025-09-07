# Multi Task Web (SSE Chat)

- 启动依赖: Node 18+
- 环境变量:
  - `NEXT_PUBLIC_API_BASE` (可选): 后端基地址，默认同源。例如 `http://localhost:8081`

## 开发

```bash
npm install
npm run dev
```

访问: http://localhost:3000

## 接口约定
- 前端会向 `POST {NEXT_PUBLIC_API_BASE}/api/chat/stream` 发送 JSON: `{ message: string }`
- 服务端需以 SSE 返回，数据行形如: `data: 内容`，以换行分割，可发送多次。
- 可发送 `data: [DONE]` 作为结束标记（可选）。



