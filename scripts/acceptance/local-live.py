"""Linux: actual control, three Agents and subscription-driven Mihomo clients.

Runs only on loopback with an isolated fixture CA. This is a reproducible
data-plane integration check, not public-server or public-CA acceptance.
Requires openssl, curl, Python 3, and built Linux control/node/Mihomo binaries.
"""
import argparse
import copy
import http.server
import json
import os
import re
from pathlib import Path
import signal
import socket
import ssl
import subprocess
import tempfile
import threading
import time
import urllib.request

from importlib.util import module_from_spec, spec_from_file_location

spec = spec_from_file_location("fleet_check", Path(__file__).with_name("check-fleet.py"))
checker = module_from_spec(spec)
spec.loader.exec_module(checker)


def port(kind=socket.SOCK_STREAM):
    with socket.socket(socket.AF_INET, kind) as listener:
        listener.bind(("127.0.0.1", 0))
        return listener.getsockname()[1]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--control", required=True, type=Path)
    parser.add_argument("--node", required=True, type=Path)
    parser.add_argument("--mihomo", required=True, type=Path)
    parser.add_argument("--with-sidecars", action="store_true",
                        help="also download and exercise managed ShadowTLS/ss-rust")
    parser.add_argument("--output", type=Path, default=Path("dist/fleet-live"))
    args = parser.parse_args()
    control, node, core = (str(p.resolve()) for p in (args.control, args.node, args.mihomo))
    args.output.mkdir(parents=True, exist_ok=True)
    processes, logs = [], []
    with tempfile.TemporaryDirectory(prefix="mui-fleet-live-") as temp:
        root = Path(temp)

        def launch(command, name, env=None):
            stream = (args.output / (name + ".log")).open("wb")
            logs.append(stream)
            process = subprocess.Popen(command, stdout=stream, stderr=subprocess.STDOUT,
                                       start_new_session=True, env=env)
            processes.append(process)
            return process

        def configure(binary, data, listen):
            subprocess.run([binary, "configure", "--data-dir", str(data),
                            "--username", "fixture", "--password-stdin", "--listen-ip",
                            "127.0.0.1", "--port", str(listen), "--core-path", core],
                           input="fixture-password", text=True, check=True, stdout=subprocess.DEVNULL)

        try:
            certificate, key = root / "cert.pem", root / "key.pem"
            subprocess.run(["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
                            "-keyout", str(key), "-out", str(certificate), "-days", "2",
                            "-subj", "/CN=localhost", "-addext", "subjectAltName=DNS:localhost,IP:127.0.0.1"],
                           check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

            class Target(http.server.BaseHTTPRequestHandler):
                def do_GET(self):
                    body = b"mui-live-hy2-data-plane"
                    self.send_response(200)
                    self.send_header("Content-Length", str(len(body)))
                    self.end_headers()
                    self.wfile.write(body)

                def log_message(self, *args):
                    pass

            target = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Target)
            context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            context.load_cert_chain(certificate, key)
            target.socket = context.wrap_socket(target.socket, server_side=True)
            threading.Thread(target=target.serve_forever, daemon=True).start()
            target_url = f"https://localhost:{target.server_port}/proof"
            panel_port = port()
            panel = f"http://127.0.0.1:{panel_port}"
            control_data = root / "control"
            configure(control, control_data, panel_port)
            controller = launch([control, "serve", "--data-dir", str(control_data)], "control")
            deadline = time.monotonic() + 30
            while True:
                try:
                    call = checker.api_client(panel, "fixture", "fixture-password", None)
                    break
                except Exception:
                    if time.monotonic() > deadline or controller.poll() is not None:
                        raise RuntimeError("control failed to start")
                    time.sleep(0.2)
            checks, node_processes, node_dirs, clients = [], [], [], []
            protocols = ["hysteria2", "anytls", "mieru"]
            if args.with_sidecars:
                protocols.append("shadowtls")
            for number in range(3):
                name = "node-" + str(number + 1)
                enrollment = call("/api/fleet/tokens", {"name": name})
                data = root / name
                configure(node, data, port())
                state_path = data / "state.json"
                state = json.loads(state_path.read_text())
                state["settings"]["apiAddress"] = f"127.0.0.1:{port()}"
                state["settings"]["apiSecret"] = name + "-fixture-secret"
                state_path.write_text(json.dumps(state))
                subprocess.run([node, "agent", "join", "--data-dir", str(data), "--control", panel,
                                "--control-key", enrollment["controlKey"], "--token-stdin"],
                               input=enrollment["token"], text=True, check=True, stdout=subprocess.DEVNULL)
                identity = json.loads((data / "agent.json").read_text())["nodeId"]
                call("/api/fleet/nodes/" + identity, {"name": name, "address": "127.0.0.1"}, method="PUT")
                process = launch([node, "serve", "--data-dir", str(data)], name)
                node_processes.append(process)
                node_dirs.append(data)
                model = copy.deepcopy(state)
                model["inbounds"] = []
                for protocol in protocols:
                    if protocol == "shadowtls":
                        continue
                    inbound = {"id": protocol + "-" + name, "name": protocol + "-" + name,
                               "type": protocol, "listen": "127.0.0.1", "enabled": True,
                               "port": port(socket.SOCK_DGRAM if protocol == "hysteria2" else socket.SOCK_STREAM),
                               "clients": [{"id": "alice", "name": "alice", "username": "alice",
                                            "password": "fixture-proxy-password", "enabled": True}]}
                    if protocol == "mieru":
                        inbound["mieru"] = {"transport": "TCP"}
                    else:
                        inbound.update({"tls": True, "tlsServerName": "localhost",
                                        "certificate": certificate.read_text(), "privateKey": key.read_text(),
                                        "certificateMode": "content"})
                    model["inbounds"].append(inbound)
                    clients.append((name, protocol, name + " / " + inbound["name"] + "-alice"))
                call("/api/fleet/nodes/" + identity + "/model", model)
                call("/api/fleet/nodes/" + identity)  # Validate the native published model is readable.
                if args.with_sidecars:
                    desired = call("/api/fleet/nodes/" + identity + "/sidecar",
                                   {"kind": "shadowtls", "port": port(), "username": "alice",
                                    "password": "fixture-shadowtls-password",
                                    "handshake": f"127.0.0.1:{target.server_port}"})
                    sidecar = next(s for s in desired["sidecars"] if s["kind"] == "shadowtls")
                    clients.append((name, "shadowtls", name + " / " + sidecar["id"]))
            # Use the actual aggregated subscription, not hand-written proxy credentials.
            control_state = json.loads((control_data / "state.json").read_text())
            tokens = call("/api/fleet/subscriptions")
            settings = control_state["settings"]
            subscription_url = panel + settings["subscriptionPath"].rstrip("/") + "/" + tokens["alice"] + settings["clashPath"]
            with urllib.request.urlopen(subscription_url, timeout=15) as response:
                subscription = response.read().decode()
            # Retain the subscription's exact proxy definitions while
            # isolating the client's listeners from the server's defaults.
            isolated = []
            drop = True
            for line in subscription.splitlines():
                match = re.match(r"^([A-Za-z0-9_-]+):", line)
                if match:
                    drop = match[1] != "proxies"
                if not drop:
                    isolated.append(line)
            subscription = "\n".join(isolated) + "\n"
            for name, protocol, proxy_name in clients:
                socks, api = port(), port()
                client_dir = root / ("client-" + name + "-" + protocol)
                client_dir.mkdir()
                client_config = client_dir / "config.yaml"
                client_config.write_text(subscription + f'\nsocks-port: {socks}\nexternal-controller: 127.0.0.1:{api}\n'
                                         + 'bind-address: 127.0.0.1\nmode: rule\nlog-level: warning\n'
                                         + 'proxy-groups: ' + json.dumps([{"name": "PROXY", "type": "select", "proxies": [proxy_name]}])
                                         + '\nrules: ["MATCH,PROXY"]\n')
                client = launch([core, "-d", str(client_dir), "-f", str(client_config)], "client-" + name + "-" + protocol,
                       env={**os.environ, "SSL_CERT_FILE": str(certificate)})
                deadline = time.monotonic() + 30
                while True:
                    try:
                        req = urllib.request.Request(f"http://127.0.0.1:{api}/proxies/PROXY",
                                                     json.dumps({"name": proxy_name}).encode(),
                                                     {"Content-Type": "application/json"}, method="PUT")
                        urllib.request.urlopen(req, timeout=2).close()
                        break
                    except Exception:
                        if time.monotonic() > deadline or client.poll() is not None:
                            raise RuntimeError("subscription client failed: " + name + " / " + protocol)
                        time.sleep(0.2)
                checks.append({"node": name, "protocol": protocol,
                               "proxy": f"socks5h://127.0.0.1:{socks}", "url": target_url,
                               "caFile": str(certificate), "contains": "mui-live-hy2-data-plane"})
            nodes = checker.check_convergence(call, 3, 120)
            manifest = root / "manifest.json"
            manifest.write_text(json.dumps({"checks": checks}))
            report = args.output / "results.json"
            required = [item for protocol in protocols for item in ("--require-protocol", protocol)]
            subprocess.run(["python3", str(Path(__file__).with_name("check-fleet.py")),
                            "--url", panel, "--username", "fixture", "--password-stdin",
                            "--manifest", str(manifest), "--output", str(report), *required],
                           input="fixture-password\n", text=True, check=True)
            # Control outage and Agent restart: all three proxies keep serving.
            os.killpg(controller.pid, signal.SIGTERM)
            controller.wait(timeout=15)
            for process in node_processes:
                os.killpg(process.pid, signal.SIGTERM)
                process.wait(timeout=15)
            for number, data in enumerate(node_dirs):
                launch([node, "serve", "--data-dir", str(data)], "restart-node-" + str(number + 1))
            for check in checks:
                deadline = time.monotonic() + 30
                while True:
                    try:
                        checker.check_proxy(check, "curl")
                        break
                    except ValueError:
                        if time.monotonic() > deadline:
                            raise
                        time.sleep(0.3)
            result = json.loads(report.read_text())
            result["controllerOutageAndAgentRestart"] = "passed"
            result["environment"] = "Linux loopback, fixture CA, actual Mihomo and aggregated subscription clients"
            result["subscriptionClients"] = "passed"
            report.write_text(json.dumps(result, indent=2))
            print("Three-node subscription data-plane, policy convergence and offline restart passed: " + ", ".join(protocols))
            target.shutdown()
        finally:
            for process in reversed(processes):
                try:
                    os.killpg(process.pid, signal.SIGTERM)
                except ProcessLookupError:
                    pass
            for process in reversed(processes):
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    os.killpg(process.pid, signal.SIGKILL)
                    process.wait(timeout=5)
            for stream in logs:
                stream.close()


if __name__ == "__main__":
    main()
