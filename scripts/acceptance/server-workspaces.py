"""Separate fleet/server pages with one control and two actual enrolled agents.

Mihomo defaults to a local fixture; --core accepts a real executable to verify
proxy listeners and live sessions. Authentication, enrollment and UI use real handlers.
"""
import argparse, importlib.util, io, json, os, signal, socket, socketserver, subprocess, tempfile, threading, time, zipfile
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

assert Path.cwd()==Path('/workspace') and Path('/.dockerenv').exists()
parser=argparse.ArgumentParser()
parser.add_argument('--control',default='dist/bootstrap/control')
parser.add_argument('--node',default='dist/bootstrap/node')
parser.add_argument('--core',help='Use a real Mihomo executable, including proxy listener checks')
parser.add_argument('--output',default='dist/server-workspaces')
parser.add_argument('--quick',action='store_true')
args=parser.parse_args()
output=Path(args.output).resolve();output.mkdir(parents=True,exist_ok=True)
control,node=[str(Path(p).resolve()) for p in (args.control,args.node)]
core=str(Path(args.core or 'dist/design-audit/fixture-core').resolve())
spec=importlib.util.spec_from_file_location('measure','scripts/acceptance/final-browser.py')
measure=importlib.util.module_from_spec(spec);spec.loader.exec_module(measure)
checks=[];errors=[];contrasts=[];processes=[];streams=[]
echo_server=None

def check(name,condition):
    assert condition,name
    checks.append(name);print('PASS '+name,flush=True)

def port():
    with socket.socket() as s:s.bind(('127.0.0.1',0));return s.getsockname()[1]

def reachable(address):
    host,number=address.rsplit(':',1)
    try:
        with socket.create_connection((host,int(number)),timeout=.2):return True
    except OSError:return False

def launch(command,name):
    stream=(output/(name+'.log')).open('w');streams.append(stream)
    p=subprocess.Popen(command,stdout=stream,stderr=stream,start_new_session=True);processes.append(p);return p

def configure(binary,data,panel_port,path='console'):
    subprocess.run([binary,'configure','--data-dir',str(data),'--username','fixture','--password-stdin',
        '--listen-ip','127.0.0.1','--port',str(panel_port),'--path',path,'--core-path',core],input='fixture-password',text=True,check=True,stdout=subprocess.DEVNULL)
    file=data/'state.json';state=json.loads(file.read_text());state['settings']['apiAddress']='127.0.0.1:'+str(port());file.write_text(json.dumps(state))

def shot(page,name):
    page.screenshot(path=str(output/(name+'.png')),full_page=True,animations='disabled')
    rows=page.evaluate(measure.MEASURE)
    contrasts.append({'surface':name,'measurements':rows,'failures':[r for r in rows if r['ratio']<r['required']]})

try:
    with tempfile.TemporaryDirectory(prefix='up-workspaces-') as temp,sync_playwright() as pw:
        temp=Path(temp);origin='http://127.0.0.1:'+str(port());base=origin+'/console'
        data=temp/'control';configure(control,data,int(origin.rsplit(':',1)[1]))
        launch([control,'serve','--data-dir',str(data)],'control')
        browser=pw.chromium.launch(headless=True,args=['--no-sandbox'])
        ctx=browser.new_context(viewport={'width':1440,'height':1050})
        page=ctx.new_page();page.on('pageerror',lambda e:errors.append(str(e)))
        for _ in range(100):
            try:
                if ctx.request.get(base+'/login',timeout=1000).ok:break
            except Exception:pass
            time.sleep(.1)
        page.goto(base+'/login');page.locator('#username').fill('fixture');page.locator('#password').fill('fixture-password');page.locator('#login-button').click();page.wait_for_url(base+'/')
        expect(page.locator('.fleet-heading')).to_be_visible()
        check('Fleet is a separate page without a hidden server dashboard',page.locator('#view-dashboard').count()==0 and page.locator('#fleet-target-select').count()==0)
        expect(page.locator('#portal-logout')).to_have_text('退出登录')

        def api(path,body=None,method=None):
            response=ctx.request.fetch(base+path,method=method or ('POST' if body is not None else 'GET'),data=body)
            assert response.ok,(path,response.status,response.text())
            return response.json().get('data')

        nodes=[]
        for i,name in enumerate(('香港入口','东京出口')):
            card=api('/api/fleet/nodes',{'name':name,'address':'203.0.113.'+str(10+i)})
            child=temp/('node-'+str(i));configure(node,child,port())
            info=api('/api/fleet/tokens',{'nodeId':card['id'],'name':name})
            subprocess.run([node,'agent','join','--data-dir',str(child),'--control',origin,'--control-key',info['controlKey'],'--token-stdin'],input=info['token'],text=True,check=True,stdout=subprocess.DEVNULL)
            model=api('/api/fleet/nodes/'+card['id'])['draft']
            model['inbounds']=[{'id':'inbound-'+str(i),'name':name+'专属入站','type':'socks','listen':'127.0.0.1','port':port(),'enabled':True,'clients':[{'id':'user-'+str(i),'name':name+'用户','username':'user-'+str(i),'password':'fixture-node-password','enabled':True}]}]
            api('/api/fleet/nodes/'+card['id']+'/model',model)
            process=launch([node,'serve','--data-dir',str(child)],'node-'+str(i))
            settings=json.loads((child/'state.json').read_text())['settings']
            addresses=[settings['apiAddress']]
            if args.core:addresses.append('127.0.0.1:'+str(model['inbounds'][0]['port']))
            nodes.append({'id':card['id'],'name':name,'data':child,'process':process,'addresses':addresses})

        deadline=time.monotonic()+60
        while True:
            fleet=api('/api/fleet')['nodes']
            if all(n['online'] and n['status'].get('details',{}).get('system',{}).get('cpuCores',0)>0 and n['status'].get('revision')==n['desiredRevision'] for n in fleet):break
            assert time.monotonic()<deadline,('agents did not report telemetry',fleet)
            time.sleep(.3)
        check('Two actual agents enroll, apply policies and report machine telemetry',True)
        page.reload(wait_until='networkidle');expect(page.locator('.fleet-node-card')).to_have_count(2)
        page.locator(f'[data-node-card="{nodes[0]["id"]}"] .fleet-card-actions .primary-btn').click()
        page.wait_for_url(base+'/nodes/'+nodes[0]['id']+'/')
        expect(page.locator('#view-dashboard')).to_be_visible()
        expect(page.locator('.server-workspace-identity strong')).to_have_text(nodes[0]['name'])
        check('A card directly opens its dashboard without a second login',page.locator('#login-form').count()==0)
        first=page
        second=ctx.new_page();second.on('pageerror',lambda e:errors.append(str(e)))
        second.goto(base+'/nodes/'+nodes[1]['id']+'/#inbounds')
        expect(second.locator('#inbounds-table')).to_contain_text(nodes[1]['name']+'专属入站')
        first.locator('#nav [data-view=inbounds]').click()
        expect(first.locator('#inbounds-table')).to_contain_text(nodes[0]['name']+'专属入站')
        check('Two browser tabs retain independent inbound lists',not first.locator('#inbounds-table').get_by_text(nodes[1]['name']+'专属入站',exact=True).count() and not second.locator('#inbounds-table').get_by_text(nodes[0]['name']+'专属入站',exact=True).count())
        for p,n in ((first,nodes[0]),(second,nodes[1])):
            p.locator('#add-inbound').click();expect(p.locator('#inbound-form')).to_be_visible()
            form=p.locator('#inbound-form');form.locator('[name=name]').fill(n['name']+'浏览器新增');form.locator('[name=type]').select_option('socks')
            p.locator('#save-inbound').click();expect(p.locator('#inbound-form')).not_to_be_visible()
        for i,n in enumerate(nodes):
            state=api('/nodes/'+n['id']+'/api/state')
            check(n['name']+': independent saved configuration and real system stats',len(state['state']['inbounds'])==2 and all(x['name'].startswith(n['name']) for x in state['state']['inbounds']) and state['system']['cpuCores']>0 and state['online'])
            history=api('/nodes/'+n['id']+'/api/server/cpuHistory/30')
            check(n['name']+': own CPU history reported by its agent',bool(history))
            p=(first,second)[i]
            check(n['name']+': copied node links use the child public address',p.evaluate('shareServerAddress()')=='203.0.113.'+str(10+i))
            host_tokens=api('/api/fleet/subscriptions')
            user='user-'+str(i)
            check(n['name']+': subscription tokens belong only to this server users',user in state['state']['subscriptionTokens'] and 'user-'+str(1-i) not in state['state']['subscriptionTokens'] and state['state']['subscriptionTokens'][user]==host_tokens[user])
            links=p.evaluate('allSubscriptionLines()')
            check(n['name']+': copied subscription opens the host public endpoint',bool(links) and all(ctx.request.get(link,headers={'Accept':'text/html'}).ok for link in links))
            rotated=api('/nodes/'+n['id']+'/api/subscriptions/token',{'username':user})['token']
            check(n['name']+': token rotation persists in the central subscription service',api('/api/fleet/subscriptions')[user]==rotated)
        def wait_for(predicate,timeout=50):
            deadline=time.monotonic()+timeout
            while not predicate():
                assert time.monotonic()<deadline,'Timed out waiting for server state'
                time.sleep(.2)

        target=nodes[0];target_base='/nodes/'+target['id']
        if args.core:
            class Echo(socketserver.BaseRequestHandler):
                def handle(self):
                    try:
                        while data:=self.request.recv(1024):self.request.sendall(data)
                    except ConnectionError:pass
            echo_server=socketserver.ThreadingTCPServer(('127.0.0.1',0),Echo)
            echo_server.daemon_threads=True
            threading.Thread(target=echo_server.serve_forever,daemon=True).start()
            def read_exact(stream,count):
                data=b''
                while len(data)<count:
                    part=stream.recv(count-len(data));assert part,'Unexpected SOCKS EOF';data+=part
                return data
            def proxy_stream(index):
                stream=socket.create_connection(('127.0.0.1',int(nodes[index]['addresses'][1].rsplit(':',1)[1])),timeout=2)
                stream.sendall(b'\x05\x01\x02');assert read_exact(stream,2)==b'\x05\x02'
                username=('user-'+str(index)).encode();password=b'fixture-node-password'
                stream.sendall(b'\x01'+bytes([len(username)])+username+bytes([len(password)])+password)
                assert read_exact(stream,2)==b'\x01\x00'
                stream.sendall(b'\x05\x01\x00\x01\x7f\x00\x00\x01'+echo_server.server_address[1].to_bytes(2,'big'))
                reply=read_exact(stream,4);assert reply[1]==0,reply
                read_exact(stream,{1:4,4:16}[reply[3]]+2)
                return stream
            def relays(stream):
                try:stream.sendall(b'live-proxy-check');return stream.recv(16)==b'live-proxy-check'
                except OSError:return False
            selected_stream=proxy_stream(0);other_stream=proxy_stream(1)
            check('Both servers carry real authenticated proxy traffic before disabling',relays(selected_stream) and relays(other_stream))
            for enabled in (False,True):
                first.evaluate('refresh()')
                switch=first.locator('[data-toggle-inbound="inbound-0"]')
                with first.expect_response(lambda r:r.url.endswith('/api/inbounds') and r.request.method=='POST'):
                    switch.locator('..').click()
                expect(switch).to_be_disabled()
                first.evaluate('renderInboundTable()')
                expect(switch).to_be_disabled()
                check('Inbound switch remains busy through table refresh '+str(enabled),switch.get_attribute('aria-busy')=='true')
                detail=api('/api/fleet/nodes/'+target['id'])
                revision=detail['desired']['revision']
                wait_for(lambda:api('/api/fleet/nodes/'+target['id'])['status']['revision']==revision)
                local=json.loads((target['data']/'state.json').read_text())
                check('Inbound switch saves the requested enabled state '+str(enabled),next(x for x in local['inbounds'] if x['id']=='inbound-0')['enabled']==enabled)
                check('Inbound switch changes the real proxy listener '+str(enabled),reachable(target['addresses'][1])==enabled)
                check('Inbound switch leaves the other server untouched '+str(enabled),all(reachable(address) for address in nodes[1]['addresses']))
                kept=next(x for x in local['inbounds'] if x['id']!='inbound-0')
                check('Inbound switch preserves another listener on the same server '+str(enabled),reachable('127.0.0.1:'+str(kept['port'])))
                if not enabled:
                    check('Disabling an inbound disconnects its already established proxy sessions',not relays(selected_stream))
                    check('Disabling an inbound preserves another server live session',relays(other_stream))
                expect(switch).not_to_be_disabled(timeout=50000)
                check('Inbound switch confirms actual application before unlocking '+str(enabled),switch.is_checked()==enabled)
            selected_stream.close();other_stream.close()
        first.locator('#nav [data-view=dashboard]').click()
        wait_for(lambda:all(reachable(address) for n in nodes for address in n['addresses']))
        first.evaluate('refresh()')
        with first.expect_response(lambda r:r.url.endswith('/api/core/stop') and r.request.method=='POST') as response:
            first.locator('#overview-core-toggle').click()
        stop_task=response.value.json()['data']
        check('Stop is dispatched only to the selected server',stop_task['nodeId']==target['id'] and stop_task['payload']['action']=='stop')
        check('Core control task is reported before state convergence',stop_task['kind']=='core-control' and stop_task['payload']['action']=='stop')
        wait_for(lambda:api(target_base+'/api/node-tasks/'+stop_task['id']).get('result'))
        wait_for(lambda:not api(target_base+'/api/state')['running'])
        check('Stopping closes the actual server listeners',not any(reachable(address) for address in target['addresses']))
        check('Stopping one server leaves the other server running',all(reachable(address) for address in nodes[1]['addresses']) and api('/nodes/'+nodes[1]['id']+'/api/state')['running'])
        expect(first.locator('#overview-core-toggle')).not_to_be_disabled(timeout=50000)
        first.evaluate('refresh()');expect(first.locator('#overview-core-toggle')).to_contain_text('启动')
        model=api('/api/fleet/nodes/'+target['id'])['draft'];model['inbounds'][0]['notes']='Saved while manually stopped'
        desired=api('/api/fleet/nodes/'+target['id']+'/model',model)
        wait_for(lambda:api('/api/fleet/nodes/'+target['id'])['status']['revision']==desired['revision'])
        check('Saving configuration keeps a manually stopped server stopped',not api(target_base+'/api/state')['running'] and not any(reachable(address) for address in target['addresses']))
        target['process'].terminate();target['process'].wait(timeout=10)
        target['process']=launch([node,'serve','--data-dir',str(target['data'])],'node-0-restarted')
        old_seen=api('/api/fleet/nodes/'+target['id'])['lastSeen']
        wait_for(lambda:api('/api/fleet/nodes/'+target['id'])['lastSeen']!=old_seen)
        check('Restarting the node service preserves manual stop and heartbeat',api(target_base+'/api/state')['online'] and not api(target_base+'/api/state')['running'] and not any(reachable(address) for address in target['addresses']))
        first.evaluate('refresh()')
        first.locator('#overview-core-toggle').click()
        wait_for(lambda:api(target_base+'/api/state')['running'] and all(reachable(address) for address in target['addresses']))
        expect(first.locator('#overview-core-toggle')).not_to_be_disabled(timeout=50000)
        first.evaluate('refresh()');expect(first.locator('#overview-core-toggle')).to_contain_text('停止')
        check('Start restores the saved configuration and actual listeners',True)
        first.locator('#overview-core-restart').click()
        expect(first.locator('#overview-core-restart')).not_to_be_disabled(timeout=50000)
        wait_for(lambda:api(target_base+'/api/state')['running'] and all(reachable(address) for address in target['addresses']))
        check('Remote restart completes with the target server healthy',True)
        first.locator('#nav [data-view=inbounds]').click()
        task=api('/nodes/'+nodes[0]['id']+'/api/mihomo/outbound/delay',{'name':'DIRECT'})
        check('Outbound testing is dispatched to the selected child',task['kind']=='outbound-delay' and task['nodeId']==nodes[0]['id'])
        foreign=ctx.request.get(base+'/nodes/'+nodes[1]['id']+'/api/node-tasks/'+task['id'])
        check('A workspace cannot read another server task',foreign.status==404)
        deadline=time.monotonic()+45
        while not task.get('result'):
            assert time.monotonic()<deadline,task
            time.sleep(.3);task=api('/nodes/'+nodes[0]['id']+'/api/node-tasks/'+task['id'])
        check('The child returns its outbound test result',task['result']['status']=='success' and 'delay' in task['result']['data'])
        backup_url=base+'/nodes/'+nodes[0]['id']+'/api/backup'
        backup=ctx.request.get(backup_url)
        assert backup.ok,backup.text()
        with zipfile.ZipFile(io.BytesIO(backup.body())) as archive:
            backed_up=json.loads(archive.read('state.json'))
        check('Server backup contains only the selected server configuration',len(backed_up['inbounds'])==2 and all(x['name'].startswith(nodes[0]['name']) for x in backed_up['inbounds']))
        restore=ctx.request.post(backup_url,multipart={'backup':{'name':'node-backup.zip','mimeType':'application/zip','buffer':backup.body()}})
        assert restore.ok,restore.text()
        check('Restoring a server backup leaves the other server intact',all(x['name'].startswith(nodes[1]['name']) for x in api('/nodes/'+nodes[1]['id']+'/api/state')['state']['inbounds']))
        check('Server changes leave host runtime configuration intact',not api('/api/state')['state']['inbounds'])
        first.reload(wait_until='networkidle');expect(first.locator('#view-inbounds')).to_be_visible()
        check('Reload preserves the server and active page',first.url.endswith('/nodes/'+nodes[0]['id']+'/#inbounds'))
        second.reload(wait_until='networkidle');expect(second.locator('.server-workspace-identity strong')).to_have_text(nodes[1]['name'])
        first.locator('.server-workspace-bar a').click();expect(first.locator('.fleet-heading')).to_be_visible()
        first.go_back(wait_until='networkidle');expect(first.locator('#view-inbounds')).to_be_visible()
        check('Back to fleet and browser Back restore the server workspace',True)
        for palette,mode in ([('t0','light')] if args.quick else [('t'+str(i),m) for i in range(5) for m in ('light','dark')]):
            theme=palette+'-'+mode
            first.evaluate('([p,m])=>{UPTheme.setColor(p);UPTheme.setMode(m)}',[palette,mode])
            first.locator('#nav [data-view=dashboard]').click();shot(first,theme+'-dashboard')
            first.locator('#nav [data-view=inbounds]').click();first.locator('#add-inbound').click()
            backdrop=first.locator('#drawer-backdrop')
            color=backdrop.evaluate('e=>getComputedStyle(e).backgroundColor')
            alpha=float(color.rstrip(')').split(',')[-1]) if color.startswith('rgba') else 1
            check(theme+': modal backdrop remains translucent',0<alpha<.7)
            shot(first,theme+'-inbound-modal')
            first.locator('#cancel-drawer').click()
            first.set_viewport_size({'width':390,'height':844})
            first.locator('#add-inbound').click();expect(first.locator('#inbound-form')).to_be_visible()
            check(theme+': mobile editor fits the screen',first.locator('.drawer').evaluate('e=>{const r=e.getBoundingClientRect();return r.left>=0&&r.right<=innerWidth+1&&r.height<=innerHeight+1}'))
            shot(first,theme+'-mobile-modal');first.locator('#cancel-drawer').click();first.set_viewport_size({'width':1440,'height':1050})
        guest=browser.new_context();deep=guest.new_page()
        deep.goto(base+'/nodes/'+nodes[1]['id']+'/')
        expect(deep.locator('#login-form')).to_be_visible()
        deep.locator('#username').fill('fixture');deep.locator('#password').fill('fixture-password');deep.locator('#login-button').click()
        deep.wait_for_url(base+'/nodes/'+nodes[1]['id']+'/')
        expect(deep.locator('.server-workspace-identity strong')).to_have_text(nodes[1]['name'])
        check('A direct link returns to its server after the first login',True)
        guest.close()
        first.goto(base+'/');expect(first.locator('.fleet-heading')).to_be_visible()
        first.locator('#host-settings').click();expect(first.locator('#view-settings')).to_be_visible()
        check('Host authentication and shared defaults remain accessible',True)
        first.locator('.server-workspace-bar a').click()
        first.locator('#portal-logout').click();expect(first.locator('#login-form')).to_be_visible()
        second.reload();expect(second.locator('#login-form')).to_be_visible()
        check('A single logout expires every server workspace',True)
        check('No browser exceptions',not errors)
        failures=[{'surface':s['surface'],**r} for s in contrasts for r in s['failures']]
        (output/'contrast.json').write_text(json.dumps(contrasts,ensure_ascii=False,indent=2))
        check('All measured workspace and modal text passes contrast',not failures)
        browser.close()
finally:
    if echo_server is not None:echo_server.shutdown();echo_server.server_close()
    for process in reversed(processes):
        if process.poll() is None:os.killpg(process.pid,signal.SIGTERM)
    for process in reversed(processes):
        try:process.wait(timeout=10)
        except subprocess.TimeoutExpired:os.killpg(process.pid,signal.SIGKILL);process.wait()
    for stream in streams:stream.close()
    (output/'results.json').write_text(json.dumps({'checks':checks,'errors':errors},ensure_ascii=False,indent=2))
print(f'Passed {len(checks)} server workspace checks',flush=True)
