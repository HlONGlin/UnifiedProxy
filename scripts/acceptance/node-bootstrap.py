"""Execute the UI's exact setup command in /workspace with real control/node.

Downloads are supplied by local release fixtures. A systemctl adapter starts
real node processes, so this checks installation and enrollment, not systemd itself.
"""
import argparse, hashlib, http.cookiejar, ipaddress, json, os, re, shutil, socket
import ssl, subprocess, tarfile, tempfile, time, urllib.request
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

assert Path.cwd()==Path('/workspace') and Path('/.dockerenv').exists() and os.geteuid()==0
parser=argparse.ArgumentParser()
parser.add_argument('--control',default='dist/bootstrap/control')
parser.add_argument('--node',default='dist/bootstrap/node')
parser.add_argument('--core',default='dist/design-audit/fixture-core')
parser.add_argument('--output',default='dist/bootstrap-acceptance')
parser.add_argument('--http',action='store_true',help='Use the container non-loopback HTTP address, without TLS')
args=parser.parse_args()
control,node,core=[Path(p).resolve() for p in (args.control,args.node,args.core)]
output=Path(args.output).resolve();output.mkdir(parents=True,exist_ok=True)
checks=[]
def check(name,value):
    assert value,name
    checks.append(name);print('PASS '+name,flush=True)
def free_port():
    with socket.socket() as s:s.bind(('127.0.0.1',0));return s.getsockname()[1]

with tempfile.TemporaryDirectory(prefix='node-bootstrap-',dir=output) as temp:
    root=Path(temp);cert=root/'cert.pem';key=root/'key.pem';data=root/'control-data'
    subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-keyout',str(key),'-out',str(cert),'-days','1','-subj','/CN=localhost','-addext','subjectAltName=DNS:localhost,IP:127.0.0.1'],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    host=socket.gethostbyname(socket.gethostname()) if args.http else 'localhost'
    if args.http:assert not ipaddress.ip_address(host).is_loopback
    origin=f'{"http" if args.http else "https"}://{host}:{free_port()}';base=origin+'/bootstrap'
    tls=[] if args.http else ['--cert',str(cert),'--key',str(key)]
    subprocess.run([str(control),'configure','--data-dir',str(data),'--username','fixture','--password-stdin','--listen-ip','0.0.0.0' if args.http else '127.0.0.1','--port',origin.rsplit(':',1)[1],'--path','bootstrap',*tls],input='fixture-password',text=True,check=True,stdout=subprocess.DEVNULL)
    context=ssl.create_default_context(cafile=str(cert))
    opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPSHandler(context=context),urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    def api(path,body=None):
        request=urllib.request.Request(base+path,None if body is None else json.dumps(body).encode(),{'Content-Type':'application/json'})
        with opener.open(request,timeout=5) as response:value=json.load(response)
        assert value.get('ok'),value
        return value.get('data')
    fixture=root/'node-linux-amd64.tar.gz'
    with tarfile.open(fixture,'w:gz') as archive:
        archive.add(node,arcname='node-linux-amd64')
        archive.add('scripts/install-fleet.sh',arcname='scripts/install-fleet.sh')
    checksum=root/'sha256sums.txt';checksum.write_text(hashlib.sha256(fixture.read_bytes()).hexdigest()+'  ./'+fixture.name+'\n')
    version=subprocess.check_output([str(node),'version'],text=True).strip()
    release=f'https://github.com/HlONGlin/UnifiedProxy/releases/download/{version}/'
    mapping=root/'downloads.json';mapping.write_text(json.dumps({release+fixture.name:str(fixture),release+checksum.name:str(checksum)}))
    commands=root/'commands';commands.mkdir()
    (commands/'curl').write_text('''#!/usr/bin/env python3
import json,os,shutil,sys
args=sys.argv[1:];url=next(v for v in args if v.startswith(('http://','https://')));dest=args[args.index('-o')+1]
if os.environ.get('UP_FAIL_DOWNLOAD')=='1':raise SystemExit(22)
with open(os.environ['UP_DOWNLOADS']) as f:mapping=json.load(f)
if url in mapping:shutil.copyfile(mapping[url],dest)
else:os.execv('/usr/bin/curl',['/usr/bin/curl']+args)
''')
    (commands/'systemctl').write_text('''#!/usr/bin/env python3
import os,signal,subprocess,sys,time
from pathlib import Path
root=Path(os.environ['UNIFIEDPROXY_FLEET_INSTALL_DIR']);pidfile=root/'node.pid';args=sys.argv[1:]
root.mkdir(parents=True,exist_ok=True)
with (root/'systemctl.log').open('a') as log:log.write(' '.join(args)+'\\n')
def alive():
    try:
        pid=int(pidfile.read_text());os.kill(pid,0)
        return Path('/proc/'+str(pid)+'/stat').read_text().split()[2]!='Z'
    except (FileNotFoundError,ProcessLookupError):return False
def stop():
    if alive():
        os.kill(int(pidfile.read_text()),signal.SIGTERM)
        for _ in range(100):
            if not alive():break
            time.sleep(.05)
        if alive():os.kill(int(pidfile.read_text()),signal.SIGKILL)
    pidfile.unlink(missing_ok=True)
if args[0] in ('show-environment','daemon-reload','enable'):raise SystemExit(0)
if args[0]=='is-active':raise SystemExit(0 if alive() else 3)
if args[0]=='stop':stop();raise SystemExit(0)
if args[0] in ('start','restart'):
    stop()
    with (root/'node.log').open('a') as log:
        p=subprocess.Popen([str(root/'node'),'serve','--data-dir',str(root/'data')],stdout=log,stderr=log,start_new_session=True)
    pidfile.write_text(str(p.pid));raise SystemExit(0)
raise SystemExit(1)
''')
    for command in commands.iterdir():command.chmod(0o755)
    install=root/'installed-node';install.mkdir();shutil.copyfile(core,install/'mihomo');(install/'mihomo').chmod(0o755)
    env={**os.environ,'PATH':str(commands)+os.pathsep+os.environ['PATH'],'SSL_CERT_FILE':str(cert),'CURL_CA_BUNDLE':str(cert),'NO_PROXY':os.environ.get('NO_PROXY','')+','+host,'TMPDIR':str(root),'UP_DOWNLOADS':str(mapping),'UNIFIEDPROXY_FLEET_INSTALL_DIR':str(install),'UNIFIEDPROXY_SYSTEMD_UNIT_DIR':str(root/'units')}
    def execute(command,extra=None):return subprocess.run(['bash','-c',command],env={**env,**(extra or {})},capture_output=True,text=True,timeout=90)
    def wait_node(node_id):
        deadline=time.monotonic()+30
        while time.monotonic()<deadline:
            view=next(n for n in api('/api/fleet')['nodes'] if n['id']==node_id)
            if view['online'] and view['status']['version']==version:return view
            time.sleep(.2)
        raise AssertionError('installed node did not send a signed status')
    with (root/'control.log').open('w') as log:
        process=subprocess.Popen([str(control),'serve','--data-dir',str(data)],stdout=log,stderr=log)
        try:
            for _ in range(100):
                try:opener.open(base+'/login',timeout=1).close();break
                except Exception:time.sleep(.1)
            api('/api/auth/login',{'username':'fixture','password':'fixture-password'})
            with sync_playwright() as pw:
                browser=pw.chromium.launch(headless=True,args=['--no-sandbox'])
                page=browser.new_page(ignore_https_errors=True)
                page.goto(base+'/login');page.locator('#username').fill('fixture');page.locator('#password').fill('fixture-password');page.locator('#login-button').click();page.wait_for_url(base+'/')
                page.evaluate("upSetLang('zh-CN')")
                page.locator('#view-fleet').get_by_role('button',name='＋ 添加子服务器',exact=True).click()
                dialog=page.locator('dialog[open]');dialog.get_by_label('服务器名称',exact=False).fill('一键接入验证')
                dialog.get_by_role('button',name='创建并获取接入命令',exact=True).click()
                expect(dialog.locator('[data-enrollment-node]')).to_be_visible()
                node_id=dialog.locator('[data-enrollment-node]').get_attribute('data-enrollment-node');command=dialog.locator('code').first.inner_text()
                check('Command pins the matching host release','--version '+"'"+version+"'" in command)
                if args.http:
                    check('Non-loopback HTTP panel needs no secure-context clipboard',not page.evaluate('window.isSecureContext') and not page.evaluate('Boolean(navigator.clipboard)'))
                    page.evaluate("document.addEventListener('copy',()=>{const f=document.activeElement;window.upCopiedSelection=f.value.substring(f.selectionStart,f.selectionEnd);},{once:true})")
                    dialog.get_by_role('button',name='复制一键接入命令',exact=True).click()
                    expect(dialog.locator('.fleet-dialog-feedback')).to_have_text('一键接入命令已复制，到子服务器粘贴回车即可。')
                    check('HTTP clipboard fallback copies the complete command',page.evaluate('window.upCopiedSelection')==command)
                    page.screenshot(path=str(output/'http-enrollment.png'),full_page=True)
                failed=execute(command,{'UP_FAIL_DOWNLOAD':'1'})
                check('Script download failure stops before installation',failed.returncode!=0 and not (install/'node').exists())
                result=execute(command)
                if result.returncode:raise AssertionError(result.stdout+result.stderr)
                view=wait_node(node_id)
                check('Exact pasted UI command installs and enrolls a real node',view['configuredAddress']=='' and (install/'node').exists() and (install/'data/agent.json').exists())
                check('Agent preserves the selected HTTP or HTTPS control origin',json.loads((install/'data/agent.json').read_text())['controlURL']==origin)
                settings=json.loads((install/'data/state.json').read_text())['settings']
                check('Fresh node panel listens on loopback',settings['panelListen'].startswith('127.0.0.1:'))
                unit=(root/'units/unifiedproxy-node.service').read_text()
                check('Service and recovery timer target the installed node',str(install/'node') in unit and (root/'units/unifiedproxy-node-recovery.timer').exists())
                identity=(install/'data/agent-identity.json').read_bytes();saved_settings=json.loads((install/'data/state.json').read_text())['settings']
                result=execute(command)
                if result.returncode:raise AssertionError(result.stdout+result.stderr)
                check('Repeating a consumed command preserves identity and settings',identity==(install/'data/agent-identity.json').read_bytes() and saved_settings==json.loads((install/'data/state.json').read_text())['settings'])
                foreign=api('/api/fleet/nodes',{'name':'another card','address':''});info=api('/api/fleet/tokens',{'nodeId':foreign['id']})
                bad=command.replace(node_id,foreign['id'])
                result=execute(bad)
                check('An existing enrollment cannot be replaced by another card',result.returncode!=0 and identity==(install/'data/agent-identity.json').read_bytes() and json.loads((install/'data/agent.json').read_text())['nodeId']==node_id)
                legacy=root/'existing-standalone';legacy.mkdir()
                subprocess.run([str(node),'configure','--data-dir',str(legacy/'data'),'--username','existing-admin','--password-stdin','--listen-ip','127.0.0.1','--port',str(free_port()),'--path','keep-existing-path','--core-path',str(core)],input='keep-existing-password',text=True,check=True,stdout=subprocess.DEVNULL)
                previous=json.loads((legacy/'data/state.json').read_text())['settings']
                old_token=re.search(r"printf '%s' '([a-fA-F0-9]+)'",command).group(1)
                result=execute(bad.replace(old_token,info['token']),{'UNIFIEDPROXY_FLEET_INSTALL_DIR':str(legacy)})
                if result.returncode:raise AssertionError(result.stdout+result.stderr)
                check('Existing standalone settings and core survive enrollment',previous==json.loads((legacy/'data/state.json').read_text())['settings'])
                backups=list((legacy/'backups').glob('*.tar.gz'))
                check('Existing data is backed up before installation',len(backups)==1 and 'data/state.json' in tarfile.open(backups[0]).getnames())
                subprocess.run([str(commands/'systemctl'),'stop','unifiedproxy-node'],env={**env,'UNIFIEDPROXY_FLEET_INSTALL_DIR':str(legacy)},check=True)
                other=root/'checksum-rejection';other.mkdir();shutil.copyfile(core,other/'mihomo');(other/'mihomo').chmod(0o755)
                corrupt=root/'corrupt.tar.gz';corrupt.write_bytes(b'not a release');mapping.write_text(json.dumps({release+fixture.name:str(corrupt),release+checksum.name:str(checksum)}))
                result=execute(command,{'UNIFIEDPROXY_FLEET_INSTALL_DIR':str(other)})
                check('Bad release checksum never creates a node identity',result.returncode!=0 and not (other/'data/agent.json').exists() and not (other/'node').exists())
                browser.close()
        finally:
            subprocess.run([str(commands/'systemctl'),'stop','unifiedproxy-node'],env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
            process.terminate();process.wait(timeout=15)
(output/'results.json').write_text(json.dumps({'checks':checks,'init':'systemctl adapter; real control/node processes'},ensure_ascii=False,indent=2))
print(f'Passed {len(checks)} bootstrap checks; evidence: {output}',flush=True)
