"""P5 real handlers, screenshots and rendered text measurements in orca-linux."""
from pathlib import Path
import argparse,json,os,socket,subprocess,tempfile,time,urllib.request,signal
from playwright.sync_api import sync_playwright
from PIL import Image
assert Path.cwd()==Path('/workspace') and Path('/.dockerenv').exists()
OUT=Path('dist/design-audit/p5');OUT.mkdir(parents=True,exist_ok=True)
checks=[];shots=[];contrast=[];errors=[];operations=[];discipline=[]
def port():
 with socket.socket() as s:s.bind(('127.0.0.1',0));return s.getsockname()[1]
MEASURE=r'''() => {
 const visible=e=>{const r=e.getBoundingClientRect(),s=getComputedStyle(e);if(!(r.width&&r.height&&s.visibility==='visible'&&s.display!=='none'))return false;for(let p=e;p;p=p.parentElement)if(+getComputedStyle(p).opacity===0)return false;const modal=document.querySelector('.modal-backdrop.open,.drawer-backdrop.open');return !modal||modal.contains(e)};
 const rgb=s=>{let m=s.match(/[\d.]+/g);return m?m.map(Number):[0,0,0,0]};
 const lum=c=>c.slice(0,3).map(v=>v/255).map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4).reduce((a,v,i)=>a+v*[.2126,.7152,.0722][i],0);
 const ratio=(a,b)=>{let x=lum(a),y=lum(b);return (Math.max(x,y)+.05)/(Math.min(x,y)+.05)};
 const rows=[];
 for(const e of document.querySelectorAll('body *')) {
  if(!visible(e)||e.closest('svg,canvas,option,script,style')||e.matches('input[type=checkbox],input[type=radio]'))continue;
  const s=getComputedStyle(e);if(s.opacity==='0'||e.closest('[disabled],[aria-disabled=true]'))continue;
  const direct=[...e.childNodes].filter(n=>n.nodeType===3).map(n=>n.textContent.trim()).join(' ');if(!direct&&!e.matches('input,textarea'))continue;
  if(e.matches('input,textarea')&&!e.value)continue;
  let bg=[255,255,255,1],chain=[];for(let p=e;p;p=p.parentElement)chain.push(p);
  for(const p of chain.reverse()){let c=rgb(getComputedStyle(p).backgroundColor),a=c.length>3?c[3]:1;if(a)bg=c.slice(0,3).map((v,i)=>v*a+bg[i]*(1-a)).concat([1]);}
  const r=ratio(rgb(s.color),bg),large=parseFloat(s.fontSize)>=24||(parseFloat(s.fontSize)>=18.66&&Number(s.fontWeight)>=700);
  rows.push({selector:e.id?'#'+e.id:e.tagName.toLowerCase()+'.'+[...e.classList].slice(0,2).join('.'),text:(direct||e.value).slice(0,90),ratio:+r.toFixed(3),required:large?3:4.5,foreground:s.color,background:bg.slice(0,3),font:s.fontSize});
 }
 return rows;
}'''
def check(name,condition,detail=None):
 if not condition:raise AssertionError((name,detail))
 checks.append({'name':name,'passed':True,'detail':detail})
def api(ctx,url,path,method='GET',data=None):
 r=ctx.request.fetch(url+path,method=method,data=data)
 if not r.ok:raise AssertionError((method,path,r.status,r.text()[:500]))
 try:return r.json().get('data',r.json())
 except:return r.body()
def close_all(page):
 page.evaluate("document.querySelectorAll('.modal-backdrop.open,.drawer-backdrop.open,#drawer-mask.open,#drawer.open').forEach(e=>e.classList.remove('open'));document.querySelectorAll('.general-menu.open,.settings-menu.open').forEach(e=>e.classList.remove('open'))")
def shot(page,folder,name):
 page.evaluate("""async () => { await document.fonts.ready; for(let i=0;i<4;i++){await new Promise(requestAnimationFrame);const animations=document.getAnimations().filter(a=>a.playState==='running'&&a.effect.getTiming().iterations!==Infinity);if(!animations.length)break;await Promise.all(animations.map(a=>a.finished.catch(()=>{})));}}""");page.wait_for_timeout(500);page.screenshot(path=str(folder/(name+'.png')),full_page=True,animations='disabled')
 rows=page.evaluate(MEASURE);contrast.append({'surface':name,'theme':folder.name,'measurements':rows,'failures':[r for r in rows if r['ratio']<r['required']]});shots.append(str(folder/(name+'.png')))
 tokens=page.evaluate("() => {const s=getComputedStyle(document.documentElement);return Object.fromEntries(['primary','highlight','primary-soft','chart-line','semantic-success','latency-good','row-height','rounded-md','duration-fast','font-mono'].map(k=>[k,s.getPropertyValue('--'+k).trim()]));}")
 def pixel_share(value):
  value=value.lstrip('#');rgb=tuple(int(value[i:i+2],16) for i in (0,2,4));im=Image.open(folder/(name+'.png')).convert('RGB');count=sum(n for n,c in im.getcolors(im.width*im.height) if max(abs(c[i]-rgb[i]) for i in range(3))<=5);return round(count/(im.width*im.height),6)
 # The supplied T2 fluorescent color is #c2ef4e in both modes. Light-mode
 # --highlight is the violet CTA color, so it must not be counted as fluorescence.
 discipline.append({'theme':folder.name,'surface':name,'tokens':tokens,'accent_pixel_share':pixel_share(tokens['primary']),'highlight_pixel_share':pixel_share(tokens['highlight']),'fluorescent_pixel_share':pixel_share('#c2ef4e') if folder.name.startswith('t2') else 0})
def main():
 global OUT
 parser=argparse.ArgumentParser();parser.add_argument('--quick',action='store_true');parser.add_argument('--fleet',action='store_true');parser.add_argument('--operations-only',action='store_true');parser.add_argument('--login-only',action='store_true');parser.add_argument('--output',default=str(OUT));args=parser.parse_args()
 OUT=Path(args.output);OUT.mkdir(parents=True,exist_ok=True)
 if args.fleet:
  OUT=OUT/'fleet';OUT.mkdir(parents=True,exist_ok=True)
 if args.operations_only:
  OUT=OUT/'operations';OUT.mkdir(parents=True,exist_ok=True)
 env=os.environ.copy();env.update(GOROOT='/root/go',GOPATH='/root/unifiedproxy-gopath');env['PATH']='/root/go/bin:'+env['PATH']
 with tempfile.TemporaryDirectory(prefix='up-p5-') as data,(OUT/'server.log').open('w') as log:
  p=port();url=f'http://127.0.0.1:{p}';env.update(UNIFIEDPROXY_DESIGN_ADDRESS=f'127.0.0.1:{p}',UNIFIEDPROXY_DESIGN_DATA=data,UNIFIEDPROXY_DESIGN_CORE='/workspace/dist/design-audit/fixture-core',UNIFIEDPROXY_DESIGN_CORE_API=f'127.0.0.1:{port()}');env['UNIFIEDPROXY_DESIGN_ROLE']='control' if args.fleet else ''
  subprocess.run(['dist/design-audit/control','configure','--data-dir',data,'--username','fixture','--password-stdin'],input='fixture-password',text=True,check=True,stdout=subprocess.DEVNULL)
  server=subprocess.Popen(['dist/design-audit/app-tests','-test.run','^TestDesignAcceptanceServer$','-test.v'],env=env,stdout=log,stderr=log,start_new_session=True)
  try:
   for _ in range(150):
    try:urllib.request.urlopen(url+'/login',timeout=1).close();break
    except:time.sleep(.1)
    if server.poll() is not None:raise RuntimeError('fixture exited: '+str(server.returncode))
   with sync_playwright() as pw:
    browser=pw.chromium.launch(headless=True,args=['--no-sandbox']);ctx=browser.new_context(viewport={'width':1440,'height':1000},permissions=['clipboard-read','clipboard-write']);page=ctx.new_page();page.on('pageerror',lambda e:errors.append(str(e)))
    page.goto(url+'/login',wait_until='networkidle');page.locator('#username').fill('fixture');page.locator('#password').fill('wrong-password');page.locator('#login-button').click();page.wait_for_timeout(400);check('F01 invalid credentials rejected',page.locator('.error').inner_text().strip()!='')
    page.locator('#password').fill('fixture-password');page.locator('#login-button').click();page.wait_for_url(url+'/');page.wait_for_load_state('networkidle');check('F01 real login',page.locator('#nav').is_visible())
    for locale in ['zh-CN','zh-TW','en','ja','ru','fa','vi','es','tr','uk','pt-BR']:
     page.evaluate('(v)=>upSetLang(v)',locale);check('F01/F02 locale '+locale,page.get_attribute('html','lang')==locale)
    page.evaluate("upSetLang('zh-CN')");page.evaluate("UPTheme.setColor('t4');UPTheme.setMode('dark')");page.reload(wait_until='networkidle');check('F02 palette persists after reload',page.get_attribute('html','data-color-theme')=='t4' and page.get_attribute('html','data-theme')=='dark')
    page.evaluate("UPTheme.setColor('t0');UPTheme.setMode('light')")
    seed={'id':'p5-mieru','name':'Acceptance Mieru','type':'mieru','listen':'127.0.0.1','port':31234,'enabled':True,'clients':[{'id':'alice','name':'Alice','username':'alice','password':'fixture-password','enabled':True,'total':1073741824}],'mieru':{'transport':'TCP'}}
    if args.fleet:
     fleet_operations(page,ctx,url,seed)
    else:
     api(ctx,url,'/api/inbounds','POST',seed);page.evaluate('refresh()');page.wait_for_timeout(300);browser_operations(page,ctx,url)
    if args.operations_only:extended_operations(page,ctx,url)
    combinations=[] if args.operations_only else ([('t0','light')] if args.quick else [('t'+str(i),m) for i in range(5) for m in ('light','dark')])
    tokens={}
    if not args.fleet:
     token=api(ctx,url,'/api/subscriptions/token','POST',{'username':'alice'});state=api(ctx,url,'/api/state')['state'];tokens={'path':state['settings']['subscriptionPath'].rstrip('/')+'/'+token['token']}
    for palette,mode in combinations:
     folder=OUT/(palette+'-'+mode);folder.mkdir(exist_ok=True);page.evaluate('([p,m])=>{UPTheme.setColor(p);UPTheme.setMode(m)}',[palette,mode]);close_all(page)
     if args.login_only:
      login_context=browser.new_context(viewport={'width':1440,'height':1000});login=login_context.new_page();login.on('pageerror',lambda e:errors.append(str(e)));login.goto(url+'/login',wait_until='networkidle');check('F01 unauthenticated login form '+palette+'-'+mode,login.locator('#login-button').is_visible());login.evaluate('([p,m])=>{UPTheme.setColor(p);UPTheme.setMode(m)}',[palette,mode]);shot(login,folder,'login');login.set_viewport_size({'width':390,'height':844});shot(login,folder,'mobile-login');login_context.close();continue
     if args.fleet:
      page.locator('[data-view=fleet]').click();page.locator('.fleet-grid tbody tr').first.wait_for();page.evaluate("document.querySelector('.app').scrollTop=0");shot(page,folder,'fleet');page.set_viewport_size({'width':390,'height':844});shot(page,folder,'mobile-fleet');check('F28 fleet mobile fits viewport',page.evaluate('document.documentElement.scrollWidth<=innerWidth+1'));page.set_viewport_size({'width':1440,'height':1000})
      for i,name in [(2,'fleet-model'),(3,'fleet-certificates'),(4,'fleet-update'),(5,'fleet-backup')]:
       block=page.locator('#view-fleet>.fleet-block').nth(i);block.evaluate("e=>{if(e.tagName==='DETAILS')e.open=true;e.querySelectorAll('details').forEach(d=>d.open=true)}");block.scroll_into_view_if_needed();shot(page,folder,name)
      page.locator('#fleet-tasks').scroll_into_view_if_needed();shot(page,folder,'fleet-tasks');continue
     for view in ['dashboard','inbounds','subtemplates','mihomo','settings','connections','logs']:
      page.evaluate('(v)=>setView(v)',view);page.wait_for_timeout(100);shot(page,folder,view)
      if view=='settings':
       for tab in page.locator('[data-settings-tab]').all():tab.click();shot(page,folder,'settings-'+tab.get_attribute('data-settings-tab'))
      if view=='mihomo':
       for tab in page.locator('[data-mihomo-tab]').all():tab.click();shot(page,folder,'mihomo-'+tab.get_attribute('data-mihomo-tab'))
     screenshots_dialogs(page,folder)
     close_all(page);page.locator('[data-view=fleet]').click();page.locator('#view-fleet .fleet-block').first.wait_for();page.evaluate("document.querySelector('.app').scrollTop=0");shot(page,folder,'standalone-fleet');page.locator('#view-fleet details').evaluate_all("items=>items.forEach(d=>d.open=true)");shot(page,folder,'standalone-sidecars')
     page.set_viewport_size({'width':390,'height':844});close_all(page);page.evaluate("setView('dashboard')");shot(page,folder,'mobile-dashboard');page.evaluate("setView('inbounds')");shot(page,folder,'mobile-inbounds');page.set_viewport_size({'width':1440,'height':1000})
     login_context=browser.new_context(viewport={'width':1440,'height':1000});login=login_context.new_page();login.on('pageerror',lambda e:errors.append(str(e)));login.goto(url+'/login',wait_until='networkidle');check('F01 unauthenticated login form '+palette+'-'+mode,login.locator('#login-button').is_visible());login.evaluate('([p,m])=>{UPTheme.setColor(p);UPTheme.setMode(m)}',[palette,mode]);shot(login,folder,'login');login.set_viewport_size({'width':390,'height':844});shot(login,folder,'mobile-login');login_context.close()
     subscription_screens(ctx,url,folder,palette,mode,tokens)
    page.emulate_media(reduced_motion='reduce');check('F36 reduced motion zero',page.evaluate("getComputedStyle(document.querySelector('#refresh')).transitionDuration.split(',').every(s=>parseFloat(s)===0)"));check('No uncaught browser errors',not errors,errors);browser.close()
  finally:os.killpg(server.pid,signal.SIGTERM);server.wait(timeout=10)
 failures=[{'theme':r['theme'],'surface':r['surface'],**f} for r in contrast for f in r['failures']];(OUT/'contrast-failures.json').write_text(json.dumps(failures,ensure_ascii=False,indent=2));check('All measured text meets contrast thresholds',not failures,failures[:5]);check('T2 fluorescent area below two percent',all(d['fluorescent_pixel_share']<.02 for d in discipline if d['theme'].startswith('t2')))
 result={'checks':checks,'operations':operations,'screenshots':shots,'contrasts':contrast,'discipline':discipline,'errors':errors}
 if args.login_only:
  previous=json.loads((OUT/'browser-results.json').read_text());surfaces={'login','mobile-login'}
  for key in ('contrasts','discipline'):previous[key]=[r for r in previous[key] if r['surface'] not in surfaces]+result[key]
  previous['screenshots']=sorted(set(previous['screenshots']+shots));previous['checks']=[r for r in previous['checks'] if not r['name'].startswith('F01 unauthenticated login form')]+[r for r in checks if r['name'].startswith('F01 unauthenticated login form')];previous['errors']+=errors;result=previous
 (OUT/'browser-results.json').write_text(json.dumps(result,ensure_ascii=False,indent=2));print(f'{len(checks)} checks, {len(shots)} screenshots, {len(failures)} measured contrast failures')
def browser_operations(page,ctx,url):
 operations.append('real login and invalid login');page.evaluate("setView('inbounds')");page.locator('#add-inbound').click();page.locator('#inbound-form').wait_for(state='visible');check('F05 inbound drawer opens',page.locator('#inbound-form').is_visible());page.locator('#cancel-drawer').click()
 check('F05 seeded inbound saved',len(api(ctx,url,'/api/state')['state']['inbounds'])==1);page.evaluate("openClientModal('p5-mieru')");check('F07 client modal',page.locator('#client-form').is_visible());close_all(page)
 client={'id':'bob','name':'Bob','username':'bob','password':'bob-password','enabled':True,'total':10000};api(ctx,url,'/api/clients','POST',{'inboundId':'p5-mieru','client':client});check('F07 client create persisted',len(api(ctx,url,'/api/state')['state']['inbounds'][0]['clients'])==2)
 client['name']='Bob edited';api(ctx,url,'/api/clients','POST',{'inboundId':'p5-mieru','client':client});api(ctx,url,'/api/traffic/reset','POST',{'inboundId':'p5-mieru','clientId':'bob'});api(ctx,url,'/api/clients?inboundId=p5-mieru&clientId=bob','DELETE');operations.append('F05/F07 create/edit/delete/reset persisted through authenticated APIs');page.evaluate('refresh()');page.wait_for_timeout(300)
 page.evaluate("setView('settings')");page.locator('[data-settings-tab=multi-control]').click();page.locator('#mc-token-open').click();page.locator('#mc-token-generate').click();page.wait_for_timeout(250);check('F25 token generated',len(page.locator('#mc-token').input_value())>=16);close_all(page)
 template={'name':'Acceptance template','content':'proxy-groups:\n  - name: Route\n    type: select\n    proxies: [DIRECT]\nrules:\n  - MATCH,Route\n'}
 preview=api(ctx,url,'/api/sub-templates/import','POST',{**template,'preview':True});check('F13 template preview',bool(preview.get('content')));api(ctx,url,'/api/sub-templates/import','POST',template);operations.append('F13 template preview/import via real endpoint')
 basics=api(ctx,url,'/api/mihomo/settings');check('F14 settings endpoint',isinstance(basics,dict))
def fleet_operations(page,ctx,url,seed):
 node=api(ctx,url,'/api/fleet/nodes','POST',{'name':'Acceptance node','address':'127.0.0.1'});node_id=node['id'];api(ctx,url,'/api/fleet/nodes/'+node_id,'PUT',{'name':'Acceptance edited','address':'127.0.0.1'})
 token=api(ctx,url,'/api/fleet/tokens','POST',{'nodeId':node_id,'name':'Acceptance enrollment'});check('F28 enrollment issued',bool(token['token']));api(ctx,url,'/api/fleet/tokens/revoke','POST',{'token':token['token']})
 detail=api(ctx,url,'/api/fleet/nodes/'+node_id);model=detail['draft'];model['inbounds']=[seed];desired=api(ctx,url,'/api/fleet/nodes/'+node_id+'/model','POST',model);check('F29 native model published',desired['revision']>0)
 detail=api(ctx,url,'/api/fleet/nodes/'+node_id);check('F29 original client credential persisted',detail['draft']['inbounds'][0]['clients'][0]['password']=='fixture-password');api(ctx,url,'/api/fleet/nodes/'+node_id+'/sidecars','POST',[]);backup=api(ctx,url,'/api/fleet/backup');check('F32 fleet backup complete',bool(backup.get('fleet')))
 batch=api(ctx,url,'/api/fleet/batch','POST',{'nodes':[node_id],'action':'quota','username':'alice','total':2147483648});check('F30 quota batch published',bool(batch));check('F30 quota batch preserved credential',api(ctx,url,'/api/fleet/nodes/'+node_id)['draft']['inbounds'][0]['clients'][0]['password']=='fixture-password');check('F30 quota batch updated selected client',api(ctx,url,'/api/fleet/nodes/'+node_id)['draft']['inbounds'][0]['clients'][0]['total']==2147483648)
 unsupported=ctx.request.post(url+'/api/fleet/batch',data={'nodes':[node_id],'action':'task','kind':'collect-status','payload':{}});check('F30 unnegotiated task rejected by capability gate',unsupported.status==400 and 'does not support' in unsupported.text());check('F30 rollout list endpoint',isinstance(api(ctx,url,'/api/fleet/batch'),list))
 retired=api(ctx,url,'/api/fleet/nodes','POST',{'name':'Revocation fixture','address':'127.0.0.1'});api(ctx,url,'/api/fleet/nodes/'+retired['id']+'/revoke','POST',{});check('F28 node revoke persisted',any(n['id']==retired['id'] and n['revoked'] for n in api(ctx,url,'/api/fleet')['nodes']))
 operations.append('F28/F29/F32 authenticated node create/edit/enrollment/revoke/model/sidecars/backup')

def extended_operations(page,ctx,url):
 page.locator('#sider-trigger').click();check('F02 sidebar collapses',page.locator('body').evaluate("e=>e.classList.contains('sider-collapsed')"));page.locator('#sider-trigger').click();check('F02 sidebar expands',not page.locator('body').evaluate("e=>e.classList.contains('sider-collapsed')"))
 page.evaluate("setView('inbounds');openDrawer('p5-mieru')");page.locator('#inbound-form').wait_for(state='visible');page.locator('#inbound-form [name=name]').fill('Edited through browser');page.locator('#inbound-form [name=notes]').fill('Browser regression record')
 with page.expect_response(lambda r:'/api/inbounds' in r.url and r.request.method=='POST') as response:page.locator('#save-inbound').click()
 check('F05 native inbound form submits',response.value.ok);page.locator('#drawer-backdrop.open').wait_for(state='hidden');check('F05 edit persisted',api(ctx,url,'/api/state')['state']['inbounds'][0]['notes']=='Browser regression record')
 page.evaluate("openClientModal('p5-mieru')");page.locator('#client-form [name=name]').fill('UI client');page.locator('#client-form [name=username]').fill('ui-client');page.locator('#client-form [name=password]').fill('ui-original-password')
 with page.expect_response(lambda r:'/api/clients' in r.url and r.request.method=='POST') as response:page.locator('#save-client').click()
 check('F07 native client form submits',response.value.ok);page.locator('#client-modal.open').wait_for(state='hidden');clients=api(ctx,url,'/api/state')['state']['inbounds'][0]['clients'];created=next(c for c in clients if c['username']=='ui-client');check('F07 original submitted credential persisted',created['password']=='ui-original-password');api(ctx,url,'/api/clients?inboundId=p5-mieru&clientId='+created['id'],'DELETE')
 page.evaluate("openBulkClientModal('p5-mieru')");page.locator('#client-bulk-form [name=prefix]').fill('acceptance-bulk-');page.locator('#client-bulk-form [name=last]').fill('2');page.locator('#client-bulk-form button[type=submit]').click();page.locator('#client-bulk-modal.open').wait_for(state='hidden');bulk=[c for c in api(ctx,url,'/api/state')['state']['inbounds'][0]['clients'] if c['username'].startswith('acceptance-bulk-')];check('F07 bulk client form persists generated credentials',len(bulk)==2 and all(c['password'] for c in bulk))
 for client in bulk:api(ctx,url,'/api/clients?inboundId=p5-mieru&clientId='+client['id'],'DELETE')
 page.evaluate("cloneInbound('p5-mieru')");items=api(ctx,url,'/api/state')['state']['inbounds'];cloned=next(i for i in items if i['id']!='p5-mieru');check('F05 clone preserves credentials and disables clone',not cloned['enabled'] and cloned['clients'][0]['password']=='fixture-password');api(ctx,url,'/api/inbounds?id='+cloned['id'],'DELETE')
 page.evaluate('refresh()');page.locator('#inbound-search').fill('no-such-inbound');check('F06 search hides nonmatches',page.locator('[data-toggle-inbound]').count()==0);page.locator('#inbound-search').fill('Edited through browser');check('F06 search restores matching inbound',page.locator('[data-toggle-inbound]').count()==1);page.locator('#inbound-search').fill('')
 for enabled in [False,True]:
  page.evaluate('(enabled)=>toggleInbound("p5-mieru",enabled)',enabled);check('F05 inbound toggle persisted '+str(enabled),api(ctx,url,'/api/state')['state']['inbounds'][0]['enabled']==enabled)
 for enabled in [False,True]:
  page.evaluate('(enabled)=>toggleClient("p5-mieru:alice",enabled)',enabled);check('F07 client toggle persisted '+str(enabled),api(ctx,url,'/api/state')['state']['inbounds'][0]['clients'][0]['enabled']==enabled)
 page.locator('#inbounds-search-toggle').click();page.locator('[data-status=disabled]').click();check('F06 disabled filter excludes enabled inbound',page.locator('[data-toggle-inbound]').count()==0);page.locator('[data-status=all]').click();check('F06 all filter restores inbound',page.locator('[data-toggle-inbound]').count()==1);page.locator('#inbounds-search-toggle').click()
 page.locator('#inbounds-auto-toggle').click();page.locator('label.switch:has(#inbounds-auto-enabled)').click();check('F06 periodic refresh starts',page.evaluate('app.inboundAutoTimer!==null'));page.locator('label.switch:has(#inbounds-auto-enabled)').click();check('F06 periodic refresh stops',page.evaluate('app.inboundAutoTimer===null'));page.locator('#inbounds-auto-toggle').click()
 page.evaluate("app.state.state.settings.pageSize=1;app.state.state.inbounds.push({...app.state.state.inbounds[0],id:'pagination-only',name:'pagination-only'});renderInboundTable()");page.locator('[data-page="2"]').last.click();check('F06 pagination selects second inbound',page.locator('[data-toggle-inbound=pagination-only]').count()==1);page.evaluate('refresh()')
 for action in ['reset-inbounds','reset-clients']:
  page.locator('#general-actions').click();page.locator('[data-general='+action+']').click();page.locator('.up-dialog.open').wait_for()
  with page.expect_response(lambda r:'/api/traffic/reset' in r.url and r.request.method=='POST') as response:page.locator('.up-dialog .danger-btn').click()
  check('F09 confirmed bulk '+action,response.value.ok);page.locator('.up-dialog.open').wait_for(state='hidden');page.wait_for_timeout(150)
 page.evaluate("setView('connections')");page.locator('#close-all-connections').click();page.locator('.up-dialog.open').wait_for()
 with page.expect_response(lambda r:'/api/core/connections' in r.url and r.request.method=='DELETE') as response:page.locator('.up-dialog .danger-btn').click()
 check('F19 confirmed close all uses DELETE handler',response.value.ok);page.locator('.up-dialog.open').wait_for(state='hidden')
 for action in ['stop','start','restart','test']:api(ctx,url,'/api/core/'+action,'POST',{})
 check('F04 core controls and config test use real handlers',True)
 for path in ['/api/state','/api/core/logs','/api/server/cpuHistory/2','/api/cross-subscriptions','/api/inbound-sync']:
  check('F03/F10/F11/F19/F20 '+path,api(ctx,url,path) is not None)
 settings=api(ctx,url,'/api/mihomo/settings');api(ctx,url,'/api/mihomo/settings','PUT',settings);check('F14/F15/F16 native settings round trip',api(ctx,url,'/api/mihomo/settings')['routingRules']==settings['routingRules'])
 snapshot=api(ctx,url,'/api/state')['state'];snapshot['settings']['remarkModel']='_ei';api(ctx,url,'/api/config','PUT',snapshot['settings']);check('F21/F24 settings save persisted',api(ctx,url,'/api/state')['state']['settings']['remarkModel']=='_ei')
 backup=api(ctx,url,'/api/backup');response=ctx.request.post(url+'/api/backup',multipart={'backup':{'name':'acceptance.zip','mimeType':'application/zip','buffer':backup}});check('F22 ZIP download and real multipart restore',response.ok,response.text()[:200]);check('F22 restore preserves admin identity',api(ctx,url,'/api/state')['state']['settings']['username']=='fixture')
 operations.append('Browser form edit/client create, clone/delete, core lifecycle, snapshots, settings persistence and ZIP restore')
 for width in [1024,768,480]:
  page.set_viewport_size({'width':width,'height':900});close_all(page);page.evaluate("setView('dashboard');setDrawerOpen(false)");shot(page,OUT,'dashboard-'+str(width));check('F02 viewport '+str(width),page.evaluate('document.documentElement.scrollWidth<=innerWidth+1'))
 page.set_viewport_size({'width':1440,'height':1000})

def screenshots_dialogs(page,folder):
 close_all(page);page.evaluate("setView('dashboard')")
 for trigger,target in [('#overview-backup','#backup-modal'),('#overview-config','#config-modal'),('#overview-logs','#logs-modal'),('#overview-import','#xray-import-modal'),('#cpu-history-open','#cpu-history-modal')]:
  close_all(page);page.locator(trigger).click();page.locator(target+'.open').wait_for();shot(page,folder,target[1:])
 close_all(page);page.evaluate("setView('inbounds');openDrawer('p5-mieru')");page.locator('#inbound-form').wait_for(state='visible');shot(page,folder,'inbound-drawer');close_all(page);page.evaluate("openClientModal('p5-mieru')");shot(page,folder,'client-modal');close_all(page);page.evaluate("setView('mihomo')")
 for trigger,target in [('#mihomo-add-outbound','#mihomo-outbound-modal'),('#mihomo-add-rule','#mihomo-rule-modal'),('#mihomo-warp','#warp-modal'),('#mihomo-vpngate','#vpngate-modal')]:
  close_all(page);page.locator('[data-mihomo-tab='+('routing' if trigger=='#mihomo-add-rule' else 'outbounds')+']').click();page.locator(trigger).click();page.locator(target+'.open').wait_for();shot(page,folder,target[1:])
 close_all(page);page.evaluate("setView('subtemplates')");page.locator('#subtpl-import').click();shot(page,folder,'subtpl-import-modal');close_all(page);page.evaluate("setView('settings')");page.locator('[data-settings-tab=multi-control]').click()
 for trigger,target in [('#mc-token-open','#mc-token-modal'),('#mc-connections-open','#mc-connections-modal'),('#mc-devices-open','#mc-devices-modal')]:
  close_all(page);page.locator(trigger).click();page.locator(target+'.open').wait_for();shot(page,folder,target[1:])
 close_all(page);page.evaluate("void(window.acceptanceDialog=upConfirm('Acceptance confirmation / 确认操作'))");shot(page,folder,'confirmation');page.keyboard.press('Escape');close_all(page)
 page.evaluate("setView('inbounds')")
 for key,target in [('sync-inbound','sync-inbound-modal'),('cross-subscriptions','cross-subscriptions-modal')]:
  close_all(page);page.locator('#general-actions').click();page.locator('[data-general='+key+']').click();page.locator('#'+target+'.open').wait_for();shot(page,folder,target)
 for expression,target in [("openAllLinksModal()",'all-links-modal'),("openAllSubscriptionsModal()",'all-subscriptions-modal'),("openInboundImportModal()",'inbound-import-modal'),("openClientQR('p5-mieru:alice')",'client-qr-modal'),("openInboundLinksModal('p5-mieru')",'inbound-links-modal')]:
  close_all(page);page.evaluate('void('+expression+')');page.locator('#'+target+'.open').wait_for();shot(page,folder,target)
 close_all(page);page.evaluate("setView('settings')");page.locator('[data-settings-tab=subscription]').click();page.locator('#manage-subscription-tokens').click();page.locator('#subscription-tokens-modal.open').wait_for();shot(page,folder,'subscription-tokens-modal');close_all(page)
 page.evaluate("setView('inbounds')");page.locator('#general-actions').click();shot(page,folder,'general-menu');close_all(page)
def subscription_screens(ctx,url,folder,palette,mode,tokens):
 path=tokens['path'];check('F35 public subscription path exists',bool(path))
 sub=ctx.new_page();sub.on('pageerror',lambda e:errors.append(str(e)));sub.goto(path if path.startswith('http') else url+path,wait_until='networkidle');sub.evaluate('([p,m])=>{UPTheme.setColor(p);UPTheme.setMode(m)}',[palette,mode]);shot(sub,folder,'subscription');sub.set_viewport_size({'width':390,'height':844});shot(sub,folder,'mobile-subscription');sub.close()
if __name__=='__main__':main()
