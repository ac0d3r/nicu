package pingx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Pingxer struct {
	id             int
	seq            int
	conn           *icmp.PacketConn
	defaultMsgData []byte
	state          sync.Map
	result         chan string
}

func NewPingxer() (*Pingxer, error) {
	p := &Pingxer{
		id:  233,
		seq: 2333,
	}
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{ID: p.id, Seq: p.seq, Data: nil},
	}
	msgData, err := msg.Marshal(nil)
	if err != nil {
		return nil, err
	}
	p.defaultMsgData = msgData
	return p, nil
}

func (p *Pingxer) Scan(ctx context.Context, ipnet net.IPNet) ([]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var err error
	p.conn, err = icmp.ListenPacket("ip4:icmp", "")
	if err != nil {
		return nil, err
	}

	// generate ip
	ips := generateIP(ipnet)
	if len(ips) <= 0 {
		return nil, errors.New("no target")
	}

	// receive icmp message
	p.result = make(chan string, 50e1)
	go func() {
		if err := p.recvICMP(ctx); err != nil {
			fmt.Println(err)
		}
	}()

	// send icmp message
	ipaddr := &net.IPAddr{}
	for i := range ips {
		ipaddr.IP = ips[i]
		p.sendICMP(ipaddr)
		p.state.Store(ips[i].String(), struct{}{})
	}

	// try to repeat send icmp message
	var allFinished bool
	fn := func(key, value interface{}) bool {
		if count, ok := value.(int); ok {
			if ip, ok := key.(string); ok && count < 3 {
				ipaddr.IP = net.ParseIP(ip)
				p.sendICMP(ipaddr)
				count++
				p.state.Store(key, count)
				allFinished = allFinished && false
			} else {
				allFinished = allFinished && true
			}
		} else {
			allFinished = allFinished && false
			p.state.Store(key, 1)
		}
		return true
	}
	go func() {
		// end condition: IP with no message response after 3 retries || all IP have message response
		for !allFinished {
			time.Sleep(time.Second)
			allFinished = true
			p.state.Range(fn)
		}
		cancel()
	}()

	result := make([]string, 0)
	for res := range p.result {
		result = append(result, res)
	}
	return result, nil
}

func (p *Pingxer) sendICMP(target net.Addr) error {
	if _, err := p.conn.WriteTo(p.defaultMsgData, target); err != nil {
		if neterr, ok := err.(*net.OpError); ok &&
			neterr.Err == syscall.ENOBUFS {
			return nil
		}
		return err
	}
	return nil
}

var defaultDelay time.Duration = time.Millisecond * 100

func (p *Pingxer) recvICMP(ctx context.Context) error {
	delay := defaultDelay
	defer close(p.result)

	for {
		select {
		case <-ctx.Done():
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

			if _, loaded := p.state.LoadAndDelete(srcaddr.String()); loaded {
				p.result <- srcaddr.String()
			}
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

func generateIP(ipnet net.IPNet) []net.IP {
	ips := make([]net.IP, 0)
	start, end := IPRange(ipnet)
	startNum := int(start[12])<<24 | int(start[13])<<16 | int(start[14])<<8 | int(start[15])
	endNum := int(end[12])<<24 | int(end[13])<<16 | int(end[14])<<8 | int(end[15])
	for num := startNum; num <= endNum; num++ {
		ips = append(ips, net.IPv4(byte((num>>24)&0xff), byte((num>>16)&0xff), byte((num>>8)&0xff), byte(num&0xff)))
	}
	return ips
}

// IPRange calculate the start and end IP addresses according to the subnet mask
// thx: https://github.com/shadow1ng/fscan/blob/main/common/ParseIP.go
func IPRange(c net.IPNet) (net.IP, net.IP) {
	var (
		start, end net.IP
		ipIdx      int
		mask       byte
	)
	start = make(net.IP, len(c.IP))
	end = make(net.IP, len(c.IP))
	copy(start, c.IP)
	copy(end, c.IP)

	for i := 0; i < len(c.Mask); i++ {
		ipIdx = len(end) - 1 - i
		mask = c.Mask[len(c.Mask)-i-1]
		end[ipIdx] = c.IP[ipIdx] | ^mask
		start[ipIdx] = c.IP[ipIdx] & mask
	}

	return start, end
}
