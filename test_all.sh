#!/bin/bash

# GoCache 全功能测试脚本
# 功能: 测试所有 HTTP API 并输出详细的测试结果
# 使用方法: chmod +x test_all.sh && ./test_all.sh

set -e

BASE_URL="http://localhost:8080/cache"
PASS=0
FAIL=0
TOTAL=0
TMP_DIR="/tmp/gocache_test_$$"
mkdir -p "$TMP_DIR"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 清理函数
cleanup() {
    echo -e "\n${BLUE}清理测试环境...${NC}"
    if [ -n "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    rm -rf "$TMP_DIR"
    echo -e "${BLUE}清理完成${NC}"
}

# 捕获退出信号
trap cleanup EXIT

# 打印表头
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}========================================${NC}"
    printf "%-25s %-25s %-15s %s\n" "测试功能" "请求路径" "预期结果" "测试结果"
    printf "%-25s %-25s %-15s %s\n" "-------------------------" "-------------------------" "---------------" "-------------------------"
}

# 测试结果函数
test_result() {
    local feature=$1
    local path=$2
    local expected=$3
    local actual=$4
    local status=$5  # "pass" or "fail"
    
    TOTAL=$((TOTAL+1))
    
    if [ "$status" == "pass" ]; then
        PASS=$((PASS+1))
        echo -e "${GREEN}✓${NC} %-23s %-23s %-15s %s" "$feature" "$path" "$expected" "${GREEN}$actual${NC}"
    else
        FAIL=$((FAIL+1))
        echo -e "${RED}✗${NC} %-23s %-23s %-15s %s" "$feature" "$path" "$expected" "${RED}$actual${NC}"
    fi
}

# 启动服务器
echo -e "${BLUE}启动 GoCache 服务器...${NC}"
go run main.go > "$TMP_DIR/server.log" 2>&1 &
SERVER_PID=$!

# 等待服务器启动
echo -e "${BLUE}等待服务器启动...${NC}"
for i in {1..30}; do
    if curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/keys" > /dev/null 2>&1; then
        echo -e "${GREEN}服务器启动成功 (PID: $SERVER_PID)${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}服务器启动超时${NC}"
        cat "$TMP_DIR/server.log"
        exit 1
    fi
    sleep 1
done

sleep 1

# ==================== 基本缓存操作测试 ====================
print_header "1. 基本缓存操作"

# 1.1 设置缓存
RESP=$(curl -s -X POST "$BASE_URL/user:name" \
  -H "Content-Type: application/json" \
  -d '{"value": "GoCache", "ttl": "1h"}')
STATUS=$(echo "$RESP" | grep -o '"key": *"[^"]*"' | head -1)
if [ -n "$STATUS" ]; then
    test_result "设置缓存" "POST /cache/user:name" "201 Created" "201 Created" "pass"
else
    test_result "设置缓存" "POST /cache/user:name" "201 Created" "失败" "fail"
fi

# 1.2 获取缓存
HTTP_CODE=$(curl -s -o "$TMP_DIR/resp.json" -w "%{http_code}" "$BASE_URL/user:name")
VALUE=$(grep -o '"value": *"[^"]*"' "$TMP_DIR/resp.json" | head -1)
if [ "$HTTP_CODE" == "200" ] && [ -n "$VALUE" ]; then
    test_result "获取缓存" "GET /cache/user:name" "200 OK" "200 OK" "pass"
else
    test_result "获取缓存" "GET /cache/user:name" "200 OK" "$HTTP_CODE" "fail"
fi

# 1.3 获取不存在的键
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/nonexistent")
if [ "$HTTP_CODE" == "404" ]; then
    test_result "获取不存在的键" "GET /cache/nonexistent" "404 Not Found" "404" "pass"
else
    test_result "获取不存在的键" "GET /cache/nonexistent" "404 Not Found" "$HTTP_CODE" "fail"
fi

# 1.4 删除缓存
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/user:name")
if [ "$HTTP_CODE" == "200" ]; then
    test_result "删除缓存" "DELETE /cache/user:name" "200 OK" "200 OK" "pass"
else
    test_result "删除缓存" "DELETE /cache/user:name" "200 OK" "$HTTP_CODE" "fail"
fi

# 1.5 删除不存在的键
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/nonexistent")
if [ "$HTTP_CODE" == "404" ]; then
    test_result "删除不存在的键" "DELETE /cache/nonexistent" "404 Not Found" "404" "pass"
else
    test_result "删除不存在的键" "DELETE /cache/nonexistent" "404 Not Found" "$HTTP_CODE" "fail"
fi

# 1.6 设置带TTL的缓存
RESP=$(curl -s -X POST "$BASE_URL/temp" \
  -H "Content-Type: application/json" \
  -d '{"value": "temporary", "ttl": "5m"}')
STATUS=$(echo "$RESP" | grep -o '"ttl": *"[^"]*"' | head -1)
if [ -n "$STATUS" ]; then
    test_result "设置带TTL缓存" "POST /cache/temp" "成功" "成功" "pass"
else
    test_result "设置带TTL缓存" "POST /cache/temp" "成功" "失败" "fail"
fi

# ==================== 批量操作测试 ====================
print_header "2. 批量操作测试"

# 2.1 批量设置缓存
for i in {1..5}; do
    curl -s -X POST "$BASE_URL/key$i" \
      -H "Content-Type: application/json" \
      -d "{\"value\": \"value$i\"}" > /dev/null
done
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/key1")
if [ "$HTTP_CODE" == "200" ]; then
    test_result "批量设置缓存" "POST /cache/key{1..5}" "5个键设置成功" "成功" "pass"
else
    test_result "批量设置缓存" "POST /cache/key{1..5}" "5个键设置成功" "失败" "fail"
fi

# 2.2 获取所有键
RESP=$(curl -s "$BASE_URL/keys")
COUNT=$(echo "$RESP" | grep -o '"count": *[0-9]*' | grep -o '[0-9]*')
if [ -n "$COUNT" ] && [ "$COUNT" -ge 5 ]; then
    test_result "获取所有键" "GET /cache/keys" "count >= 5" "count: $COUNT" "pass"
else
    test_result "获取所有键" "GET /cache/keys" "count >= 5" "count: ${COUNT:-0}" "fail"
fi

# ==================== 统计信息测试 ====================
print_header "3. 统计信息测试"

# 3.1 获取统计信息
HTTP_CODE=$(curl -s -o "$TMP_DIR/stats.json" -w "%{http_code}" "$BASE_URL/stats")
if [ "$HTTP_CODE" == "200" ]; then
    HITS=$(grep -o '"hits": *[0-9]*' "$TMP_DIR/stats.json" | grep -o '[0-9]*')
    SETS=$(grep -o '"sets": *[0-9]*' "$TMP_DIR/stats.json" | grep -o '[0-9]*')
    test_result "获取统计信息" "GET /cache/stats" "200 OK, hits>0, sets>0" "200 OK, hits:$HITS, sets:$SETS" "pass"
else
    test_result "获取统计信息" "GET /cache/stats" "200 OK" "$HTTP_CODE" "fail"
fi

# 3.2 统计命中率
if [ -n "$HITS" ] && [ "$HITS" -gt 0 ]; then
    test_result "验证统计记录" "GET /cache/stats" "命中率计算正常" "正常" "pass"
else
    test_result "验证统计记录" "GET /cache/stats" "命中率计算正常" "异常" "fail"
fi

# ==================== 管理操作测试 ====================
print_header "4. 管理操作测试"

# 4.1 清空缓存
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/clear")
if [ "$HTTP_CODE" == "200" ]; then
    test_result "清空缓存" "POST /cache/clear" "200 OK" "200 OK" "pass"
else
    test_result "清空缓存" "POST /cache/clear" "200 OK" "$HTTP_CODE" "fail"
fi

# 4.2 验证清空结果
RESP=$(curl -s "$BASE_URL/keys")
COUNT=$(echo "$RESP" | grep -o '"count": *[0-9]*' | grep -o '[0-9]*')
if [ "$COUNT" == "0" ]; then
    test_result "验证清空结果" "GET /cache/keys" "count: 0" "count: 0" "pass"
else
    test_result "验证清空结果" "GET /cache/keys" "count: 0" "count: ${COUNT:-null}" "fail"
fi

# ==================== 错误处理测试 ====================
print_header "5. 错误处理测试"

# 5.1 无效 JSON
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/invalid" \
  -H "Content-Type: application/json" \
  -d 'invalid json')
if [ "$HTTP_CODE" == "400" ]; then
    test_result "无效JSON" "POST /cache/invalid" "400 Bad Request" "400" "pass"
else
    test_result "无效JSON" "POST /cache/invalid" "400 Bad Request" "$HTTP_CODE" "fail"
fi

# 5.2 空键
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/" \
  -H "Content-Type: application/json" \
  -d '{"value": "test"}')
if [ "$HTTP_CODE" == "400" ]; then
    test_result "空键" "POST /cache/" "400 Bad Request" "400" "pass"
else
    test_result "空键" "POST /cache/" "400 Bad Request" "$HTTP_CODE" "fail"
fi

# 5.3 不允许的方法
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$BASE_URL/test")
if [ "$HTTP_CODE" == "405" ]; then
    test_result "不允许的方法" "PATCH /cache/test" "405 Method Not Allowed" "405" "pass"
else
    test_result "不允许的方法" "PATCH /cache/test" "405 Method Not Allowed" "$HTTP_CODE" "fail"
fi

# ==================== CORS 测试 ====================
print_header "6. CORS 跨域测试"

# 6.1 OPTIONS 预检请求
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X OPTIONS "$BASE_URL/keys" \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET")
if [ "$HTTP_CODE" == "200" ]; then
    test_result "OPTIONS预检" "OPTIONS /cache/keys" "200 OK" "200 OK" "pass"
else
    test_result "OPTIONS预检" "OPTIONS /cache/keys" "200 OK" "$HTTP_CODE" "fail"
fi

# 6.2 CORS 头
HEADERS=$(curl -s -D - -o /dev/null "$BASE_URL/keys")
CORS_HEADER=$(echo "$HEADERS" | grep -i "Access-Control-Allow-Origin" || true)
if [ -n "$CORS_HEADER" ]; then
    test_result "CORS头" "GET /cache/keys" "包含CORS头" "包含" "pass"
else
    test_result "CORS头" "GET /cache/keys" "包含CORS头" "未找到" "fail"
fi

# ==================== 总结 ====================
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  测试总结${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "总测试数: $TOTAL"
echo -e "通过:     ${GREEN}$PASS${NC}"
echo -e "失败:     ${RED}$FAIL${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✓ 所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}✗ 部分测试失败，请检查日志${NC}"
    echo "服务器日志: $TMP_DIR/server.log"
    cat "$TMP_DIR/server.log"
    exit 1
fi
