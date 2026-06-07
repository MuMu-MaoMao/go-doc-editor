#!/bin/bash
# =============================================
# 测试套件：文档 CRUD（三重测试 · 第一层：脚本测试）
# 测试范围：ListFiles / ReadFile / SaveFile / DeleteFile
# 前置条件：服务已运行（默认 localhost:3000）
# 关联功能：文档数据库存储（2026-06-06）
# 参考文档：docs/08-测试要求.md
# =============================================

BASE_URL="${1:-http://localhost:3000}"
PASS=0
FAIL=0

# ---- 辅助函数 ----

check() {
    local desc="$1"
    local expected="$2"
    local actual="$3"
    if echo "$actual" | grep -q "$expected"; then
        echo "  ✅ $desc"
        PASS=$((PASS + 1))
    else
        echo "  ❌ $desc"
        echo "     期望包含: $expected"
        echo "     实际响应: $actual"
        FAIL=$((FAIL + 1))
    fi
}

section() {
    echo ""
    echo "=== $1 ==="
}

cleanup() {
    # 清理测试用户和数据
    local tok="$1"
    curl -s -X DELETE "$BASE_URL/api/file/test.md" \
      -H "Authorization: Bearer $tok" > /dev/null 2>&1
    curl -s -X DELETE "$BASE_URL/api/file/userA-file.md" \
      -H "Authorization: Bearer $tok" > /dev/null 2>&1
}

# ---- 1. 注册与登录 ----

section "注册与登录"

RESP=$(curl -s -X POST "$BASE_URL/api/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test123"}')
check "注册新用户" "success" "$RESP"

RESP=$(curl -s -X POST "$BASE_URL/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"test123"}')
TOKEN=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin).get('token',''))" 2>/dev/null)
check "登录成功获取 Token" "token" "$RESP"

if [ -z "$TOKEN" ]; then
    echo "  ⛔ 无法获取 Token，终止测试"
    exit 1
fi

# ---- 2. 文档 CRUD ----

section "文档 CRUD"

# 2a. 初始文件列表（应为空）
RESP=$(curl -s "$BASE_URL/api/files" -H "Authorization: Bearer $TOKEN")
check "初始文件列表为空" "success" "$RESP"

# 2b. 保存新文件
RESP=$(curl -s -X POST "$BASE_URL/api/file/test.md" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content":"# Hello World\n\n这是存入数据库的文档内容！"}')
check "保存新文件 test.md" "文件保存成功" "$RESP"

# 2c. 读取文件
RESP=$(curl -s "$BASE_URL/api/file/test.md" -H "Authorization: Bearer $TOKEN")
check "读取文件 test.md" "Hello World" "$RESP"

# 2d. 保存后文件列表
RESP=$(curl -s "$BASE_URL/api/files" -H "Authorization: Bearer $TOKEN")
check "文件列表包含 test.md" "test.md" "$RESP"

# 2e. 覆盖保存（更新内容）
RESP=$(curl -s -X POST "$BASE_URL/api/file/test.md" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content":"更新后的内容"}')
check "覆盖保存 test.md" "文件保存成功" "$RESP"

# 2f. 验证更新后的内容
RESP=$(curl -s "$BASE_URL/api/file/test.md" -H "Authorization: Bearer $TOKEN")
check "读取验证更新内容" "更新后的内容" "$RESP"

# 2g. 删除文件
RESP=$(curl -s -X DELETE "$BASE_URL/api/file/test.md" \
  -H "Authorization: Bearer $TOKEN")
check "删除文件 test.md" "文件删除成功" "$RESP"

# 2h. 删除后文件列表
RESP=$(curl -s "$BASE_URL/api/files" -H "Authorization: Bearer $TOKEN")
check "文件列表为空（删除后）" "success" "$RESP"

# ---- 3. 边界条件 ----

section "边界条件"

# 3a. 读取不存在的文件
RESP=$(curl -s "$BASE_URL/api/file/nonexistent.md" \
  -H "Authorization: Bearer $TOKEN")
check "读取不存在的文件返回错误" "文件不存在" "$RESP"

# 3b. 删除不存在的文件
RESP=$(curl -s -X DELETE "$BASE_URL/api/file/nonexistent.md" \
  -H "Authorization: Bearer $TOKEN")
check "删除不存在的文件返回错误" "文件不存在" "$RESP"

# 3c. 保存空内容
RESP=$(curl -s -X POST "$BASE_URL/api/file/empty.md" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content":""}')
check "保存空内容文件" "文件保存成功" "$RESP"

# 清理空内容文件
curl -s -X DELETE "$BASE_URL/api/file/empty.md" \
  -H "Authorization: Bearer $TOKEN" > /dev/null 2>&1

# ---- 4. 用户隔离 ----

section "用户隔离"

# 用户A保存一个文件
curl -s -X POST "$BASE_URL/api/file/userA-file.md" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content":"用户A的文档"}' > /dev/null

# 注册用户B
RESP=$(curl -s -X POST "$BASE_URL/api/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuserB","password":"test123"}')

# 用户B登录
RESP=$(curl -s -X POST "$BASE_URL/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuserB","password":"test123"}')
TOKEN_B=$(echo "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin).get('token',''))" 2>/dev/null)

# 用户B列文件（应该为空）
RESP=$(curl -s "$BASE_URL/api/files" -H "Authorization: Bearer $TOKEN_B")
check "用户B看不到用户A的文件" "success" "$RESP"

# 用户B读取用户A的文件（应该报 文件不存在）
RESP=$(curl -s "$BASE_URL/api/file/userA-file.md" \
  -H "Authorization: Bearer $TOKEN_B")
check "用户B不能读取用户A的文件" "文件不存在" "$RESP"

# 清理
cleanup "$TOKEN"

# ---- 5. 总结 ----

section "测试结果"
echo "  通过: $PASS"
echo "  失败: $FAIL"
if [ $FAIL -eq 0 ]; then
    echo "  状态: ✅ 全部通过"
else
    echo "  状态: ❌ 存在失败项"
fi
echo ""
exit $FAIL
