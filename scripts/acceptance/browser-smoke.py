"""Headless fleet UI check using an isolated local control and test credentials.

Requires: python -m pip install playwright; an installed Chromium/Edge executable.
No existing panel, data directory or credentials are used.
"""
import argparse
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import time
import urllib.request

from playwright.sync_api import sync_playwright

parser = argparse.ArgumentParser()
parser.add_argument("--binary", required=True)
parser.add_argument("--browser", required=True)
parser.add_argument("--output", default="dist/fleet-browser")
args = parser.parse_args()
binary = str(Path(args.binary).resolve())
output = Path(args.output).resolve()
output.mkdir(parents=True, exist_ok=True)
with socket.socket() as listener:
    listener.bind(("127.0.0.1", 0))
    port = listener.getsockname()[1]
url = f"http://127.0.0.1:{port}"
results = []
with tempfile.TemporaryDirectory(prefix="up-fleet-browser-") as data:
    subprocess.run([binary, "configure", "--data-dir", data, "--username", "fixture",
                    "--password-stdin", "--listen-ip", "127.0.0.1", "--port", str(port)],
                   input="fixture-password", text=True, check=True, stdout=subprocess.DEVNULL)
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(executable_path=args.browser, headless=True)
        try:
            for theme in ("shadcn", "md3", "liquid"):
                state_path = Path(data) / "state.json"
                state = json.loads(state_path.read_text(encoding="utf-8"))
                state["settings"]["uiTheme"] = theme
                state_path.write_text(json.dumps(state), encoding="utf-8")
                process = subprocess.Popen([binary, "serve", "--data-dir", data],
                                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                try:
                    for attempt in range(100):
                        try:
                            urllib.request.urlopen(url + "/login", timeout=1).close()
                            break
                        except Exception:
                            time.sleep(0.1)
                    context = browser.new_context(viewport={"width": 1440, "height": 1000})
                    response = context.request.post(url + "/api/auth/login",
                                                    data={"username": "fixture", "password": "fixture-password"})
                    assert response.ok, response.status
                    response = context.request.post(url + "/api/fleet/nodes",
                                                    data={"name": "fixture-node", "address": "node.example"})
                    assert response.ok, response.status
                    page = context.new_page()
                    errors = []
                    page.on("pageerror", lambda error: errors.append(str(error)))
                    page.goto(url, wait_until="networkidle")
                    page.locator('#nav button[data-view="fleet"]').click()
                    page.locator("#view-fleet h2").wait_for()
                    locales = page.evaluate("window.UPFleet.locales")
                    missing = page.evaluate("Object.entries(UPFleet.words).filter(([key,v])=>v.length!==11||v.some(x=>!x)).map(([key])=>key)")
                    assert not missing, missing
                    for locale in locales:
                        page.evaluate("lang => upSetLang(lang)", locale)
                        assert page.locator("#view-fleet h2").inner_text().strip()
                        assert page.locator("#view-fleet textarea").count() >= 5
                        assert not errors, errors
                        results.append({"theme": theme, "locale": locale, "status": "passed"})
                    page.evaluate("upSetLang('zh-CN')")
                    assert page.locator(".fleet-grid tbody tr").count() >= 1
                    page.set_viewport_size({"width": 390, "height": 844})
                    page.evaluate("setDrawerOpen(false)")
                    page.wait_for_timeout(300)
                    assert page.evaluate("document.documentElement.scrollWidth <= window.innerWidth + 1"), "mobile page overflows"
                    page.screenshot(path=str(output / (theme + "-mobile.png")), full_page=True)
                    page.set_viewport_size({"width": 1440, "height": 1000})
                    page.screenshot(path=str(output / (theme + ".png")), full_page=True)
                    context.close()
                finally:
                    process.terminate()
                    process.wait(timeout=15)
        finally:
            browser.close()
(output / "results.json").write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
print(f"Passed {len(results)} theme/language combinations; evidence: {output}")
