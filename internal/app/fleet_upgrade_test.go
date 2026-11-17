package app

import (
	"reflect"
	"testing"
)

func TestFleetServiceCommandSupportsSystemdAndOpenRC(t *testing.T) {
	t.Setenv("UNIFIEDPROXY_SERVICE_MANAGER", "openrc")
	target, err := resolveFleetService("unifiedproxy-node")
	if err != nil {
		t.Fatal(err)
	}
	command, args, err := fleetServiceCommand("is-active", target)
	if err != nil {
		t.Fatal(err)
	}
	if command != "rc-service" || !reflect.DeepEqual(args, []string{"unifiedproxy-node", "status"}) {
		t.Fatalf("unexpected OpenRC command: %s %v", command, args)
	}

	t.Setenv("UNIFIEDPROXY_SERVICE_MANAGER", "")
	target, err = resolveFleetService("unifiedproxy-control.service")
	if err != nil {
		t.Fatal(err)
	}
	command, args, err = fleetServiceCommand("restart", target)
	if err != nil {
		t.Fatal(err)
	}
	if command != "systemctl" || !reflect.DeepEqual(args, []string{"restart", "unifiedproxy-control.service"}) {
		t.Fatalf("unexpected systemd command: %s %v", command, args)
	}
}

func TestFleetServiceRejectsUnknownManager(t *testing.T) {
	t.Setenv("UNIFIEDPROXY_SERVICE_MANAGER", "launchd")
	for _, service := range []string{"unifiedproxy-node", "unifiedproxy-node.service"} {
		if _, err := resolveFleetService(service); err == nil {
			t.Fatalf("unknown service manager was accepted for %s", service)
		}
	}
}

func TestFleetServiceRejectsManagerNameMismatch(t *testing.T) {
	t.Setenv("UNIFIEDPROXY_SERVICE_MANAGER", "openrc")
	if _, err := resolveFleetService("unifiedproxy-node.service"); err == nil {
		t.Fatal("OpenRC accepted a systemd service name")
	}
	t.Setenv("UNIFIEDPROXY_SERVICE_MANAGER", "systemd")
	if _, err := resolveFleetService("unifiedproxy-node"); err == nil {
		t.Fatal("systemd accepted an OpenRC service name")
	}
}
