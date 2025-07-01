# EventBridge

[简体中文](README_zh.md)

A Go implementation of the EventBridge, providing both gRPC and HTTP APIs.

EventBridge is a service that allows you to build event-driven architectures
by connecting different services and applications through events.

![eventbridge.svg](docs/en/img/eventbridge.svg)

EventBridge supports the following features:

- **Schema Management**: Define and manage schemas for events.
- **Event Buses**: Create event buses to route events between sources and targets.
- **Rules**: Define rules to filter, transform and deliver events to targets.

## Documentation

- [Quick Start](docs/en/quick-start.md)
- [Concepts](docs/en/concepts.md)
- [Architecture](docs/en/architecture.md)
- [Event Flowchart](docs/en/event-flow.md)
- [Event Rules](docs/en/rule.md)
- [Entity Relationship Diagram](docs/en/erd.md)
- [gRPC](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto)
  and [HTTP](https://github.com/tianping526/apis/blob/main/openapi.yaml)
  API documentation