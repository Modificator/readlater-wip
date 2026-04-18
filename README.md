# readlater-wip

基于 Go + Echo 的稍后阅读归档系统 MVP（API-only）。

## 已实现能力（MVP 基础版）

- URL 提交异步归档任务
- 任务状态查询与手动重试
- 规则管理（prefix/regex）与 URL 命中测试
- GitHub 链接识别（repo 与非 repo 区分）
- Gitea 多目标配置（alias）
- GitHub repo -> Gitea mirror 调用（通过 Gitea migrate API）
- HTML 抓取、正文提取（基础）、Markdown 生成（基础）
- 正文图片下载与本地化（基础）
- 归档落盘：`content.md` / `metadata.json` / `raw.html` / `assets/`

## 运行

```bash
go run ./cmd/server
```

默认监听：`0.0.0.0:8080`

可选环境变量：

- `LISTEN_ADDR`：服务监听地址（默认 `:8080`）
- `ARCHIVE_STORAGE_ROOT`：归档目录根路径（默认 `./archive`）

## API 草案（当前实现）

### 健康检查

- `GET /health`

### 任务

- `POST /api/v1/tasks`
- `GET /api/v1/tasks/:id`
- `POST /api/v1/tasks/:id/retry`

创建任务示例：

```json
{
  "url": "https://example.com/article",
  "save_web_content": true,
  "download_assets": true,
  "extract_github_links": true,
  "clone_to_gitea": false,
  "gitea_target_alias": "",
  "rule_id": "",
  "force_browser_render": false,
  "overwrite_existing": false,
  "repo_name_override": ""
}
```

### 规则

- `POST /api/v1/rules`
- `GET /api/v1/rules`
- `POST /api/v1/rules/test-match`

### Gitea 目标

- `POST /api/v1/gitea-targets`
- `GET /api/v1/gitea-targets`

## 说明

当前版本为可运行的后端 MVP 骨架，强调接口与异步链路可用性；正文提取与 Markdown 转换为基础实现，便于后续按规则系统逐步增强。
