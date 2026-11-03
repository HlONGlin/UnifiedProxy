"""Run only inside a fresh disposable Linux systemd host/container as root.

Exercises the real installer, HTTPS update API, transient upgrade worker,
health-gated rollback and persistent interruption recovery. Never use an
existing installation: this deliberately installs /usr/local/m-ui-control.
"""
import argparse
import datetime
import hashlib
import http.server
import json
from pathlib import Path
import shutil
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


def command(*args, **kwargs):
    return subprocess.run(args, check=True, stdout=subprocess.DEVNULL, **kwargs)


def wait_result(path, expected, timeout=180):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            result = json.loads(path.read_text())
            if result["status"] == expected:
                return result
            if result["status"] in ("failed", "rolled-back", "success") and result["status"] != expected:
                raise RuntimeError("unexpected upgrade result: " + result["status"])
        except FileNotFoundError:
            pass
        time.sleep(0.5)
    raise RuntimeError("upgrade did not settle")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--control", required=True, type=Path)
    parser.add_argument("--candidate", required=True, type=Path)
    parser.add_argument("--output", type=Path, default=Path("dist/fleet-systemd"))
    args = parser.parse_args()
    install = Path("/usr/local/m-ui-control")
    if install.exists():
        raise SystemExit("requires a fresh disposable host: existing installation found")
    args.output.mkdir(parents=True, exist_ok=True)
    data = install / "data"
    with tempfile.TemporaryDirectory(prefix="mui-systemd-") as temp:
        root = Path(temp)
        certificate, key = root / "cert.pem", root / "key.pem"
        subprocess.run(["openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes",
                        "-keyout", str(key), "-out", str(certificate), "-days", "2",
                        "-subj", "/CN=localhost", "-addext", "subjectAltName=DNS:localhost,IP:127.0.0.1"],
                       check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        candidate = root / "candidate"
        shutil.copyfile(args.candidate, candidate)
        bad = root / "bad-candidate"
        bad.write_text('#!/bin/sh\nif [ "$1" = version ]; then echo fixture-unhealthy; exit 0; fi\nexit 1\n')
        assets = {"candidate": candidate.read_bytes(), "bad": bad.read_bytes()}

        class Assets(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                body = assets.get(self.path.lstrip("/"))
                if body is None:
                    self.send_error(404)
                    return
                self.send_response(200)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

            def log_message(self, *args):
                pass

        server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Assets)
        ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        ctx.load_cert_chain(certificate, key)
        server.socket = ctx.wrap_socket(server.socket, server_side=True)
        threading.Thread(target=server.serve_forever, daemon=True).start()
        binary = str(args.control.resolve())
        command("bash", str(Path(__file__).parents[1] / "install-fleet.sh"),
                "control", binary, hashlib.sha256(args.control.read_bytes()).hexdigest())
        command(str(install / "control"), "configure", "--data-dir", str(data),
                "--username", "fixture", "--password-stdin", "--listen-ip", "127.0.0.1",
                "--port", "2053", "--domain", "localhost", "--cert", str(certificate),
                "--key", str(key), input="fixture-password", text=True)
        command("systemctl", "set-environment", "SSL_CERT_FILE=" + str(certificate))
        command("systemctl", "start", "m-ui-control.service")
        panel = "https://127.0.0.1:2053"
        result_path = data / "pending-upgrade.json"

        def login():
            deadline = time.monotonic() + 30
            while time.monotonic() < deadline:
                try:
                    return checker.api_client(panel, "fixture", "fixture-password", str(certificate))
                except Exception:
                    time.sleep(0.2)
            raise RuntimeError("TLS control panel did not start")

        def health(expected):
            context = ssl.create_default_context(cafile=str(certificate))
            with urllib.request.urlopen(panel + "/fleet/v1/health", context=context, timeout=10) as response:
                value = json.load(response)
            if value["sha256"] != expected:
                raise RuntimeError("running executable checksum differs")

        report = {}
        try:
            call = login()
            expected = hashlib.sha256(assets["candidate"]).hexdigest()
            call("/api/fleet/update", {"url": f"https://localhost:{server.server_port}/candidate",
                                      "sha256": expected})
            wait_result(result_path, "success")
            health(expected)
            report["TLSHealthVerifiedUpgrade"] = "passed"
            call = login()
            call("/api/fleet/update", {"url": f"https://localhost:{server.server_port}/bad",
                                      "sha256": hashlib.sha256(assets["bad"]).hexdigest()})
            wait_result(result_path, "rolled-back")
            health(expected)
            report["UnhealthyCandidateRollback"] = "passed"
            record = json.loads(result_path.read_text())
            record["status"] = "pending-restart"
            record["phase"] = "verifying"
            record["startedAt"] = (datetime.datetime.now(datetime.timezone.utc)
                                   - datetime.timedelta(minutes=11)).isoformat()
            record["workerUnit"] = "m-ui-upgrade-0123456789abcdef.service"
            result_path.write_text(json.dumps(record))
            command("systemctl", "stop", "m-ui-control.service")
            (install / "control").write_bytes(assets["bad"])
            (install / "control").chmod(0o755)
            command("systemctl", "start", "m-ui-control-recovery.service")
            wait_result(result_path, "rolled-back", 30)
            login()
            health(expected)
            report["InterruptedWorkerRecovery"] = "passed"
            command("systemctl", "is-active", "m-ui-control-recovery.timer")
            report["PersistentRecoveryTimer"] = "passed"
            print("Real systemd upgrade, TLS health rollback and interrupted-worker recovery passed.")
        finally:
            (args.output / "results.json").write_text(json.dumps(report, indent=2))
            command("systemctl", "stop", "m-ui-control.service", "m-ui-control-recovery.timer")
            server.shutdown()


if __name__ == "__main__":
    main()
