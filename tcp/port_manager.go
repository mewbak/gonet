package tcp

import (
	"errors"
	"network/ipv4"

	"network/ipv4/ipv4tps"

	"github.com/hsheth2/logs"
)

// Global src, dst port and ip registry for TCP binding
type TCP_Port_Manager_Type struct {
	tcp_reader *ipv4.IP_Reader
	incoming   map[uint16](map[uint16](map[ipv4tps.IPhash](chan *TCP_Packet))) // dst, src port, remote ip
}

// TODO TCP_Port_Manager_Type should have an unbind function
func (m *TCP_Port_Manager_Type) bind(rport, lport uint16, ip *ipv4tps.IPaddress) (chan *TCP_Packet, error) {
	// lport is the local one here, rport is the remote
	if _, ok := m.incoming[lport]; !ok {
		m.incoming[lport] = make(map[uint16](map[ipv4tps.IPhash](chan *TCP_Packet)))
	}

	// TODO add an option (for servers) for all srcports
	if _, ok := m.incoming[lport][rport]; !ok {
		m.incoming[lport][rport] = make(map[ipv4tps.IPhash](chan *TCP_Packet))
	}

	if _, ok := m.incoming[lport][rport][ip.Hash()]; ok {
		return nil, errors.New("Ports and IP already binded to")
	}

	ans := make(chan *TCP_Packet, TCP_INCOMING_BUFF_SZ)
	m.incoming[lport][rport][ip.Hash()] = ans
	return ans, nil
}

func (m *TCP_Port_Manager_Type) readAll() {
	for {
		rip, lip, _, payload, err := m.tcp_reader.ReadFrom()
		if err != nil {
			logs.Error.Println("TCP readAll error", err)
			continue
		}

		p, err := Extract_TCP_Packet(payload, rip, lip)
		if err != nil {
			logs.Error.Println(err)
			continue
		}

		rport := p.header.srcport
		lport := p.header.dstport

		var output chan *TCP_Packet = nil

		//logs.Trace.Printf("readAll tcp packet manager dealing with packet or rport: %d and lport %d", rport, lport)
		if _, ok := m.incoming[lport]; ok {
			//logs.Trace.Printf("readAll: promising packet rport: %d and lport %d", rport, lport)
			if p, ok := m.incoming[lport][rport]; ok {
				//logs.Trace.Println("readAll: exact port number match")
				if x, ok := p[rip.Hash()]; ok {
					output = x
				} else if x, ok := p[ipv4tps.IP_ALL_HASH]; ok {
					output = x
				}
			} else if p, ok := m.incoming[lport][0]; ok {
				//logs.Trace.Println("readAll: forwarding to a listening server")
				if x, ok := p[ipv4tps.IP_ALL_HASH]; ok {
					output = x
				} else if x, ok := p[rip.Hash()]; ok {
					output = x
				}
			}
		}

		if output != nil {
			go func() { output <- p }()
		} else {
			// TODO send a rst to sender if nothing is binded to the dst port, src port, and remote ip
			//fmt.Println(errors.New("Dst/Src port + ip not binded to"))
		}
	}
}

var TCP_Port_Manager = func() *TCP_Port_Manager_Type {
	irm := ipv4.GlobalIPReadManager

	ipr, err := ipv4.NewIP_Reader(irm, ipv4tps.IP_ALL, ipv4.TCP_PROTO)
	if err != nil {
		logs.Error.Println(err)
		return nil
	}

	m := &TCP_Port_Manager_Type{
		tcp_reader: ipr,
		incoming:   make(map[uint16](map[uint16](map[ipv4tps.IPhash](chan *TCP_Packet)))),
	}
	go m.readAll()
	return m
}()