"""
直接使用 OpenAI 客户端测试 LM Studio
"""
from openai import OpenAI

# 创建 OpenAI 客户端，指向 LM Studio
client = OpenAI(
    base_url="http://192.168.8.100:11452/v1",
    api_key="not-needed"
)

model = "qwen3.5-9b-uncensored-hauhaucs-aggressive"

print(f"测试模型：{model}")
print(f"API 端点：{client.base_url}")
print()

# 测试简单对话
print("测试简单对话...")
try:
    response = client.chat.completions.create(
        model=model,
        messages=[{"role": "user", "content": "你好！"}],
        max_tokens=50
    )
    print(f"✓ 对话成功：{response.choices[0].message.content}")
except Exception as e:
    print(f"✗ 对话失败：{e}")

# 测试 function calling
print("\n测试 Function Calling...")
try:
    tools = [
        {
            "type": "function",
            "function": {
                "name": "get_weather",
                "description": "Get the weather in a given city",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "city": {"type": "string", "description": "The city to get the weather for"}
                    },
                    "required": ["city"]
                }
            }
        }
    ]
    
    response = client.chat.completions.create(
        model=model,
        messages=[{"role": "user", "content": "北京天气怎么样？"}],
        tools=tools,
        max_tokens=100
    )
    print(f"✓ Function Calling 成功")
    
    if response.choices[0].message.tool_calls:
        tool_call = response.choices[0].message.tool_calls[0]
        print(f"  调用了工具：{tool_call.function.name}")
        print(f"  参数：{tool_call.function.arguments}")
    else:
        print(f"  无工具调用，内容：{response.choices[0].message.content}")
except Exception as e:
    print(f"✗ Function Calling 失败：{e}")
