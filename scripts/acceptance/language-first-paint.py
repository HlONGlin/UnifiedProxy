"""Check the first visible workspace frame using the real control server."""
import argparse, socket, subprocess, tempfile, time
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

assert Path.cwd() == Path('/workspace') and Path('/.dockerenv').exists()
parser = argparse.ArgumentParser()
parser.add_argument('--control', default='dist/localization/control')
args = parser.parse_args()
control = str(Path(args.control).resolve())
checks = []

def check(name, condition):
    assert condition, name
    checks.append(name)
    print('PASS ' + name, flush=True)

observe = """(() => {
  window.languageFrames = [];
  function frame() {
    const nav = document.querySelector('#nav [data-view="inbounds"] span:last-child');
    if (nav && getComputedStyle(nav).visibility === 'visible') {
      const text = nav.textContent.trim();
      if (window.languageFrames.at(-1) !== text) window.languageFrames.push(text);
    }
    requestAnimationFrame(frame);
  }
  requestAnimationFrame(frame);
})();"""

with tempfile.TemporaryDirectory(prefix='language-', dir='/workspace/dist') as temp, sync_playwright() as pw:
    temp = Path(temp)
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        port = sock.getsockname()[1]
    base = f'http://127.0.0.1:{port}/console/'
    subprocess.run([control, 'configure', '--data-dir', str(temp/'data'), '--username', 'fixture',
                    '--password-stdin', '--listen-ip', '127.0.0.1', '--port', str(port), '--path', 'console'],
                   input='fixture-password', text=True, check=True, stdout=subprocess.DEVNULL)
    with (temp/'control.log').open('w') as log:
        process = subprocess.Popen([control, 'serve', '--data-dir', str(temp/'data')], stdout=log, stderr=log)
        browser = pw.chromium.launch(headless=True, args=['--no-sandbox'])
        try:
            ctx = browser.new_context()
            for _ in range(100):
                try:
                    if ctx.request.get(base+'login', timeout=1000).ok: break
                except Exception: pass
                time.sleep(.1)
            def api(path, body=None):
                response = ctx.request.fetch(base+path, method='POST' if body is not None else 'GET', data=body)
                assert response.ok, (path, response.status, response.text())
                return response.json()['data']
            api('api/auth/login', {'username':'fixture', 'password':'fixture-password'})
            cards = []
            for language in ('en', 'zh-TW'):
                card = api('api/fleet/nodes', {'name':language+' node', 'address':''})
                model = api('api/fleet/nodes/'+card['id'])['draft']
                model['settings']['language'] = language
                api('api/fleet/nodes/'+card['id']+'/model', model)
                cards.append(card)
            ctx.add_init_script(observe)
            page = ctx.new_page()
            errors = []
            page.on('pageerror', lambda error: errors.append(str(error)))
            # Slow the last parser-blocking script: English template nodes exist
            # for several browser frames before DOMContentLoaded can translate.
            def slow_script(route):
                time.sleep(.7)
                route.continue_()
            page.route('**/static/xray-import.js', slow_script)
            for language, label in (('zh-CN','入站列表'), ('en','Inbounds'), ('zh-TW','入站')):
                api('api/language', {'language':language})
                page.goto(base)
                page.evaluate("localStorage.setItem('up-language', 'en')")
                for card in cards:
                    page.goto(base+'nodes/'+card['id']+'/')
                    expect(page.locator('html')).not_to_have_attribute('data-i18n-pending', '')
                    expect(page.locator('#nav [data-view=inbounds]')).to_have_text(label)
                    page.evaluate('refresh()')
                    page.wait_for_timeout(150)
                    frames = page.evaluate('window.languageFrames')
                    check(language+' / '+card['name']+': first visible frame uses host language', bool(frames) and set(frames)=={label})
                    check(language+' / '+card['name']+': node refresh does not change language', page.evaluate('upGetLang()')==language)
            check('No browser exceptions during normal workspace use', not errors)
            # Failed translation downloads must release the loading gate.
            page.route('**/static/i18n.js', lambda route: route.abort())
            page.goto(base)
            expect(page.locator('html')).not_to_have_attribute('data-i18n-pending', '', timeout=7000)
            check('Translation failure releases the loading screen', page.locator('#view-fleet').is_visible())
            page.unroute('**/static/i18n.js')
            errors.clear()
            # The live language must also work when storage writes are denied.
            page.goto(base+'nodes/'+cards[0]['id']+'/')
            page.evaluate("Storage.prototype.setItem = function(){throw new Error('storage disabled')}; upSetLang('en')")
            expect(page.locator('#nav [data-view=inbounds]')).to_have_text('Inbounds')
            page.evaluate('refresh()')
            check('Language switching works without localStorage writes', page.evaluate('upGetLang()')=='en')
            check('No browser exceptions after switching language with storage writes disabled', not errors)
        finally:
            browser.close()
            process.terminate()
            process.wait(timeout=15)
print(f'Passed {len(checks)} language checks.', flush=True)
