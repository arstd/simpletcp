package main

import (
	"encoding/binary"
	"flag"
	"io"
	"math/rand"
	"net"
	"time"

	"github.com/arstd/log"
	"github.com/arstd/simpletcp"
)

var (
	addr  string
	times int
	conns int
	rest  int64
)

func main() {
	flag.StringVar(&addr, "a", "127.0.0.1:8090", "acserver tcp address")
	flag.IntVar(&times, "t", 20e4, "send times of one connection")
	flag.IntVar(&conns, "c", 32, "connection numbers")
	flag.Int64Var(&rest, "r", 19e4, "rest time(ns) before send another body")
	flag.Parse()

	log.Infof("addr=%s conns=%d times=%d rest=%d", addr, conns, times, rest)

	single()

	if conns < 0 || times < 10 {
		return
	}

	for i := 1; i < conns; i++ {
		go send(uint32(times))
	}
	send(uint32(times * 11 / 10))
}

func single() {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	log.Fataln(err)
	tcpConn, err := net.DialTCP("tcp", nil, raddr)
	log.Fataln(err)

	defer tcpConn.Close()
	tcpConn.SetNoDelay(true)

	f := simpletcp.NewFrameDefault()
	f.SetMessageId(15)
	f.SetBodyWithLength(body)

	log.Fataln(tcpConn.Write(f.Head()))
	log.Fataln(tcpConn.Write(f.Body))

	// f = simpletcp.NewFrameDefault()
	log.Debug(io.ReadFull(tcpConn, f.Head()))
	f.SetBody(make([]byte, f.BodyLength()))
	log.Debug(io.ReadFull(tcpConn, f.Body))

	log.Debug(f.String())
}

func send(count uint32) {
	raddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	tcpConn, err := net.DialTCP("tcp", nil, raddr)
	if err != nil {
		log.Fatal(err)
	}
	defer tcpConn.Close()

	var start time.Time
	go func() {
		f := simpletcp.NewFrameDefault()
		start = time.Now()
		log.Debug("start send")
		// f.SetVersion(simpletcp.VersionPing)
		tcpConn.SetWriteBuffer(4096 * 1024)
		for i := uint32(1); i <= count; i++ {
			f.SetMessageId(i)
			// f.SetBodyWithLength([]byte("hello"))
			f.SetBodyWithLength(body[:1+rand.Intn(len(body))])

			reserved := uint32(time.Now().UnixNano() / 1000)
			binary.BigEndian.PutUint32(f.Reserved(), reserved)
			log.Fataln(tcpConn.Write(f.Head()))
			log.Fataln(tcpConn.Write(f.Body))

			// log.Debug(f.String())
			time.Sleep(time.Duration(rest))
		}
		log.Debug("send over")
	}()

	var total int64
	recv := simpletcp.NewFrame()
	buf := make([]byte, simpletcp.MaxLength)
	tcpConn.SetReadBuffer(4096 * 1024)
	for i := uint32(1); i <= count; i++ {
		log.Fataln(io.ReadFull(tcpConn, recv.Head()))
		recv.SetBody(buf[:recv.BodyLength()])
		log.Fataln(io.ReadFull(tcpConn, recv.Body))

		reserved := binary.BigEndian.Uint32(recv.Reserved())

		now := uint32(time.Now().UnixNano() / 1000)
		delta := int64(now - reserved)
		total += delta

		if i%(count/10) == 0 {
			log.Debugf("%d: %d %s, %dus", i, recv.MessageId(), recv.Body, delta)
		}
	}

	used := time.Now().Sub(start)

	log.Infof("Requests/sec: %d/%s = %d", count, used, time.Duration(count)*1e9/used)
	log.Infof("Latency: %s/%d = %s", time.Duration(total*1000), count,
		time.Duration(total*1000)/time.Duration(count))
}

var body = []byte("hellohellohellohellohellohellohellohellohellohellohellohello")
