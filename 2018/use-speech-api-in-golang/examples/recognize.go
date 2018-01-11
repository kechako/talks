package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gordonklaus/portaudio"
	"golang.org/x/net/context"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

func readAudio() ([]byte, error) {
	var err error

	// initialize portaudio
	portaudio.Initialize()
	defer portaudio.Terminate()

	const bitDepth = 16
	const channels = 1
	const sampleRate = 16000
	const length = 10 // second

	// open default stream
	buf := make([]int16, sampleRate)
	stream, err := portaudio.OpenDefaultStream(channels, 0, sampleRate, len(buf), buf)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	fmt.Print("Input voice (10s)...")
	err = stream.Start()
	if err != nil {
		return nil, err
	}
	defer stream.Stop()

	var byteBuf bytes.Buffer
	for i := 0; i < 10; i++ {
		err = stream.Read()
		if err != nil && err != portaudio.InputOverflowed {
			return nil, err
		}

		binary.Write(&byteBuf, binary.LittleEndian, buf)
		if err != nil {
			return nil, err
		}
	}

	fmt.Println("Finished")

	return byteBuf.Bytes(), nil
}

func main() {
	buf, err := readAudio()
	if err != nil {
		log.Fatalf("Error : %v\n", err)
	}

	// create new client
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// request recognize
	resp, err := client.Recognize(ctx, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:              speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz:       16000,
			LanguageCode:          "ja-JP",
			EnableWordTimeOffsets: true,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: buf,
			},
		},
	})
	if err != nil {
		log.Fatalf("Could not recognize: %v", err)
	}
	// show result
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			fmt.Printf("Transcript: %s\n", alt.Transcript)
			for _, w := range alt.Words {
				fmt.Printf("  Word: %s\n", w.Word)
			}
		}
	}
}
