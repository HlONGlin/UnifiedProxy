"""Native-control behavioral equivalence; must run in orca-linux /workspace."""
from pathlib import Path
import functools,json,threading
from http.server import ThreadingHTTPServer,SimpleHTTPRequestHandler
from playwright.sync_api import sync_playwright

assert Path.cwd()==Path('/workspace') and Path('/.dockerenv').exists()
html='''<!doctype html><html data-theme="light" data-color-theme="t0"><head>
<link rel="stylesheet" href="/web/themes.css"><link rel="stylesheet" href="/web/panel.css">
<link rel="stylesheet" href="/web/components.css"><link rel="stylesheet" href="/web/controls.css">
<script defer src="/web/controls.js"></script><script defer src="/web/native-interactions.js"></script></head>
<body style="padding:40px"><form id="form">
<label class="switch"><input id="normal" name="normal" type="checkbox"><span></span></label>
<label class="switch switch-sm"><input id="small" checked type="checkbox"><span></span></label>
<label class="switch"><input id="disabled" checked disabled type="checkbox"><span></span></label>
<select id="picker"><option value="a">Alpha</option><option value="b">Beta</option><option value="c" disabled>Disabled</option></select>
</form><div class="segmented" id="segments"><button class="active">First</button><button>Second</button><button disabled>Disabled</button></div>
<div id="dynamic"></div><script>window.changes=0;document.querySelector('#normal').onchange=()=>window.changes++;
window.activations=0;document.querySelector('#segments').onclick=e=>{if(e.target.matches('button:not(:disabled)')){window.activations++;document.querySelectorAll('#segments button').forEach(b=>b.classList.toggle('active',b===e.target));}};</script></body></html>'''
class Handler(SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path=='/fixture':
            self.send_response(200);self.send_header('Content-Type','text/html; charset=utf-8');self.end_headers();self.wfile.write(html.encode())
        else:super().do_GET()
    def log_message(self,*args):pass
server=ThreadingHTTPServer(('127.0.0.1',0),functools.partial(Handler,directory='/workspace'))
threading.Thread(target=server.serve_forever,daemon=True).start()
checks=[]
with sync_playwright() as p:
    browser=p.chromium.launch(headless=True,args=['--no-sandbox']);page=browser.new_page()
    errors=[];page.on('pageerror',lambda e:errors.append(str(e)))
    page.goto(f'http://127.0.0.1:{server.server_port}/fixture',wait_until='networkidle')
    def check(condition,name):
        assert condition,name
        checks.append(name)
    check(page.evaluate("document.querySelectorAll('.liquid-toggle-canvas').length===0"),'native controls require no canvas')
    page.locator('#normal').evaluate('(el)=>el.click()')
    check(page.locator('#normal').is_checked() and page.evaluate('changes')==1,'tap changes native value once')
    check(page.evaluate("new FormData(document.querySelector('#form')).get('normal')==='on'"),'form serialization')
    page.locator('#normal').focus();page.keyboard.press('Space')
    check(not page.locator('#normal').is_checked() and page.evaluate('changes')==2,'Space keyboard toggle')
    page.locator('#disabled').evaluate('(el)=>el.click()');check(page.locator('#disabled').is_checked(),'disabled remains checked')
    page.evaluate("document.querySelector('#form').reset()")
    check(not page.locator('#normal').is_checked() and page.locator('#small').is_checked(),'form reset')
    def drag(cancel=False,rtl=False,disable=False):
        return page.evaluate('''({cancel,rtl,disable})=>{const el=document.querySelector('#normal'),shell=el.parentElement;
        document.documentElement.dir=rtl?'rtl':'ltr';el.checked=false;let r=shell.getBoundingClientRect();
        const x1=rtl?r.right-3:r.left+3,x2=rtl?r.left+3:r.right-3;
        const fire=(target,type,x)=>target.dispatchEvent(new PointerEvent(type,{bubbles:true,pointerId:7,isPrimary:true,button:0,clientX:x,clientY:r.top+10}));
        let before=changes;fire(el,'pointerdown',x1);fire(document,'pointermove',x2);if(disable)el.disabled=true;
        fire(window,cancel?'pointercancel':'pointerup',x2);el.disabled=false;
        return {checked:el.checked,delta:changes-before};}''',{'cancel':cancel,'rtl':rtl,'disable':disable})
    check(drag()=={'checked':True,'delta':1},'drag commits once')
    check(drag(cancel=True)=={'checked':False,'delta':0},'pointer cancellation does not change native value')
    check(drag(disable=True)=={'checked':False,'delta':0},'disabled during drag cannot commit')
    check(drag(rtl=True)=={'checked':True,'delta':1},'RTL drag')
    page.evaluate("document.documentElement.dir='ltr'")
    page.locator('#segments button').nth(1).click();check(page.evaluate('activations')==1,'segment click once')
    page.locator('#segments button').nth(1).press('ArrowRight');check(page.locator('#segments button').first.get_attribute('class')=='active','arrow wraps and skips disabled')
    segment=page.evaluate('''()=>{let b=document.querySelector('#segments'),items=b.querySelectorAll('button'),r=items[0].getBoundingClientRect(),s=items[1].getBoundingClientRect();
    const fire=(t,k,x)=>t.dispatchEvent(new PointerEvent(k,{bubbles:true,pointerId:9,isPrimary:true,button:0,clientX:x,clientY:r.top+5}));let before=activations;
    fire(items[0],'pointerdown',r.left+5);fire(document,'pointermove',s.left+5);fire(window,'pointerup',s.left+5);
    return items[1].classList.contains('active')&&activations===before+1;}''')
    check(segment,'segment drag commits once')
    page.locator('#picker').select_option('b');check(page.locator('#picker').input_value()=='b','native select change')
    page.locator('.swift-picker-selection').click();page.keyboard.press('Home');page.keyboard.press('Enter')
    check(page.locator('#picker').input_value()=='a','picker keyboard selection')
    page.locator('.swift-picker-selection').click();page.keyboard.press('Escape')
    check(page.locator('.swift-picker-menu').is_hidden(),'picker escape closes')
    page.evaluate("document.querySelector('#dynamic').innerHTML='<label class=\"switch\"><input id=\"added\" type=\"checkbox\"><span></span></label>'")
    page.locator('#added').evaluate('(el)=>el.click()');check(page.locator('#added').is_checked(),'dynamic switch works without optical controller')
    check(not errors,'zero browser errors');browser.close()
server.shutdown()
Path('dist/design-audit/native-controls.json').write_text(json.dumps({'passed':checks,'errors':errors},indent=2))
print(f'{len(checks)} native interaction checks PASS')
