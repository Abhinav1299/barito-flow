package cmds

import (
	"fmt"

	"github.com/BaritoLog/barito-flow/flow"
	"github.com/BaritoLog/go-boilerplate/envkit"
	"github.com/BaritoLog/go-boilerplate/timekit"
	"github.com/BaritoLog/instru"
	cluster "github.com/bsm/sarama-cluster"
	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	EnvForwarderKafkaBrokers         = "BARITO_FORWARDER_KAFKA_BROKERS"
	EnvForwarderKafkaConsumerGroupId = "BARITO_FORWARDER_KAFKA_CONSUMER_GROUP_ID"
	EnvForwarderKafkaConsumerTopic   = "BARITO_FORWARDER_KAFKA_CONSUMER_TOPIC"
	EnvForwarderElasticsearchUrl     = "BARITO_FORWARDER_ELASTICSEARCH_URL"

	EnvPushMetricUrl      = "BARITO_PUSH_METRIC_URL"
	EnvPushMetricToken    = "BARITO_PUSH_METRIC_TOKEN"
	EnvPushMetricInterval = "BARITO_PUSH_METRIC_INTERVAL"
)

func Forwarder(c *cli.Context) (err error) {

	brokers := envkit.GetSlice(EnvForwarderKafkaBrokers, ",", []string{"localhost:9092"})
	groupID := envkit.GetString(EnvForwarderKafkaConsumerGroupId, "barito-group")
	topics := envkit.GetSlice(EnvForwarderKafkaConsumerTopic, ",", []string{"topic01"})
	esUrl := envkit.GetString(EnvForwarderElasticsearchUrl, "http://localhost:9200")

	pushMetricUrl := envkit.GetString(EnvPushMetricUrl, "http://localhost:3000/api/increase_log_count")
	pushMetricToken := envkit.GetString(EnvPushMetricToken, "")
	pushMetricInterval := envkit.GetString(EnvPushMetricInterval, "30s")

	log.Infof("Start Forwarder")
	log.Infof("%s=%v", EnvForwarderKafkaBrokers, brokers)
	log.Infof("%s=%s", EnvForwarderKafkaConsumerGroupId, groupID)
	log.Infof("%s=%v", EnvForwarderKafkaConsumerTopic, topics)
	log.Infof("%s=%v", EnvForwarderElasticsearchUrl, esUrl)
	log.Infof("%s=%v", EnvPushMetricUrl, pushMetricUrl)
	log.Infof("%s=%v", EnvPushMetricToken, pushMetricToken)
	log.Infof("%s=%v", EnvPushMetricInterval, pushMetricInterval)

	if pushMetricToken != "" && pushMetricUrl != "" {
		log.Infof("Set callback to instrumentation")
		instru.SetCallback(timekit.Duration(pushMetricInterval), flow.NewMetricMarketCallback(pushMetricUrl, pushMetricToken))
	} else {
		log.Infof("No callback for instrumentation")
	}

	// elastic client
	client, err := elastic.NewClient(
		elastic.SetURL(esUrl),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	)

	storeman := flow.NewElasticStoreman(client)

	// consumer config
	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true

	// kafka consumer
	consumer, err := cluster.NewConsumer(brokers, groupID, topics, config)
	if err != nil {
		return
	}

	agent := flow.KafkaAgent{
		Consumer: consumer,
		Store:    storeman.Store,
		OnError: func(err error) {
			fmt.Println(err.Error())
		},
	}

	return agent.Start()

}
