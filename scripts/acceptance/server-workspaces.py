"""Separate fleet/server pages with one control and two actual enrolled agents.

Only the Mihomo executable is a local fixture. Authentication, signed enrollment,
heartbeats, telemetry collection, edits and browser navigation use real handlers.
"""
import argparse, importlib.util, io, json, os, signal, socket, subprocess, tempfile, time, zipfile
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

assert Path.cwd()==Path('/workspace') and Path('/.dockerenv').exists()
parser=argparse.ArgumentParser()
parser.add_argument('--control',default='dist/bootstrap/control')
parser.add_argument('--node',default='dist/bootstrap/node')
parser.add_argument('--output',default='dist/server-workspaces')
parser.add_argument('--quick',action='store_true')
args=parser.parse_args()
output=Path(args.output).resolve();output.mkdir(parents=True,exist_ok=True)
control,node=[str(Path(p).resolve()) for p in (args.control,args.node)]
core=str(Path('dist/design-audit/fixture-core').resolve())
spec=importlib.util.spec_from_file_location('measure','scripts/acceptance/final-browser.py')
measure=importlib.util.module_from_spec(spec);spec.loader.exec_module(measure)
checks=[];errors=[];contrasts=[];processes=[];streams=[]

def check(name,condition):
    assert condition,name
    checks.append(name);print('PASS '+name,flush=True)

def port():
    with socket.socket() as s:s.bind(('127.0.0.1',0));return s.getsockname()[1]

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
            launch([node,'serve','--data-dir',str(child)],'node-'+str(i))
            nodes.append({'id':card['id'],'name':name,'data':child})

        deadline=time.monotonic()+60
        while True:
            fleet=api('/api/fleet')['nodes']
            if all(n['online'] and n['status'].get('details',{}).get('system',{}).get('cpuCores',0)>0 and n['status'].get('revision')==n['desiredRevision'] for n in fleet):break
            assert time.monotonic()<deadline,('agents did not report telemetry',fleet)
            time.sleep(.3)
        check('Two actual agents enroll, apply policies and report machine telemetry',True)
        page.reload(wait_until='networkidle');expect(page.locator('.fleet-node-card')).to_have_count(2)
        page.locator(f'[data-node-card="{nodes[0]["id"]}"] .fleet-node-link').click()
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
    for process in reversed(processes):
        if process.poll() is None:os.killpg(process.pid,signal.SIGTERM)
    for process in reversed(processes):
        try:process.wait(timeout=10)
        except subprocess.TimeoutExpired:os.killpg(process.pid,signal.SIGKILL);process.wait()
    for stream in streams:stream.close()
    (output/'results.json').write_text(json.dumps({'checks':checks,'errors':errors},ensure_ascii=False,indent=2))
print(f'Passed {len(checks)} server workspace checks',flush=True)
