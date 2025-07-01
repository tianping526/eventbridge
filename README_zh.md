# EventBridge

一个 Go 实现的 EventBridge，提供 gRPC 和 HTTP 两种协议的接口。

EventBridge 是一个允许您通过事件连接不同的服务和应用程序，从而构建事件驱动架构的服务。

![eventbridge.svg](docs/zh/img/eventbridge.svg)

EventBridge 支持以下功能：

- **Schema 管理**：定义和管理事件的 Schema。
- **事件总线**：创建事件总线以在源和目标之间路由事件。
- **事件规则**：定义规则以过滤、转换和发送事件到目标。

## 文档

- [快速开始](docs/zh/quick-start.md)
- [概念](docs/zh/concepts.md)
- [架构](docs/zh/architecture.md)
- [事件流程图](docs/zh/event-flow.md)
- [事件规则](docs/zh/rule.md)
- [实体关系图](docs/zh/erd.md)
- [gRPC](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto)
  和 [HTTP](https://github.com/tianping526/apis/blob/main/openapi.yaml)
  接口文档
