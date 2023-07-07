当前milvus上存在的问题

1、架构复杂，虽然让节点无状态，但是无端多了很多的概念。（proxy/datanode/indexnode/lognode/queryco...）

2、过多的外部依赖，etcd、mq、object storage、k8s

3、不同厂商的k8s可能不同，对交付会有很大的问题


期望：

1、架构足够简单，没有那么多节点概念（少于等于3个）

2、外部依赖足够简单，减少外部依赖，聚焦核心能力

3、可扩展的架构，支持
