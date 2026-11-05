"""Fresh disposable orca-linux systemd container only; runs under /workspace.

Uses real systemd, HTTP(S) control, node and upstream pinned Mihomo. Only the
not-yet-published UnifiedProxy release download is supplied as a local fixture.
"""
import argparse, hashlib, http.cookiejar, ipaddress, json, os, shlex, socket, ssl, subprocess, tarfile
import tempfile, time, urllib.request
from pathlib import Path

assert Path.cwd()==Path('/workspace') and Path('/.dockerenv').exists() and os.geteuid()==0
parser=argparse.ArgumentParser()
parser.add_argument('--http',action='store_true',help='Use the container non-loopback HTTP address, without TLS')
parser.add_argument('--published',action='store_true',help='Download the actual published node package instead of a local fixture')
args=parser.parse_args()
install=Path('/usr/local/unifiedproxy-node')
assert not install.exists(),'Requires a fresh disposable container'
output=Path('dist/bootstrap-systemd');output.mkdir(parents=True,exist_ok=True)
control=Path('dist/bootstrap/control').resolve();node=Path('dist/bootstrap/node').resolve()
checks=[]
def check(name,value):
    assert value,name
    checks.append(name);print('PASS '+name,flush=True)
with tempfile.TemporaryDirectory(prefix='bootstrap-systemd-',dir='/workspace') as temp:
    root=Path(temp);data=root/'control-data';cert=root/'cert.pem';key=root/'key.pem'
    subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-keyout',str(key),'-out',str(cert),'-days','1','-subj','/CN=localhost','-addext','subjectAltName=DNS:localhost,IP:127.0.0.1'],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
    with socket.socket() as listener:listener.bind(('127.0.0.1',0));port=listener.getsockname()[1]
    host=socket.gethostbyname(socket.gethostname()) if args.http else 'localhost'
    if args.http:assert not ipaddress.ip_address(host).is_loopback
    origin=f'{"http" if args.http else "https"}://{host}:{port}';base=origin+'/bootstrap'
    tls=[] if args.http else ['--cert',str(cert),'--key',str(key)]
    subprocess.run([str(control),'configure','--data-dir',str(data),'--username','fixture','--password-stdin','--listen-ip','0.0.0.0' if args.http else '127.0.0.1','--port',str(port),'--path','bootstrap',*tls],input='fixture-password',text=True,check=True,stdout=subprocess.DEVNULL)
    opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPSHandler(context=ssl.create_default_context(cafile=str(cert))),urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    def api(path,body=None):
        request=urllib.request.Request(base+path,None if body is None else json.dumps(body).encode(),{'Content-Type':'application/json'})
        with opener.open(request,timeout=10) as response:value=json.load(response)
        assert value.get('ok'),value
        return value.get('data')
    archive=root/'node-linux-amd64.tar.gz'
    with tarfile.open(archive,'w:gz') as bundle:
        bundle.add(node,arcname='node-linux-amd64');bundle.add('scripts/install-fleet.sh',arcname='scripts/install-fleet.sh')
    checksum=root/'sha256sums.txt';checksum.write_text(hashlib.sha256(archive.read_bytes()).hexdigest()+'  ./'+archive.name+'\n')
    version=subprocess.check_output([str(node),'version'],text=True).strip()
    downloads=root/'downloads.json';release=f'https://github.com/HlONGlin/UnifiedProxy/releases/download/{version}/'
    downloads.write_text(json.dumps({} if args.published else {release+archive.name:str(archive),release+checksum.name:str(checksum)}))
    commands=root/'bin';commands.mkdir()
    (commands/'curl').write_text('''#!/usr/bin/env python3
import json,os,shutil,sys
args=sys.argv[1:]
with open(os.environ['UP_DOWNLOADS']) as f:mapping=json.load(f)
url=next(v for v in args if v.startswith(('http://','https://')))
if url in mapping:shutil.copyfile(mapping[url],args[args.index('-o')+1])
else:os.execv('/usr/bin/curl',['/usr/bin/curl']+args)
''');(commands/'curl').chmod(0o755)
    env={**os.environ,'PATH':str(commands)+os.pathsep+os.environ['PATH'],'UP_DOWNLOADS':str(downloads),'SSL_CERT_FILE':str(cert),'CURL_CA_BUNDLE':str(cert),'NO_PROXY':os.environ.get('NO_PROXY','')+','+host}
    # A public root bundle is needed for the actual Mihomo download as well.
    bundle=root/'ca.pem';bundle.write_bytes(Path('/etc/ssl/certs/ca-certificates.crt').read_bytes()+cert.read_bytes())
    env['SSL_CERT_FILE']=env['CURL_CA_BUNDLE']=str(bundle)
    subprocess.run(['systemctl','set-environment','SSL_CERT_FILE='+str(bundle)],check=True)
    with (output/'control.log').open('w') as log:
        process=subprocess.Popen([str(control),'serve','--data-dir',str(data)],stdout=log,stderr=log)
        try:
            for _ in range(100):
                try:opener.open(base+'/login',timeout=1).close();break
                except Exception:time.sleep(.1)
            api('/api/auth/login',{'username':'fixture','password':'fixture-password'})
            card=api('/api/fleet/nodes',{'name':'systemd fresh node','address':''});info=api('/api/fleet/tokens',{'nodeId':card['id']})
            q=shlex.quote
            protocols='=http,https' if args.http else '=https'
            command='( set -euo pipefail; up_install_script="$(mktemp)"; trap \'rm -f "$up_install_script"\' EXIT; curl -fsSL --proto '+q(protocols)+' --proto-redir '+q(protocols)+' --connect-timeout 10 --max-time 60 '+q(origin+'/fleet/v1/install.sh')+' -o "$up_install_script"; printf \'%s\' '+q(info['token'])+' | bash "$up_install_script" --control '+q(origin)+' --control-key '+q(info['controlKey'])+' --token-stdin --node-id '+q(card['id'])+' --version '+q(version)+' )'
            result=subprocess.run(['bash','-c',command],env=env,text=True,capture_output=True,timeout=300)
            (output/'installer.log').write_text(result.stdout+result.stderr)
            if result.returncode:raise AssertionError(result.stdout+result.stderr)
            check('Fresh server installs node and pinned real Mihomo',(install/'node').exists() and 'v1.19.33' in subprocess.check_output([str(install/'mihomo'),'-v'],text=True))
            check('Agent uses the selected HTTP or HTTPS control origin',json.loads((install/'data/agent.json').read_text())['controlURL']==origin)
            subprocess.run(['systemctl','is-active','--quiet','unifiedproxy-node','unifiedproxy-node-recovery.timer'],check=True)
            check('Real systemd service and recovery timer are active',True)
            deadline=time.monotonic()+45
            while time.monotonic()<deadline:
                view=api('/api/fleet')['nodes'][0]
                if view['online'] and view['status'].get('running') and view['status'].get('version')==version:break
                time.sleep(.5)
            check('Fresh node is online with its real Mihomo running',view['online'] and view['status'].get('running') and view['status'].get('version')==version)
            before=view['lastSeen'];deadline=time.monotonic()+35
            while time.monotonic()<deadline:
                view=api('/api/fleet')['nodes'][0]
                if view['lastSeen']!=before and view.get('seenIP'):break
                time.sleep(.5)
            check('Heartbeat continues and auto public IP resolves',view['lastSeen']!=before and bool(view.get('seenIP')) and view['addressSource']=='auto')
            original=(install/'data/agent-identity.json').read_bytes()
            result=subprocess.run(['bash','-c',command],env=env,text=True,capture_output=True,timeout=60)
            check('Repeating the same command keeps the real enrollment',result.returncode==0 and (install/'data/agent-identity.json').read_bytes()==original)
        finally:
            subprocess.run(['systemctl','stop','unifiedproxy-node','unifiedproxy-node-recovery.timer'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
            subprocess.run(['systemctl','unset-environment','SSL_CERT_FILE'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
            process.terminate();process.wait(timeout=15)
(output/'results.json').write_text(json.dumps({'checks':checks,'transport':'http' if args.http else 'https','downloads':'published node release and actual pinned upstream Mihomo' if args.published else 'local node release fixture; actual pinned upstream Mihomo'},ensure_ascii=False,indent=2))
print(f'Passed {len(checks)} real systemd bootstrap checks',flush=True)
