package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gordonklaus/portaudio"
	"golang.org/x/net/context"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

var DefaultSampleRate = 16000

type AudioInput struct {
	Data []byte
	Err  error
}

type AudioStream struct {
	Input   chan AudioInput
	stream  *portaudio.Stream
	buf     []int16
	closed  bool
	canExit chan struct{}
	mu      sync.Mutex
}

func NewAudioStream() (*AudioStream, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}

	buf := make([]int16, DefaultSampleRate/2)

	stream, err := portaudio.OpenDefaultStream(1, 0, float64(DefaultSampleRate), len(buf), buf)
	if err != nil {
		return nil, err
	}

	return &AudioStream{
		stream:  stream,
		Input:   make(chan AudioInput, 10),
		buf:     buf,
		closed:  false,
		canExit: make(chan struct{}),
	}, nil
}

func (s *AudioStream) read() {
	defer func() {
		close(s.Input)
		close(s.canExit)
	}()

	for {
		var input AudioInput

		s.mu.Lock()
		if s.closed {
			return
		}

		err := s.stream.Read()
		if err == nil {
			var buf bytes.Buffer
			err = binary.Write(&buf, binary.LittleEndian, s.buf)
			if err == nil {
				input.Data = buf.Bytes()
			}
		}

		if err != nil {
			input.Err = err
		}

		s.mu.Unlock()

		s.Input <- input
	}
}

func (s *AudioStream) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("audio stream is already closed")
	}

	err := s.stream.Start()
	if err != nil {
		return err
	}

	go s.read()

	return nil
}

func (s *AudioStream) Close() (err error) {
	s.mu.Lock()

	if s.closed {
		return
	}

	err = s.stream.Stop()
	if err != nil {
		return
	}

	err = s.stream.Close()
	if err != nil {
		return
	}

	s.closed = true

	s.mu.Unlock()

	<-s.canExit

	err = portaudio.Terminate()
	if err != nil {
		return
	}

	return nil
}

func run() (int, error) {
	ctx := context.Background()

	// [START speech_streaming_mic_recognize]
	client, err := speech.NewClient(ctx)
	if err != nil {
		return 1, err
	}
	stream, err := client.StreamingRecognize(ctx)
	if err != nil {
		return 1, err
	}
	// Send the initial configuration message.
	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_LINEAR16,
					SampleRateHertz: 16000,
					LanguageCode:    "ja-JP",
				},
				InterimResults: true,
			},
		},
	}); err != nil {
		return 1, err
	}

	audio, err := NewAudioStream()
	if err != nil {
		return 1, err
	}
	defer audio.Close()

	err = audio.Start()
	if err != nil {
		return 1, err
	}
	fmt.Println("Input voice")

	ctx, cancel := context.WithTimeout(ctx, 59*time.Second)
	sig := make(chan os.Signal)

	go func() {
		<-sig
		cancel()
	}()

	signal.Notify(sig, os.Interrupt)

	go send(ctx, stream, audio)

	err = receive(ctx, stream)
	if err != nil {
		return 1, err
	}

	return 0, nil
}

func send(ctx context.Context, stream speechpb.Speech_StreamingRecognizeClient, audio *AudioStream) {
	defer func() {
		if err := stream.CloseSend(); err != nil {
			log.Printf("Could not close stream: %v\n", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				switch err {
				case context.Canceled:
					log.Println("Aborted.")
				case context.DeadlineExceeded:
					log.Println("Timeout.")
				}
			}
			return
		case input, ok := <-audio.Input:
			if !ok {
				return
			}
			if input.Err != nil {
				log.Printf("Could not read audio: %v\n", input.Err)
				return
			}

			if err := stream.Send(&speechpb.StreamingRecognizeRequest{
				StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
					AudioContent: input.Data,
				},
			}); err != nil {
				log.Printf("Could not send audio: %v\n", err)
				return
			}
		}
	}
}

func receive(ctx context.Context, stream speechpb.Speech_StreamingRecognizeClient) error {
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Cannot stream results: %v", err)
		}
		if err := resp.Error; err != nil {
			return fmt.Errorf("Could not recognize: %v", err)
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
}

func main() {
	code, err := run()
	if err != nil {
		log.Printf("Error : %v\n", err)
	}
	if code != 0 {
		os.Exit(code)
	}
}
