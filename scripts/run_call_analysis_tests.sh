#!/bin/bash

# 调用分析测试运行脚本
# 用于运行所有跨语言调用分析测试

set -e

echo "=========================================="
echo "CodeAtlas 调用分析测试"
echo "=========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 测试函数
run_test() {
    local test_name=$1
    local test_pattern=$2
    
    echo -e "${YELLOW}运行测试: ${test_name}${NC}"
    if go test -v ./tests/integration -run "${test_pattern}" -timeout 30s 2>&1 | grep -q "PASS"; then
        echo -e "${GREEN}✓ ${test_name} 通过${NC}"
        return 0
    else
        echo -e "${RED}✗ ${test_name} 失败${NC}"
        return 1
    fi
}

# 计数器
total=0
passed=0
failed=0

echo "1. 运行基于 Fixture 的测试（推荐）"
echo "-------------------------------------------"

# C++ → C
total=$((total + 1))
if run_test "C++ → C" "TestCallAnalysis_CPPCallsC_Fixture"; then
    passed=$((passed + 1))
else
    failed=$((failed + 1))
fi
echo ""

# Objective-C → C
total=$((total + 1))
if run_test "Objective-C → C" "TestCallAnalysis_ObjCCallsC_Fixture"; then
    passed=$((passed + 1))
else
    failed=$((failed + 1))
fi
echo ""

# Objective-C++ → C++
total=$((total + 1))
if run_test "Objective-C++ → C++" "TestCallAnalysis_ObjCppCallsCpp_Fixture"; then
    passed=$((passed + 1))
else
    failed=$((failed + 1))
fi
echo ""

# Kotlin → Java
total=$((total + 1))
if run_test "Kotlin → Java" "TestCallAnalysis_KotlinCallsJava_Fixture"; then
    passed=$((passed + 1))
else
    failed=$((failed + 1))
fi
echo ""

# Swift → Objective-C
total=$((total + 1))
if run_test "Swift → Objective-C" "TestCallAnalysis_SwiftCallsObjC_Fixture"; then
    passed=$((passed + 1))
else
    failed=$((failed + 1))
fi
echo ""

# TypeScript → JavaScript
total=$((total + 1))
if run_test "TypeScript → JavaScript" "TestCallAnalysis_TypeScriptCallsJS_Fixture"; then
    passed=$((passed + 1))
else
    failed=$((failed + 1))
fi
echo ""

# 总结
echo "=========================================="
echo "测试总结"
echo "=========================================="
echo -e "总计: ${total} 个测试"
echo -e "${GREEN}通过: ${passed}${NC}"
echo -e "${RED}失败: ${failed}${NC}"
echo ""

if [ $failed -eq 0 ]; then
    echo -e "${GREEN}✓ 所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}✗ 有 ${failed} 个测试失败${NC}"
    exit 1
fi
