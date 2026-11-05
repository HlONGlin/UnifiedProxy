"""Exercise the real inbound/client/outbound editors on non-loopback HTTP.

Run only in orca-linux /workspace. No browser APIs or backend responses are mocked.
"""
import argparse, ipaddress, json, os, re, socket, subprocess, tempfile, time, urllib.request
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

assert Path.cwd()==Path('/workspace') and Path('/.dockerenv').exists()
parser=argparse.ArgumentParser()
parser.add_argument('--control',default='dist/bootstrap/control')
parser.add_argument('--node',default='dist/bootstrap/node')
parser.add_argument('--role',choices=['control','standalone','both'],default='both')
parser.add_argument('--transport',choices=['http','https','loopback','all'],default='http')
parser.add_argument('--output',default='dist/http-editors')
args=parser.parse_args()
output=Path(args.output).resolve();output.mkdir(parents=True,exist_ok=True)
checks=[]
uuid_pattern=re.compile(r'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$')

def check(name,value):
    assert value,name
    checks.append(name);print('PASS '+name,flush=True)

roles=['control','standalone'] if args.role=='both' else [args.role]
transports=['http','https','loopback'] if args.transport=='all' else [args.transport]
with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,args=['--no-sandbox'])
    for role in roles:
        for transport in transports:
            case=f'{role}-{transport}';errors=[]
            with tempfile.TemporaryDirectory(prefix=case+'-',dir=output) as temp:
                data=Path(temp)/'data';binary=str(Path(args.control if role=='control' else args.node).resolve())
                host='127.0.0.1' if transport=='loopback' else socket.gethostbyname(socket.gethostname())
                if transport!='loopback':assert not ipaddress.ip_address(host).is_loopback
                with socket.socket() as listener:listener.bind(('0.0.0.0',0));port=listener.getsockname()[1]
                origin=f'{"https" if transport=="https" else "http"}://{host}:{port}';base=origin+'/editors'
                tls=[]
                if transport=='https':
                    cert=Path(temp)/'cert.pem';key=Path(temp)/'key.pem'
                    subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-keyout',str(key),'-out',str(cert),'-days','1','-subj','/CN=localhost','-addext',f'subjectAltName=IP:{host}'],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
                    tls=['--cert',str(cert),'--key',str(key)]
                subprocess.run([binary,'configure','--data-dir',str(data),'--username','fixture','--password-stdin','--listen-ip','0.0.0.0','--port',str(port),'--path','editors','--core-path',str(Path('dist/design-audit/fixture-core').resolve()),*tls],input='fixture-password',text=True,check=True,stdout=subprocess.DEVNULL)
                with (output/(case+'.log')).open('w') as log:
                    process=subprocess.Popen([binary,'serve','--data-dir',str(data)],stdout=log,stderr=log)
                    context=browser.new_context(ignore_https_errors=True,viewport={'width':1440,'height':1050})
                    page=context.new_page();page.on('pageerror',lambda error:errors.append(str(error)))
                    try:
                        for _ in range(100):
                            try:
                                if context.request.get(base+'/login',timeout=1000).ok:break
                            except Exception:pass
                            if process.poll() is not None:raise RuntimeError((output/(case+'.log')).read_text())
                            time.sleep(.1)
                        page.goto(base+'/login');page.locator('#username').fill('fixture');page.locator('#password').fill('fixture-password');page.locator('#login-button').click();page.wait_for_url(base+'/')
                        if role=='standalone':page.wait_for_function('Boolean(app.state)')
                        else:expect(page.locator('.fleet-heading')).to_be_visible()
                        page.evaluate("upSetLang('zh-CN')")
                        capabilities=page.evaluate("({secure:isSecureContext,uuid:typeof crypto.randomUUID,clone:typeof structuredClone,random:typeof crypto.getRandomValues})")
                        check(case+': expected browser context',capabilities['secure']==(transport!='http') and capabilities['random']=='function')
                        target=''
                        def api(path,body=None):
                            headers={'X-Fleet-Node':target} if target and not path.startswith('/api/fleet') else {}
                            response=context.request.fetch(base+path,method='POST' if body is not None else 'GET',data=body,headers=headers)
                            assert response.ok,(response.status,response.text())
                            payload=response.json();return payload.get('data',payload)
                        if role=='control':
                            card=api('/api/fleet/nodes',{'name':'HTTP editor node','address':'203.0.113.10'})
                            target=card['id'];page.goto(base+'/nodes/'+target+'/');page.wait_for_function('Boolean(app.state)')
                            expect(page.locator('html')).to_have_attribute('data-node-id',target)
                        page.locator('#nav [data-view=inbounds]').click()
                        page.locator('#add-inbound').click()
                        expect(page.locator('#inbound-form')).to_be_visible(timeout=5000)
                        check(case+': Add inbound opens the drawer',True)
                        if transport=='http':page.screenshot(path=str(output/(case+'-add-inbound.png')),full_page=True)
                        form=page.locator('#inbound-form')
                        for protocol in ('vless','vmess','tuic','mieru','shadowsocks','socks'):
                            form.locator('[name=type]').select_option(protocol)
                            clients=page.evaluate('app.draftInboundClients')
                            check(case+': '+protocol+' generates a client',len(clients)==1 and bool(clients[0]['id']))
                            if protocol in ('vless','vmess','tuic'):check(case+': '+protocol+' has a valid v4 UUID',bool(uuid_pattern.fullmatch(clients[0]['uuid'])))
                        form.locator('[name=type]').select_option('vless')
                        form.locator('[name=name]').fill('HTTP VLESS')
                        page.locator('#save-inbound').click();expect(page.locator('#inbound-form')).not_to_be_visible()
                        inbound=next(item for item in api('/api/state')['state']['inbounds'] if item['name']=='HTTP VLESS')
                        check(case+': inbound and its generated UUID persist',len(inbound['clients'])==1 and bool(uuid_pattern.fullmatch(inbound['clients'][0]['uuid'])))
                        menu=page.locator(f'[data-inbound-menu="{inbound["id"]}"]')
                        menu.click();page.locator('[data-popover-add-client]').click();expect(page.locator('#client-modal')).to_be_visible()
                        client_form=page.locator('#client-form');client_form.locator('[name=name]').fill('extra-user');page.locator('#save-client').click()
                        expect(page.locator('#client-modal')).not_to_be_visible()
                        menu.click();page.locator('[data-popover-add-bulk]').click();expect(page.locator('#client-bulk-modal')).to_be_visible()
                        bulk=page.locator('#client-bulk-form');bulk.locator('[name=last]').fill('2');bulk.locator('button[type=submit]').click()
                        expect(page.locator('#client-bulk-modal')).not_to_be_visible()
                        saved=api('/api/state')['state']['inbounds']
                        inbound=next(item for item in saved if item['name']=='HTTP VLESS')
                        check(case+': inbound and four valid UUID clients persist',len(inbound['clients'])==4 and all(uuid_pattern.fullmatch(c['uuid']) for c in inbound['clients']))
                        workspace=base+'/nodes/'+target if target else base
                        with page.expect_request(lambda request:request.url==workspace+'/api/state') as initial_state:
                            page.reload()
                        if target:
                            check(case+': first state request restores the selected node','/nodes/'+target+'/api/state' in initial_state.value.url)
                            defaults=context.request.get(base+'/api/state').json()['data']['state']
                            check(case+': child edits leave host global inbounds intact',not defaults['inbounds'])
                        page.locator('#nav [data-view=inbounds]').click()
                        expect(page.locator('#inbounds-table')).to_contain_text('HTTP VLESS')
                        check(case+': inbound survives reload',True)
                        page.locator('#nav [data-view=mihomo]').click();page.locator('[data-mihomo-tab=outbounds]').click()
                        page.locator('#mihomo-add-outbound').click();expect(page.locator('#mihomo-outbound-modal')).to_be_visible()
                        outbound=page.locator('#mihomo-outbound-form');outbound.locator('[name=name]').fill('HTTP direct');outbound.locator('[name=type]').select_option('direct');outbound.locator('button[type=submit]').click()
                        expect(page.locator('#mihomo-outbound-modal')).not_to_be_visible()
                        check(case+': outbound draft accepts a generated ID',True)
                        page.locator('[data-mihomo-tab=routing]').click();page.locator('#mihomo-add-rule').click()
                        rule=page.locator('#mihomo-rule-form');rule.locator('[name=payload]').fill('example.com');rule.locator('button[type=submit]').click()
                        expect(page.locator('#mihomo-rule-modal')).not_to_be_visible()
                        page.locator('#mihomo-save').click();expect(page.locator('#mihomo-save')).to_be_disabled()
                        state=api('/api/state')['state']
                        check(case+': outbound and routing rule persist',any(item['name']=='HTTP direct' for item in state['outbounds']) and any(item['payload']=='example.com' and bool(item['id']) for item in state['routingRules']))
                        page.screenshot(path=str(output/(case+'.png')),full_page=True)
                        check(case+': no browser exceptions',not errors)
                    finally:
                        (output/(case+'-errors.json')).write_text(json.dumps(errors,ensure_ascii=False,indent=2))
                        if errors:print('BROWSER ERRORS: '+json.dumps(errors),flush=True)
                        context.close();process.terminate();process.wait(timeout=15)
    browser.close()
(output/'results.json').write_text(json.dumps({'checks':checks},ensure_ascii=False,indent=2))
print(f'Passed {len(checks)} editor checks',flush=True)
