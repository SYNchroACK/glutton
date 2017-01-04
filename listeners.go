package glutton

import (
	"bufio"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hectane/go-nonblockingchan"
	"github.com/synchroack/glutton/protocols"
)

var counters Connections

type bufferedConn struct {
	r        *bufio.Reader
	net.Conn // So that most methods are embedded
}

func newBufferedConn(c net.Conn) bufferedConn {
	return bufferedConn{bufio.NewReader(c), c}
}

func newBufferedConnSize(c net.Conn, n int) bufferedConn {
	return bufferedConn{bufio.NewReaderSize(c, n), c}
}

func (b bufferedConn) Peek(n int) ([]byte, error) {
	return b.r.Peek(n)
}

func (b bufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

func handleTCPClient(conn net.Conn, ch *nbc.NonBlockingChan, counter ConnCounter) {
	// Splitting address to compare with conntrack logs
	srcAddr := conn.RemoteAddr().String()
	if srcAddr == "<nil>" {
		log.Println("Error: handleTCPClient - Address:port == nil - conn.RemoteAddr().String()")
		return
	}

	addr := strings.Split(srcAddr, ":")

	dp := protocols.GetTCPDesPort(addr, ch)

	if dp == -1 {
		log.Println("Warning: Packet dropped! [TCP] destPort == -1")
		return
	}

	// TCP client for destination server
	// CHANGED: handler := GetHandler(dp)
	handler := portConf.Ports[dp]

	if len(handler) < 2 {
		log.Println("No explicit handler found")

		// CHANGED: handler = GetDefaultHandler()
		handler = portConf.Default

		if handler == "" {
			log.Println("No default handler found. Packet dropped!")
			return
		}
	}

	if strings.HasPrefix(handler, "handle") {
		if strings.HasSuffix(handler, "telnet") {
			log.Printf("New TCP connection from %s to port %d -> glutton:telnet\n", addr[0], dp)
			counter.incrCon()
			protocols.HandleTelnet(time.Now().Unix(), conn)
			counter.decrCon()
		}
		if strings.HasSuffix(handler, "default") {
			log.Printf("New TCP connection from %s to port %d -> glutton:default\n", addr[0], dp)
			counter.incrCon()
			bufConn := newBufferedConn(conn)
			snip, err := bufConn.Peek(4)
			if err != nil {
				log.Println(err)
			}
			httpMap := map[string]bool{"GET ": true, "POST": true, "HEAD": true}
			if _, ok := httpMap[string(snip)]; ok == true {
				log.Println("Handling HTTP")
				protocols.HandleHTTP(bufConn)
			} else {
				protocols.HandleDefault(conn)
			}
			counter.decrCon()
		}
	}

	if strings.HasPrefix(handler, "proxy") {
		proxyConn := protocols.TCPClient(handler[6:])
		if proxyConn == nil {
			return
		}

		log.Printf("New TCP connection from %s to port %d -> glutton:Proxy\n", addr[0], dp)
		counter.incrCon()

		// Data Transfer between Connections
		clossedBy, err := protocols.ProxyServer(time.Now().Unix(), conn.(*net.TCPConn), proxyConn)
		counter.connectionClosed(srcAddr, handler[6:], clossedBy, err)
	}
}

// TCPListener listens for new TCP connections
func TCPListener(ch *nbc.NonBlockingChan, wg *sync.WaitGroup) {
	defer wg.Done()

	counters = Connections{}

	service := ":5000"

	addr, err := net.ResolveTCPAddr("tcp", service)
	CheckError("[*] ResolveTCPAddr Error.", err)

	// Listener for incoming TCP connections
	listener, err := net.ListenTCP("tcp", addr)
	CheckError("[*] Error in net.ListenTCP", err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		// Goroutines to handle multiple connections
		go handleTCPClient(conn, ch, &counters)
	}
}

func handleUDPClient(conn *net.UDPConn, ch *nbc.NonBlockingChan) {

	for {
		var b [1500]byte
		n, addr, err := conn.ReadFromUDP(b[0:])
		if err != nil {
			return
		}

		c := UDPConn{conn, addr, ch, b, n}
		go c.UDPBroker(portConf) //, &counters)
	}
}

// UDPListener listens for new UDP connections
func UDPListener(ch *nbc.NonBlockingChan, wg *sync.WaitGroup) {
	defer wg.Done()

	service := ":5000"

	addr, err := net.ResolveUDPAddr("udp", service)
	CheckError("[*] Error in UDP address resolving", err)

	// Listener for incoming UDP connections
	conn, err := net.ListenUDP("udp", addr)
	CheckError("[*] Error in UDP listener", err)

	handleUDPClient(conn, ch)

}
