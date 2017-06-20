package main

import (
	"bufio"
	"math/rand"
	"net"
	"time"

	"github.com/arstd/log"
	"github.com/arstd/simpletcp"
	"github.com/arstd/simpletcp/example/random"
	"github.com/arstd/simpletcp/example/util"
	"github.com/arstd/simpletcp/simple"
)

const randLength = 2048

func main() {
	// useBytes(30000 * time.Millisecond)
	// useFrame(1000 * time.Millisecond)
	// asyncFrame(6e5)

	single()
}

func single() {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8623")
	log.Fataln(err)
	tcpConn, err := net.DialTCP("tcp", nil, addr)
	log.Fataln(err)

	defer tcpConn.Close()

	tcpConn.SetNoDelay(true)

	pack := simple.NewPacketMessageId(0)
	pack.Data = random.Bytes(rand.Intn(randLength))

	log.Fataln(tcpConn.Write(pack.GetHeader()))
	log.Fataln(tcpConn.Write(pack.GetBody()))

	proto := &simple.Protocol{}

	res, err := proto.ReadPacket(tcpConn)
	pack = res.(*simple.Packet)
	log.Fataln(err)
	log.Debugf("% x", res.GetHeader())
	log.Debug(pack.MessageId(), pack.DataLength(), pack.Data)
}

func useBytes(period time.Duration) {
	client := &simpletcp.Client{
		Host: "0.0.0.0",
		Port: 8623,

		Fixed:     simpletcp.Fixed,
		MaxLength: simpletcp.MaxLength,

		Version:  simpletcp.Version1,
		DataType: simpletcp.DataTypeJSON,
	}
	defer client.Close()

	var stop bool
	var count time.Duration

	go func() {
		<-time.After(period)
		stop = true
	}()

	for !stop {
		count++
		message := random.Bytes(rand.Intn(randLength))
		_, err := client.Send(message)
		if err != nil {
			log.Fatal(err)
		}
		// log.Printf("%s", received)
	}
	log.Infof("%s %d => qps: %d, latency: %s", period, count, count*time.Second/period, period/count)
}

func useFrame(period time.Duration) {
	client := &simpletcp.Client{
		Host: "0.0.0.0",
		Port: 8623,
	}
	defer client.Close()

	var stop bool
	var count time.Duration

	go func() {
		<-time.After(period)
		stop = true
	}()

	frame := simpletcp.Frame{
		Header: simpletcp.Header{
			Version:  simpletcp.Version1,
			DataType: simpletcp.DataTypeJSON,
		},
	}

	for !stop {
		count++
		frame.MessageId = client.NextId()
		// frame.MessageId = uint32(i + 1)

		frame.Data = random.Bytes(rand.Intn(randLength))

		_, err := client.SendFrame(&frame)
		if err != nil {
			log.Fatal(err)
		}
		// log.Debugf("%d %d %s", received.MessageId, received.DataLength, received.Data)
		// log.Debugf("%#v", received)
	}
	log.Infof("%s %d => qps: %d, latency: %s", period, count, count*time.Second/period, period/count)
}

func asyncFrame(count uint32) {
	raddr, err := net.ResolveTCPAddr("tcp", ":8623")
	if err != nil {
		log.Fatal(err)
	}

	tcpConn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		log.Fatal(err)
	}
	defer tcpConn.Close()

	tcpConn.SetNoDelay(true)

	// tcpConn.SetKeepAlive(true)
	// tcpConn.SetKeepAlivePeriod(time.Second)

	// tcpConn.SetReadBuffer(20480)
	// tcpConn.SetWriteBuffer(20480)

	frame := simpletcp.Frame{
		Header: simpletcp.Header{
			Version:  simpletcp.Version1,
			DataType: simpletcp.DataTypeJSON,
		},
	}

	var start time.Time
	go func() {
		bw := bufio.NewWriter(tcpConn)
		start = time.Now()
		log.Info("start send")
		for i := uint32(0); i < count; i++ {
			frame.MessageId = i
			frame.Data = random.Bytes(1024)
			frame.Version = simpletcp.Version1

			if i%10000 == 0 {
				frame.Version = simpletcp.VersionPing
				frame.Data = nil
			}

			reserved := uint32(time.Now().UnixNano() / 1000)
			frame.Reserved = util.Bytes(reserved)

			if err = simpletcp.Write(bw, simpletcp.Fixed, &frame); err != nil {
				log.Error(err)
			}
			// time.Sleep(1 * time.Nanosecond)
			// log.Infof("%d: %d %d %s", scount, frame.MessageId, frame.DataLength, frame.Data)
		}
	}()

	var total int64
	br := bufio.NewReader(tcpConn)
	for i := uint32(0); i < count; i++ {
		received, err := simpletcp.Read(br, simpletcp.Fixed, simpletcp.MaxLength)
		if err != nil {
			log.Fatal(err)
		}
		if received.Version == simpletcp.VersionPing {
			log.Debugf("%d: %d %d %s", i, received.MessageId, received.DataLength, received.Data)
		}

		reserved := util.Uint32(received.Reserved)

		now := uint32(time.Now().UnixNano() / 1000)
		delta := int64(now - reserved)
		// log.Info(delta, reserved)
		total += delta
		if i%(count/10) == 0 {
			log.Debugf("%d: %d %d %s", i, received.MessageId, received.DataLength, received.Data)
			log.Infof("%06d %d-%d=%d", received.MessageId, now, reserved, delta)
		}
	}

	used := time.Since(start)

	log.Infof("Requests/sec: %d/%s = %d", count, used, time.Duration(count)*1e9/used)
	log.Infof("Latency: %s/%d = %s", time.Duration(total*1000), count,
		time.Duration(total*1000)/time.Duration(count))
}
