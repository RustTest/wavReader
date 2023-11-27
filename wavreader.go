package main

import (
	"log"
	"os"

	"github.com/go-audio/wav"
)

type AudioData struct {
	pcm_bytes   []int16 `json:"pcm_bytes"`
	sample_rate uint32  `json:"sample_rate"`
}

type AudioMessage struct {
	id       string         `json:"id"`
	seq_no   uint32         `json:"seq_no"`
	channels []AudioChannel `json:"channels"`
}

type AudioChannel struct {
	channel_id uint32      `json:"channel_id"`
	data       []AudioData `json:"data"`
}

func wavReaderVoxflo(audioFile string, durationMillisec int) []AudioMessage {

	// Replace 'yourfile.wav' with the actual path to your WAV file
	filePath := audioFile

	audioMessageArr, err := extractWavChunk(filePath, durationMillisec)
	if err != nil {
		log.Fatal(err)
	}

   return audioMessageArr
}

func extractWavChunk(inputFilePath string, durationMillisec int) ([]AudioMessage, error) {
	// Open the input WAV file
	file, err := os.Open(inputFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode the WAV file
	decoder := wav.NewDecoder(file)
	if decoder == nil {
		return nil, nil
	}
	segmentSamples := ((int(decoder.SampleRate) * durationMillisec) / 1000)
	// Calculate the number of samples for the desired duration
	// Read audio data
	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, err
	}

	var int16buf []int16
	for _, value := range buf.Data {
		int16buf = append(int16buf, int16(value))
	}
	count := 0
	var audioMessageArr []AudioMessage
	for i := 0; i < len(int16buf); i += segmentSamples {
		end := i + segmentSamples
		if end > len(int16buf) {
			end = len(int16buf)
		}
		audioData := AudioData{
			pcm_bytes:   int16buf[i:end],
			sample_rate: 16,
		}
		count++
		audioChannel := AudioChannel{
			channel_id: 1,
			data:       []AudioData{audioData},
		}
		audioMessage := AudioMessage{
			id:       "1",
			seq_no:   uint32(count),
			channels: []AudioChannel{audioChannel},
		}
		audioMessageArr = append(audioMessageArr, audioMessage)
	}

	return audioMessageArr, nil
}
