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
	filter      string        = "port 53" // Only capturing DNS packets, both queries and responses
	snaplen     int32         = 1600
	promiscuous bool          = true
	timeout     time.Duration = pcap.BlockForever
)

var (
	VERSION       string = "0.0.0"
	REVISION      string = "develop"
	device        string // Where DNS packets are forwarded
	dbAddr        string // Postgresql: IP address and port number pair
	dbName        string // Postgresql: Database name
	dbUser        string // Postgresql: Login username
	dbPassFile    string // Postgresql: Login password file
	quietFlag     bool
	containerFlag bool
	helpFlag      bool
	sniffFlag     bool
	versionFlag   bool
	err           error
	errCounter    uint16
)

type telescreenLog interface {
	String() string
	Colorize() string
}

type telescreenLogCommon struct {
	Timestamp time.Time `pg:"received_at"`
	SrcIP     net.IP    `pg:"src_ip"`
	DstIP     net.IP    `pg:"dst_ip"`
	SrcPort   uint16    `pg:"src_port"`
	DstPort   uint16    `pg:"dst_port"`
	TransTCP  bool      `pg:"tcp_transport,notnull,use_zero"`
}

type QueryLog struct {
	telescreenLogCommon
	QString   string `pg:"query_string"`
	QType     string `pg:"query_type"`
	hasAnswer bool   `pg:"-"`
}

type ResponseLog struct {
	QueryLog
	AnsIP     net.IP `pg:"answer_ip"`
	IPv6Ready bool   `pg:"ipv6_ready,notnull,use_zero"`
}

func (q *QueryLog) String() string {
	ts := q.Timestamp.Format(time.RFC3339)
	src := fmt.Sprintf("%s.%d", q.SrcIP.String(), q.SrcPort)
	dst := fmt.Sprintf("%s.%d", q.DstIP.String(), q.DstPort)
	qtype := fmt.Sprintf("%s?", q.QType)
	trans := "UDP"
	if q.TransTCP {
		trans = "TCP"
	}
	return fmt.Sprintf("%s | %-43s > %-25s %s %-8s %s", ts, src, dst, trans, qtype, q.QString)
}

func (q *QueryLog) Colorize() string {
	switch q.QType {
	case "A":
		return fmt.Sprintf("\033[0;31m%s\033[0m", q.String())
	case "AAAA":
		return fmt.Sprintf("\033[0;32m%s\033[0m", q.String())
	default:
		return q.String()
	}
}

func (r *ResponseLog) String() string {
	ts := r.Timestamp.Format(time.RFC3339)
	src := fmt.Sprintf("%s.%d", r.SrcIP.String(), r.SrcPort)
	dst := fmt.Sprintf("%s.%d", r.DstIP.String(), r.DstPort)
	qtype := fmt.Sprintf("%s?", r.QType)
	trans := "UDP"
	if r.TransTCP {
		trans = "TCP"
	}
	return fmt.Sprintf("%s | %-43s < %-25s %s %-8s %s (%s)", ts, dst, src, trans, qtype, r.QString, r.AnsIP)
}

func (r *ResponseLog) Colorize() string {
	if r.IPv6Ready {
		return fmt.Sprintf("\033[0;34m%s\033[0m", r.String())
	}
	return fmt.Sprintf("\033[0;35m%s\033[0m", r.String())
}

func newTelescreenLogCommon(packet gopacket.Packet) *telescreenLogCommon {
	c := new(telescreenLogCommon)
	c.Timestamp = time.Now()

	if err := packet.ErrorLayer(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode some part of the packet: %v\n", err)
		return nil
	}

	if ip6Layer := packet.Layer(layers.LayerTypeIPv6); ip6Layer != nil {
		ip6, _ := ip6Layer.(*layers.IPv6)
		c.SrcIP = ip6.SrcIP
		c.DstIP = ip6.DstIP
	} else {
		return nil
	}

	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		c.SrcPort = uint16(udp.SrcPort)
		c.DstPort = uint16(udp.DstPort)
	}

	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		c.SrcPort = uint16(tcp.SrcPort)
		c.DstPort = uint16(tcp.DstPort)
		c.TransTCP = true
	}

	return c
}

func newQueryLog(packet gopacket.Packet, c *telescreenLogCommon) *QueryLog {
	q := new(QueryLog)
	q.telescreenLogCommon = *c

	if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
		dns, _ := dnsLayer.(*layers.DNS)
		if len(dns.Questions) > 0 {
			question := dns.Questions[0]
			q.QString = string(question.Name)
			q.QType = question.Type.String()
			q.hasAnswer = len(dns.Answers) > 0
			return q
		}
	}

	return nil
}

func newResponseLog(packet gopacket.Packet, q *QueryLog) *ResponseLog {
	r := new(ResponseLog)
	r.QueryLog = *q
	_, nat64_prefix, _ := net.ParseCIDR("64:ff9b::/96")

	if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
		dns, _ := dnsLayer.(*layers.DNS)
		if len(dns.Answers) > 0 {
			answer := dns.Answers[0]
			r.AnsIP = answer.IP
			r.IPv6Ready = !nat64_prefix.Contains(r.AnsIP)
			r.hasAnswer = answer.IP != nil
			return r
		}
	}

	return nil
}

func stdExporter(qr telescreenLog) {
	if qr != nil {
		fmt.Println(qr.Colorize())
	}
}

func newDBExporter(options *pg.Options) (func(qr telescreenLog), func()) {
	db := pg.Connect(options)
	schemas := []interface{}{
		(*QueryLog)(nil),
		(*ResponseLog)(nil),
	}
	for _, schema := range schemas {
		db.Model(schema).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
	}

	exporter := func(qr telescreenLog) {
		if _, err = db.Model(qr).Insert(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to issue INSERT: %v\n", err)
			errCounter += 1
			if errCounter > 5 {
				fmt.Fprintf(os.Stderr, "Exit with DB connection problem\n")
				os.Exit(1)
			}
			return
		}
		errCounter = 0
	}

	closer := func() {
		fmt.Println("Closing database connection...")
		db.Close()
	}

	return exporter, closer
}

func telescreen(exporters []func(telescreenLog)) {
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
		c := newTelescreenLogCommon(packet)
		if c == nil {
			continue
		}
		q := newQueryLog(packet, c)
		if q == nil {
			continue
		}
		r := newResponseLog(packet, q)

		is_valid_query := c.DstPort == 53 && q != nil
		is_valid_response := c.SrcPort == 53 && r != nil
		has_aaaa_answer := is_valid_response && r.QType == "AAAA" && r.hasAnswer

		var log telescreenLog = q
		switch {
		case !is_valid_query && !has_aaaa_answer:
			continue
		case sniffFlag && has_aaaa_answer:
			log = r
		}

		for _, exporter := range exporters {
			exporter(log)
		}
	}
}

func init() {
	flag.StringVarP(&device, "dev", "i", "", "Interface name")
	flag.BoolVarP(&quietFlag, "quiet", "q", false, "Suppress standard output")
	flag.BoolVarP(&sniffFlag, "with-response", "A", false, "Store responses to AAAA queries")
	flag.StringVarP(&dbAddr, "db-host", "H", "", "Postgres server address to store logs (e.g., localhost:5432)")
	flag.StringVarP(&dbName, "db-name", "N", "", "Database name to store")
	flag.StringVarP(&dbUser, "db-user", "U", "", "Username to login")
	flag.StringVarP(&dbPassFile, "db-password-file", "P", "", "Password to login - path of a plaintext password file")
	flag.BoolVarP(&containerFlag, "container", "c", false, "Run inside a container - load options from environment variables")
	flag.BoolVarP(&helpFlag, "help", "h", false, "Show help message")
	flag.BoolVarP(&versionFlag, "version", "v", false, "Show build version")
	flag.CommandLine.SortFlags = false
}

func main() {
	flag.Parse()

	exporters := []func(telescreenLog){}

	if containerFlag {
		device = os.Getenv("TELESCREEN_DEVICE")
		dbAddr = os.Getenv("TELESCREEN_DB_HOST")
		dbName = os.Getenv("TELESCREEN_DB_NAME")
		dbUser = os.Getenv("TELESCREEN_DB_USER")
		dbPassFile = os.Getenv("TELESCREEN_DB_PASSWORD_FILE")
		quietFlag = true
		switch os.Getenv("TELESCREEN_STORE_RESPONSES") {
		case "yes", "Yes", "YES", "true", "True", "TRUE":
			sniffFlag = true
		default:
			sniffFlag = true
		}
	}

	if versionFlag {
		fmt.Println(VERSION + "-" + REVISION)
		os.Exit(0)
	}

	show_help := helpFlag || device == ""
	if show_help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if !quietFlag {
		exporters = append(exporters, stdExporter)
	}

	use_psql := dbAddr != "" && dbName != "" && dbUser != "" && dbPassFile != ""
	if use_psql {
		f, err := os.Open(dbPassFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open password file for DB login: %v\n", err)
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

		fmt.Printf("Prepared database connection: %s", dbAddr)
		exporters = append(exporters, dbExporter)
		defer dbCloser()
	}

	telescreen(exporters)
}
