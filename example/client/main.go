package main

import (
	"log"
	"math"
	"time"

	"github.com/arstd/simpletcp"
	"github.com/arstd/simpletcp/example/random"
)

func main() {

	useBytes(time.Second)

	useFrame(time.Second)
}

func useBytes(period time.Duration) {
	client := &simpletcp.Client{
		Host: "0.0.0.0",
		Port: 8623,

		FixedHeader: simpletcp.FixedHeader,
		Version:     simpletcp.Version1,
		DataType:    simpletcp.DataTypePlain,
		MaxLength:   simpletcp.MaxLength,
	}
	defer client.Close()

	var count = math.MaxUint32

	go func() {
		<-time.After(period)
		count = 0
	}()

	for i := 0; i < count; i++ {
		message := []byte(random.String(3))

		received, err := client.Send(message)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s", received)
	}
}

func useFrame(period time.Duration) {
	client := &simpletcp.Client{
		Host: "0.0.0.0",
		Port: 8623,
	}
	defer client.Close()

	var count = math.MaxUint32

	go func() {
		<-time.After(period)
		count = 0
	}()

	var frame, received simpletcp.Frame
	frame = simpletcp.Frame{
		Header: simpletcp.Header{
			FixedHeader: simpletcp.FixedHeader,
			Version:     simpletcp.Version1,
			DataType:    simpletcp.DataTypePlain,
		},
	}
	received = frame
	buf := make([]byte, simpletcp.MaxLength)

	for i := 0; i < count; i++ {
		frame.Data = random.Bytes(3)
		received.Data = buf // 不会重新分配内存

		err := client.SendFrame(&frame, &received)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%+v", received)
	}
}
