package ping

import (
	"testing"
	"time"

	"network/ipv4/ipv4tps"

	"github.com/hsheth2/logs"
)

func ping_tester(t *testing.T, ip *ipv4tps.IPaddress, num uint16) {
	err := GlobalPingManager.SendPing(ip, time.Second, time.Second, num)
	if err != nil {
		logs.Error.Println(err)
		t.Error(err)
	} else {
		t.Log("Success")
	}
	time.Sleep(500 * time.Millisecond)
}

func TestLocalPing(t *testing.T) {
	ping_tester(t, ipv4tps.MakeIP("127.0.0.1"), 5)
}

func TestTapPing(t *testing.T) {
	ping_tester(t, ipv4tps.MakeIP("10.0.0.2"), 5)
}

func TestExternalPing(t *testing.T) {
	ping_tester(t, ipv4tps.MakeIP("192.168.1.2"), 5) // TODO decide dynamically based on ip address
}