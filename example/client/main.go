package main

import (
	"bufio"
	"math"
	"math/rand"
	"net"
	"time"

	"github.com/arstd/log"
	"github.com/arstd/simpletcp"
	"github.com/arstd/simpletcp/example/random"
)

const randLength = 2048

func main() {

	// useBytes(1 * time.Millisecond)
	//
	// useFrame(1 * time.Millisecond)

	asyncFrame(10000 * time.Millisecond)
}

func useBytes(period time.Duration) {
	client := &simpletcp.Client{
		Host: "0.0.0.0",
		Port: 8623,

		FixedHeader: simpletcp.FixedHeader,
		Version:     simpletcp.Version1,
		DataType:    simpletcp.DataTypeJSON,
		MaxLength:   simpletcp.MaxLength,
	}
	defer client.Close()

	var count = math.MaxUint32

	go func() {
		<-time.After(period)
		count = 0
	}()

	for i := 0; i < count; i++ {
		message := []byte(random.String(rand.Intn(randLength)))

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

	frame := simpletcp.Frame{
		Header: simpletcp.Header{
			FixedHeader: simpletcp.FixedHeader,
			Version:     simpletcp.Version1,
			DataType:    simpletcp.DataTypeJSON,
		},
	}

	for i := 0; i < count; i++ {
		// frame.MessageId = client.NextMessageId()
		// frame.MessageId = uint32(i + 1)
		frame.MessageId = 0 // 如果未设置， client 会分配 MessageId

		frame.Data = random.Bytes(rand.Intn(randLength))

		received, err := client.SendFrame(&frame)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("%d %d %s", received.MessageId, received.DataLength, received.Data)
	}
}

func asyncFrame(period time.Duration) {
	raddr, err := net.ResolveTCPAddr("tcp", ":8623")
	if err != nil {
		log.Fatal(err)
	}

	tcpConn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		log.Fatal(err)
	}
	defer tcpConn.Close()

	var stop bool
	var scount, rcount uint32

	go func() {
		<-time.After(period)
		stop = true
	}()

	frame := simpletcp.Frame{
		Header: simpletcp.Header{
			FixedHeader: simpletcp.FixedHeader,
			Version:     simpletcp.Version1,
			DataType:    simpletcp.DataTypeJSON,
		},
	}

	var start time.Time
	go func() {
		bw := bufio.NewWriter(tcpConn)
		start = time.Now()
		log.Info("start send")
		for !stop {
			scount++
			frame.MessageId = scount
			frame.Data = random.Bytes(1024)

			reserved := uint32(time.Now().UnixNano() / 1000)
			frame.Reserved = [4]byte{
				byte(reserved >> 24),
				byte(reserved >> 16),
				byte(reserved >> 8),
				byte(reserved),
			}

			if err = simpletcp.Write(bw, &frame); err != nil {
				log.Error(err)
			}
			time.Sleep(1 * time.Microsecond)
			// log.Infof("%d: %d %d %s", scount, frame.MessageId, frame.DataLength, frame.Data)
		}
	}()

	var total int64
	br := bufio.NewReader(tcpConn)
	for {
		received, err := simpletcp.Read(br, simpletcp.FixedHeader, simpletcp.MaxLength)
		if err != nil {
			log.Fatal(err)
		}
		rcount++
		// log.Debugf("%d: %d %d %s", rcount, received.MessageId, received.DataLength, received.Data)

		reserved := uint32(received.Reserved[0])<<24 +
			uint32(received.Reserved[1])<<16 +
			uint32(received.Reserved[2])<<8 +
			uint32(received.Reserved[3])

		delta := int64(uint32(time.Now().UnixNano()/1000) - reserved)
		// log.Info(delta, reserved)
		total += delta

		if stop && rcount == scount {
			break
		}
	}

	log.Infof("Requests/sec: %d/%s = %d", rcount, time.Now().Sub(start),
		1e9*time.Duration(rcount)/time.Now().Sub(start))
	log.Infof("Latency: %s/%d = %s", time.Duration(1000*total), rcount,
		1000*time.Duration(total)/time.Duration(rcount))
}
