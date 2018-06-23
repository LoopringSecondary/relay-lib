# Relay Library

Relay library contains common codes of all micro services like relay-cluster and extractor. Also other system can use those modules too. The module list in the library is :

* broadcast  - Order broadcast module, contains ipfs and matrix.
* cache - The wrapped redis cache.
* cloudwatch - Aws cloudwatch client, can report metric data and system alarm infomation.
* crypto - The wrapped ethereum crypto module, has `verifySign`, `sign`, `generateHash` methods.
* dao - The common database access layer, only support mysql now.
* eth - The ethereum network access client, includes lot of useful class like `eth accessor` `ABI` `gasPriceEvaluator`
* eventemitter - A in-process event driven programming tool.
* extractor - The motan rpc interface of extractor microService.
* kafka - The wrapped Kafka message producer/comsumer client.
* log - The common log module with loopring log config.
* marketcap - The coinmarketcap data collector.
* marketutil - Loopring market and token common util module, contains meta data of tokens and market.
* motan - The wrapped motan rpc client.
* sns - The aws sns notify client.
* types - Contains all common model definition in Loopring relay microService.
* zklock - A distributed lock implement by zookeeper.
