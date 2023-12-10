package wavreader

import (
	"github.com/apache/pulsar-client-go/pulsar"
	plog "github.com/apache/pulsar-client-go/pulsar/log"
	"github.com/sirupsen/logrus"
)

// create a singleton client for all the producers
var pulsarClient *pulsar.Client

func (p *PubSub) CreatePulsarClient(clientConfig PulsarClientConfig) *pulsar.Client {

	if pulsarClient == nil {
		logger := logrus.StandardLogger()
		logger.SetLevel(logrus.ErrorLevel)
		var err error
		var pulsarC pulsar.Client
		pulsarC, err = pulsar.NewClient(pulsar.ClientOptions{
			URL: clientConfig.URL,
			//	ConnectionTimeout: 3 * time.Second,
			Logger: plog.NewLoggerWithLogrus(logger),
		})
		pulsarClient = &pulsarC
		if err != nil {
			//return nil, fmt.Errorf("failed to create pulsar producer, topic: %s, error: %+v", producerConfig.Topic, err)
		}
	}

	return pulsarClient
}
