# Description
Fetches all deribit options contracts of a user specified currency (e.g. BTC), and writes the data to a local kafka in docker. Data from Deribit is processed concurrently in batches of 50 options contracts. Each batch is a separate goroutine, that has its own WebSocket to subscribe to Deribit options data, and has its own producer to publish to the kafka topic for efficiency.


# Pre-requisite (Windows)

## Code
- `git clone` this repository
- Run `go mod tidy`

## Docker
- Download https://www.docker.com/products/docker-desktop/


# How to run

## Kafka and Zookeeper
- Open docker-desktop
- In a terminal, start Kafka and Zookeeper `docker-compose up -d`
- Create a topic `docker exec -it kafka kafka-topics --create --topic deribitOrderbook --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1`
- Verify topic was created `docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092`
- For testing, start up a consumer `docker exec -it kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic deribitOrderbook --from-beginning`

## Subscribe to Deribit options data and Produce to Kafka
- In a separate terminal, run `go run main.go`, `-currency` flag can be specified to specify the currency, default: BTC.
- Observe that messages are produced to the Kafka topic, and are consumed by the consumer in the Kafka terminal.

## Clean up
- Terminate kafka, zookeeper and docker `docker-compose down`


# Considerations
- **Efficiency**: There are a lot of incoming data, and we need to maximize efficiency via concurrency. The data is split based on the options contracts, for example BTC has about 1300 options contracts now. A goroutine will be created for every 50 options contracts, with its own websocket to subscribe to these 50 datasets, and its own producer to publish to the Kafka topic.   
- **Robustness**: As mentioned above, processing is done in batches of 50, with its own goroutine, websocket and producer. If a particular goroutine, websocket or producer fails, it will not affect other batches. Moreover, retries have been implemented for the websocket connection failures and publishing failures, to handle transient errors.  
- **Flexibility**: `-currency` flag allows user to specify any currency to subscribe to. 

# Improvements
- Message Batching: The data from Deribit is quite high-volume, batching messages can be considered before publishing to Kafka within each goroutine, which helps to reduce the number of requests sent to Kafka and improve throughput.
- Partitions: Depending on the requirements, number of partitions can be increased. E.g. one partition for each currency, or partitioning based on dates. With increased partitions, we can increase the respective number of consumers to increase parallelization.
- Retry/error topic: If publishing of a particular message exceeds the number of retry, we could add this to a retry or error topic to investigate the root cause further.
- Observability: Observability and alarms should be added as well, to monitor the health of the system.
- Exponential backoff: Currently some of the retries uses a static 5 seconds delay, we can improve this by adding exponential backoff.
- Tests: Unit tests, integration tests and end-to-end testing should be added as well before deploying to production.
