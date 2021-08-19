package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

const (
	device       string        = "eth1"
	filter       string        = "dst port 53" // Only capturing DNS query packets
	snapshot_len int32         = 262144        // The same default as tcpdump
	promiscuous  bool          = true
	timeout      time.Duration = 30 * time.Second
)

var (
	err    error
	handle *pcap.Handle
)

func main() {
	handle, err = pcap.OpenLive(device, snapshot_len, promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	if err = handle.SetBPFFilter(filter); err != nil {
		log.Fatal(err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		fmt.Println(packet)
	}
}
