# Mem0 Server - 本地部署版本

## 概述

这是一个独立部署的 Mem0 REST API 服务器，使用 FAISS 作为向量存储，支持通过 Ollama 协议连接本地 LLM（如 LM Studio）。

## 特性

- **独立可执行文件**: 无需安装 Python 环境，直接运行
- **本地部署**: 所有数据存储在本地，无需云端服务
- **REST API**: 提供完整的记忆管理 API
- **支持 LM Studio**: 可以通过 Ollama 协议连接 LM Studio 运行的本地模型

## 构建方法

### 1. 安装依赖

```bash
cd test-available/mem0-integration/mem0-server-local
pip install -r requirements.txt
```

### 2. 构建可执行文件

```bash
# Windows
pyinstaller --clean mem0_server.spec

# 构建完成后，可执行文件位于 dist/mem0-server.exe
```

### 3. 使用虚拟环境构建（推荐）

```bash
# 创建虚拟环境
python -m venv venv
venv\Scripts\activate

# 安装依赖
pip install -r requirements.txt

# 构建
pyinstaller --clean mem0_server.spec

# 取消激活虚拟环境
deactivate
```

## 使用方法

### 环境变量配置

创建 `.env` 文件或在命令行中设置以下环境变量：

```bash
# FAISS 向量存储路径
FAISS_PATH=./data/faiss_memories

# LLM 配置（连接 LM Studio）
OLLAMA_BASE_URL=http://localhost:1234
OLLAMA_MODEL=local-model

# Embedder 配置（如果使用 LM Studio 提供 embedding）
EMBEDDER_MODEL=text-embedding-3-small
EMBEDDER_API_KEY=not-needed
```

### 启动服务器

#### 使用可执行文件

```bash
# Windows
dist\mem0-server\mem0-server.exe

# 或直接运行打包后的单文件
dist\mem0-server.exe
```

#### 使用 Python 脚本（开发模式）

```bash
python run_server.py
```

### API 使用

服务器启动后，访问：

- **API 文档**: http://localhost:8000/docs
- **健康检查**: http://localhost:8000

#### 示例：创建记忆

```bash
curl -X POST "http://localhost:8000/memories" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "我喜欢看科幻电影"},
      {"role": "assistant", "content": "好的，我记住了您喜欢科幻电影"}
    ],
    "user_id": "user123"
  }'
```

#### 示例：搜索记忆

```bash
curl -X POST "http://localhost:8000/search" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "电影偏好",
    "user_id": "user123"
  }'
```

#### 示例：获取所有记忆

```bash
curl "http://localhost:8000/memories?user_id=user123"
```

## 与 LM Studio 集成

### 1. 启动 LM Studio

1. 打开 LM Studio
2. 加载模型（如 `qwen3.5-9b-uncensored-hauhaucs-aggressive`）
3. 启动本地服务器（默认端口 1234）
4. 确保开启了 Function Calling 支持

### 2. 配置 Mem0 Server

在 `.env` 文件中设置：

```bash
OLLAMA_BASE_URL=http://localhost:1234
OLLAMA_MODEL=local-model
```

### 3. 测试连接

启动 Mem0 Server 后，通过 API 创建和搜索记忆来验证连接。

## API 接口

### POST /memories
创建新的记忆

### GET /memories
获取指定用户/代理/运行的所有记忆

### GET /memories/{memory_id}
获取特定记忆

### POST /search
搜索记忆

### PUT /memories/{memory_id}
更新记忆

### GET /memories/{memory_id}/history
获取记忆历史

### DELETE /memories/{memory_id}
删除特定记忆

### DELETE /memories
删除所有记忆

### POST /reset
重置所有记忆

## 技术栈

- **FastAPI**: Web 框架
- **Mem0**: 记忆管理库
- **FAISS**: 向量存储
- **Ollama Protocol**: LLM 通信协议
- **PyInstaller**: 打包工具

## 注意事项

1. **首次启动**: 首次运行时会创建 FAISS 索引文件，可能需要几秒钟
2. **模型要求**: 使用的 LLM 需要支持 function calling
3. **端口占用**: 默认使用 8000 端口，如有冲突请修改
4. **数据持久化**: FAISS 数据存储在 `FAISS_PATH` 指定的目录

## 故障排除

### 问题：找不到模块

确保所有依赖都已正确安装，使用 `pip install -r requirements.txt` 重新安装。

### 问题：无法连接 LLM

检查 LM Studio 是否已启动，并且端口配置正确。

### 问题：打包后文件过大

PyInstaller 会打包所有依赖，文件较大是正常的。可以考虑使用 UPX 压缩。

## 开发计划

- [ ] 支持更多向量存储后端
- [ ] 添加 Web 管理界面
- [ ] 支持多用户认证
- [ ] 添加记忆导出/导入功能
