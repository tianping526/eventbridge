# 概念

## EDA - 事件驱动架构

事件驱动型架构有三个主要组成部分：事件生成器、事件路由器和事件使用者。
生成器将事件发布至路由器，由路由器对事件进行筛选并推送给使用者。
生成器服务和使用者服务将会解耦，从而使它们能够独立扩展、更新和部署。

## EventBridge

EventBridge 是EDA的实现，提供了Event的过滤、转换和路由功能。

### Event

Event⽤来表⽰分布式系统中发⽣的状态变化。Event数据结构所遵循的schema可以通过 source + type 在Schema中索引。
Event包含以下字段：

- `id`: Event在Source中的唯一标识符。如果不指定，EventBridge会自动生成一个。
- `source`: Event的源头，通常是服务或应用程序的名称。
- `subject`: 可选，在Source内部的更细粒度的上下⽂标识。
- `type`: Event的类型，用于区分不同类型的Event。
- `time`: 事件的发⽣时间（⾮EventBridge接收时间），由发送端设置。
- `data`: 与特定领域相关的场景数据，遵循由source + type 确定的Schema.
- `datacontenttype`: 仅支持 `application/json`，表示data的内容类型。

向EventBridge发送Event时，可以指定 `pub_time` 和 `retry_strategy`，
`pub_time`决定Event何时发送到Target，`retry_strategy`决定发送失败后采取什么样的重试策略。

### Source

Event的源头，通常是一个服务或应用程序，它会生成Event并将其发送到EventBridge。

### Schema

Schema定义了Event的结构和格式，使用JSON Schema来描述。
想要将Event发送到EventBridge，必须先注册Schema。
Schema包含以下字段：

- `source`: Event的源头，通常是服务或应用程序的名称。
- `type`: Event的类型，用于区分不同类型的Event。
- `bus_name`: Event所属的Bus名称，EventBridge会根据Bus名称将Event路由到对应的Bus。
- `spec`: 序列化的JSON Schema，用于描述Event的结构。
- `version`: Schema的版本号，每次Schema变更时，版本号会递增。

### Bus

⽤来存储和传输事件的中转站，Bus和Bus的资源完全隔离，
可以存在多个Bus，EventBridge⾃带 Default Bus。
Bus有并发和有序两种模式，如果是有序模式，Event会按照发送顺序进行处理。
Bus被移除时，和Bus相关的Schema和Rule也会被删除。

### Rule

Rule用于对Event进行过滤、转换和路由。

#### Pattern

Pattern是Rule的⼦概念，⽤来过滤符合特定条件的事件。
Pattern的整体结构与Event相同，区别在于Event描述内容，⽽Pattern描述的是匹配规则。

#### Target

Target是Rule的⼦概念，包含Event的转换规则和发送目标。
下面是Target的示例：

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

上面定义了一个使用HTTP POST请求，将`{"code":"10188:{$.data.a}"}`发送到`http://127.0.0.1:10188/target/event`
的Target。`Params`定义了将Event转换成`HTTPDispatcher`参数的细节，`RetryStrategy`的指定则会覆盖Event中`RetryStrategy`。

##### Dispatcher

Dispatcher是Target的类型，负责将Event发送到指定的目标。目前EventBridge支持以下Dispatcher类型：

- `HTTPDispatcher`: 通过HTTP请求发送Event。
- `gRPCDispatcher`: 通过gRPC请求发送Event。

你可以使用接口`rpc ListDispatcherSchema`来获取所有Dispatcher的DispatcherSchema。
下面是`HTTPDispatcher`的DispatcherSchema:

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

上面的DispatcherSchema描述了`HTTPDispatcher`的参数结构，
包含`method`、`url`、`header`和`body`四个字段， 其中`method`、`url`是必选字段。
