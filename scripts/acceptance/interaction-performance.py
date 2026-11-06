"""Measure polling and navigation with a large real control-side node model."""
import argparse, json, socket, statistics, subprocess, tempfile, time
from pathlib import Path
from playwright.sync_api import sync_playwright, expect

assert Path.cwd() == Path('/workspace') and Path('/.dockerenv').exists()
parser=argparse.ArgumentParser()
parser.add_argument('--control',default='dist/interaction/control')
parser.add_argument('--output',default='dist/interaction/results.json')
parser.add_argument('--record-baseline',action='store_true')
args=parser.parse_args()
control=str(Path(args.control).resolve())
output=Path(args.output);output.parent.mkdir(parents=True,exist_ok=True)
checks=[]
def check(name,value):
    if not args.record_baseline:assert value,name
    checks.append({'name':name,'passed':bool(value)})
    print(('PASS ' if value else 'BASELINE ')+name,flush=True)

with tempfile.TemporaryDirectory(prefix='interaction-',dir='/workspace/dist') as directory,sync_playwright() as pw:
    directory=Path(directory)
    with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
    base=f'http://127.0.0.1:{port}/console/'
    subprocess.run([control,'configure','--data-dir',str(directory/'data'),'--username','fixture','--password-stdin',
                    '--listen-ip','127.0.0.1','--port',str(port),'--path','console'],input='fixture-password',text=True,check=True,stdout=subprocess.DEVNULL)
    with (directory/'control.log').open('w') as log:
        process=subprocess.Popen([control,'serve','--data-dir',str(directory/'data')],stdout=log,stderr=log)
        browser=pw.chromium.launch(headless=True,args=['--no-sandbox'])
        try:
            ctx=browser.new_context(viewport={'width':1440,'height':1000})
            for _ in range(100):
                try:
                    if ctx.request.get(base+'login',timeout=1000).ok:break
                except Exception:pass
                time.sleep(.1)
            def api(path,body=None):
                response=ctx.request.fetch(base+path,method='POST' if body is not None else 'GET',data=body)
                assert response.ok,(path,response.status,response.text())
                return response.json()['data']
            api('api/auth/login',{'username':'fixture','password':'fixture-password'})
            card=api('api/fleet/nodes',{'name':'Large model','address':''})
            model=api('api/fleet/nodes/'+card['id'])['draft']
            model['settings']['pageSize']=0
            model['inbounds']=[{'id':f'in-{i}','name':f'Inbound {i}','type':'socks','listen':'127.0.0.1','port':20000+i,'enabled':True,
                               'clients':[{'id':f'user-{i}-{j}','name':f'User {i}-{j}','username':f'u-{i}-{j}','password':'fixture','enabled':True} for j in range(4)]} for i in range(160)]
            model['outbounds']=[{'id':f'out-{i}','kind':'proxy','type':'socks5','name':f'Outbound {i}','server':'127.0.0.1','port':1080} for i in range(200)]
            model['routingRules']=[{'id':f'rule-{i}','type':'DOMAIN-SUFFIX','payload':f'site-{i}.example','target':'DIRECT','noResolve':False} for i in range(320)]+[{'id':'final','type':'MATCH','target':'DIRECT'}]
            api('api/fleet/nodes/'+card['id']+'/model',model)
            page=ctx.new_page();errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
            page.goto(base+'nodes/'+card['id']+'/')
            expect(page.locator('.server-workspace-identity strong')).to_have_text('Large model')
            page.evaluate('refresh()')
            metrics=page.evaluate("""async () => {
              while(app.refreshing) await new Promise(r=>setTimeout(r,10));
              const areas={inbounds:'#inbounds-table',outbounds:'#mihomo-outbound-table',routing:'#mihomo-rule-table',settings:'#settings-form'};
              const mutations={},mutationTargets={};const observers=[];
              for(const [key,selector] of Object.entries(areas)) {
                mutations[key]=0;
                mutationTargets[key]={};
                const observer=new MutationObserver(records=>{mutations[key]+=records.length;for(const record of records){const target=record.target,owner=target.nodeType===1?target.closest('[name], [id]'):target.parentElement?.closest('[name], [id]');const label=(owner?.id||owner?.name||target.parentElement?.id||target.nodeName)+':'+record.type+(record.attributeName?':'+record.attributeName:'');mutationTargets[key][label]=(mutationTargets[key][label]||0)+1;}});
                observer.observe(document.querySelector(selector),{childList:true,subtree:true,attributes:true,characterData:true});
                observers.push(observer);
              }
              const polling=[];
              for(let i=0;i<6;i++){const start=performance.now();await refresh();polling.push(performance.now()-start);await new Promise(requestAnimationFrame);}
              observers.forEach(observer=>observer.disconnect());
              const navigation=[];
              for(let i=0;i<6;i++)for(const view of ['inbounds','mihomo','dashboard']) {
                const start=performance.now();document.querySelector('#nav [data-view='+view+']').click();
                await new Promise(requestAnimationFrame);navigation.push(performance.now()-start);
              }
              return {mutations,mutationTargets,polling,navigation};
            }""")
            check('Dashboard polling leaves hidden inbound, outbound and routing tables untouched',all(metrics['mutations'][key]==0 for key in ('inbounds','outbounds','routing')))
            check('Unchanged settings do not mutate their controls during polling',metrics['mutations']['settings']==0)
            page.locator('#nav [data-view=inbounds]').click()
            expect(page.locator('#inbounds-table tbody > tr')).to_have_count(160)
            stable=page.evaluate("""async()=>{const row=document.querySelector('#inbounds-table tbody > tr');await refresh();return row===document.querySelector('#inbounds-table tbody > tr')}""")
            check('Unchanged inbound rows survive refresh with their controls intact',stable)
            page.locator('.inbound-expand[data-expand-inbound="in-0"]').click()
            expect(page.locator('.client-detail-row')).to_be_visible()
            check('Inbound expansion still shows its clients',page.locator('[data-edit-client="in-0:user-0-0"]').is_visible())
            page.locator('#nav [data-view=mihomo]').click()
            page.locator('[data-mihomo-tab=outbounds]').click()
            expect(page.locator('[data-outbound-edit]')).to_have_count(200)
            stable=page.evaluate("""async()=>{const row=document.querySelector('#mihomo-outbound-table tbody > tr');await refresh();return row===document.querySelector('#mihomo-outbound-table tbody > tr')}""")
            check('Unchanged outbound controls survive telemetry refresh',stable)
            page.locator('[data-mihomo-tab=routing]').click()
            expect(page.locator('[data-rule-edit]')).to_have_count(321)
            stable=page.evaluate("""async()=>{const row=document.querySelector('#mihomo-rule-table tbody > tr');await refresh();return row===document.querySelector('#mihomo-rule-table tbody > tr')}""")
            check('Unchanged routing controls survive telemetry refresh',stable)
            page.locator('#nav [data-view=dashboard]').click()
            model['inbounds'][0]['notes']='Changed remotely'
            model['outbounds'][0]['name']='Changed outbound'
            api('api/fleet/nodes/'+card['id']+'/model',model)
            page.evaluate('refresh()')
            page.locator('#nav [data-view=inbounds]').click()
            expect(page.locator('#inbounds-table')).to_contain_text('Changed remotely')
            page.locator('#nav [data-view=mihomo]').click();page.locator('[data-mihomo-tab=outbounds]').click()
            expect(page.locator('#mihomo-outbound-table')).to_contain_text('Changed outbound')
            check('Returning to a view immediately shows changes received while it was hidden',True)
            page.locator('#mihomo-add-outbound').click()
            expect(page.locator('#mihomo-outbound-modal')).to_be_visible()
            check('Outbound editor remains available after cached refreshes',True)
            check('No browser exceptions',not errors)
            metrics['pollingMedianMs']=statistics.median(metrics['polling'])
            metrics['navigationMedianMs']=statistics.median(metrics['navigation'])
            output.write_text(json.dumps({'checks':checks,'metrics':metrics,'errors':errors},ensure_ascii=False,indent=2))
            print(json.dumps({'pollingMedianMs':metrics['pollingMedianMs'],'navigationMedianMs':metrics['navigationMedianMs'],'mutations':metrics['mutations']}),flush=True)
        finally:
            browser.close();process.terminate();process.wait(timeout=15)
