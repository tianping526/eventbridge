# 快速开始

## 部署

<details>
<summary><span style="font-size:1.5em; font-weight:bold;">使用 Docker 部署</span></summary>

#### 启动 Postgres，Redis 和 RocketMQ

准备 `docker-compose.yaml` 文件：

```yaml
services:
  db:
    image: postgres
    environment:
      POSTGRES_PASSWORD: example
    depends_on:
      - redis
    networks:
      - eventbridge

  redis:
    image: redis
    networks:
      - eventbridge

  mq-namesrv:
    restart: always
    image: apache/rocketmq:5.3.1
    environment:
      - JAVA_OPT_EXT=-server -Xms256m -Xmx256m -Xmn128m
    command: sh mqnamesrv
    networks:
      - eventbridge

  mq-broker:
    restart: always
    image: apache/rocketmq:5.3.1
    depends_on:
      - mq-namesrv
    environment:
      - NAMESRV_ADDR=mq-namesrv:9876
      - JAVA_OPT_EXT=-server -Xms512m -Xmx512m -Xmn256m
    command: sh mqbroker
    networks:
      - eventbridge

  mq-proxy:
    restart: always
    image: apache/rocketmq:5.3.1
    depends_on:
      - mq-namesrv
      - mq-broker
    environment:
      - NAMESRV_ADDR=mq-namesrv:9876
      - JAVA_OPT_EXT=-server -Xms256m -Xmx256m -Xmn128m
    command: sh mqproxy
    networks:
      - eventbridge

  create-default-data-bus:
    restart: on-failure
    image: apache/rocketmq:5.3.1
    depends_on:
      - mq-namesrv
      - mq-broker
      - mq-proxy
    networks:
      - eventbridge
    command:
      - sh
      - -c
      - |
        set -e

        # Create Default data bus
        until ./mqadmin updateTopic -n mq-namesrv:9876 -t EBInterBusDefault -c DefaultCluster -r 8 -w 8 | tee /dev/stderr | grep success; do
        echo "Retrying updateTopic for EBInterBusDefault..."
        sleep 1
        done

        ./mqadmin updateTopic -n mq-namesrv:9876 -t EBInterDelayBusDefault -c DefaultCluster -r 8 -w 8 -a +message.type=DELAY | tee /dev/stderr | grep success
        ./mqadmin updateTopic -n mq-namesrv:9876 -t EBInterTargetExpDecayBusDefault -c DefaultCluster -r 8 -w 8 | tee /dev/stderr | grep success
        ./mqadmin updateTopic -n mq-namesrv:9876 -t EBInterTargetBackoffBusDefault -c DefaultCluster -r 8 -w 8 | tee /dev/stderr | grep success

        ./mqadmin updateSubGroup -n mq-namesrv:9876 -c DefaultCluster -g DefaultSource -r 3 | tee /dev/stderr | grep success
        ./mqadmin updateSubGroup -n mq-namesrv:9876 -c DefaultCluster -g DefaultSourceDelay -r 3 | tee /dev/stderr | grep success
        ./mqadmin updateSubGroup -n mq-namesrv:9876 -c DefaultCluster -g DefaultTargetExpDecay -r 176 | tee /dev/stderr | grep success
        ./mqadmin updateSubGroup -n mq-namesrv:9876 -c DefaultCluster -g DefaultTargetBackoff -r 3 | tee /dev/stderr | grep success

networks:
  eventbridge:
    name: eventbridge
    driver: bridge
```

有一些重要的信息需要关注：

- `db`：Postgres 数据库服务，使用密码 `example`，端口 5432。
- `redis`：Redis 服务，端口 6379。
- `mq-proxy`：RocketMQ Proxy 服务，端口 8081。
- `create-default-data-bus`：创建 Default Bus 的 Topic 并配置订阅组。
    - 为 Default Bus 创建了四个 Topic：
        - `EBInterBusDefault`：用于接收实时 Event 的 Topic。
        - `EBInterDelayBusDefault`：用于接收延迟 Event 的 Topic，额外添加了 `message.type=DELAY` 的属性。
        - `EBInterTargetExpDecayBusDefault`：用于存放需要进行指数衰减策略重试的 Event 的 Topic。
        - `EBInterTargetBackoffBusDefault`：用于存放需要进行退避策略重试的 Event 的 Topic。
    - 为每个 Topic 创建了对应的订阅组：
        -
        订阅组的名字是[{host}{port}{topic}](https://github.com/tianping526/eventbridge/blob/main/app/job/internal/data/rocketmq.go#L101)
        的格式。
        - `EBInterTargetExpDecayBusDefault` 订阅组的重试次数设置为 176 次，如果设置错误，Job 将无法正确处理指数衰减策略的重试。
        - `EBInterTargetBackoffBusDefault` 订阅组的重试次数设置为 3 次，如果设置错误，Job 将无法正确处理退避策略的重试。
        - 其他订阅组的重试次数设置为 3 次，代表 Event 在 Job 内部流转失败时的重试次数。

启动Docker Compose：

> 确保当前目录下有 docker-compose.yaml 文件。

```bash
docker-compose -f docker-compose.yaml up -d
```

查看服务状态：

```bash
docker-compose -f docker-compose.yaml ps -a
```

    NAME                                    IMAGE                   COMMAND                  SERVICE                   CREATED          STATUS                     PORTS
    eventbridge-create-default-data-bus-1   apache/rocketmq:5.3.1   "./docker-entrypoint…"   create-default-data-bus   43 seconds ago   Exited (0) 3 seconds ago   
    eventbridge-db-1                        postgres                "docker-entrypoint.s…"   db                        43 seconds ago   Up 43 seconds              5432/tcp
    eventbridge-mq-broker-1                 apache/rocketmq:5.3.1   "./docker-entrypoint…"   mq-broker                 44 seconds ago   Up 43 seconds              9876/tcp, 10909/tcp, 10911-10912/tcp
    eventbridge-mq-namesrv-1                apache/rocketmq:5.3.1   "./docker-entrypoint…"   mq-namesrv                44 seconds ago   Up 43 seconds              9876/tcp, 10909/tcp, 10911-10912/tcp
    eventbridge-mq-proxy-1                  apache/rocketmq:5.3.1   "./docker-entrypoint…"   mq-proxy                  43 seconds ago   Up 31 seconds              9876/tcp, 10909/tcp, 10911-10912/tcp
    eventbridge-redis-1                     redis                   "docker-entrypoint.s…"   redis                     44 seconds ago   Up 43 seconds              6379/tcp

`eventbridge-create-default-data-bus-1` 状态为 `Exited (0)` 表示创建 Default Bus 的 Topic 和配置订阅组成功。

#### 启动 Service

> 确保当前目录下有 `service.yaml` 文件。

```bash
docker run -d --network eventbridge -p 8011:8011 -p 9011:9011 -v $(pwd)/service.yaml:/data/conf/service.yaml linktin/eb-service:1.0.0
```

下面是 `service.yaml` 的内容，你还可以查看 Service 的 [配置文件示例](../../app/service/configs/service.yaml)
和 [schema](../../app/service/internal/conf/conf.proto)。

```yaml
bootstrap:
  server:
    http:
      addr: 0.0.0.0:8011 # 监听 HTTP 请求的端口
      timeout: 1s
    grpc:
      addr: 0.0.0.0:9011 # 监听 gRPC 请求的端口
      timeout: 1s
  data:
    database:
      driver: postgres
      source: postgresql://postgres:example@db:5432/postgres # Postgres 数据库连接字符串
      max_open: 100
      max_idle: 10
      conn_max_life_time: 0s
      conn_max_idle_time: 300s
    redis:
      addrs:
        - redis:6379 # Redis 服务地址
      password:
      db_index: 0
      dial_timeout: 1s
      read_timeout: 0.2s
      write_timeout: 0.2s
    default_mq: rocketmq://mq-proxy:8081 # RocketMQ Proxy 服务地址
```

查看服务状态：

```bash
docker ps -a
```

    CONTAINER ID   IMAGE                      COMMAND                  CREATED         STATUS                     PORTS                                            NAMES
    0cfa5a79afb8   linktin/eb-service:1.0.0   "./server -conf /dat…"   5 seconds ago   Up 4 seconds               0.0.0.0:8011->8011/tcp, 0.0.0.0:9011->9011/tcp   sweet_yalow

Service 状态为 `Up` 表示启动成功。

#### 启动 Job

> 确保当前目录下有 `job.yaml` 文件。

```bash
docker run -d --network eventbridge -v $(pwd)/job.yaml:/data/conf/job.yaml linktin/eb-job:1.0.0
```

下面是 `job.yaml` 的内容，你还可以查看 Job 的 [配置文件示例](../../app/job/configs/service.yaml)
和 [schema](../../app/job/internal/conf/conf.proto)。

```yaml
bootstrap:
  server:
    http:
      addr: 0.0.0.0:8012 # Metrics HTTP 端口
      timeout: 1s
    event:
      source_timeout: 1s # 处理 source_topic 中 Event 的超时时间
      delay_timeout: 1s # 处理 source_delay_topic 中 Event 的超时时间
      target_exp_decay_timeout: 3s # 处理 target_exp_decay_topic 中 Event 的超时时间
      target_backoff_timeout: 3s # 处理 target_backoff_topic 中 Event 的超时时间
  data:
    database:
      driver: postgres
      source: postgresql://postgres:example@db:5432/postgres # Postgres 数据库连接字符串
      max_open: 100
      max_idle: 10
      conn_max_life_time: 0s
      conn_max_idle_time: 300s
    default_mq: rocketmq://mq-proxy:8081 # RocketMQ Proxy 服务地址
```

查看服务状态：

```bash
docker ps -a
```

    CONTAINER ID   IMAGE                      COMMAND                  CREATED          STATUS                      PORTS                                            NAMES
    b7c280bfde43   linktin/eb-job:1.0.0       "./server -conf /dat…"   5 seconds ago    Up 5 seconds                                                                 happy_hugle

Job 状态为 `Up` 表示启动成功。

</details>

<details>
<summary><span style="font-size:1.5em; font-weight:bold;">使用 Helm chart 部署</span></summary>

> 演示中使用的 Helm chart 启动了一个高可用的 EventBridge 集群，包括了 Service、Job、Postgres、Redis 和 RocketMQ。

#### 添加 Helm 仓库

```bash
helm repo add tianping526 https://tianping526.github.io/helm-charts
helm repo update
```

#### 安装 EventBridge

```bash
helm install eventbridge tianping526/eventbridge --namespace eventbridge --create-namespace
```

#### 查看服务状态

```bash
kubectl -n eventbridge get pod
```

    NAME                                READY   STATUS    RESTARTS        AGE
    eb-job-66f946b9f6-s9rz6             1/1     Running   3 (4m3s ago)    4m33s
    eb-job-66f946b9f6-t24gv             1/1     Running   3 (4m6s ago)    4m33s
    eb-job-66f946b9f6-vz8wf             1/1     Running   3 (3m51s ago)   4m33s
    eb-pg-ha-pgpool-58959774c7-42sgk    1/1     Running   0               4m33s
    eb-pg-ha-pgpool-58959774c7-lgb9g    1/1     Running   0               4m33s
    eb-pg-ha-postgresql-0               1/1     Running   0               4m32s
    eb-pg-ha-postgresql-1               1/1     Running   0               4m32s
    eb-pg-ha-postgresql-2               1/1     Running   0               4m32s
    eb-redis-cluster-0                  1/1     Running   0               4m32s
    eb-redis-cluster-1                  1/1     Running   0               4m31s
    eb-redis-cluster-2                  1/1     Running   0               4m31s
    eb-redis-cluster-3                  1/1     Running   0               4m30s
    eb-redis-cluster-4                  1/1     Running   0               4m31s
    eb-redis-cluster-5                  1/1     Running   0               4m31s
    eb-rmq-broker-master-0              1/1     Running   0               4m32s
    eb-rmq-broker-master-1              1/1     Running   0               2m52s
    eb-rmq-broker-replica-id1-0         1/1     Running   0               4m31s
    eb-rmq-broker-replica-id1-1         1/1     Running   0               2m50s
    eb-rmq-broker-replica-id2-0         1/1     Running   0               4m31s
    eb-rmq-broker-replica-id2-1         1/1     Running   0               3m31s
    eb-rmq-controller-0                 1/1     Running   0               4m32s
    eb-rmq-controller-1                 1/1     Running   0               4m32s
    eb-rmq-controller-2                 1/1     Running   0               4m32s
    eb-rmq-dashboard-6bcbb4dd4b-jwp8n   1/1     Running   0               4m33s
    eb-rmq-nameserver-0                 1/1     Running   0               4m33s
    eb-rmq-nameserver-1                 1/1     Running   0               4m32s
    eb-rmq-nameserver-2                 1/1     Running   0               4m32s
    eb-rmq-proxy-bcd8968-2mfq4          1/1     Running   4 (3m28s ago)   4m33s
    eb-rmq-proxy-bcd8968-2vjt6          1/1     Running   4 (3m30s ago)   4m33s
    eb-rmq-proxy-bcd8968-dtmx2          1/1     Running   3 (3m32s ago)   4m33s
    eb-service-56cd698777-cbb5q         1/1     Running   2 (4m9s ago)    4m33s
    eb-service-56cd698777-qqfs2         1/1     Running   3 (3m50s ago)   4m18s
    eb-service-56cd698777-sdmjr         1/1     Running   3 (3m54s ago)   4m18s

所有服务都处于 `Running` 状态，表示启动成功。你可能观察到部分 Pod 的 `RESTARTS` 数量大于 0，
这是因为它们依赖的服务还未就绪，导致它们重启了几次，但只要最终状态是 `Running` 即可。

</details>

## 创建 Bus

### 创建 Default Bus

创建名为 Default 的 Bus 会失败，因为 EventBridge 启动时会自动创建一个名为 Default 的 Bus。
你可以查看 [HTTP Create Bus](https://github.com/tianping526/apis/blob/main/openapi.yaml#L10)
和 [gRPC Create Bus](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L47)
的定义，了解创建 Bus 的请求格式。

```bash
curl --location '127.0.0.1:8011/v1/eventbridge/bus' \
--header 'Content-Type: application/json' \
--header 'Accept: application/json' \
--data '{
  "name": "Default",
  "mode": 1
}'

# {"code":400, "reason":"BUS_NAME_REPEAT", "message":"bus name repeat. name: Default", "metadata":{}}
```

查看 Default Bus 的信息， 你可以查看 [HTTP List Buses](https://github.com/tianping526/apis/blob/main/openapi.yaml#L59)
和 [gRPC List Buses](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L43)
的定义，了解查询 Bus 的请求格式。

```bash

```Bash
curl --location '127.0.0.1:8011/v1/eventbridge/buses?prefix=Default' \
--header 'Accept: application/json'

# {"buses":[{"name":"Default", "mode":"BUS_WORK_MODE_CONCURRENTLY", "sourceTopic":"EBInterBusDefault", "sourceDelayTopic":"EBInterDelayBusDefault", "targetExpDecayTopic":"EBInterTargetExpDecayBusDefault", "targetBackoffTopic":"EBInterTargetBackoffBusDefault"}], "nextToken":"0"}
```

Default Bus 的 `source_topic`、`source_delay_topic`、`target_exp_decay_topic` 和 `target_backoff_topic`
分别是 `EBInterBusDefault`、`EBInterDelayBusDefault`、`EBInterTargetExpDecayBusDefault`
和 `EBInterTargetBackoffBusDefault`。
这四个 Topic 的作用，你可以查看[架构](architecture.md#job)和[实体关系图](erd.md#bus)以了解更多。

### 创建 Orderly Bus

我们可以创建其他 Bus，例如 Orderly，模式为顺序处理（`BUS_WORK_MODE_ORDERLY`）。
你可以查看 [HTTP Create Bus](https://github.com/tianping526/apis/blob/main/openapi.yaml#L10)
和 [gRPC Create Bus](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L47)
的定义，了解创建 Bus 的请求格式。

```bash
curl --location '127.0.0.1:8011/v1/eventbridge/bus' \
--header 'Content-Type: application/json' \
--header 'Accept: application/json' \
--data '{
  "name": "Orderly",
  "mode": 2
}'

# {"id":"28866794466836484"}
```

如果 Bus 的工作模式是顺序处理（`BUS_WORK_MODE_ORDERLY`），则需要为其创建顺序处理的 Topic。
如下所示， `EBInterBusOrderly`、`EBInterTargetExpDecayBusOrderly`
和 `EBInterTargetBackoffBusOrderly` 额外添加了 `-a +message.type=FIFO -o true` 的参数，
它们的消费组额外添加了 `-o true` 的参数。
如果不添加这些参数，RocketMQ 会将这些 Topic 视为普通 Topic，Event 的处理将不再是顺序的。

```bash
./mqadmin updateTopic -n mq-namesrv:9876 -t EBInterBusOrderly -c DefaultCluster -r 8 -w 8 -a +message.type=FIFO -o true | tee /dev/stderr | grep success
./mqadmin updateTopic -n mq-namesrv:9876 -t EBInterDelayBusOrderly -c DefaultCluster -r 8 -w 8 -a +message.type=DELAY | tee /dev/stderr | grep success        
./mqadmin updateTopic -n mq-namesrv:9876 -t EBInterTargetExpDecayBusOrderly -c DefaultCluster -r 8 -w 8 -a +message.type=FIFO -o true | tee /dev/stderr | grep success
./mqadmin updateTopic -n mq-namesrv:9876 -t EBInterTargetBackoffBusOrderly -c DefaultCluster -r 8 -w 8 -a +message.type=FIFO -o true | tee /dev/stderr | grep success

./mqadmin updateSubGroup -n mq-namesrv:9876 -c DefaultCluster -g OrderlySource -r 3 -o true | tee /dev/stderr | grep success                          
./mqadmin updateSubGroup -n mq-namesrv:9876 -c DefaultCluster -g OrderlySourceDelay -r 3 | tee /dev/stderr | grep success                     
./mqadmin updateSubGroup -n mq-namesrv:9876 -c DefaultCluster -g OrderlyTargetExpDecay -r 176 -o true | tee /dev/stderr | grep success          
./mqadmin updateSubGroup -n mq-namesrv:9876 -c DefaultCluster -g OrderlyTargetBackoff -r 3 -o true | tee /dev/stderr | grep success
```

## 创建 Schema

创建 Schema 的请求格式，你可以查看 [HTTP Create Schema](https://github.com/tianping526/apis/blob/main/openapi.yaml#L280)
和 [gRPC Create Schema](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L24)。

```bash
curl --location '127.0.0.1:8011/v1/eventbridge/schema' \
--header 'Content-Type: application/json' \
--header 'Accept: application/json' \
--data '{
  "source": "testSource1",
  "type": "testSourceType1",
  "busName": "Default",
  "spec": "{\"$schema\":\"https://json-schema.org/draft/2020-12/schema\",\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\"}}}"
}'
```

这个请求会为 Source `testSource1` 的 `testSourceType1` 类型的 Event 创建一个 Schema。
这种 Event 有一个 `string` 类型的属性 `a`， 要发往 Default Bus。

## 创建 Rule

创建 Rule 的请求格式，你可以查看 [HTTP Create Rule](https://github.com/tianping526/apis/blob/main/openapi.yaml#L152)
和 [gRPC Create Rule](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L63)。

```bash
curl --location '127.0.0.1:8011/v1/eventbridge/rule' \
--header 'Content-Type: application/json' \
--header 'Accept: application/json' \
--data '{
  "name": "TestRule",
  "busName": "Default",
  "status": "RULE_STATUS_ENABLE",
  "pattern": "{\"source\":[{\"prefix\":\"testSource1\"}]}",
  "targets": [
    {
      "id": 1,
      "type": "HTTPDispatcher",
      "params": [
        {
          "key": "url",
          "form": "CONSTANT",
          "value": "http://192.168.30.143:10188/target/event"
        },
        {
          "key": "method",
          "form": "CONSTANT",
          "value": "POST"
        },
        {
          "key": "body",
          "form": "TEMPLATE",
          "value": "{\"subject\":\"$.data.a\"}",
          "template": "{\"code\":\"10188:${subject}\"}"
        }
      ],
      "retryStrategy": 1
    }
  ]
}'
```

我们在 Default Bus 上创建了一个名为 `TestRule` 的 Rule。这个 Rule 会匹配 `source` 前缀为 `testSource1` 的 Event，并将
Event 通过 HTTP POST 方法发送到 `http://192.168.30.143:10188/target/event` ，HTTP Body 的内容是
`{"code":"10188:{$.data.a}"}`。

为了能顺利接收 `TestRule` 发送的 Event，我们需要在 `192.168.30.143` 这台主机上启动下面这个 HTTP 服务。

```go
package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/target/event", func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		body, _ := io.ReadAll(request.Body)
		fmt.Println("Received event:", string(body))
	})
	server := &http.Server{
		Addr:    ":10188",
		Handler: mux,
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Error listening on port 10188:", err)
	}
}
```

## 发送 Event

发送 Event 的请求格式，你可以查看 [HTTP Post Event](https://github.com/tianping526/apis/blob/main/openapi.yaml#L127)
和 [gRPC Post Event](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L12)。

```bash
curl --location '127.0.0.1:8011/v1/eventbridge/event' \
--header 'Content-Type: application/json' \
--header 'Accept: application/json' \
--data '{
    "event": {
        "id": "1",
        "source": "testSource1",
        "type": "testSourceType1",
        "time": "2025-06-22T05:28:28.974Z",
        "data": "{\"a\": \"i am test content\"}",
        "datacontenttype": "application/json"
    },
    "retryStrategy": "RETRY_STRATEGY_BACKOFF"
}'
```

主机 `192.168.30.143` 上运行的 HTTP 服务会接收到如下内容：

    Received event: {"code":"10188:i am test content"}
