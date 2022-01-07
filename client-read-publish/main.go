package main

import (
	"fmt"
	"sync"

	"github.com/aler9/gortsplib"
)

type Packet struct {
	TrackID int
	Payload []byte
}

func main() {
	var waitGroup sync.WaitGroup
	tracks := make(chan gortsplib.Tracks)
	rtpPacketChan := make(chan Packet, 10)
	rtcpPacketChan := make(chan Packet, 10)

	readURL := "rtsp://localhost:8554/ExampleStream"
	publishURL := "rtsp://localhost:8554/PublishedStream"

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		Reader(readURL, tracks, rtpPacketChan, rtcpPacketChan)
	}()
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		Publisher(publishURL, tracks, rtpPacketChan, rtcpPacketChan)
	}()

	waitGroup.Wait()
}

func Reader(url string, tracks chan<- gortsplib.Tracks, rtpPacketChan chan<- Packet, rtcpPacketChan chan<- Packet) {
	c := gortsplib.Client{
		OnPacketRTP: func(trackID int, payload []byte) {
			rtpPacketChan <- Packet{
				TrackID: trackID,
				Payload: payload,
			}
		},
		OnPacketRTCP: func(trackID int, payload []byte) {
			rtcpPacketChan <- Packet{
				TrackID: trackID,
				Payload: payload,
			}
		},
	}

	c.StartReading(url)
	tracks <- c.Tracks()
	panic(c.Wait())
}

func Publisher(url string, tracksChan <-chan gortsplib.Tracks, rtpPacketChan <-chan Packet, rtcpPacketChan <-chan Packet) {
	c := gortsplib.Client{}

	tracks := <-tracksChan
	fmt.Printf("Got %d tracks from tracksChan\n", len(tracks))

	err := c.StartPublishing(url, tracks)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	for {
		select {
		case rtpPacket := <-rtpPacketChan:
			err = c.WritePacketRTP(rtpPacket.TrackID, rtpPacket.Payload)
			if err != nil {
				panic(err)
			}

		case rtcpPacket := <-rtcpPacketChan:
			err = c.WritePacketRTCP(rtcpPacket.TrackID, rtcpPacket.Payload)
			if err != nil {
				panic(err)
			}
		}
	}
}
