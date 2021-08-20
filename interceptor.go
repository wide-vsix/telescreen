package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	flag "github.com/spf13/pflag"
)

const (
	_device     string        = "vsix"                //Where DNS packets are forwarded
	filter      string        = "udp and dst port 53" // Only capturing UDP DNS queries
	snaplen     int32         = 1600
	promiscuous bool          = true
	timeout     time.Duration = pcap.BlockForever
)

var (
	VERSION     string = "0.0.0"
	REVISION    string = "develop"
	device      string // Where DNS packets are forwarded
	dbAddr      string // Postgresql: IP address and port number pair
	dbName      string // Postgresql: Database name
	dbUser      string // Postgresql: Login username
	dbPassFile  string // Postgresql: Login password file
	quietFlag   bool
	helpFlag    bool
	versionFlag bool
	err         error
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

func stdExporter(q *QueryLog) {
	fmt.Println(q.Colorize())
}

func newDBExporter(options *pg.Options) (func(q *QueryLog), func()) {
	db := pg.Connect(options)
	schema := (*QueryLog)(nil)
	db.Model(schema).CreateTable(&orm.CreateTableOptions{
		IfNotExists:   true,
		FKConstraints: true,
	})

	exporter := func(q *QueryLog) {
		if _, err = db.Model(q).Insert(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to issue INSERT: %v\n", err)
		}
	}

	closer := func() {
		db.Close()
		fmt.Println("DB connection closed")
	}

	return exporter, closer
}

func interceptor(exporters []func(*QueryLog)) {
	handle, err := pcap.OpenLive(device, snaplen, promiscuous, timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start capturing: %v\n", err)
		return
	}
	defer handle.Close()

	if err = handle.SetBPFFilter(filter); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set BPF filter: %v\n", err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		q := newQueryLog(packet)
		for _, exporter := range exporters {
			exporter(q)
		}
	}
}

func init() {
	flag.StringVarP(&device, "dev", "i", "", "Caputing interface name")
	flag.BoolVarP(&quietFlag, "quiet", "q", false, "Suppress standard output")
	flag.StringVar(&dbAddr, "db-addr", "", "Postgresql server address to store queries (e.g., localhost:5432)")
	flag.StringVar(&dbName, "db-name", "", "Database name to store queries")
	flag.StringVar(&dbUser, "db-user", "", "Username to login DB")
	flag.StringVar(&dbPassFile, "db-password-file", "", "Path of plaintext password file")
	flag.BoolVarP(&helpFlag, "help", "h", false, "Show help message")
	flag.BoolVarP(&versionFlag, "version", "v", false, "Show build version")
	flag.CommandLine.SortFlags = false
}

func main() {
	flag.Parse()

	if versionFlag {
		fmt.Println(VERSION + "-" + REVISION)
		os.Exit(0)
	}

	if helpFlag || device == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}

	exporters := []func(*QueryLog){}

	if dbAddr != "" && dbName != "" && dbUser != "" && dbPassFile != "" {
		f, err := os.Open(dbPassFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open DB password file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		b, err := ioutil.ReadAll(f)
		password := string(b)
		dbExporter, dbCloser := newDBExporter(&pg.Options{
			Addr:     dbAddr,
			User:     dbUser,
			Password: password,
			Database: dbName,
		})

		exporters = append(exporters, dbExporter)
		defer dbCloser()
	}

	if !quietFlag {
		exporters = append(exporters, stdExporter)
	}

	interceptor(exporters)
}
