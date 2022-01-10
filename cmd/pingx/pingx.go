package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Buzz2d0/nicu/pkg/network"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func banner() {
	t := `
 ____  ____  _  _  ___  _  _ 
(  _ \(_  _)( \( )/ __)( \/ )
 )___/ _)(_  )  (( (_-. )  ( 
(__)  (____)(_)\_)\___/(_/\_)
`
	fmt.Println(t)
}

func main() {
	banner()
	fmt.Println("load local interface...")

	ipnets, err := network.GetLocalIPV4Net()
	if err != nil || len(ipnets) < 1 {
		log.Fatal(err)
	}
	pingxer := NewPingxer()
	if err := pingxer.Listen(); err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pingxer.recvICMP(stop); err != nil {
			return
		}
	}()

	target := make(chan net.IP, 10)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range ipnets {
			genip(ipnets[i], target)
		}
	}()

	for ip := range target {
		err := pingxer.sendICMP(&net.IPAddr{IP: ip, Zone: ""})
		if err != nil {
			fmt.Println(err)
		}
	}
	wg.Wait()
	pingxer.conn.Close()
}

func genip(ipnet net.IPNet, target chan<- net.IP) {
	fmt.Printf("load ipnet: ip=%s,mask=%s\n", ipnet.IP.String(), ipnet.Mask.String())
	defer close(target)

	start, end := network.IPRange(ipnet)
	startNum := int(start[12])<<24 | int(start[13])<<16 | int(start[14])<<8 | int(start[15])
	endNum := int(end[12])<<24 | int(end[13])<<16 | int(end[14])<<8 | int(end[15])
	for num := startNum; num <= endNum; num++ {
		target <- net.IPv4(byte((num>>24)&0xff), byte((num>>16)&0xff), byte((num>>8)&0xff), byte(num&0xff))
	}
}

var (
	defaultDelay time.Duration = time.Millisecond * 100
)

type Pingxer struct {
	id   int
	seq  int
	conn *icmp.PacketConn
}

func NewPingxer() *Pingxer {
	return &Pingxer{
		id:  233,
		seq: 2333,
	}
}

func (p *Pingxer) Listen() error {
	conn, err := icmp.ListenPacket("ip4:icmp", "")
	if err != nil {
		return err
	}

	p.conn = conn
	return nil
}

func (p *Pingxer) sendICMP(target net.Addr) error {
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   p.id,
			Seq:  p.seq,
			Data: nil,
		},
	}

	msgData, err := msg.Marshal(nil)
	if err != nil {
		return err
	}
	if _, err := p.conn.WriteTo(msgData, target); err != nil {
		if neterr, ok := err.(*net.OpError); ok {
			if neterr.Err == syscall.ENOBUFS {
				return nil
			}
		}
		return err
	}
	return nil
}

func (p *Pingxer) recvICMP(done chan os.Signal) error {
	delay := defaultDelay
	for {
		select {
		case <-done:
			return nil
		default:
			if err := p.conn.SetReadDeadline(time.Now().Add(delay)); err != nil {
				return err
			}
			// TODO data size
			data := make([]byte, 1024)
			n, srcaddr, err := p.conn.ReadFrom(data)
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						// Read timeout
						delay = time.Second
						continue
					}
				}
				return err
			} else {
				delay = defaultDelay
			}

			if n <= 0 {
				continue
			}

			ok, err := p.messageFiltering(data[:])
			if !ok {
				if err != nil {
					fmt.Printf("recvIcmp - processPacket error:%s\n", err)
				}
				continue
			}
			fmt.Printf("[+] %s\n", srcaddr.String())
		}
	}
}

func (p *Pingxer) messageFiltering(bytes []byte) (bool, error) {
	var (
		m   *icmp.Message
		err error
	)
	if m, err = icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), bytes); err != nil {
		return false, fmt.Errorf("error parsing icmp message: %w", err)
	}
	if m.Type != ipv4.ICMPTypeEchoReply {
		return false, nil
	}

	switch pkt := m.Body.(type) {
	case *icmp.Echo:
		if p.id != pkt.ID || p.seq != pkt.Seq {
			return false, nil
		}
	default:
		return false, fmt.Errorf("invalid ICMP echo reply; type: '%T', '%v'", pkt, pkt)
	}
	return true, nil
}
