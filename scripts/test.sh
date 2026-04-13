#!/bin/bash
# test.sh - 统一测试运行脚本
# 用法:
#   ./scripts/test.sh           # 运行所有测试
#   ./scripts/test.sh -v        # 详细输出
#   ./scripts/test.sh -cover    # 生成覆盖率报告
#   ./scripts/test.sh -cover -v # 详细输出 + 覆盖率

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认参数
VERBOSE=""
COVER=""
COVERPROFILE=""

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -v)
            VERBOSE="-v"
            shift
            ;;
        -cover)
            COVER="-cover"
            COVERPROFILE="-coverprofile=coverage.out"
            shift
            ;;
        *)
            echo "未知参数: $1"
            echo "用法: $0 [-v] [-cover]"
            exit 1
            ;;
    esac
done

echo -e "${YELLOW}=== Go 测试运行器 ===${NC}"
echo ""

# 切换到项目根目录
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "项目目录: $(pwd)"
echo ""

# 运行测试
echo -e "${YELLOW}运行 Go 测试...${NC}"
echo "命令: go test ./... -count=1 $VERBOSE $COVER $COVERPROFILE"
echo ""

TEST_ARGS="./... -count=1 $VERBOSE $COVER $COVERPROFILE"

if go test $TEST_ARGS 2>&1 | tee test_output.txt; then
    echo ""
    echo -e "${GREEN}=== 所有测试通过 ===${NC}"

    # 如果生成了覆盖率报告，显示摘要
    if [ -f "coverage.out" ]; then
        echo ""
        echo -e "${YELLOW}=== 覆盖率摘要 ===${NC}"
        go tool cover -func=coverage.out | tail -1
        echo ""
        echo "详细覆盖率报告: go tool cover -html=coverage.out -o coverage.html"
        echo "查看覆盖率: go tool cover -func=coverage.out"
    fi
else
    echo ""
    echo -e "${RED}=== 测试失败 ===${NC}"
    # 统计失败数量
    FAIL_COUNT=$(grep -c "^FAIL" test_output.txt 2>/dev/null || echo "未知")
    echo "失败模块数: $FAIL_COUNT"
    exit 1
fi

# 清理临时文件
rm -f test_output.txt
