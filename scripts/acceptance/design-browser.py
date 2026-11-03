"""Real browser snapshots and UI regression, only in orca-linux /workspace."""
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


def free_port():
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        return sock.getsockname()[1]


def main():
    assert Path.cwd() == Path('/workspace') and Path('/.dockerenv').exists()
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--phase', required=True)
    parser.add_argument('--binary', default='dist/design-audit/control')
    parser.add_argument('--all-themes', action='store_true')
    args = parser.parse_args()
    output = Path('dist/design-audit') / args.phase
    output.mkdir(parents=True, exist_ok=True)
    binary = str(Path(args.binary).resolve())
    checks = []
    contrasts = []
    with tempfile.TemporaryDirectory(prefix='up-ui-') as data:
        port = free_port()
        url = f'http://127.0.0.1:{port}'
        subprocess.run([binary, 'configure', '--data-dir', data, '--username', 'fixture', '--password-stdin',
                        '--listen-ip', '127.0.0.1', '--port', str(port)], input='fixture-password', text=True,
                       check=True, stdout=subprocess.DEVNULL)
        with (output / 'server.log').open('w') as log:
            process = subprocess.Popen([binary, 'serve', '--data-dir', data], stdout=log, stderr=log)
            try:
                for _ in range(100):
                    try:
                        urllib.request.urlopen(url+'/login', timeout=1).close()
                        break
                    except Exception:
                        time.sleep(.1)
                with sync_playwright() as pw:
                    browser = pw.chromium.launch(headless=True, args=['--no-sandbox'])
                    context = browser.new_context(viewport={'width': 1440, 'height': 1000}, locale='zh-CN')
                    page = context.new_page()
                    errors = []
                    page.on('pageerror', lambda err: errors.append(str(err)))
                    page.goto(url+'/login', wait_until='networkidle')
                    page.screenshot(path=str(output/'login.png'), full_page=True)
                    page.locator('#username').fill('fixture')
                    page.locator('#password').fill('fixture-password')
                    page.locator('#login-button').click()
                    page.wait_for_url(url+'/', timeout=15000)
                    page.wait_for_load_state('networkidle')
                    combinations = [(None,None)] if args.phase == 'before' else [('t'+str(i), mode) for i in range(5) for mode in ('light','dark')]
                    if args.phase != 'before':
                        assert page.evaluate('UPTheme.color') == 't0'
                        assert page.evaluate('UPTheme.mode') == 'light'
                        page.evaluate("UPTheme.setColor('t2')")
                        assert page.evaluate('UPTheme.mode') == 'dark'
                        page.evaluate("UPTheme.setMode('light'); UPTheme.setColor('t0'); UPTheme.setColor('t2')")
                        assert page.evaluate('UPTheme.mode') == 'light'
                    for palette,mode in combinations:
                        combo = f'{palette}-{mode}' if palette else ''
                        folder = output/combo
                        folder.mkdir(exist_ok=True)
                        if palette:
                            page.evaluate('([p,m])=>{UPTheme.setColor(p);UPTheme.setMode(m)}', [palette,mode])
                            page.reload(wait_until='networkidle')
                            assert page.get_attribute('html','data-color-theme') == palette
                            assert page.get_attribute('html','data-theme') == mode
                        if palette:
                            colors = page.evaluate("() => {let s=getComputedStyle(document.documentElement); return Object.fromEntries(['canvas','surface-1','surface-2','surface-3','ink','ink-muted','ink-subtle','primary','primary-hover','primary-active','primary-deep','primary-soft','on-primary'].map(k=>[k,s.getPropertyValue('--'+k).trim()]));}")
                            def luminance(value):
                                values = [int(value.lstrip('#')[i:i+2],16)/255 for i in (0,2,4)]
                                return sum(c*w for c,w in zip([v/12.92 if v<=.04045 else ((v+.055)/1.055)**2.4 for v in values],[.2126,.7152,.0722]))
                            def ratio(a,b):
                                x,y=sorted([luminance(colors[a]),luminance(colors[b])]); return (y+.05)/(x+.05)
                            pairs=[(fg,bg) for fg in ['ink','ink-muted','ink-subtle'] for bg in ['canvas','surface-1','surface-2','surface-3']]+[('on-primary',bg) for bg in ['primary','primary-hover','primary-active']]+[('primary-deep','primary-soft')]
                            for fg,bg in pairs:
                                result={'theme':combo,'foreground':fg,'background':bg,'ratio':round(ratio(fg,bg),3)}
                                contrasts.append(result)
                                assert result['ratio'] >= 4.5, result
                        for view in ['dashboard', 'inbounds', 'subtemplates', 'mihomo', 'settings', 'fleet']:
                            page.locator(f'#nav [data-view="{view}"]').click()
                            page.wait_for_timeout(100)
                            page.screenshot(path=str(folder/f'{view}.png'), full_page=True)
                            checks.append({'entry': view, 'theme':combo,'passed': page.locator('#view-'+view).is_visible()})
                    page.locator('#nav [data-view="settings"]').click()
                    for tab in page.locator('[data-settings-tab]').all():
                        tab.click()
                        page.screenshot(path=str(output/f'settings-{tab.get_attribute("data-settings-tab")}.png'), full_page=True)
                    assert not errors, errors
                    (output/'results.json').write_text(json.dumps({'checks': checks, 'errors': errors, 'contrasts':contrasts}, indent=2))
                    browser.close()
            finally:
                process.terminate()
                process.wait(timeout=15)
    print(f'{args.phase}: {len(checks)} pages, no browser errors; screenshots: {output}')


if __name__ == '__main__':
    main()
