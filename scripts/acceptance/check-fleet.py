"""Live convergence and data-plane acceptance. Requires Python 3 and curl.

Actual clients (hy2/Mieru/AnyTLS/ShadowTLS/Naive, etc.) must already expose
the SOCKS5 or HTTPS CONNECT endpoints named in the manifest. Credentials are
read from the manifest and stdin, never included in reports or curl argv.
"""
import argparse
import getpass
import hashlib
import http.cookiejar
import ipaddress
import json
from pathlib import Path
import ssl
import subprocess
import sys
import tempfile
import time
import urllib.parse
import urllib.request


def safe_url(value, schemes=("https",), allow_remote_http=False):
    parsed = urllib.parse.urlsplit(value)
    if parsed.scheme not in schemes or not parsed.hostname or parsed.fragment:
        raise ValueError("invalid URL scheme or host")
    if parsed.scheme == "http" and not allow_remote_http:
        try:
            local = ipaddress.ip_address(parsed.hostname).is_loopback
        except ValueError:
            local = parsed.hostname == "localhost"
        if not local:
            raise ValueError("HTTP is allowed only on loopback")
    return parsed


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise ValueError("panel redirects are disabled; pass the canonical panel URL")


def api_client(panel, username, password, ca_file):
    parsed = safe_url(panel, ("https", "http"), allow_remote_http=True)
    if parsed.username or parsed.password or parsed.query:
        raise ValueError("panel URL must not contain credentials or query")
    opener = urllib.request.build_opener(
        NoRedirect(), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()),
        urllib.request.HTTPSHandler(context=ssl.create_default_context(cafile=ca_file)))

    def call(path, data=None, method=None):
        body = None if data is None else json.dumps(data).encode()
        request = urllib.request.Request(panel.rstrip("/") + path, body,
                                         {"Content-Type": "application/json"}, method=method)
        with opener.open(request, timeout=30) as response:
            raw = response.read((64 << 20) + 1)
        if len(raw) > 64 << 20:
            raise ValueError("oversized panel response")
        result = json.loads(raw)
        if not result.get("ok"):
            raise ValueError("panel operation failed")
        return result.get("data")
    call("/api/auth/login", {"username": username, "password": password})
    return call


def check_convergence(call, minimum, timeout):
    deadline = time.monotonic() + timeout
    failures = []
    while True:
        view = call("/api/fleet")
        nodes = [n for n in view["nodes"] if not n.get("revoked")]
        failures = []
        if len(nodes) < minimum:
            failures.append(f"expected at least {minimum} active nodes, found {len(nodes)}")
        for node in nodes:
            status = node.get("status") or {}
            detail = call("/api/fleet/nodes/" + urllib.parse.quote(node["id"], safe=""))
            desired = detail.get("desired")
            reasons = []
            if not node.get("online"):
                reasons.append("offline")
            if not status.get("running"):
                reasons.append("core stopped")
            if status.get("error"):
                reasons.append("runtime error")
            if not desired:
                reasons.append("no published policy")
            elif status.get("revision") != desired["revision"] or status.get("digest") != desired["digest"]:
                reasons.append("revision/digest mismatch")
            if reasons:
                failures.append(node["id"] + ": " + ", ".join(reasons))
        if not failures:
            return nodes
        if time.monotonic() >= deadline:
            raise ValueError("; ".join(failures))
        time.sleep(min(2, max(0, deadline - time.monotonic())))


def curl_quote(value):
    value = str(value)
    if any(ord(c) < 32 for c in value):
        raise ValueError("control characters in curl option")
    return '"' + value.replace("\\", "\\\\").replace('"', '\\"') + '"'


def check_proxy(check, curl):
    proxy = safe_url(check["proxy"], ("socks5", "socks5h", "http", "https"))
    target = safe_url(check["url"])
    if target.username or target.password:
        raise ValueError("target URL must not contain credentials")
    if not check.get("contains") and not check.get("sha256"):
        raise ValueError("each data-plane check needs contains or sha256 evidence")
    options = {"proxy": check["proxy"], "url": check["url"], "noproxy": "",
               "proto": "=https", "proto-redir": "=https", "max-time": "30",
               "max-filesize": str(2 << 20)}
    if check.get("caFile"):
        options["cacert"] = str(Path(check["caFile"]).resolve())
    if check.get("proxyCAFile"):
        options["proxy-cacert"] = str(Path(check["proxyCAFile"]).resolve())
    with tempfile.TemporaryDirectory(prefix="up-proxy-check-") as temp:
        path = Path(temp) / "response"
        options["output"] = str(path)
        config = "\n".join(key + " = " + curl_quote(value) for key, value in options.items())
        result = subprocess.run([curl, "--silent", "--show-error", "--fail", "--config", "-",
                                 "--write-out", "%{http_code}"],
                                input=config, capture_output=True, text=True, timeout=40)
        if result.returncode != 0:
            # Do not expose a URL or proxy credentials from diagnostic output.
            raise ValueError(f"proxy request failed (curl exit {result.returncode})")
        if result.stdout.strip() != str(check.get("status", 200)):
            raise ValueError("unexpected target HTTP status")
        body = path.read_bytes()
        if len(body) > 2 << 20:
            raise ValueError("oversized target response")
        digest = hashlib.sha256(body).hexdigest()
        if check.get("contains") and check["contains"].encode() not in body:
            raise ValueError("target response marker missing")
        if check.get("sha256") and check["sha256"].lower() != digest:
            raise ValueError("target response checksum mismatch")
        return {"node": check["node"], "protocol": check["protocol"],
                "transport": proxy.scheme, "status": "passed", "responseSHA256": digest}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--url", required=True, help="panel URL including panel path")
    parser.add_argument("--username", required=True)
    parser.add_argument("--password-stdin", action="store_true")
    parser.add_argument("--ca-file", help="trusted panel CA bundle; verification stays enabled")
    parser.add_argument("--manifest", type=Path)
    parser.add_argument("--status-only", action="store_true")
    parser.add_argument("--minimum-nodes", type=int, default=3)
    parser.add_argument("--require-protocol", action="append", default=[])
    parser.add_argument("--timeout", type=float, default=120)
    parser.add_argument("--curl", default="curl")
    parser.add_argument("--output", type=Path, default=Path("fleet-acceptance.json"))
    args = parser.parse_args()
    if args.minimum_nodes < 1 or args.timeout <= 0:
        parser.error("minimum nodes and timeout must be positive")
    if not args.status_only and not args.manifest:
        parser.error("--manifest is required for data-plane acceptance; --status-only explicitly skips it")
    report = {"status": "failed", "scope": "status-only" if args.status_only else "manifest",
              "checks": [], "protocols": []}
    try:
        password = sys.stdin.readline().rstrip("\r\n") if args.password_stdin else getpass.getpass()
        call = api_client(args.url, args.username, password, args.ca_file)
        nodes = check_convergence(call, args.minimum_nodes, args.timeout)
        report["nodes"] = [{"id": n["id"], "revision": n["status"]["revision"],
                            "digest": n["status"]["digest"]} for n in nodes]
        if not args.status_only:
            manifest = json.loads(args.manifest.read_text(encoding="utf-8-sig"))
            checks = manifest.get("checks")
            if not checks:
                raise ValueError("manifest has no data-plane checks")
            active_ids = {n["id"] for n in nodes}
            active_names = {n["name"] for n in nodes}
            for check in checks:
                if check.get("node") not in active_ids | active_names:
                    raise ValueError("manifest references an unknown active node")
                if not check.get("protocol"):
                    raise ValueError("manifest check needs an explicit protocol")
                report["checks"].append(check_proxy(check, args.curl))
            covered_nodes = {c["node"] for c in checks}
            if any(n["id"] not in covered_nodes and n["name"] not in covered_nodes for n in nodes):
                raise ValueError("manifest must test every active node")
            protocols = sorted({c["protocol"] for c in checks})
            report["protocols"] = protocols
            if set(args.require_protocol) - set(protocols):
                raise ValueError("required protocol missing from manifest")
        report["status"] = "passed"
    except Exception as error:
        # Keep secrets in manifests out of exception messages and reports.
        report["error"] = str(error) if isinstance(error, ValueError) else type(error).__name__
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"Fleet acceptance {report['status']} ({report['scope']}); report: {args.output}")
    return 0 if report["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
