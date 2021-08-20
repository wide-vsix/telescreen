package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
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

type QueryLog struct {
	Timestamp time.Time `db:"received_at"`
	SrcIP     net.IP    `db:"src_ip"`
	DstIP     net.IP    `db:"dst_ip"`
	SrcPort   uint16    `db:"src_port"`
	DstPort   uint16    `db:"dst_port"`
	Query     string    `db:"query_string"`
	RRType    string    `db:"query_type"`
	OverTCP   bool      `db:"query_over_tcp"`
}

func (q QueryLog) String() string {
	ts := q.Timestamp.Format(time.RFC3339)
	src := fmt.Sprintf("%s.%d", q.SrcIP.String(), q.SrcPort)
	dst := fmt.Sprintf("%s.%d", q.DstIP.String(), q.DstPort)
	qtype := fmt.Sprintf("%s?", q.RRType)
	trans := "UDP"
	if q.OverTCP {
		trans = "TCP"
	}
	return fmt.Sprintf("%s | %-43s > %-25s %s %-5s %s", ts, src, dst, trans, qtype, q.Query)
}

func (q QueryLog) Colorize() string {
	switch q.RRType {
	case "A":
		return fmt.Sprintf("\033[31m%s\033[0m", q.String())
	case "AAAA":
		return fmt.Sprintf("\033[32m%s\033[0m", q.String())
	default:
		return q.String()
	}
}

func newQueryLog(packet gopacket.Packet) *QueryLog {
	q := new(QueryLog)
	q.Timestamp = time.Now()

	if err := packet.ErrorLayer(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode some part of the packet: %v\n", err)
		return q
	}

	if ip6Layer := packet.Layer(layers.LayerTypeIPv6); ip6Layer != nil {
		ip6, _ := ip6Layer.(*layers.IPv6)
		q.SrcIP = ip6.SrcIP
		q.DstIP = ip6.DstIP
	}

	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		q.SrcPort = uint16(udp.SrcPort)
		q.DstPort = uint16(udp.DstPort)
	}

	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		q.SrcPort = uint16(tcp.SrcPort)
		q.DstPort = uint16(tcp.DstPort)
		q.OverTCP = true
	}

	if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
		dns, _ := dnsLayer.(*layers.DNS)
		for _, question := range dns.Questions {
			q.Query = string(question.Name)
			q.RRType = question.Type.String()
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

	db := pg.Connect(&pg.Options{
		Addr:     "localhost:5432",
		User:     "vsix",
		Password: "changeme",
		Database: "interception",
	})
	defer db.Close()

	schema := (*QueryLog)(nil)
	db.Model(schema).CreateTable(&orm.CreateTableOptions{
		IfNotExists:   true,
		FKConstraints: true,
	})

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		item := newQueryLog(packet)
		fmt.Println(item.Colorize())
		if _, err = db.Model(item).Insert(); err != nil {
			panic(err)
		}
	}
}

func displayBanner() {
	v := VERSION + "-" + REVISION
	fmt.Printf("Version: %s\n", v)
}
