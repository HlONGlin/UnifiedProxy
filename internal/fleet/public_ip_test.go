package fleet

import "testing"

func TestPublicIP(t *testing.T) {
	for _, value := range []string{"", "<html>8.8.8.8</html>", "127.0.0.1", "10.0.0.1", "100.64.0.1", "192.168.1.1", "169.254.0.1", "::1", "fe80::1%eth0", "fc00::1", "224.0.0.1", "203.0.113.1", "2001:db8::1", "8.8.8.8 1.1.1.1", "example.com"} {
		if got := PublicIP(value); got != "" {
			t.Errorf("accepted %q: %s", value, got)
		}
	}
	for _, value := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		if got := PublicIP("\n" + value + "\n"); got != value {
			t.Errorf("rejected %s: %s", value, got)
		}
	}
	if PublicIP("::ffff:8.8.8.8") != "8.8.8.8" {
		t.Fatal("mapped IPv4 not normalized")
	}
}
