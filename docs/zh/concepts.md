# 概念

## EDA - 事件驱动架构

事件驱动型架构有三个主要组成部分：事件生成器、事件路由器和事件使用者。
生成器将事件发布至路由器，由路由器对事件进行筛选并推送给使用者。
生成器服务和使用者服务将会解耦，从而使它们能够独立扩展、更新和部署。

## EventBridge

EventBridge 是 EDA 的实现，提供了 Event 的匹配、转换和路由功能。

### Event

Event 用来表示分布式系统中发生的状态变化。Event 数据结构所遵循的 schema 可以通过 source + type 在 Schema 中索引。
Event 包含以下字段：

- `id`: Event 在 Source 中的唯一标识符。如果不指定，EventBridge 会自动生成一个。
- `source`: Event 的源头，通常是服务或应用程序的名称。
- `subject`: 可选，在 Source 内部的更细粒度的上下文标识。
- `type`: Event 的类型，用于区分不同类型的 Event。
- `time`: 事件的发生时间（非 EventBridge 接收时间），由发送端设置。
- `data`: 与特定领域相关的场景数据，遵循由 source + type 确定的 Schema。
- `datacontenttype`: 仅支持 `application/json`，表示 data 的内容类型。

向 EventBridge 发送 Event 时，可以指定 `pub_time` 和 `retry_strategy`，
`pub_time` 决定 Event 何时发送到 Target，`retry_strategy` 决定发送失败后采取什么样的重试策略。

### Source

Event 的源头，通常是一个服务或应用程序，它会生成 Event 并将其发送到 EventBridge。

### Schema

Schema 定义了 Event 的结构和格式，使用 JSON Schema 来描述。
想要将 Event 发送到 EventBridge，必须先注册 Schema。
Schema 包含以下字段：

- `source`: Event 的源头，通常是服务或应用程序的名称。
- `type`: Event 的类型，用于区分不同类型的 Event。
- `bus_name`: Event 所属的 Bus 名称，EventBridge 会根据 Bus 名称将 Event 路由到对应的 Bus。
- `spec`: 序列化的 JSON Schema，用于描述 Event 的结构。
- `version`: Schema 的版本号，每次 Schema 变更时，版本号会递增。

### Bus

用来存储和传输事件的中转站，Bus 和 Bus 的资源完全隔离，
可以存在多个 Bus，EventBridge 自带 Default Bus。
Bus 有并发和有序两种模式，如果是有序模式，Event 会按照发送顺序进行处理。
Bus 被移除时，和 Bus 相关的 Schema 和 Rule 也会被删除。

### Rule

Rule 用于对 Event 进行匹配、转换和路由。

#### Pattern

Pattern 是 Rule 的子概念，用来匹配符合特定条件的事件。
Pattern 的整体结构与 Event 相同，区别在于 Event 描述内容，而 Pattern 描述的是匹配规则。

#### Target

Target 是 Rule 的子概念，包含 Event 的转换规则和发送目标。
下面是 Target 的示例：

```json
{
  "ID": 1,
  "Type": "HTTPDispatcher",
  "Params": [
    {
      "Key": "url",
      "Form": "CONSTANT",
      "Value": "http://127.0.0.1:10188/target/event",
      "Template": null
    },
    {
      "Key": "method",
      "Form": "CONSTANT",
      "Value": "POST",
      "Template": null
    },
    {
      "Key": "body",
      "Form": "TEMPLATE",
      "Value": "{\"subject\":\"$.data.a\"}",
      "Template": "{\"code\":\"10188:${subject}\"}"
    }
  ],
  "RetryStrategy": 1
}
```

上面定义了一个使用 HTTP POST 请求，将 `{"code":"10188:{$.data.a}"}` 发送到 `http://127.0.0.1:10188/target/event`
的 Target。`Params` 定义了将 Event 转换成 `HTTPDispatcher` 参数的细节，`RetryStrategy` 的指定则会覆盖 Event 中 `RetryStrategy`。

##### Dispatcher

Dispatcher 是 Target 的类型，负责将 Event 发送到指定的目标。目前 EventBridge 支持以下 Dispatcher 类型：

- `HTTPDispatcher`: 通过 HTTP 请求发送 Event。
- `gRPCDispatcher`: 通过 gRPC 请求发送 Event。

你可以使用接口 `rpc ListDispatcherSchema` 来获取所有 Dispatcher的 DispatcherSchema。
下面是 `HTTPDispatcher` 的 DispatcherSchema:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "HTTP dispatcher",
  "description": "The data format of the HTTP dispatcher",
  "type": "object",
  "properties": {
    "method": {
      "description": "method specifies the HTTP method (GET, POST, PUT, etc.)",
      "type": "string"
    },
    "url": {
      "description": "url specifies the URL to access",
      "type": "string"
    },
    "header": {
      "description": "header contains the request header fields",
      "type": "object",
      "patternProperties": {
        ".*": {
          "type": "string"
        }
      }
    },
    "body": {
      "description": "body is the request's body",
      "type": "object"
    }
  },
  "required": [
    "method",
    "url"
  ]
}
```

上面的 DispatcherSchema 描述了 `HTTPDispatcher` 的参数结构，
包含 `method`、`url`、`header` 和 `body` 四个字段，其中 `method`、`url` 是必选字段。
