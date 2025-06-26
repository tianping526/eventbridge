# 实体关系图

<p style="text-align: center;">
  <img src="img/eb-erd.svg" alt="eb-erd.svg" />
</p>

## Schema

`source` + `type` 唯一标识一个 Schema，`spec` 是序列化的 JSON Schema，用于描述 Event 的结构。
`bus_name` 是 Bus 的名称，Schema 验证后的 Event 会被发送到对应的 Bus。`version` 是 Schema 的版本号，
每次 Schema 变更时，版本号会递增。

## Bus

`name` 是 Bus 的名称，用于唯一标识一个 Bus。`mode` 定义了 Bus 的工作模式，是并发还是有序发送 Event。
`source_topic`、`source_delay_topic`、`target_exp_decay_topic` 和 `target_backoff_topic`
定义了用于存储不同阶段 Event 的 MQ Topic。

## Rule

`name` 是 Rule 的名称，用于唯一标识一个 Rule。`bus_name` 是 Rule 所关联的 Bus 名称，指定 Rule 作用的 Bus。
`status` 可以将 Rule 标记为启用或禁用。`pattern` 定义了 Rule 的匹配模式，用来从 Bus 中筛选 Event。
`target` 定义了 Rule 匹配到的 Event 应该如何进行转换和发送。

## Version

Bus 和 Rule 的版本信息。Bus 和 Rule 在 Version 表中有个固定的 `id`，每当 Bus 或 Rule 发生变更时，
对应 id 的 `version` 会递增。有了 Version，Job 可以高效地跟踪 Bus 和 Rule 的变更，然后将最新的变更应用到 Job 中。
