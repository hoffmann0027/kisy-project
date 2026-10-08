package dashboard

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestStunAddr(t *testing.T) {
	for in, want := range map[string]string{
		"turn:188.245.179.84:3478?transport=udp": "188.245.179.84:3478",
		"turn:relay.example:3478":                "relay.example:3478",
		"stun:relay.example":                     "relay.example:3478",
	} {
		got, err := stunAddr(in)
		if err != nil || got != want {
			t.Fatalf("%s: got %q %v, want %q", in, got, err, want)
		}
	}
	if _, err := stunAddr("https://relay.example"); err == nil {
		t.Fatal("accepted a non-STUN url")
	}
}

// A relay that answers a binding request is up; silence is down.
func TestStunProbe(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	go func() {
		buf := make([]byte, 512)
		for {
			n, from, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			if n < 20 {
				continue
			}
			resp := make([]byte, 20)
			binary.BigEndian.PutUint16(resp[0:], 0x0101)
			copy(resp[4:20], buf[4:20]) // cookie and transaction id
			_, _ = pc.WriteTo(resp, from)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := stunProbe(ctx, "turn:"+pc.LocalAddr().String()+"?transport=udp"); err != nil {
		t.Fatalf("a live relay looked down: %v", err)
	}

	silent, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel2()
	if err := stunProbe(ctx2, "turn:"+silent.LocalAddr().String()); err == nil {
		t.Fatal("a silent relay looked up")
	}
}

func TestSeverity(t *testing.T) {
	for reason, want := range map[string]string{"fraud": "high", "illegal": "high", "abuse": "medium", "spam": "low", "other": "low"} {
		if got := severity(reason); got != want {
			t.Fatalf("%s: %s, want %s", reason, got, want)
		}
	}
}
