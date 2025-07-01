# Entity Relationship Diagram (ERD)

<p style="text-align: center;">
  <img src="img/eb-erd.svg" alt="eb-erd.svg" />
</p>

## Schema

`source` + `type` indicates a unique Schema,
where `spec` is the serialized JSON Schema that describes the structure of an Event.
`bus_name` is the name of the Bus to which the Schema-validated Event will be sent.
`version` is the version number of the Schema, which increments each time the Schema changes.

## Bus

`name` is the name of the Bus, used to uniquely identify a Bus.
`mode` defines the working mode of the Bus, whether it sends Events concurrently or in order.
`source_topic`, `source_delay_topic`, `target_exp_decay_topic`, and `target_backoff_topic`
define the MQ Topics used to store Events at different stages.

## Rule

`name` is the name of the Rule, used to uniquely identify a Rule.
`bus_name` is the name of the Bus associated with the Rule, specifying the Bus to which the Rule applies.
`status` can enable or disable the Rule.
`pattern` defines the matching pattern of the Rule, used to filter Events from the Bus.
`target` defines how the matched Events should be transformed and dispatched.

## Version

Version for Bus and Rule. Each Bus and Rule has a fixed `id` in the Version table,
whenever a Bus or Rule changes, the corresponding `version` for that `id` increments.
With Version, Jobs can efficiently track changes to Bus and Rule, and apply the latest changes to the Job.
