# Rule

## Pattern

Pattern is a sub-concept of Rule, used to match events that meet specific conditions.

### Notes

- Pattern is matched character by character, case-sensitive,
  and no processing is performed on the strings during matching.
- Values to be matched follow JSON rules: strings are quoted,
  numbers, `true`, `false`, and `null` are not quoted.
- Pattern matches multiple specific fields of an Event,
  where the fields to be matched are combined with "AND" logic,
  meaning all participating fields must match successfully for
  the Pattern to be considered a match.

> The match succeeds only if the `data.name` field is not equal to `test`
> and the `data.scope` field is not equal to 100.

```json
{
  "data": {
    "name": [
      {
        "anything-but": "test"
      }
    ],
    "scope": [
      {
        "anything-but": 100
      }
    ]
  }
}
```

- Pattern matches specific fields of an event.
  For multiple matching rules on the same field, a logical "OR" is applied,
  meaning if any one of the matching rules succeeds, the Pattern is considered a match.

> The match succeeds if the suffix of the `subject` field is either `est` or `xxx`.

```json
{
  "subject": [
    {
      "suffix": "est"
    },
    {
      "suffix": "xxx"
    }
  ]
}
```

- Pattern matches specific fields of an event.
  Multiple matching rules for the same field can also use "AND" logic,
  meaning all matching rules must succeed for the pattern to be considered a match.

> The match succeeds only if the `source` field starts with `aa` and ends with `bb`.
> Note the difference in writing between logical "AND" and "OR" in the `source` field.

```json
{
  "source": [
    {
      "prefix": "aa",
      "suffix": "bb"
    }
  ]
}
```

### Matching Rules

#### Prefix

Prefix is used to match the prefix part of a string.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "source": [
    {
      "prefix": "testSource1"
    }
  ]
}
```

</td>
</tr>
</table>

Above, the `source` field of the Event matches the prefix `testSource1`, matching succeeds.

#### Suffix

Suffix is used to match the suffix part of a string.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "subject": [
    {
      "suffix": "est"
    },
    {
      "suffix": "xxx"
    }
  ]
}
```

</td>
</tr>
</table>

Above, the `subject` field of the Event matches the suffix `est` or `xxx`, matching succeeds.

#### Anything-But

Anything-but is used to filter specific values or take "not" for other matching rules.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit xxx",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"tes\",\"scope\":100}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "data": {
    "name": [
      {
        "anything-but": "test"
      }
    ],
    "scope": [
      {
        "anything-but": 100
      }
    ]
  }
}
```

</td>
</tr>
</table>

Above, matching the Event's `data.name` field not equal to `test`
and `data.scope` field not equal to `100`, matching fails.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "data": {
    "name": [
      {
        "anything-but": [
          "test",
          "test1"
        ]
      }
    ]
  }
}
```

</td>
</tr>
</table>

Above, matching the Event's `data.name` field not equal to `test` and `test1`, matching fails.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "data": {
    "name": [
      {
        "anything-but": {
          "prefix": "tes"
        }
      }
    ]
  }
}
```

</td>
</tr>
</table>

Above, matching the Event's `data.name` field prefix not equal to `tes`, matching fails.

#### Exists

Exists is used to check if a specific field exists in the Event.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "data": {
    "name": [
      {
        "exists": true
      }
    ]
  }
}
```

</td>
</tr>
</table>

Above, matching the Event's `data.name` field exists, matching succeeds.

#### CIDR

CIDR is used to match IP addresses in the Event.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "data": {
    "source-ip": [
      {
        "cidr": "10.0.0.0/24"
      }
    ]
  }
}
```

</td>
</tr>
</table>

Above, matching the Event's `data.source-ip` field within the CIDR range `10.0.0.0/24`, matching succeeds.

#### Exact

Exact is used to match the exact value of a field in the Event.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"a\":\"i am test content ad\"}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "source": [
    "testSource1"
  ]
}
// eqivalent to
{
  "source": "testSource1"
}
```

</td>
</tr>
</table>

Above, matching the Event's `source` field exactly equal to `testSource1`, matching succeeds.

#### Numeric

Numeric is used to match numeric fields in the Event.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test\",\"scope\":100,\"count1\":3,\"count2\":8,\"count3\":301.8}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "data": {
    "count1": [
      {
        "numeric": [
          ">",
          0,
          "<=",
          5
        ]
      }
    ],
    "count2": [
      {
        "numeric": [
          "<",
          10
        ]
      }
    ],
    "count3": [
      {
        "numeric": [
          "=",
          301.8
        ]
      }
    ]
  }
}
```

</td>
</tr>
</table>

Above, matching `0 < data.count1 <= 5`, `data.count2 < 10` and `data.count3 = 301.8`, matching succeeds.

#### Array

Array uses multiple values to match fields in the Event.
There are two cases for Event fields:

- If the Event field is an array,
  the match succeeds as long as there is any overlap
  between the values in the array matching rule and the field's values.
- If the Event field is not an array,
  the match succeeds as long as the field's value is included in the values
  specified by the array matching rule.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "source": [
    "testSource1",
    "testSource2",
    "testSource3"
  ]
}
```

</td>
</tr>
</table>

Above, matching the Event's `source` field equal to
`testSource1`, `testSource2`, or `testSource3`, matching succeeds.

#### Empty

Empty is used to match fields in the Event that are empty string or `null`.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\",\"value1\":\"\",\"value2\":null}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "data": {
    "value1": [
      ""
    ],
    "value2": [
      null
    ]
  }
}
```

</td>
</tr>
</table>

Above, matching the Event's `data.value1` field as an empty string
and `data.value2` field as `null`, matching succeeds.

#### Composite

Composite is used to combine multiple matching rules for a field in the Event.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "source": [
    {
      "prefix": "testSource1"
    }
  ],
  "data": {
    "source-ip": [
      {
        "cidr": "10.0.0.0/24"
      }
    ],
    "name": [
      {
        "anything-but": "test"
      }
    ]
  }
}
```

</td>
</tr>
</table>

Above, matching the Event's `source` field prefix as `testSource1`,
the `data.source-ip` field within the CIDR range `10.0.0.0/24`
and the `data.name` field not equal to `test`, matching succeeds.

<table>
<tr>
<td>

```json
{
  "id": 123,
  "source": "aa-bb",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
{
  "source": [
    {
      "prefix": "aa",
      "suffix": "bb"
    },
    {
      "prefix": "cc",
      "suffix": "dd"
    },
    {}
  ]
}
```

</td>
</tr>
</table>

Above, matching the Event's `source` field with a prefix of `aa` and a suffix of `bb`,
or a prefix of `cc` and a suffix of `dd`, matching succeeds.
`{}` indicates no matching rules and will always fail to match.

## Transform

Transform is a sub-concept of Rule,
used to transform an Event into a different format or structure before sending it to the Target.

### Transform Rules

#### Full Event

Full Event transformation rule is used to pass all fields of the Event directly to the Target.

<table>
<tr>
<td>

```json
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
// Note: When target.params is empty, it means the Event is sent directly to the target.
[]
```

</td>
<td>

```json
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
```

</td>
</tr>
</table>

#### Partial Event

Partial Event transformation rule is used to pass only specific fields of the Event to the Target.

<table>
<tr>
<td>

```json
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ips\":[\"10.0.0.123\", \"10.0.0.124\"]}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
[
  {
    "key": "resKey",
    "form": "JSONPATH",
    "value": "$.data.name"
  },
  {
    "key": "resKey1",
    "form": "JSONPATH",
    "value": "$.data.source-ips"
  }
]
```

</td>
<td>

```json
{
  "resKey": "test1",
  "resKey1": [
    "10.0.0.123",
    "10.0.0.124"
  ]
}
```

</td>
</tr>
</table>

#### Constant

Constant transformation rule is used to pass constant values to the Target.

<table>
<tr>
<td>

```json
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
[
  {
    "key": "resKey",
    "form": "CONSTANT",
    "value": "constantValue"
  }
]
```

</td>
<td>

```json
{
  "resKey": "constantValue"
}
```

</td>
</tr>
</table>

#### Template

Template transformation rule is used to process the Event's field values using templates.

<table>
<tr>
<td>

```json
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
// Note: Custom variable declarations and usage will automatically remove leading and trailing whitespace
[
  {
    "key": "resKey",
    "form": "TEMPLATE",
    "value": "{\"name  \":\"$.data.name\",\"ip\":\"$.data.source-ip\"}",
    "template": "\"i am ${name}, my ip is ${  ip  }.\""
  }
]
```

</td>
<td>

```json
{
  "resKey": "i am test1, my ip is 10.0.0.123."
}
```

</td>
</tr>
</table>

Above is a simple example of using a template to output a string.
You can also use templates to output JSON objects.

<table>
<tr>
<td>

```json
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
[
  {
    "key": "resKey",
    "form": "TEMPLATE",
    "value": "{\"name  \":\"$.data.name\",\"ip\":\"$.data.source-ip\"}",
    "template": "{\"name\": \"${name}\", \"ip\": \"${  ip  }\"}"
  }
]
```

</td>
<td>

```json
{
  "resKey": {
    "name": "test1",
    "ip": "10.0.0.123"
  }
}
```

</td>
</tr>
<tr>
<td>

```json
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"source-ip\":\"10.0.0.123\"}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
[
  {
    "key": "resKey",
    "form": "TEMPLATE",
    "value": "{\"name  \":\"$.data.name\",\"ip\":\"$.data.source-ip\"}",
    "template": "{\"name\": \"${name}\", \"ips\": [\"${  ip  }\", \"10.251.11.1\"], \"cc\":\"bb\"}"
  }
]
```

</td>
<td>

```json
{
  "resKey": {
    "name": "test1",
    "ips": [
      "10.0.0.123",
      "10.251.11.1"
    ],
    "cc": "bb"
  }
}
```

</td>
</tr>
<tr>
<td>

```json
{
  "id": "123",
  "source": "testSource1",
  "subject": "dolor mollit reprehenderit velit est",
  "type": "testSourceType1",
  "time": "2020-08-17T16:04:46.149Z",
  "data": "{\"name\":\"test1\",\"ips\":[{\"host\":\"10.0.0.123\", \"port\":\"8080\"}]}",
  "datacontenttype": "application/json"
}
```

</td>
<td>

```json
[
  {
    "key": "resKey",
    "form": "TEMPLATE",
    "value": "{\"name  \":\"$.data.name\",\"ips\":\"$.data.ips\"}",
    "template": "{\"name\": \"${name}\", \"ips\": ${  ips  }}"
  }
]
```

</td>
<td>

```json
{
  "resKey": {
    "name": "test1",
    "ips": [
      {
        "host": "10.0.0.123",
        "port": "8080"
      }
    ]
  }
}
```

</td>
</tr>
</table>