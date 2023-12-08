package wavreader

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	plog "github.com/apache/pulsar-client-go/pulsar/log"
	"github.com/go-audio/wav"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v5"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/stats"
)

var (
	errNilState        = errors.New("xk6-pubsub: publisher's state is nil")
	errNilStateOfStats = errors.New("xk6-pubsub: stats's state is nil")
)

type PublisherStats struct {
	Topic        string
	ProducerName string
	Messages     int
	Errors       int
	Bytes        int64
	Duration     time.Duration
}

type PubSub struct{}

func init() {
	modules.Register("k6/x/wavreader", new(PubSub))
}

type PulsarClientConfig struct {
	URL string
}

type ProducerConfig struct {
	Topic string
}

func (p *PubSub) CreateClient(clientConfig PulsarClientConfig) (pulsar.Client, error) {
	logger := logrus.StandardLogger()
	logger.SetLevel(logrus.ErrorLevel)
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               clientConfig.URL,
		ConnectionTimeout: 3 * time.Second,
		Logger:            plog.NewLoggerWithLogrus(logger),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pulsar client, url: %s, error: %+v", clientConfig.URL, err)

	}
	return client, nil
}

func (p *PubSub) CloseClient(client pulsar.Client) {
	client.Close()
}

func (p *PubSub) CloseProducer(producer pulsar.Producer) {
	producer.Close()
}

func (p *PubSub) CreateProducer(client pulsar.Client, config ProducerConfig) pulsar.Producer {
	option := pulsar.ProducerOptions{
		Topic:               config.Topic,
		Schema:              pulsar.NewStringSchema(nil),
		CompressionType:     pulsar.LZ4,
		CompressionLevel:    pulsar.Faster,
		BatchingMaxMessages: 100,
		MaxPendingMessages:  100,
		SendTimeout:         time.Second,
	}

	producer, err := client.CreateProducer(option)
	if err != nil {
		log.Fatalf("failed to create producer, error: %+v", err)
	}
	return producer
}

func (p *PubSub) Publish(
	ctx context.Context,
	producer pulsar.Producer,
	body []byte,
	properties map[string]string,
	async bool,
	inputFilePath string,
	durationMillisec int,
) error {
	state := lib.GetState(ctx)
	if state == nil {
		return errNilState
	}
	var err error
	currentStats := PublisherStats{
		Topic:        producer.Topic(),
		ProducerName: producer.Name(),
		Bytes:        int64(len(body)),
		Messages:     1,
	}

	var mes [][]byte = wavReaderVoxflo(inputFilePath, durationMillisec)
	//log.Printf("sending total byte arr of length %d", len(mes))

	for lop := 0; lop < len(mes); lop++ {
		msg := &pulsar.ProducerMessage{
			Value:      "",
			Payload:    mes[lop],
			Properties: properties,
		}
		if async {
			producer.SendAsync(
				ctx,
				msg,
				func(mi pulsar.MessageID, pm *pulsar.ProducerMessage, e error) {
					if e != nil {
						err = e
						currentStats.Errors++
					}
				},
			)
			return err
		}
		msg.EventTime = time.Now()
		var iterationStartTime = time.Now()
		_, err = producer.Send(ctx, msg)
		currentStats.Duration = time.Since(iterationStartTime)
		currentStats.Bytes = (int64(len(mes[lop])))
		currentStats.Messages++

		//log.Printf("sending byte arr of length in loop %d ", len(msg.Payload))
		if err != nil {
			currentStats.Errors++
		}
		if errStats := ReportPubishMetrics(ctx, currentStats); errStats != nil {
			log.Fatal(errStats)
		}
		log.Printf("no error delivered")
	}
	producer.Close()
	return err
}

func ReportPubishMetrics(ctx context.Context, currentStats PublisherStats) error {
	state := lib.GetState(ctx)
	if state == nil {
		return errNilStateOfStats
	}

	tags := make(map[string]string)
	tags["producer_name"] = currentStats.ProducerName
	tags["topic"] = currentStats.Topic

	now := time.Now()

	stats.PushIfNotDone(ctx, state.Samples, stats.Sample{
		Time:   now,
		Metric: PublishMessages,
		Tags:   stats.IntoSampleTags(&tags),
		Value:  float64(currentStats.Messages),
	})

	stats.PushIfNotDone(ctx, state.Samples, stats.Sample{
		Time:   now,
		Metric: PublishErrors,
		Tags:   stats.IntoSampleTags(&tags),
		Value:  float64(currentStats.Errors),
	})

	stats.PushIfNotDone(ctx, state.Samples, stats.Sample{
		Time:   now,
		Metric: PublishBytes,
		Tags:   stats.IntoSampleTags(&tags),
		Value:  float64(currentStats.Bytes),
	})

	stats.PushIfNotDone(ctx, state.Samples, stats.Sample{
		Time:   now,
		Metric: PublishDuration,
		Tags:   stats.IntoSampleTags(&tags),
		Value:  float64(currentStats.Duration),
	})
	return nil
}

type AudioData struct {
	PcmBytes   []int16 `msgpack:"pcm_bytes"`
	SampleRate uint32  `msgpack:"sample_rate"`
}

// func (ad AudioData) GetPcm_bytes() []int16 {
// 	return ad.pcm_bytes
// }

//	func (ad AudioData) GetSample_rate() uint32 {
//		return ad.sample_rate
//	}
type AudioChannel struct {
	ChannelId uint32      `msgpack:"channel_id"`
	Data      []AudioData `msgpack:"data"`
}

// func (ac AudioChannel) GetChannel_id() uint32 {
// 	return ac.channel_id
// }

// func (ac AudioChannel) GetData() []AudioData {
// 	return ac.data
// }

type AudioMessage struct {
	Id       string         `msgpack:"id"`
	SeqNo    uint32         `msgpack:"seq_no"`
	Channels []AudioChannel `msgpack:"channels"`
}

// func (am AudioMessage) GetId() string {
// 	return am.id
// }

// func (am AudioMessage) GetSeqNo() uint32 {
// 	return am.seqNo
// }

// func (am AudioMessage) GetChannels() []AudioChannel {
// 	return am.channels
// }

func NewAudioData(pcmBytes []int16, sampleRate uint32) *AudioData {
	return &AudioData{
		PcmBytes:   pcmBytes,
		SampleRate: sampleRate,
	}
}

func NewAudioChannel(channelID uint32, data []AudioData) *AudioChannel {
	return &AudioChannel{
		ChannelId: channelID,
		Data:      data,
	}
}

func NewAudioMessage(id string, seqNo uint32, channels []AudioChannel) *AudioMessage {
	return &AudioMessage{
		Id:       id,
		SeqNo:    seqNo,
		Channels: channels,
	}
}

func wavReaderVoxflo(inputFilePath string, durationMillisec int) [][]byte {
	// Open the input WAV file
	log.Println("geting the wavReader")
	file, err := os.Open(inputFilePath)
	var audioMessageBytesArr [][]byte
	if err != nil {
		log.Fatal("error opening file")
		return nil
	}
	defer file.Close()

	// Decode the WAV file
	decoder := wav.NewDecoder(file)
	if decoder == nil {
		log.Fatal("error in decoder")
		return nil
	}
	segmentSamples := 5328
	// Calculate the number of samples for the desired duration
	// Read audio data
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		log.Fatal("error in pcm buf")
		return nil
	}

	var int16buf []int16
	for _, value := range buf.Data {
		int16buf = append(int16buf, int16(value))
	}
	var count uint32 = 0
	for i := 0; i < len(int16buf); i += segmentSamples {
		end := i + segmentSamples
		if end > len(int16buf) {
			end = len(int16buf)
		}
		audioData := NewAudioData(int16buf[i:end], 16)
		audioChannel := NewAudioChannel(1, []AudioData{*audioData})
		audioMessage := NewAudioMessage("load", count, []AudioChannel{*audioChannel})
		// AudioData{
		// 	pcm_bytes:   int16buf[i:end],
		// 	sample_rate: 16,
		// }
		// audioChannel := AudioChannel{
		// 	channel_id: 1,
		// 	data:       []AudioData{audioData},
		// }
		// audioMessage := AudioMessage{
		// 	id:       "1",
		// 	seq_no:   uint32(count),
		// 	channels: []AudioChannel{audioChannel},
		// }
		count++
		data, err := msgpack.Marshal(&audioMessage)
		if nil != err {
			log.Printf("error in serializing: %s", err.Error())
			return nil
		}
		//log.Printf("size of bytes are with the new encoder%d", len(data))
		audioMessageBytesArr = append(audioMessageBytesArr, data)
	}
	//log.Fatal("length isaudioMessageArr %d", len(audioMessageArr))
	audioMessageBytesArr = append(audioMessageBytesArr, []byte{})
	return audioMessageBytesArr
}
