# Architecture

## Service

Service is an API service that supports both gRPC and HTTP protocols. 
All requests to EventBridge are sent to Service.

<p style="text-align: center;">
  <img src="img/service-arch.svg" alt="service-arch.svg" />
</p>

Entities such as Schema, Bus, Rule, and Version are stored in the Postgres, 
Event is stored in message queues, and Redis is used to cache Schema.

## Job

Job is a component that consumes events from a message queue 
and applies rules to match, transform and dispatch events.

<p style="text-align: center;">
  <img src="img/job-arch.svg" alt="img/job-arch.svg" />
</p>
