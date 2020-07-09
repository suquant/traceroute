package traceroute

import (
	"context"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"os"
	"time"
)

type Options struct {
	MaxTTL   int
	WaitTime time.Duration
}

type Result struct {
	Hop     int
	Src     net.IP
	Dst     net.IP
	Rtt     time.Duration
	Success bool
}

func Traceroute(ctx context.Context, dst net.IPAddr, opts Options) ([]Result, error) {
	c, err := net.ListenPacket("ip4:1", "0.0.0.0")
	if err != nil {
		return nil, fmt.Errorf("%w: network listen packet", err)
	}
	defer c.Close()

	p := ipv4.NewPacketConn(c)
	if err := p.SetControlMessage(ipv4.FlagTTL|ipv4.FlagSrc|ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
		return nil, fmt.Errorf("%w: set control message", err)
	}

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getegid() & 0xfff,
			Data: []byte("HELLO-R-U-THERE"),
		},
	}

	var results []Result
	rb := make([]byte, 1500)
	for i := 1; i <= opts.MaxTTL; i++ {
		results = append(results, Result{
			Hop: i,
		})

		if err := ctx.Err(); err != nil {
			return results, err
		}

		wm.Body.(*icmp.Echo).Seq = i
		wb, err := wm.Marshal(nil)
		if err != nil {
			return results, fmt.Errorf("%w: marhsal icmp message", err)
		}
		if err := p.SetTTL(i); err != nil {
			return results, fmt.Errorf("%w: set ttl", err)
		}

		// In the real world usually there are several
		// multiple traffic-engineered paths for each hop.
		// You may need to probe a few times to each hop.
		begin := time.Now()
		if _, err := p.WriteTo(wb, nil, &dst); err != nil {
			return results, fmt.Errorf("%w: write icmp message", err)
		}
		if err := p.SetReadDeadline(time.Now().Add(opts.WaitTime)); err != nil {
			return results, fmt.Errorf("%w: set read deadline", err)
		}

		n, cm, _, err := p.ReadFrom(rb)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				continue
			}
			return results, fmt.Errorf("%w: read bytes", err)
		}

		results[i-1].Src = cm.Src
		results[i-1].Dst = cm.Dst

		rm, err := icmp.ParseMessage(1, rb[:n])
		if err != nil {
			return nil, fmt.Errorf("%w: parse icmp message", err)
		}
		results[i-1].Rtt = time.Since(begin)
		results[i-1].Success = true

		// In the real world you need to determine whether the
		// received message is yours using ControlMessage.Src,
		// ControlMessage.Dst, icmp.Echo.ID and icmp.Echo.Seq.
		switch rm.Type {
		case ipv4.ICMPTypeTimeExceeded:
			continue
		case ipv4.ICMPTypeEchoReply:
			return results, nil
		default:
			results[i-1].Success = false
			return results, fmt.Errorf("unknown ICMP message: %+v\n", rm)
		}
	}

	return results, nil
}
