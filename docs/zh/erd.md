# 实体关系图

<p style="text-align: center;">
  <img src="img/eb-erd.svg" alt="eb-erd.svg" />
</p>

## Schema

`source` + `type`唯一标识一个Schema，`spec`是序列化的JSON Schema，用于描述Event的结构。
`bus_name`是Bus的名称，Schema验证后的Event会被发送到对应的Bus。`version`是Schema的版本号,
每次Schema变更时，版本号会递增。

## Bus

`name`是Bus的名称，用于唯一标识一个Bus。`mode`定义了Bus的工作模式，是并发还是有序发送Event。
`source_topic`, `source_delay_topic`, `target_exp_decay_topic` 和 `target_backoff_topic`
定义了用于存储不同阶段Event的MQ Topic。

## Rule

`name`是Rule的名称，用于唯一标识一个Rule。`bus_name`是Rule所关联的Bus名称，指定Rule作用的Bus。
`status`可以将Rule标记为启用或禁用。`pattern`定义了Rule的匹配模式，用来从Bus中筛选Event。
`target`定义了Rule匹配到的Event应该如何进行转换和发送。

## Version

Bus和Rule的版本信息。Bus和Rule在Version表中有个固定的`id`，每当Bus或Rule发生变更时，
对应id的`version`会递增。有了Version，Job可以高效的跟踪Bus和Rule的变更，然后将最新的变更应用到Job中。
