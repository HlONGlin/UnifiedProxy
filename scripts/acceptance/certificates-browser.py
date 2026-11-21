"""Certificate UI regression in orca-linux; node responses and ACME tasks are simulated.

Build: go build -o dist/certificate-audit/control ./cmd/control
Run with a Python environment containing playwright and cryptography.
"""
import argparse
import base64
import hashlib
import json
import socket
import subprocess
import tempfile
import time
import urllib.request
from pathlib import Path

from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric.x25519 import X25519PrivateKey, X25519PublicKey
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.kdf.hkdf import HKDF
from playwright.sync_api import expect, sync_playwright

assert Path.cwd() == Path('/workspace') and Path('/.dockerenv').exists()
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--binary', default='dist/certificate-audit/control')
parser.add_argument('--browser')
args = parser.parse_args()
binary = str(Path(args.binary).resolve())
output = Path('dist/certificate-audit')
output.mkdir(parents=True, exist_ok=True)
with socket.socket() as sock:
    sock.bind(('127.0.0.1', 0))
    port = sock.getsockname()[1]
origin = f'http://127.0.0.1:{port}'
errors, checks, tasks, operations = [], [], [], []


def check(name, condition=True):
    assert condition, name
    checks.append(name)
    print('PASS ' + name, flush=True)


with tempfile.TemporaryDirectory(prefix='up-certificate-browser-') as data:
    subprocess.run([binary, 'configure', '--data-dir', data, '--username', 'fixture', '--password-stdin',
                    '--listen-ip', '127.0.0.1', '--port', str(port)], input='fixture-password', text=True,
                   check=True, stdout=subprocess.DEVNULL)
    with (output / 'server.log').open('w') as log:
        process = subprocess.Popen([binary, 'serve', '--data-dir', data], stdout=log, stderr=log)
        try:
            for _ in range(100):
                try:
                    urllib.request.build_opener(urllib.request.ProxyHandler({})).open(origin + '/login', timeout=1).close()
                    break
                except Exception:
                    if process.poll() is not None:
                        raise RuntimeError((output / 'server.log').read_text())
                    time.sleep(.1)
            with sync_playwright() as pw:
                launch = {'headless': True, 'args': ['--no-sandbox']}
                if args.browser:
                    launch['executable_path'] = args.browser
                browser = pw.chromium.launch(**launch)
                ctx = browser.new_context(viewport={'width': 1440, 'height': 1100})
                page = ctx.new_page()
                page.on('pageerror', lambda error: errors.append(str(error)))
                page.goto(origin + '/login', wait_until='networkidle')
                page.locator('#username').fill('fixture')
                page.locator('#password').fill('fixture-password')
                page.locator('#login-button').click()
                page.wait_for_url(origin + '/')
                page.evaluate("upSetLang('zh-CN')")

                def api(path, payload=None):
                    response = ctx.request.fetch(origin + '/api/fleet' + path, method='GET' if payload is None else 'POST', data=payload)
                    assert response.ok, response.text()
                    return response.json()['data']

                node = api('/nodes', {'name': '证书测试节点', 'address': 'edge.example'})
                old_node = api('/nodes', {'name': '旧版节点', 'address': 'old.example'})
                node_id = node['id']
                identity = X25519PrivateKey.generate()
                public = base64.b64encode(identity.public_key().public_bytes(serialization.Encoding.Raw, serialization.PublicFormat.Raw)).decode()
                profiles = [{'name': 'default'}, {'name': 'account-b'}]
                certs = [{'name': 'edge-cert', 'domain': 'edge.example', 'dns': 'dns_cf', 'credentialName': 'account-b',
                          'automatic': True, 'expiresAt': '2027-02-01T00:00:00Z', 'inboundIds': ['inbound-1']}]
                modern_caps = ['cert-issue', 'cert-renew', 'cert-bind', 'cert-delete', 'cert-update', 'cert-credentials', 'cert-credentials-manage']

                def fulfill(route, data, status=200):
                    route.fulfill(status=status, json={'ok': status < 400, 'data': data, 'message': data if status >= 400 else ''})

                def handle(route):
                    path = route.request.url.removeprefix(origin + '/api/fleet')
                    if route.request.method == 'GET' and path in ['/nodes/' + node_id, '/nodes/' + old_node['id']]:
                        old = path.endswith(old_node['id'])
                        fulfill(route, {**(old_node if old else node), 'encryptionPublicKey': public, 'online': True,
                                        'status': {'capabilities': ['cert-issue', 'cert-credentials'] if old else modern_caps,
                                                   'details': {'cloudflareCredentialsConfigured': True,
                                                               **({} if old else {'cloudflareCredentials': profiles}),
                                                               'managedCertificates': certs}}, 'draft': {'inbounds': []}})
                    elif path == '/tasks' and route.request.method == 'GET':
                        fulfill(route, tasks)
                    elif route.request.method == 'POST' and path.startswith('/nodes/' + node_id + '/certificates/'):
                        body = route.request.post_data_json
                        action = path.rsplit('/', 1)[-1]
                        operations.append((action, body))
                        if action == 'dns-credentials':
                            kind, task_id, envelope = body['kind'], body['taskId'], body['payload']
                            assert kind == 'cert-credentials-manage'
                            shared = identity.exchange(X25519PublicKey.from_public_bytes(base64.b64decode(envelope['ephemeralPublicKey'])))
                            key = HKDF(algorithm=hashes.SHA256(), length=32,
                                       salt=hashlib.sha256(('UnifiedProxy-fleet-secret-salt-v1\n' + node_id).encode()).digest(),
                                       info=('UnifiedProxy-fleet-secret-key-v1\n' + task_id + '\n' + kind).encode()).derive(shared)
                            plain = json.loads(AESGCM(key).decrypt(base64.b64decode(envelope['nonce']), base64.b64decode(envelope['ciphertext']),
                                                                  ('UnifiedProxy-fleet-secret-v1\n' + node_id + '\n' + task_id + '\n' + kind).encode()))
                            assert 'browser-profile-secret' not in json.dumps(body)
                            if plain.get('action') == 'delete':
                                profiles[:] = [item for item in profiles if item['name'] != plain['name']]
                            else:
                                assert plain['cloudflareToken'] == 'browser-profile-secret'
                                profiles.append({'name': plain['name']})
                        elif action == 'update':
                            certs[0].update(body)
                        elif action == 'delete':
                            certs[:] = [item for item in certs if item['name'] != body['name']]
                        elif action == 'issue':
                            certs.append({**body, 'expiresAt': '2027-02-01T00:00:00Z'})
                        task = {'id': str(len(tasks) + 1), 'status': 'success'}
                        tasks.append(task)
                        fulfill(route, {'task': task}, 202)
                    else:
                        route.continue_()

                page.route('**/api/fleet/**', handle)
                page.reload(wait_until='networkidle')
                root = page.locator('#view-fleet')
                root.locator('[data-fleet-tab=certificates]').click()
                root.get_by_label('子服务器', exact=True).select_option(node_id)
                card = root.locator('[data-certificate-name=edge-cert]')
                expect(card).to_be_visible()
                expect(card.locator('dl')).to_contain_text('account-b')
                expect(card.locator('dl div').filter(has=page.locator('dt', has_text='已绑定')).locator('dd')).to_have_text('1')
                check('Card displays actual node bindings and selected credential')
                root.get_by_label('凭据名称', exact=False).fill('account-c')
                root.get_by_label('Cloudflare API Token', exact=False).fill('browser-profile-secret')
                root.get_by_role('button', name='加密保存到此子服务器', exact=True).click()
                expect(root.locator('.fleet-certificate-panel-dns').get_by_text('account-c', exact=True)).to_be_visible(timeout=10000)
                check('Multiple credentials save and refresh after task completion; browser encryption decrypts correctly')
                row = root.locator('.fleet-certificate-panel-dns .fleet-certificate-item').filter(has_text='account-c')
                row.get_by_role('button', name='删除凭据', exact=True).click()
                page.locator('.up-dialog .danger-btn').click()
                expect(row).to_have_count(0, timeout=10000)
                check('Credential deletion confirms and updates the list')
                card.locator('summary').click()
                card.get_by_label('自动续期', exact=True).uncheck()
                card.get_by_label('续期凭据', exact=True).select_option('default')
                card.get_by_role('button', name='保存设置', exact=True).click()
                expect(card.locator('.fleet-certificate-badge')).to_have_text('手动续期', timeout=10000)
                check('Renewal settings persist selected credential and automatic switch')
                root.get_by_label('验证方式', exact=True).select_option('dns_cf')
                root.get_by_label('签发与续期凭据', exact=True).select_option('account-b')
                refresh = root.get_by_role('button', name='刷新证书', exact=True)
                refresh.click()
                expect(refresh).to_be_enabled()
                expect(root.get_by_label('签发与续期凭据', exact=True)).to_have_value('account-b')
                check('Refreshing node status preserves the selected issuance credential')
                root.get_by_label('证书名称', exact=True).fill('second-cert')
                root.get_by_label('证书域名', exact=True).fill('second.example')
                root.get_by_role('button', name='在此子服务器申请证书', exact=True).click()
                second = root.locator('[data-certificate-name=second-cert]')
                expect(second).to_be_visible(timeout=10000)
                check('Issue request selects a named credential', next(body for action, body in operations if action == 'issue')['credentialName'] == 'account-b')
                second.get_by_role('button', name='删除证书', exact=True).click()
                page.locator('.up-dialog .danger-btn').click()
                expect(second).to_have_count(0, timeout=10000)
                check('Certificate deletion confirms and refreshes the list')
                page.screenshot(path=str(output / 'certificates-desktop.png'), full_page=True)
                page.set_viewport_size({'width': 390, 'height': 844})
                page.screenshot(path=str(output / 'certificates-mobile.png'), full_page=True)
                check('Mobile certificate page fits viewport', page.evaluate('document.documentElement.scrollWidth <= innerWidth + 1'))
                page.set_viewport_size({'width': 1440, 'height': 1100})
                root.get_by_label('子服务器', exact=True).select_option(old_node['id'])
                expect(root.get_by_label('凭据名称', exact=False)).to_be_disabled()
                expect(root.get_by_role('button', name='删除凭据', exact=True)).to_be_disabled()
                expect(root.get_by_role('button', name='删除证书', exact=True)).to_be_disabled()
                check('Legacy node capability gates prevent incompatible management tasks')
                root.get_by_label('子服务器', exact=True).select_option('')
                expect(root.get_by_role('button', name='加密保存到此子服务器', exact=True)).to_be_disabled()
                expect(root.get_by_role('button', name='在此子服务器申请证书', exact=True)).to_be_disabled()
                check('Clearing node selection disables certificate and credential submission')
                check('No browser script errors', not errors)
                browser.close()
        finally:
            process.terminate()
            process.wait(timeout=10)
(output / 'browser-results.json').write_text(json.dumps({'checks': checks, 'errors': errors}, ensure_ascii=False, indent=2))
