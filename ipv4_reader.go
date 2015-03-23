package main

import (
	"errors"
	"fmt"
	"net"
	"time"
	//    "syscall"
	//"golang.org/x/net/ipv4"
)

type IP_Reader struct {
	incomingPackets <-chan []byte
	nr              *Network_Reader
	protocol        uint8
	ip              string
}

func (nr *Network_Reader) NewIP_Reader(ip string, protocol uint8) (*IP_Reader, error) {
	c, err := nr.bind(ip, protocol)
	if err != nil {
		return nil, err
	}

	return &IP_Reader{
		incomingPackets: c,
		nr:              nr,
		protocol:        protocol,
		ip:              ip,
	}, nil
}

func slicePacket(b []byte) (hrd, payload []byte) {
	hdrLen := int(b[0]&0x0f) * 4
	//fmt.Println("HdrLen: ", hdrLen)
	return b[:hdrLen], b[hdrLen:]
}

func (ipr *IP_Reader) ReadFrom() (ip string, b, payload []byte, e error) {
	b = <-ipr.incomingPackets
	//fmt.Println("Read Length: ", len(b))
	//fmt.Println("Full Read Data: ", b)

	ip = net.IPv4(b[12], b[13], b[14], b[15]).String()
	hdr, p := slicePacket(b)

	if hdr[6]>>5 == 0 { // if not fragment
		// verify checksum
		if calcChecksum(hdr, false) != 0 {
			//fmt.Println("Header checksum verification failed. Packet dropped.")
			//fmt.Println("Wrong header: ", hdr)
			//fmt.Println("Payload (dropped): ", p)
			return "", nil, nil, errors.New("Header checksum incorrect, packet dropped")
		}

		//fmt.Println("Payload Length: ", len(p))
		//fmt.Println("Full payload: ", p)

		return ip, b, p, nil
	} else {
		payload := p
		extraFrags := make(map[uint64]([]byte))
		t := time.Now()
		for time.Since(t).Seconds() < 0.25 {
			select {
			case frag := <-ipr.incomingPackets:
				hdr, p := slicePacket(frag)
				fmt.Println("RECEIVED FRAG")
				fmt.Println("Offset:", 8*(uint64(hdr[6]&0x1f)<<8+uint64(hdr[7])))
				fmt.Println(len(payload))
				if (int(hdr[6]&0x1f)<<8+int(hdr[7]))*8 == len(payload) {
					payload = append(payload, p...)
					for storedFrag, found := extraFrags[uint64(len(payload))]; found; {
						delete(extraFrags, uint64(len(payload)))
						payload = append(payload, storedFrag...)
					}
					if hdr[6]>>5 == 1 {
						return ip, b, payload, nil
					}
				} else {
					extraFrags[8*(uint64(p[6])<<3>>11+uint64(p[7]))] = p
				}
			}
		}
		return "", nil, nil, errors.New("Fragments took too long, packet dropped")
	}
}

func (ipr *IP_Reader) Close() error {
	return ipr.nr.unbind(ipr.ip, ipr.protocol)
}

/* h := &ipv4.Header{
	Version:  ipv4.Version,      // protocol version
	Len:      20,                // header length
	TOS:      0,                 // type-of-service (0 is everything normal)
	TotalLen: len(x) + 20,       // packet total length (octets)
	ID:       0,                 // identification
	Flags:    ipv4.DontFragment, // flags
	FragOff:  0,                 // fragment offset
	TTL:      8,                 // time-to-live (maximum lifespan in seconds)
	Protocol: 17,                // next protocol (17 is UDP)
	Checksum: 0,                 // checksum (apparently autocomputed)
	//Src:    net.IPv4(127, 0, 0, 1), // source address, apparently done automatically
	Dst: net.ParseIP(c.manager.ipAddress), // destination address
	//Options                         // options, extension headers
}
*/
