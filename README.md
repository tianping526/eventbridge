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