# Quick Start

## Deployment

<details>
<summary><span style="font-size:1.5em; font-weight:bold;">Deploying with Docker</span></summary>

#### Starting Postgres, Redis and RocketMQ

Prepare the `docker-compose.yaml` file:

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

There is some important information to focus on:

- `db`: Postgres service, using password `example`, port 5432.
- `redis`: Redis service, port 6379.
- `mq-proxy`: RocketMQ Proxy service, port 8081.
- `create-default-data-bus`: Creates Topics and configures subscription groups for the Default Bus.
    - Four Topics are created for the Default Bus:
        - `EBInterBusDefault`: Topic for receiving direct Events.
        - `EBInterDelayBusDefault`: Topic for receiving delayed Events, with an additional property
          `message.type=DELAY`.
        - `EBInterTargetExpDecayBusDefault`: Topic for storing Events that require exponential decay retry strategy.
        - `EBInterTargetBackoffBusDefault`: Topic for storing Events that require backoff retry strategy.
    - Corresponding subscription groups are created for each Topic:
        - The subscription group name follows the
          format [{host}{port}{topic}](https://github.com/tianping526/eventbridge/blob/main/app/job/internal/data/rocketmq.go#L101)
        - The retry count for the `EBInterTargetExpDecayBusDefault` subscription group is set to 176. If set
          incorrectly, the Job will not handle exponential decay retries correctly.
        - The retry count for the `EBInterTargetBackoffBusDefault` subscription group is set to 3. If set incorrectly,
          the Job will not handle backoff retries correctly.
        - The retry count for other subscription groups is set to 3, representing the retry count for Event flow
          failures within the Job.

Start Docker Compose:

> Ensure you have a `docker-compose.yaml` file in the current directory.

```bash
docker-compose -f docker-compose.yaml up -d
```

After starting, you can check the status of the containers:

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

Successful startup of the `eventbridge-create-default-data-bus-1` container with status `Exited (0)`
indicates that the Default Bus's Topics and subscription groups were created successfully.

#### Start Service

> Ensure the current directory contains a `service.yaml` file.

```bash
docker run -d --network eventbridge -p 8011:8011 -p 9011:9011 -v $(pwd)/service.yaml:/data/conf/service.yaml linktin/eb-service:1.0.0
```

Below is the content of `service.yaml`.
You can also view the [service configuration example](../../app/service/configs/service.yaml)
and [schema](../../app/service/internal/conf/conf.proto).

```yaml
bootstrap:
  server:
    http:
      addr: 0.0.0.0:8011 # HTTP Server listening port
      timeout: 1s
    grpc:
      addr: 0.0.0.0:9011 # gRPC Server listening port
      timeout: 1s
  data:
    database:
      driver: postgres
      source: postgresql://postgres:example@db:5432/postgres # Postgres connection string
      max_open: 100
      max_idle: 10
      conn_max_life_time: 0s
      conn_max_idle_time: 300s
    redis:
      addrs:
        - redis:6379 # Redis connection address
      password:
      db_index: 0
      dial_timeout: 1s
      read_timeout: 0.2s
      write_timeout: 0.2s
    default_mq: rocketmq://mq-proxy:8081 # RocketMQ Proxy connection address
```

After starting, you can check the status of the Service container:

```bash
docker ps -a
```

    CONTAINER ID   IMAGE                      COMMAND                  CREATED         STATUS                     PORTS                                            NAMES
    0cfa5a79afb8   linktin/eb-service:1.0.0   "./server -conf /dat…"   5 seconds ago   Up 4 seconds               0.0.0.0:8011->8011/tcp, 0.0.0.0:9011->9011/tcp   sweet_yalow

Successful startup of the Service container with status `Up` indicates that the Service is running correctly.

#### Start Job

> Ensure the current directory contains a `job.yaml` file.

```bash
docker run -d --network eventbridge -v $(pwd)/job.yaml:/data/conf/job.yaml linktin/eb-job:1.0.0
```

Below is the content of `job.yaml`.
You can also view the [Job configuration example](../../app/job/configs/service.yaml)
and [schema](../../app/job/internal/conf/conf.proto).

```yaml
bootstrap:
  server:
    http:
      addr: 0.0.0.0:8012 # Metrics Server listening port
      timeout: 1s
    event:
      source_timeout: 1s # The timeout for processing Events from the source_topic
      delay_timeout: 1s # The timeout for processing Events from the source_delay_topic
      target_exp_decay_timeout: 3s # The timeout for processing Events from the target_exp_decay_topic
      target_backoff_timeout: 3s # The timeout for processing Events from the target_backoff_topic
  data:
    database:
      driver: postgres
      source: postgresql://postgres:example@db:5432/postgres # Postgres connection string
      max_open: 100
      max_idle: 10
      conn_max_life_time: 0s
      conn_max_idle_time: 300s
    default_mq: rocketmq://mq-proxy:8081 # RocketMQ Proxy connection address
```

After starting, you can check the status of the Job container:

```bash
docker ps -a
```

    CONTAINER ID   IMAGE                      COMMAND                  CREATED          STATUS                      PORTS                                            NAMES
    b7c280bfde43   linktin/eb-job:1.0.0       "./server -conf /dat…"   5 seconds ago    Up 5 seconds                                                                 happy_hugle

Successful startup of the Job container with status `Up` indicates that the Job is running correctly.

</details>

<details>
<summary><span style="font-size:1.5em; font-weight:bold;">Deploying with Helm chart</span></summary>

> The Helm chart used in the demonstration launches a highly available EventBridge cluster, 
> which includes Service, Job, Postgres, Redis, and RocketMQ.

#### Add Helm repository

```bash
helm repo add tianping526 https://tianping526.github.io/helm-charts
helm repo update
```

#### Install EventBridge

```bash
helm install eventbridge tianping526/eventbridge --namespace eventbridge --create-namespace
```

#### Check the status of the EventBridge cluster

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

All services are in the `Running` status, indicating they have started successfully. 
You may notice that some Pods have a `RESTARTS` count greater than 0; 
this is because their dependent services were not ready, causing them to restart a few times. 
As long as the final status is `Running` , this is expected and not an issue.

</details>

## Create Bus

### Create Default Bus

However, if you attempt to create a Bus named `Default`,
it will fail because EventBridge automatically creates a Bus named `Default` upon startup.
You can view the definitions for creating a Bus in
the [HTTP Create Bus](https://github.com/tianping526/apis/blob/main/openapi.yaml#L10)
and [gRPC Create Bus](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L47).

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

As you can see, the error message indicates that the Bus name `Default` already exists.
After creating the Default Bus, you can view its information.
The definition for viewing Bus information can be found in
the [HTTP List Buses](https://github.com/tianping526/apis/blob/main/openapi.yaml#L59)
and [gRPC List Buses](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L43).

```bash

```Bash
curl --location '127.0.0.1:8011/v1/eventbridge/buses?prefix=Default' \
--header 'Accept: application/json'

# {"buses":[{"name":"Default", "mode":"BUS_WORK_MODE_CONCURRENTLY", "sourceTopic":"EBInterBusDefault", "sourceDelayTopic":"EBInterDelayBusDefault", "targetExpDecayTopic":"EBInterTargetExpDecayBusDefault", "targetBackoffTopic":"EBInterTargetBackoffBusDefault"}], "nextToken":"0"}
```

The `source_topic`, `source_delay_topic`, `target_exp_decay_topic`, and `target_backoff_topic` of the Default Bus are
`EBInterBusDefault`, `EBInterDelayBusDefault`, `EBInterTargetExpDecayBusDefault`, and `EBInterTargetBackoffBusDefault`,
respectively.
For more details about the purpose of these four topics, refer to the [Architecture](architecture.md#job)
and [Entity Relationship Diagram (ERD)](erd.md#bus).

### Create an Orderly Bus

We can create other Buses, such as `Orderly`, with a working mode of `BUS_WORK_MODE_ORDERLY`.
You can view the definitions for creating a Bus in
the [HTTP Create Bus](https://github.com/tianping526/apis/blob/main/openapi.yaml#L10)
and [gRPC Create Bus](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L47)

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

If the Bus's working mode is `BUS_WORK_MODE_ORDERLY`, you need to create orderly processing Topics for it.
As shown below, the topics `EBInterBusOrderly`, `EBInterTargetExpDecayBusOrderly`, and `EBInterTargetBackoffBusOrderly`
require the additional parameters `-a +message.type=FIFO -o true`.
Their corresponding consumer groups also require the `-o true` parameter.
Without these parameters, RocketMQ will treat these topics as regular topics,
and event processing will no longer be sequential.

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

## Create Schema

The definition for creating a Schema can be found in
the [HTTP Create Schema](https://github.com/tianping526/apis/blob/main/openapi.yaml#L280)
and [gRPC Create Schema](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L24).

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

This request creates a Schema for the Event of type `testSourceType1` from the source `testSource1`.
Such an Event has a property `a` of type `string`, and it will be sent to the Default Bus.

## Create Rule

The definition for creating a Rule can be found in
the [HTTP Create Rule](https://github.com/tianping526/apis/blob/main/openapi.yaml#L152)
and [gRPC Create Rule](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L63).

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

We have created a Rule named `TestRule` on the Default Bus.
This Rule matches Events with the `source` prefix `testSource1`
and sends the Event via HTTP POST method to `http://192.168.30.143:10188/target/event`
with the HTTP Body content being `{"code":"10188:{$.data.a}"}`.

To successfully receive the Event sent by `TestRule`,
we need to start the following HTTP server on the host `192.168.30.143`
to listen for incoming requests.

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

## Send Event

The definition for sending an Event can be found in
the [HTTP Post Event](https://github.com/tianping526/apis/blob/main/openapi.yaml#L127)
and [gRPC Post Event](https://github.com/tianping526/apis/blob/main/api/eventbridge/service/v1/eventbridge_service_v1.proto#L12).

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

You will see the following output in the HTTP server running on `192.168.30.143`:

    Received event: {"code":"10188:i am test content"}
