package wavreader

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	url2 "net/url"
	"os"
	"os/signal"
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
	Duration     int64
	vu           int64
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
			Logger:                  plog.NewLoggerWithLogrus(logger),
			MaxConnectionsPerBroker: 300,
		})
		pulsarClient = &pulsarC
		if err != nil {
			//return nil, fmt.Errorf("failed to create pulsar producer, topic: %s, error: %+v", producerConfig.Topic, err)
		}
	}

	return pulsarClient
}

// func (p *PubSub) CreateClient(clientConfig PulsarClientConfig) (pulsar.Client, error) {
// 	if pulsarClient == nil {
// 		logger := logrus.StandardLogger()
// 		logger.SetLevel(logrus.ErrorLevel)
// 		pulsarClient, err := pulsar.NewClient(pulsar.ClientOptions{
// 			URL: clientConfig.URL,
// 			//	ConnectionTimeout: 3 * time.Second,
// 			Logger: plog.NewLoggerWithLogrus(logger),
// 		})
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to create pulsar client, url: %s, error: %+v", clientConfig.URL, err)

// 		}
// 	}
// 	return pulsarClient, nil
// }

func (p *PubSub) CloseClient(client pulsar.Client) {
	client.Close()
}

func (p *PubSub) CloseProducer(producer pulsar.Producer) {
	producer.Close()
}

func (p *PubSub) CreateProducer(client pulsar.Client, config ProducerConfig) pulsar.Producer {
	log.Printf("topic name in create producer %s", config.Topic)
	option := pulsar.ProducerOptions{
		Topic: config.Topic,
		Name:  config.Topic,
		//Schema:              pulsar.NewStringSchema(nil),
		// CompressionType:     pulsar.LZ4,
		// CompressionLevel:    pulsar.Faster,
		// BatchingMaxMessages: 100,
		// MaxPendingMessages:  5,
		// SendTimeout:         time.Second/300,
	}

	producer, err := client.CreateProducer(option)
	if err != nil {
		log.Fatalf("failed to create producer, error: %+v", err)
	}
	return producer
}

func (p *PubSub) Publish(
        ctx context.Context,
        connectionId string,
        body []byte,
        properties map[string]string,
        async bool,
        inputFilePath string,
        durationMillisec int,
) error  {
	    state := lib.GetState(ctx)
        if state == nil {
                return errNilState
        }
        var err error
	webSocketVoxflo(inputFilePath, durationMillisec, connectionId)
	return  err
}

func (p *PubSub) PublishStreamPulsar(
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
		// need to get the elasped time in micro secon
		currentStats.Duration = time.Since(iterationStartTime).Milliseconds()
		currentStats.Bytes = (int64(len(mes[lop])))
		currentStats.Messages = 1

		//log.Printf("message %d", currentStats.Duration)
		//log.Printf("sending byte arr of length in loop %d ", len(msg.Payload))
		if err != nil {
			currentStats.Errors++
		}
		if errStats := ReportPubishMetrics(ctx, currentStats); errStats != nil {
			log.Fatal(errStats)
		}
		if time.Since(iterationStartTime).Milliseconds() < (int64(durationMillisec)) {
			time.Sleep(time.Millisecond *
				time.Duration((int64(durationMillisec))-time.Since(iterationStartTime).Milliseconds()))
		}

		//log.Printf("no error delivered")
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

func openAudioFile(filePath string) []int16 {
	file, err := os.Open(filePath)
	const samples_per_ms = 16
	if err != nil {
		log.Fatal("error opening file")
		return nil
	}
	decoder := wav.NewDecoder(file)
	if decoder == nil {
		log.Fatal("error in decoder")
		return nil
	}
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
	return int16buf
}

//func int16ArrayToByteArray(intArray []int16) ([]byte, error) {
//	byteArray := make([]byte, len(intArray)*2) // 2 bytes per int16
//
//	for i, val := range intArray {
//		binary.LittleEndian.PutUint16(byteArray[i*2:], uint16(val))
//	}
//
//	return byteArray, nil
//}

//func webSocketVoxflo(filePath string, durationMillisec int) {
//
//	url := url2.URL{Scheme: "ws", Host: "127.0.0.1:5555", Path: "", RawQuery: "connectId=1234"}
//	conn, resp, err := websocket.DefaultDialer.Dial(url.String(), nil)
//	int16buf := openAudioFile(filePath)
//	if err != nil {
//		log.Printf("error connecting")
//		log.Fatal("details", err)
//	} else {
//		log.Printf("resposne from {}", resp.Status)
//	}
//	defer conn.Close()
//	done := make(chan struct{})
//
//	go func() {
//		defer close(done)
//		for {
//			_, message, err := conn.ReadMessage()
//			if err != nil {
//				log.Printf("read", err)
//				return
//			}
//			log.Printf("message is {}", message)
//		}
//	}()
//	const samples_per_ms = 16
//	segmentSamples := samples_per_ms * durationMillisec
//	ticker := time.NewTicker(time.Millisecond * 60)
//	defer ticker.Stop()
//	interrupt := make(chan os.Signal, 1)
//	signal.Notify(interrupt, os.Interrupt)
//
//	for {
//		count := 0
//		end := segmentSamples
//		select {
//		case <-done:
//			return
//		case t := <-ticker.C:
//			if count <= len(int16buf) {
//				res, _ := int16ArrayToByteArray(int16buf[count:end])
//
//				var err = conn.WriteMessage(websocket.BinaryMessage, res)
//				count++
//				log.Printf("sending message in {}", t.String())
//				if err != nil {
//					log.Printf("error is {}", err)
//					return
//				}
//			} else {
//				log.Printf("finished messaging, count value is {}", count)
//				return
//			}
//		case <-interrupt:
//			var myEmptyArrByte []byte
//			var err = conn.WriteMessage(websocket.BinaryMessage, myEmptyArrByte)
//			if err != nil {
//				log.Printf("error is {}", err)
//				return
//			}
//
//		}
//	}
//
//}

func wavReaderVoxflo(inputFilePath string, durationMillisec int) [][]byte {
	// Open the input WAV file
	const samples_per_ms = 16
	var audioMessageBytesArr [][]byte
	log.Println("geting the wavReader")
	segmentSamples := samples_per_ms * durationMillisec
	var int16buf = openAudioFile(inputFilePath)
	var count uint32 = 0
	for i := 0; i < len(int16buf); i += segmentSamples {
		end := i + segmentSamples
		if end > len(int16buf) {
			end = len(int16buf)
		}
		count++
		audioData := NewAudioData(int16buf[i:end], 16)
		audioChannel := NewAudioChannel(1, []AudioData{*audioData})
		audioMessage := NewAudioMessage("load", count, []AudioChannel{*audioChannel})
		data, err := msgpack.Marshal(&audioMessage)
		if nil != err {
			log.Printf("error in serializing: %s", err.Error())
			return nil
		}
		audioMessageBytesArr = append(audioMessageBytesArr, data)
	}
	var close16buf []int16 = []int16{}
	audioData := NewAudioData(close16buf, 16)
	audioChannel := NewAudioChannel(1, []AudioData{*audioData})
	count++
	audioMessage := NewAudioMessage("load", count, []AudioChannel{*audioChannel})
	endData, err := msgpack.Marshal(&audioMessage)
	if nil != err {
		log.Printf("error in serializing: %s", err.Error())
		return nil
	}
	audioMessageBytesArr = append(audioMessageBytesArr, endData)
	return audioMessageBytesArr
}

func openAudioFileForBytes(filePath string) []int16 {
	file, err := os.Open(filePath)
	const samples_per_ms = 16
	if err != nil {
		log.Fatal("error opening file")
		return nil
	}
	decoder := wav.NewDecoder(file)
	if decoder == nil {
		log.Fatal("error in decoder")
		return nil
	}
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
	return int16buf
}

func int16ArrayToByteArray(intArray []int16) ([]byte, error) {
	byteArray := make([]byte, len(intArray)*2) // 2 bytes per int16

	for i, val := range intArray {
		binary.LittleEndian.PutUint16(byteArray[i*2:], uint16(val))
	}

	return byteArray, nil
}

func webSocketVoxflo(filePath string, durationMillisec int, connectionId string) {

	url := url2.URL{Scheme: "ws", Host: "127.0.0.1:5555", Path: "", RawQuery: fmt.Sprintf("connectId=%s", connectionId)}
	conn, resp, err := websocket.DefaultDialer.Dial(url.String(), nil)
	int16buf := openAudioFile(filePath)
	if err != nil {
		log.Printf("error connecting")
		log.Fatal("details", err)
	} else {
		log.Printf("resposne from {}", resp.Status)
	}
	defer conn.Close()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("read", err)
				return
			}
			log.Printf("message is {}", string(message))
		}
	}()
	const samples_per_ms = 16
	segmentSamples := samples_per_ms * durationMillisec
	ticker := time.NewTicker(time.Millisecond * 60)
	defer ticker.Stop()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		count := 0
		end := segmentSamples
		select {
		case <-done:
			return
		case t := <-ticker.C:
			if count <= len(int16buf) {
				res, _ := int16ArrayToByteArray(int16buf[count:end])

				var err = conn.WriteMessage(websocket.BinaryMessage, res)
				count++
				log.Printf("sending message in {}", t.String())
				if err != nil {
					log.Printf("error is {}", err)
					return
				}
			} else {
				log.Printf("finished messaging, count value is {}", count)
				return
			}
		case <-interrupt:
			var myEmptyArrByte []byte
			var err = conn.WriteMessage(websocket.BinaryMessage, myEmptyArrByte)
			if err != nil {
				log.Printf("error is {}", err)
				return
			}

		}
	}

}
