# 架构

## Service

Service 是个API服务，支持 gRPC 和 HTTP 两种协议。对EventBridge的所有操作都通过Service进行。

<p style="text-align: center;">
  <img src="img/service-arch.svg" alt="service-arch.svg" />
</p>

Schema、Bus、Rule和Version等实体都存储在Postgres数据库中，Event数据存储在消息队列中，Redis则会用于缓存Schema。

## Job

Job 是处理Event的工作组件，负责从消息队列中消费Event，应用Rule进行过滤、转换和发送。

<p style="text-align: center;">
  <img src="img/job-arch.svg" alt="img/job-arch.svg" />
</p>

