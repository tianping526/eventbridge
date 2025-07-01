# Rule

## Pattern

Pattern 是 Rule 的子概念，用于匹配 Event。

### 注意事项

- Pattern 是逐个字符精确匹配的，需注意大小写，匹配过程中不会对字符串进行任何处理。
- 要匹配的值遵循 JSON 规则：字符串带引号，数字、`true`、`false` 和 `null` 不带引号。
- Pattern 对 Event 多个特定字段进行匹配，待匹配的字段之间是逻辑"与"，
  即所有参与匹配的字段都必须匹配成功，才能认为 Pattern 匹配成功。

> `data.name != test` 且 `data.scope != 100` 才能匹配成功。

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

- Pattern 对 Event 特定字段进行匹配，同一个字段的多个匹配规则之间是逻辑"或"，
  即只要有一个匹配成功，就认为 Pattern 匹配成功。

> `subject` 后缀是 `est` 或 `xxx`，都能匹配成功。

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

- Pattern 对 Event 特定字段进行匹配，同一个字段的多个匹配规则之间，也可以是逻辑"与"，
  即所有匹配规则都必须匹配成功，才能认为 Pattern 匹配成功。

> `source` 前缀是 `aa` 且后缀是 `bb`，才能匹配成功。
> 注意 `source` 字段逻辑"与"和逻辑"或"的写法区别。

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

### 匹配规则

#### 前缀匹配

前缀匹配规则用于匹配字符串的前缀部分。

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

上述示例中，匹配Event的`source`字段前缀为`testSource1`，匹配成功。

#### 后缀匹配

后缀匹配规则用于匹配字符串的后缀部分。

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

上述示例中，匹配Event的`subject`字段后缀为`est`或`xxx`，匹配成功。

#### 除外匹配

除外匹配规则用于过滤特定的值或对其他匹配规则取“非”。

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

上述示例中，匹配Event的`data.name`字段不等于`test`，且`data.scope`字段不等于`100`，匹配失败。

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

上述示例中，匹配Event的`data.name`字段不等于`test`和`test1`，匹配失败。

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

上述示例中，匹配Event的`data.name`字段前缀不等于`tes`，匹配失败。

#### 存在匹配

存在匹配规则用于检查Event中是否存在特定的字段。

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

上述示例中，匹配Event的`data.name`字段存在，匹配成功。

#### IP地址匹配

IP地址匹配规则用于匹配Event中的IP地址字段。

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

上述示例中，匹配Event的`data.source-ip`字段在CIDR范围`10.0.0.0/24`内，匹配成功。

#### 精确匹配

精确匹配规则用于匹配Event中字段的具体值。

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
```

等价写法:

```json
{
  "source": "testSource1"
}
```

</td>
</tr>
</table>

上述示例中，匹配Event的`source`字段等于`testSource1`，匹配成功。

#### 数值匹配

数值匹配规则用于匹配Event中数值类型的字段。

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

上述示例中，匹配同时满足`0 < data.count1 <= 5`，`data.count2 < 10`和`data.count3 = 301.8`的Event，匹配成功。

#### 数组匹配

数组匹配规则使用多个值来匹配Event中的字段。
Event的字段分两种情况：

- 如果Event字段是数组，只要数组匹配规则的值和字段的值的交集不为空，则匹配成功；
- 如果Event字段不是数组，只要字段的值在数组匹配规则的值中，则匹配成功。

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

上述示例中，匹配Event的`source`字段等于`testSource1`、`testSource2`或`testSource3`，匹配成功。

#### 空值匹配

空值匹配规则用于匹配Event中字段的值为`""`（空字符串）或`null`。

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

上述示例中，匹配Event的`data.value1`字段为空字符串，且`data.value2`字段为`null`，匹配成功。

#### 多模式匹配

多模式匹配规则允许在一个Pattern中使用多种匹配方式。

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

上述示例中，匹配Event的`source`字段前缀为`testSource1`，
且`data.source-ip`字段在CIDR范围`10.0.0.0/24`内，
且`data.name`字段不等于`test`，匹配成功。

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

上述示例中，匹配Event的`source`字段前缀为`aa`且后缀为`bb`，或者前缀为`cc`且后缀为`dd`，匹配成功。
`{}`表示没有匹配规则，匹配失败。

## Transform

Transform 是 Rule 的子概念，用于转换 Event。

### 转换规则

#### 完整事件

完整事件转换规则用于将Event的所有字段直接传递到目标。

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

> target.params 为空时，表示将完整事件传递到目标。

```json
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

#### 部分事件

部分事件转换规则用于将Event的部分字段传递到目标。

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

#### 常量

常量转换规则用于将固定值传递到目标。

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

#### 模版

模版转换规则用于将Event的字段值模版化处理。

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

> 注意：⾃定义变量的声明和使⽤，都会⾃动删除前后空白字符，所以有⼀定的容错能⼒。

```json
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

上面是通过模版输出字符串，你也可以通过模版输出JSON对象。

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