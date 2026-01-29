# Antigravity 权重选择方案（方案 1）

## 设计目的

- 支持 Antigravity 多账号的静态权重选择，避免纯轮询。
- 保持现有 `priority` 分组逻辑：先分组，再组内选择。
- 配置方式贴近当前文件化账号体系（`auth-dir` JSON）。
- 最小侵入：只扩展文件合成与 selector。

## 方案 1 概述（基于文件合成）

**核心思路**

- 在每个 `auth-dir` JSON 文件中新增 `weight` 字段。
- 文件合成阶段读取 `metadata` 中的 `priority/weight`，写入 `auth.Attributes`。
- selector 始终只读取 `auth.Attributes`，保持一致的读取路径。

**配置示例**

```json
{
  "type": "antigravity",
  "email": "foo@bar.com",
  "access_token": "...",
  "refresh_token": "...",
  "priority": 10,
  "weight": 3
}
```

## 改动方案（详细）

### 1) 文件合成阶段写入权重与优先级

**位置**

- `internal/watcher/synthesizer/file.go`

**改动**

- 从 `metadata["priority"]` 读取优先级。
- 从 `metadata["weight"]` 读取权重。
- `weight` 缺失默认 1，`weight < 0` 时打 warning。
- 统一写入 `auth.Attributes["priority"]` 与 `auth.Attributes["weight"]`。

**目的**

- 让文件型账号与配置型账号使用同一条“读取路径”。
- selector 无需关心来源差异。

### 2) selector 支持权重选择

**位置**

- `sdk/cliproxy/auth/selector.go`

**改动**

- 新增 `weightOf(auth)`，读取 `Attributes["weight"]`，默认 1。
- `weight` 必须是数字，`weight <= 0` 视为不可选。
- 新增 `WeightedSelector`，在“最高 priority 组”内进行加权选择。
- 维持 `getAvailableAuths` 不变，仅使用其返回的候选集合。

**权重选择伪代码（加权随机）**

```go
total := 0
for _, a := range available {
  w := weightOf(a)
  if w <= 0 { continue }
  total += w
}
if total == 0 { return nil, ErrNoAvailableAuth }
r := rand.Intn(total)
sum := 0
for _, a := range available {
  w := weightOf(a)
  if w <= 0 { continue }
  sum += w
  if r < sum { return a, nil }
}
```

### 3) 支持路由策略配置

**位置**

- `sdk/cliproxy/builder.go`
- `sdk/cliproxy/service.go`

**改动**

- 新增 `routing.strategy = weighted`。
- 映射到 `WeightedSelector{}`。
- 默认策略为 `weighted`（未配置时按 `weighted` 处理）。

## 行为说明

- 先过滤不可用账号（禁用/冷却/配额）。
- 按 `priority` 分组，取最高组。
- 组内按 `weight` 选择账号。
- `weight` 越大命中越高。
- `weight` 缺失默认 1，`weight = 0` 表示不调用。

## 测试建议

- `sdk/cliproxy/auth/selector_test.go`
  - 组内权重分布测试（多次采样）。
  - `weight` 缺失默认 1。
  - `weight = 0` 不会被选中。
  - priority 分组优先级测试。
  - 冷却/禁用过滤保持不变。

## 备注

- 本方案只处理“静态权重”，不会自动根据错误率或配额动态调整。
