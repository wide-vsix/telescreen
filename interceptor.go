package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const (
	device      string        = "eth1"                // Where DNS packets are forwarded
	filter      string        = "udp and dst port 53" // Only capturing UDP DNS queries
	snaplen     int32         = 1600
	promiscuous bool          = true
	timeout     time.Duration = pcap.BlockForever
)

var (
	VERSION  string = "0.0.0"
	REVISION string = "develop"
	err      error
)

type QueryLogItem struct {
	timestamp time.Time
	srcIP     net.IP
	dstIP     net.IP
	srcPort   uint16
	dstPort   uint16
	query     string
	rrType    string
	overTCP   bool
}

func (q QueryLogItem) String() string {
	ts := q.timestamp.Format(time.RFC3339)
	src := fmt.Sprintf("%s.%d", q.srcIP.String(), q.srcPort)
	dst := fmt.Sprintf("%s.%d", q.dstIP.String(), q.dstPort)
	qtype := fmt.Sprintf("%s?", q.rrType)
	trans := "UDP"
	if q.overTCP {
		trans = "TCP"
	}
	return fmt.Sprintf("%s | %-43s > %-25s %s %-5s %s", ts, src, dst, trans, qtype, q.query)
}

func newQueryLogItem(packet gopacket.Packet) *QueryLogItem {
	q := new(QueryLogItem)
	q.timestamp = time.Now()

	if err := packet.ErrorLayer(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode some part of the packet: %v\n", err)
		return q
	}

	if ip6Layer := packet.Layer(layers.LayerTypeIPv6); ip6Layer != nil {
		ip6, _ := ip6Layer.(*layers.IPv6)
		q.srcIP = ip6.SrcIP
		q.dstIP = ip6.DstIP
	}

	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		q.srcPort = uint16(udp.SrcPort)
		q.dstPort = uint16(udp.DstPort)
	}

	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		q.srcPort = uint16(tcp.SrcPort)
		q.dstPort = uint16(tcp.DstPort)
		q.overTCP = true
	}

	if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
		dns, _ := dnsLayer.(*layers.DNS)
		for _, question := range dns.Questions {
			q.query = string(question.Name)
			q.rrType = question.Type.String()
		}
	}

	return q
}

func main() {
	displayBanner()

	handle, err := pcap.OpenLive(device, snaplen, promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	if err = handle.SetBPFFilter(filter); err != nil {
		log.Fatal(err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		item := newQueryLogItem(packet)
		fmt.Println(item.String())
	}
}

func displayBanner() {
	v := VERSION + "-" + REVISION
	fmt.Printf("Version: %s\n", v)
}
