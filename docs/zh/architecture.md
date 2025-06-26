# 架构

## Service

Service 是个 API 服务，支持 gRPC 和 HTTP 两种协议。对 EventBridge 的所有操作都通过 Service 进行。

<p style="text-align: center;">
  <img src="img/service-arch.svg" alt="service-arch.svg" />
</p>

Schema、Bus、Rule 和 Version 等实体都存储在 Postgres 数据库中，Event 数据存储在消息队列中，Redis 则会用于缓存 Schema。

## Job

Job 是处理 Event 的工作组件，负责从消息队列中消费 Event，应用 Rule 进行过滤、转换和发送。

<p style="text-align: center;">
  <img src="img/job-arch.svg" alt="img/job-arch.svg" />
</p>
