"""
测试 LiteLLM 与 LM Studio 的连接
"""
import litellm

# 设置 LM Studio 的 API 端点
api_base = "http://192.168.8.100:11452"
api_key = "not-needed"

# 测试模型是否支持 function calling
model = "openai/qwen3.5-9b-uncensored-hauhaucs-aggressive"

print(f"测试模型：{model}")
print(f"API 端点：{api_base}")
print()

# 检查是否支持 function calling
try:
    supports_fc = litellm.supports_function_calling(model=model)
    print(f"✓ 支持 Function Calling: {supports_fc}")
except Exception as e:
    print(f"✗ 检查 Function Calling 失败：{e}")

# 尝试简单的 completion 请求
print("\n测试简单对话...")
try:
    response = litellm.completion(
        model=model,
        messages=[{"role": "user", "content": "Hello!"}],
        max_tokens=50,
        api_base=api_base,
        api_key=api_key
    )
    print(f"✓ 对话成功：{response.choices[0].message.content[:100]}")
except Exception as e:
    print(f"✗ 对话失败：{e}")

# 尝试带 function calling 的请求
print("\n测试 Function Calling...")
try:
    tools = [
        {
            "type": "function",
            "function": {
                "name": "test_function",
                "description": "A test function",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "name": {"type": "string"}
                    }
                }
            }
        }
    ]
    
    response = litellm.completion(
        model=model,
        messages=[{"role": "user", "content": "Call test function"}],
        tools=tools,
        max_tokens=100,
        api_base=api_base,
        api_key=api_key
    )
    print(f"✓ Function Calling 成功")
    if response.choices[0].message.tool_calls:
        print(f"  调用了工具：{response.choices[0].message.tool_calls[0].function.name}")
    else:
        print(f"  无工具调用，内容：{response.choices[0].message.content[:100]}")
except Exception as e:
    print(f"✗ Function Calling 失败：{e}")
