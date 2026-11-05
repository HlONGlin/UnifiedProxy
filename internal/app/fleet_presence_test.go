package app

import (
	"github.com/hlonglin/unifiedproxy/internal/fleet"
	"testing"
)

func TestFleetAutomaticAddressFeedsSubscriptions(t *testing.T) {
	a := fleetTestApp(t)
	n, err := a.fleet.store.AddNode("auto", "")
	if err != nil {
		t.Fatal(err)
	}
	f := a.fleet
	f.mu.Lock()
	ed, err := f.editor(n.ID)
	f.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	in := Inbound{ID: "auto-in", Name: "socks", Type: "socks", Listen: "0.0.0.0", Port: 12345, Enabled: true, Clients: []Client{{ID: "alice", Name: "alice", Username: "alice", Password: "password", Enabled: true}}}
	if err = ed.manager.saveInbound(in); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	_, err = f.publish(n.ID, ed, nil)
	f.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if err = a.refreshFleetSubscriptions(); err != nil {
		t.Fatal(err)
	}
	a.manager.mu.Lock()
	path := subscriptionPublicPath(a.manager.state.Settings, a.manager.state.SubscriptionTokens["alice"])
	a.manager.mu.Unlock()
	assertAddress := func(want string) {
		t.Helper()
		snapshot, _, _ := a.fleetSubscriptionSnapshot(path)
		if want == "" {
			if len(snapshot.Clients) != 0 {
				t.Fatal("unresolved address exported")
			}
			return
		}
		if len(snapshot.Clients) != 1 || snapshot.Clients[0].Server != want {
			t.Fatal("subscription address mismatch", snapshot.Clients)
		}
	}
	assertAddress("")
	if err = f.store.UpdateStatus(n.ID, fleet.Status{PublicIP: "1.1.1.1"}); err != nil {
		t.Fatal(err)
	}
	assertAddress("1.1.1.1")
	if err = f.store.UpdateNode(n.ID, n.Name, "proxy.example"); err != nil {
		t.Fatal(err)
	}
	assertAddress("proxy.example")
	if err = f.store.UpdateNode(n.ID, n.Name, ""); err != nil {
		t.Fatal(err)
	}
	assertAddress("1.1.1.1")
}
