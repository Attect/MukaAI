import logging
import os
from typing import Any, Dict, List, Optional

from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse, RedirectResponse
from pydantic import BaseModel, Field

from mem0 import Memory

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

# Load environment variables
load_dotenv()

# 配置使用 FAISS 作为向量存储，LM Studio 作为 LLM 和 Embedder
FAISS_PATH = os.environ.get("FAISS_PATH", "./data/faiss_memories")  # 使用本地路径
LM_STUDIO_BASE_URL = os.environ.get("LM_STUDIO_BASE_URL", "http://192.168.8.100:11452")  # LM Studio 地址
LM_STUDIO_MODEL = os.environ.get("LM_STUDIO_MODEL", "qwen3.5-9b-uncensored-hauhaucs-aggressive")  # LM Studio 中的 LLM 模型
EMBEDDER_MODEL = os.environ.get("EMBEDDER_MODEL", "nomic-embed-text-v1.5")  # Embedding 模型名称
EMBEDDER_API_KEY = os.environ.get("EMBEDDER_API_KEY", "not-needed")  # 如果使用本地 embedder，可以设置为任意值

# 设置 OpenAI 兼容的环境变量（用于 embedder 和 litellm）
os.environ["OPENAI_API_KEY"] = EMBEDDER_API_KEY
os.environ["OPENAI_BASE_URL"] = LM_STUDIO_BASE_URL

DEFAULT_CONFIG = {
    "version": "v1.1",
    "vector_store": {
        "provider": "faiss",
        "config": {
            "collection_name": "mem0_memories",
            "path": FAISS_PATH,
            "distance_strategy": "cosine"
        }
    },
    "llm": {
        "provider": "lmstudio",
        "config": {
            "model": LM_STUDIO_MODEL,
            "temperature": 0.1,
            "max_tokens": 2000,
            "api_key": EMBEDDER_API_KEY,
            "lmstudio_base_url": f"{LM_STUDIO_BASE_URL}/v1",
            # 使用 text 格式，因为 LM Studio 不支持 json_object
            "lmstudio_response_format": {"type": "text"}
        }
    },
    "embedder": {
        "provider": "lmstudio",
        "config": {
            "model": EMBEDDER_MODEL,
            "api_key": EMBEDDER_API_KEY,
            "lmstudio_base_url": f"{LM_STUDIO_BASE_URL}/v1"
        }
    }
}

MEMORY_INSTANCE = Memory.from_config(DEFAULT_CONFIG)

# 修复 FAISS 维度问题 - nomic-embed-text-v1.5 使用 768 维
try:
    MEMORY_INSTANCE.vector_store.embedding_model_dims = 768
    # 如果 FAISS 索引已创建，需要重新创建
    if MEMORY_INSTANCE.vector_store.index and MEMORY_INSTANCE.vector_store.index.d != 768:
        logging.info("重新创建 FAISS 索引以匹配 embedding 维度 (768)")
        MEMORY_INSTANCE.vector_store.create_col(MEMORY_INSTANCE.vector_store.collection_name)
except Exception as e:
    logging.warning(f"无法修改 FAISS 维度：{e}")

app = FastAPI(
    title="Mem0 REST APIs (Local)",
    description="A REST API for managing and searching memories for your AI Agents and Apps. (Local deployment with FAISS + Ollama/LM Studio)",
    version="1.0.0",
)


class Message(BaseModel):
    role: str = Field(..., description="Role of the message (user or assistant).")
    content: str = Field(..., description="Message content.")


class MemoryCreate(BaseModel):
    messages: List[Message] = Field(..., description="List of messages to store.")
    user_id: Optional[str] = None
    agent_id: Optional[str] = None
    run_id: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None


class SearchRequest(BaseModel):
    query: str = Field(..., description="Search query.")
    user_id: Optional[str] = None
    run_id: Optional[str] = None
    agent_id: Optional[str] = None
    filters: Optional[Dict[str, Any]] = None


@app.post("/configure", summary="Configure Mem0")
def set_config(config: Dict[str, Any]):
    """Set memory configuration."""
    global MEMORY_INSTANCE
    MEMORY_INSTANCE = Memory.from_config(config)
    return {"message": "Configuration set successfully"}


@app.post("/memories", summary="Create memories")
def add_memory(memory_create: MemoryCreate):
    """Store new memories."""
    if not any([memory_create.user_id, memory_create.agent_id, memory_create.run_id]):
        raise HTTPException(status_code=400, detail="At least one identifier (user_id, agent_id, run_id) is required.")

    params = {k: v for k, v in memory_create.model_dump().items() if v is not None and k != "messages"}
    try:
        response = MEMORY_INSTANCE.add(messages=[m.model_dump() for m in memory_create.messages], **params)
        return JSONResponse(content=response)
    except Exception as e:
        logging.exception("Error in add_memory:")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/memories", summary="Get memories")
def get_all_memories(
    user_id: Optional[str] = None,
    run_id: Optional[str] = None,
    agent_id: Optional[str] = None,
):
    """Retrieve stored memories."""
    if not any([user_id, run_id, agent_id]):
        raise HTTPException(status_code=400, detail="At least one identifier is required.")
    try:
        params = {
            k: v for k, v in {"user_id": user_id, "run_id": run_id, "agent_id": agent_id}.items() if v is not None
        }
        return MEMORY_INSTANCE.get_all(**params)
    except Exception as e:
        logging.exception("Error in get_all_memories:")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/memories/{memory_id}", summary="Get a memory")
def get_memory(memory_id: str):
    """Retrieve a specific memory by ID."""
    try:
        return MEMORY_INSTANCE.get(memory_id)
    except Exception as e:
        logging.exception("Error in get_memory:")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/search", summary="Search memories")
def search_memories(search_req: SearchRequest):
    """Search for memories based on a query."""
    try:
        params = {k: v for k, v in search_req.model_dump().items() if v is not None and k != "query"}
        return MEMORY_INSTANCE.search(query=search_req.query, **params)
    except Exception as e:
        logging.exception("Error in search_memories:")
        raise HTTPException(status_code=500, detail=str(e))


@app.put("/memories/{memory_id}", summary="Update a memory")
def update_memory(memory_id: str, updated_memory: Dict[str, Any]):
    """Update an existing memory with new content.
    
    Args:
        memory_id (str): ID of the memory to update
        updated_memory (str): New content to update the memory with
        
    Returns:
        dict: Success message indicating the memory was updated
    """
    try:
        return MEMORY_INSTANCE.update(memory_id=memory_id, data=updated_memory)
    except Exception as e:
        logging.exception("Error in update_memory:")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/memories/{memory_id}/history", summary="Get memory history")
def memory_history(memory_id: str):
    """Retrieve memory history."""
    try:
        return MEMORY_INSTANCE.history(memory_id=memory_id)
    except Exception as e:
        logging.exception("Error in memory_history:")
        raise HTTPException(status_code=500, detail=str(e))


@app.delete("/memories/{memory_id}", summary="Delete a memory")
def delete_memory(memory_id: str):
    """Delete a specific memory by ID."""
    try:
        MEMORY_INSTANCE.delete(memory_id=memory_id)
        return {"message": "Memory deleted successfully"}
    except Exception as e:
        logging.exception("Error in delete_memory:")
        raise HTTPException(status_code=500, detail=str(e))


@app.delete("/memories", summary="Delete all memories")
def delete_all_memories(
    user_id: Optional[str] = None,
    run_id: Optional[str] = None,
    agent_id: Optional[str] = None,
):
    """Delete all memories for a given identifier."""
    if not any([user_id, run_id, agent_id]):
        raise HTTPException(status_code=400, detail="At least one identifier is required.")
    try:
        params = {
            k: v for k, v in {"user_id": user_id, "run_id": run_id, "agent_id": agent_id}.items() if v is not None
        }
        MEMORY_INSTANCE.delete_all(**params)
        return {"message": "All relevant memories deleted"}
    except Exception as e:
        logging.exception("Error in delete_all_memories:")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/reset", summary="Reset all memories")
def reset_memory():
    """Completely reset stored memories."""
    try:
        MEMORY_INSTANCE.reset()
        return {"message": "All memories reset"}
    except Exception as e:
        logging.exception("Error in reset_memory:")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/", summary="Redirect to the OpenAPI documentation", include_in_schema=False)
def home():
    """Redirect to the OpenAPI documentation."""
    return RedirectResponse(url="/docs")
