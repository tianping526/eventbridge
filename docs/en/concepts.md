# Concepts

## EDA - Event-Driven Architecture

Event-Driven Architecture (EDA) consists of three main components: 
event generators, event routers, and event consumers.
Event generators publish events to routers, which filter and push events to consumers.
The generator and consumer services are decoupled, allowing them to scale, update, and deploy independently.

## EventBridge

EventBridge is an implementation of EDA, providing event matching, transformation, and dispatching capabilities.

### Event

Event is used to represent state changes in a distributed system.
The Event data structure follows a schema that can be indexed in the Schema by its `source` and `type`.
Event contains the following fields:

- `id`: A unique identifier for the Event within its Source. If not specified, 
  EventBridge will automatically generate one.
- `source`: The source of the Event, typically the name of a service or application.
- `subject`: Optional, a finer-grained context identifier within the Source.
- `type`: The type of the Event, used to distinguish between different types of Events.
- `time`: The time when the event occurred (not when EventBridge received it), set by the sender.
- `data`: Domain-specific scenario data that follows the schema defined by `source` and `type`.
- `datacontenttype`: Only supports `application/json`, indicating the content type of the data.

When sending an Event to EventBridge, you can specify `pub_time` and `retry_strategy`.
`pub_time` determines when the Event is sent to the Target, 
and `retry_strategy` determines the retry policy in case of failure.

### Source

The source of an Event, typically a service or application that generates the Event and sends it to EventBridge.

### Schema

Schema defines the structure of an Event, described using JSON Schema.
To send an Event to EventBridge, you must first register the Schema.
Schema contains the following fields:

- `source`: The source of the Event, typically the name of a service or application.
- `type`: The type of the Event, used to distinguish between different types of Events.
- `bus_name`: The name of the Bus to which the Event belongs; EventBridge routes the Event based on this name.
- `spec`: The serialized JSON Schema that describes the structure of the Event.
- `version`: The version number of the Schema, which increments each time the Schema changes.

### Bus

Bus is a transit station for storing and transmitting events, 
with complete isolation of resources between different Buses.
Multiple Buses can exist, and EventBridge comes with a Default Bus.
Bus can work in either concurrent or ordered mode; in ordered mode, Events are processed in the order they are sent.
When a Bus is removed, all Schemas and Rules associated with that Bus are also deleted.

### Rule

Rule is used to match, transform, and dispatch Events.

#### Pattern

Pattern is a sub-concept of Rule, used to match events that meet specific conditions.
The overall structure of a Pattern is the same as that of an Event,
but while an Event describes content, a Pattern describes matching rules.

#### Target

Target is a sub-concept of Rule, defines how to transform an Event and where to send it.

Example:

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

Below defines a Target that uses an HTTP POST request to send `{"code":"10188:{$.data.a}"}` to `http://127.0.0.1:10188/target/event`
The `Params` define the details of transforming the Event into `HTTPDispatcher` parameters,
and the specified `RetryStrategy` will override the `RetryStrategy` in the Event.

##### Dispatcher

Dispatcher is the type of Target, responsible for dispatching Events to the specified destination.
Currently, EventBridge supports the following Dispatcher types:

- `HTTPDispatcher`: Dispatches Events via HTTP requests.
- `gRPCDispatcher`: Dispatches Events via gRPC requests.

You can use the `rpc ListDispatcherSchema` API to get the DispatcherSchema of all Dispatchers.
Here is the DispatcherSchema for `HTTPDispatcher`:

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

Below is the description of the `HTTPDispatcher` parameters structure in the DispatcherSchema,
which includes four fields: `method`, `url`, `header`, and `body` 
where `method` and `url` are required fields.
