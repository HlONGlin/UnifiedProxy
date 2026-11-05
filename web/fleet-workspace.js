/* UnifiedProxy fleet workspace. Node policies continue to use the native editors. */
'use strict';
window.UPFleetWorkspace = function ({request, load, selectTarget, hostSettings, t, copyText}) {
  const root = document.querySelector('#view-fleet');
  const ui = {tab:'nodes', query:'', filter:'all', selected:new Set(), enrollment:new Map(), modelID:''};
  let fleet, dialog, notice = null;
  const tr = (zh,en) => (window.upGetLang?.() || 'zh-CN').startsWith('zh') ? zh : en;
  function el(tag,text,parent,props={}) {
    const node=document.createElement(tag);
    if(text!==undefined)node.textContent=text;
    for(const [key,value] of Object.entries(props)) {
      if(key.startsWith('aria-')||key==='role')node.setAttribute(key,value);else node[key]=value;
    }
    parent?.append(node);return node;
  }
  function output(data,error=false) {
    notice={data,error};
    const box=root.querySelector('#fleet-feedback');
    if(box){box.hidden=false;box.classList.toggle('fleet-error',error);box.setAttribute('role',error?'alert':'status');box.replaceChildren();
      el('span',typeof data==='string'?data:(error?tr('操作未完成，请查看详情。','The operation failed. See details.'):tr('操作已受理。任务执行情况可在“任务记录”查看。','Request accepted. Check Task history for progress.')),box);
      if(typeof data!=='string'){const detail=details(box,tr('查看返回详情','Response details'));el('pre',JSON.stringify(data,null,2),detail,{className:'fleet-result'});}
      const close=el('button','×',box,{type:'button',className:'fleet-dismiss','aria-label':tr('关闭提示','Dismiss')});close.onclick=()=>{notice=null;box.hidden=true;};
    }
    if(dialog?.open){const problem=dialog.querySelector('.fleet-dialog-feedback');problem.classList.toggle('fleet-error',error);problem.setAttribute('role',error?'alert':'status');problem.textContent=typeof data==='string'?data:JSON.stringify(data);problem.hidden=false;}
  }
  function button(label,parent,fn,primary=false) {
    const b=el('button',label,parent,{type:'button',className:primary?'primary-btn':'outline-btn'});
    b.onclick=async()=>{if(b.disabled)return;b.disabled=true;b.setAttribute('aria-busy','true');try{await fn();}catch(e){output(e.message,true);}finally{b.disabled=false;b.removeAttribute('aria-busy');}};return b;
  }
  function details(parent,title) {const box=el('details',undefined,parent,{className:'fleet-disclosure'});el('summary',title,box);return el('div',undefined,box,{className:'fleet-disclosure-body'});}
  function block(parent,title,hint) {const box=el('section',undefined,parent,{className:'fleet-block'});if(title)el('h3',title,box);if(hint)el('p',hint,box,{className:'fleet-hint'});return box;}
  function field(parent,label,value='',type='text',hint='') {
    const wrap=el('label',undefined,parent,{className:'fleet-field'});el('span',label,wrap);
    const input=el(type==='textarea'?'textarea':'input',undefined,wrap,{value});if(type!=='textarea')input.type=type;
    if(hint)el('small',hint,wrap);return input;
  }
  function codeEditor(parent,label,value) {const input=field(parent,label,JSON.stringify(value,null,2),'textarea');input.spellcheck=false;input.className='fleet-code-editor';return input;}
  function pick(parent,label,options,value) {const wrap=el('label',undefined,parent,{className:'fleet-field'});el('span',label,wrap);const select=el('select',undefined,wrap,{'aria-label':label});select.setAttribute('data-no-swift-picker','');for(const [key,text] of options)el('option',text,select,{value:key});if(value!==undefined)select.value=value;return select;}
  function empty(parent,title,hint) {const box=el('div',undefined,parent,{className:'fleet-empty'});el('h3',title,box);el('p',hint,box,{className:'fleet-hint'});return box;}
  function closeDialog() {if(dialog?.open)dialog.close();}
  function modal(title,hint) {
    closeDialog();dialog?.remove();const previous=document.activeElement;
    dialog=el('dialog',undefined,document.body,{className:'fleet-dialog','aria-labelledby':'fleet-dialog-title'});
    const head=el('header',undefined,dialog,{className:'fleet-dialog-head'});el('h2',title,head,{id:'fleet-dialog-title'});button(tr('关闭','Close'),head,closeDialog);
    const body=el('div',undefined,dialog,{className:'fleet-dialog-body'});if(hint)el('p',hint,body,{className:'fleet-hint'});
    el('p','',dialog,{className:'fleet-dialog-feedback',hidden:true,role:'status','aria-live':'polite'});
    dialog.addEventListener('click',e=>{if(e.target===dialog){const r=dialog.getBoundingClientRect();if(e.clientX<r.left||e.clientX>r.right||e.clientY<r.top||e.clientY>r.bottom)closeDialog();}});
    dialog.addEventListener('close',()=>{if(!dialog?.open&&previous?.isConnected)previous.focus();});
    dialog.showModal();return body;
  }
  function footer(body) {return el('div',undefined,body,{className:'fleet-dialog-actions'});}
  function validateServer(form,name,address) {
    for(const input of [name,address]){input.setCustomValidity(input.value.trim()?'':tr('请填写有效内容，不能只有空格。','Enter a value containing more than spaces.'));input.oninput=()=>input.setCustomValidity('');}
    return form.reportValidity();
  }
  function go(tab) {ui.tab=tab;render(fleet);}
  function state(node) {
    if(node.revoked)return 'revoked';if(node.online)return 'online';
    return !node.lastSeen||node.lastSeen.startsWith('0001-')?'pending':'offline';
  }
  function stateText(node) {return ({online:t('online'),offline:t('offline'),pending:tr('待接入','Awaiting connection'),revoked:tr('已吊销','Revoked')})[state(node)];}
  function bytes(value) {let n=Number(value)||0,i=0;while(n>=1024&&i<4){n/=1024;i++;}return `${n.toLocaleString(undefined,{maximumFractionDigits:i?1:0})} ${['B','KB','MB','GB','TB'][i]}`;}
  function traffic(node) {let up=0,down=0,legacy=0;for(const [key,v] of Object.entries(node.status?.traffic||{}))if(!key.includes(':')){up+=v.up||0;down+=v.down||0;legacy+=v.legacyCombined||0;}return {up,down,legacy};}
  function formatTime(value) {const date=new Date(value);return !value||value.startsWith('0001-')||!Number.isFinite(+date)?tr('尚未连接','Never connected'):date.toLocaleString(window.upGetLang?.()||'zh-CN');}
  function syncText(node) {const desired=node.desiredRevision||0,applied=node.status?.revision||0;
    if(node.revoked)return tr('接入权限已撤销','Access revoked');if(node.status?.error)return tr('配置应用失败，请检查任务记录','Apply failed; check task history');
    if(!desired)return tr('尚未发布配置','No configuration published');
    return applied===desired?tr(`配置已同步 · 版本 ${applied}`,`In sync · revision ${applied}`):tr(`等待同步 · ${applied} → ${desired}`,`Awaiting sync · ${applied} → ${desired}`);
  }
  async function manage(node,view='inbounds') {if(await selectTarget(node.id)){closeDialog();document.querySelector(`#nav [data-view="${view}"]`)?.click();if(view==='mihomo')document.querySelector('[data-mihomo-tab="outbounds"]')?.click();}}
  function steps(parent) {const list=el('ol',undefined,parent,{className:'fleet-steps'});for(const [title,hint] of [
    [tr('创建服务器卡片','Create a server card'),tr('为子服务器填写名称与公网地址','Give your server a name and public address')],
    [tr('在子服务器执行接入命令','Connect the child server'),tr('复制命令与一次性令牌，等待上线','Copy the command and token, then wait for it to appear online')],
    [tr('在这里管理代理配置','Manage proxy settings here'),tr('入站、用户、出口和路由统一操作','Configure inbounds, users, outbounds and routing from this panel')]
  ]){const item=el('li',undefined,list);el('strong',title,item);el('span',hint,item);}return list;}
  async function enroll(node) {
    let info=ui.enrollment.get(node.id);
    if(!info||info.until<Date.now()){const result=await request('/tokens',{nodeId:node.id,name:node.name});info={...result,until:Date.now()+result.expiresIn*1000};ui.enrollment.set(node.id,info);}
    showEnrollment(node,info);
  }
  function showEnrollment(node,info) {
    const body=modal(tr(`接入子服务器 · ${node.name}`,`Connect server · ${node.name}`),tr('这张卡片已创建。接下来在子服务器终端完成接入，主机页面会自动更新在线状态。','The card is ready. Complete enrollment in the child server terminal; this page updates automatically.'));
    body.dataset.enrollmentNode=node.id;
    const quote=value=>"'"+String(value).replaceAll("'","'\"'\"'")+"'";
    el('p',tr('适用于已安装 node 的子服务器；接入使用 HTTPS。','Requires node to be installed on the child server; enrollment uses HTTPS.'),body,{className:'fleet-callout'});
    const origin=field(body,tr('主机公网地址','Host public URL'),location.origin,'url',tr('子服务器必须能够访问这个地址。HTTPS 反向代理需同时转发 /fleet/v1/。','Must be reachable from the child server. An HTTPS reverse proxy must also forward /fleet/v1/.'));origin.required=true;
    if(location.protocol!=='https:')el('p',tr('当前主机使用 HTTP：请填写已配置证书的 HTTPS 地址后再执行接入。','This panel uses HTTP: enter the configured HTTPS address before enrollment.'),body,{className:'fleet-warning'});
    el('h3',tr('1. 在子服务器终端运行','1. Run on the child server'),body);
    const command=el('code','',body,{className:'fleet-command'});
    const setCommand=()=>{command.textContent='/usr/local/unifiedproxy-node/node agent join --data-dir /usr/local/unifiedproxy-node/data --control '+quote(origin.value.trim())+' --control-key '+quote(info.controlKey)+' --token-stdin';};origin.oninput=setCommand;setCommand();
    button(tr('复制接入命令','Copy enrollment command'),body,async()=>{const url=new URL(origin.value);if(url.protocol!=='https:')throw new Error(tr('接入地址需要使用 HTTPS，请先配置主机证书或 HTTPS 反向代理。','Enrollment requires HTTPS. Configure a certificate or HTTPS reverse proxy first.'));await copyText(command.textContent);output(tr('接入命令已复制','Enrollment command copied'));});
    el('h3',tr('2. 粘贴一次性令牌，按 Ctrl+D 结束输入','2. Paste the one-time token, then press Ctrl+D'),body);
    const token=field(body,tr('一次性接入令牌','One-time enrollment token'),info.token,'password');token.readOnly=true;
    const tokenActions=el('div',undefined,body,{className:'fleet-toolbar'});button(tr('复制令牌','Copy token'),tokenActions,async()=>{await copyText(info.token);output(tr('令牌已复制；粘贴后按 Ctrl+D','Token copied; press Ctrl+D after pasting'));});button(tr('显示 / 隐藏','Show / hide'),tokenActions,()=>{token.type=token.type==='password'?'text':'password';});
    const expiry=el('p',tr(`有效期至 ${new Date(info.until).toLocaleTimeString()}，仅限此节点使用。`,`Expires at ${new Date(info.until).toLocaleTimeString()}; valid for this node only.`),body,{className:'fleet-hint'});
    const timer=setInterval(()=>{if(!dialog?.open||!dialog.contains(expiry)){clearInterval(timer);return;}if(Date.now()>info.until){expiry.textContent=tr('令牌已过期，请关闭窗口后重新生成。','Token expired. Close this window and generate another.');tokenActions.querySelectorAll('button').forEach(b=>b.disabled=true);clearInterval(timer);}},1000);
    el('h3',tr('3. 重启子服务器节点服务','3. Restart the child node service'),body);el('code','systemctl restart unifiedproxy-node',body,{className:'fleet-command'});
    el('p',tr('接入后通常在 15 秒内更新。子服务器只运行代理服务，日常配置都在此主机完成。','Status usually updates within 15 seconds. Manage day-to-day settings from this host.'),body,{className:'fleet-hint'});
    button(tr('完成，返回服务器列表','Done, return to servers'),footer(body),()=>{closeDialog();go('nodes');},true);
  }
  function addServer() {
    const body=modal(tr('添加子服务器','Add child server'),tr('先创建卡片，再把接入命令复制到子服务器。这里只登记服务器，不会直接连接 SSH。','Create a card, then run its enrollment command on the child server. This registers a server without connecting over SSH.'));
    const form=el('form',undefined,body,{id:'fleet-add-form'});
    const name=field(form,tr('服务器名称','Server name'),'','text',tr('例如：香港 01、东京中转','For example: Hong Kong 01 or Tokyo relay'));name.required=true;name.maxLength=120;name.autofocus=true;
    const address=field(form,tr('子服务器公网地址','Child server public address'),'','text',tr('填写子服务器的 IP 或域名，用于生成分享节点；不是主机面板地址。','Child server IP or domain for shared nodes, not this host panel URL.'));address.required=true;address.placeholder='203.0.113.10';
    const actions=footer(form);button(tr('取消','Cancel'),actions,closeDialog);
    const submit=el('button',tr('创建并获取接入命令','Create and get enrollment command'),actions,{type:'submit',className:'primary-btn'});
    form.onsubmit=async e=>{e.preventDefault();if(!validateServer(form,name,address))return;submit.disabled=true;try{const node=await request('/nodes',{name:name.value.trim(),address:address.value.trim()});closeDialog();ui.tab='nodes';ui.query='';ui.filter='all';await load();await enroll(node);}catch(error){output(error.message,true);}finally{submit.disabled=false;}};
    name.focus();
  }
  function editServer(node) {
    const body=modal(tr('编辑服务器信息','Edit server details'),node.name),form=el('form',undefined,body);
    const name=field(form,t('name'),node.name);name.required=true;
    const address=field(form,tr('子服务器公网地址','Child server public address'),node.address);address.required=true;
    const actions=footer(form);button(tr('取消','Cancel'),actions,closeDialog);
    const save=el('button',tr('保存信息','Save details'),actions,{type:'submit',className:'primary-btn'});
    form.onsubmit=async e=>{e.preventDefault();if(!validateServer(form,name,address))return;save.disabled=true;try{await request('/nodes/'+node.id,{name:name.value.trim(),address:address.value.trim()},'PUT');closeDialog();await load();output(tr('服务器信息已保存','Server details saved'));}catch(e){output(e.message,true);}finally{save.disabled=false;}};
  }
  function render(data) {
    fleet=data;root.classList.add('fleet-workspace');root.replaceChildren();
    for(const id of ui.selected)if(!fleet.nodes.some(n=>n.id===id&&!n.revoked))ui.selected.delete(id);
    const heading=el('div',undefined,root,{className:'fleet-heading'}),text=el('div',undefined,heading);
    el('span','UNIFIEDPROXY / FLEET',text,{className:'eyebrow'});el('h2',t('title'),text);
    el('p',tr('一个主机面板，管理所有子服务器的代理配置。','One host panel for every child server and its proxy settings.'),text,{className:'fleet-hint'});
    const actions=el('div',undefined,heading,{className:'fleet-toolbar'});button(t('refresh'),actions,load);button(tr('＋ 添加子服务器','＋ Add child server'),actions,addServer,true);
    const feedback=el('div',undefined,root,{id:'fleet-feedback',className:'fleet-feedback',hidden:true,role:'status'});feedback.setAttribute('aria-live','polite');if(notice)output(notice.data,notice.error);
    const tabs=el('nav',undefined,root,{className:'fleet-tabs','aria-label':tr('舰队工作区','Fleet workspace')});
    for(const [key,label] of [['nodes',tr('子服务器','Servers')],['tasks',tr('任务记录','Task history')],['subscriptions',t('subscriptions')],['backup',tr('备份与迁移','Backup & migration')],['maintenance',tr('主机维护','Host maintenance')],['advanced',tr('高级配置','Advanced')]]) {
      const b=button(label,tabs,()=>go(key));b.dataset.fleetTab=key;if(key===ui.tab)b.setAttribute('aria-current','page');
    }
    const content=el('div',undefined,root,{id:'fleet-content'});
    ({nodes:renderNodes,tasks:renderTasks,subscriptions:renderSubscriptions,backup:renderBackup,maintenance:renderMaintenance,advanced:renderAdvanced})[ui.tab](content);
  }
  function renderNodes(content) {
    const stats=el('div',undefined,content,{className:'fleet-stats'});
    for(const [key,label] of [['all',tr('子服务器总数','Total servers')],['online',tr('在线','Online')],['pending',tr('待接入','Awaiting connection')],['offline',tr('离线','Offline')]]) {
      const b=button('',stats,()=>{ui.filter=key;root.querySelector('#fleet-filter').value=key;renderCards();});b.className='fleet-stat';el('span',label,b);el('strong',String(key==='all'?fleet.nodes.length:fleet.nodes.filter(n=>state(n)===key).length),b,{id:'fleet-count-'+key});
    }
    if(!fleet.nodes.length){const box=block(content,tr('接入你的第一台子服务器','Connect your first child server'),tr('按下面三步完成接入，然后就能在这个网页生成节点、设置出口和路由。','Follow these three steps to create proxy nodes and configure outbounds and routing here.'));steps(box);button(tr('添加第一台子服务器','Add your first child server'),box,addServer,true);}
    else{const help=details(content,tr('如何接入和管理子服务器？','How do I connect and manage servers?'));steps(help);}
    const tools=el('div',undefined,content,{className:'fleet-list-tools'});
    const search=field(tools,tr('搜索服务器','Search servers'),ui.query,'search');search.placeholder=tr('名称、地址或节点 ID','Name, address or node ID');search.id='fleet-search';search.oninput=()=>{ui.query=search.value;renderCards();};
    const filter=pick(tools,tr('连接状态','Connection status'),[['all',tr('全部状态','All states')],['online',t('online')],['pending',tr('待接入','Awaiting connection')],['offline',t('offline')],['revoked',tr('已吊销','Revoked')]],ui.filter);filter.id='fleet-filter';filter.onchange=()=>{ui.filter=filter.value;renderCards();};
    const choose=el('label',undefined,tools,{className:'fleet-check'}),all=el('input',undefined,choose,{type:'checkbox',id:'fleet-select-all'});el('span',tr('选择当前列表','Select visible servers'),choose);all.onchange=()=>{for(const n of filtered())if(!n.revoked){if(all.checked)ui.selected.add(n.id);else ui.selected.delete(n.id);}renderCards();};
    const batch=el('div',undefined,content,{className:'fleet-selection',id:'fleet-selection',hidden:true});el('span','',batch,{id:'fleet-selection-count'});button(t('batch'),batch,()=>go('tasks'),true);button(tr('清除选择','Clear selection'),batch,()=>{ui.selected.clear();renderCards();});
    el('div',undefined,content,{className:'fleet-cards',id:'fleet-cards'});renderCards();
  }
  function filtered(data=fleet){const q=ui.query.trim().toLowerCase();return (data.nodes||[]).filter(n=>(ui.filter==='all'||state(n)===ui.filter)&&(!q||[n.name,n.address,n.id].some(v=>String(v||'').toLowerCase().includes(q))));}
  function renderCards() {
    const cards=root.querySelector('#fleet-cards');if(!cards)return;cards.replaceChildren();
    const nodes=filtered();for(const node of nodes)renderCard(cards,node);
    if(!nodes.length&&fleet.nodes.length)empty(cards,tr('没有匹配的服务器','No matching servers'),tr('试试其他名称或清除状态筛选。','Try a different search or connection filter.'));
    const count=ui.selected.size;root.querySelector('#fleet-selection').hidden=!count;root.querySelector('#fleet-selection-count').textContent=tr(`已选择 ${count} 台服务器`,`${count} servers selected`);
    const available=nodes.filter(n=>!n.revoked),all=root.querySelector('#fleet-select-all');all.checked=available.length>0&&available.every(n=>ui.selected.has(n.id));all.indeterminate=!all.checked&&available.some(n=>ui.selected.has(n.id));
  }
  function renderCard(parent,node) {
    const card=el('article',undefined,parent,{className:'fleet-node-card'});card.dataset.nodeCard=node.id;
    const head=el('div',undefined,card,{className:'fleet-node-head'}),name=el('div',undefined,head);
    el('h3',node.name||node.id,name,{className:'fleet-node-title'});const badge=el('span',stateText(node),name,{className:'fleet-node-state'});badge.dataset.state=state(node);badge.dataset.nodeState=node.id;
    const label=el('label',undefined,head,{className:'fleet-check'}),check=el('input',undefined,label,{type:'checkbox',className:'fleet-selected',checked:ui.selected.has(node.id),value:node.id,disabled:!!node.revoked,'aria-label':tr('选择 ','Select ')+node.name});check.onchange=()=>{check.checked?ui.selected.add(node.id):ui.selected.delete(node.id);renderCards();root.querySelector(`[data-node-card="${node.id}"] .fleet-selected`)?.focus();};
    el('p',node.address||'—',card,{className:'fleet-node-address'});
    const facts=el('dl',undefined,card,{className:'fleet-node-facts'});
    const row=(label,value,key)=>{el('dt',label,facts);const out=el('dd',value,facts);if(key)out.dataset[key]=node.id;};
    row(tr('代理内核','Proxy core'),state(node)==='pending'?tr('等待接入','Awaiting connection'):t(node.status?.running?'running':'stopped'),'fleetCore');
    row(tr('配置状态','Configuration'),syncText(node),'fleetSync');
    const usage=traffic(node);row(tr('累计流量','Total traffic'),bytes(usage.up+usage.down+usage.legacy),'fleetTraffic');
    row(tr('上传 / 下载','Upload / download'),`${bytes(usage.up)} / ${bytes(usage.down)}`,'fleetDirections');
    row(tr('节点版本','Node version'),node.status?.version||'—','fleetVersion');
    row(tr('最近连接','Last seen'),formatTime(node.lastSeen),'fleetSeen');
    if(node.needsUpgrade)el('p',tr('节点版本较旧，部分操作需要先升级。','An older node version may need an upgrade for some actions.'),card,{className:'fleet-warning'});
    const actions=el('div',undefined,card,{className:'fleet-card-actions'});
    const main=button(state(node)==='pending'?tr('获取接入命令','Get enrollment command'):tr('管理节点','Manage node'),actions,()=>{const current=fleet.nodes.find(n=>n.id===node.id);if(current&&!current.revoked)return state(current)==='pending'?enroll(current):manage(current);},true);main.disabled=!!node.revoked;
    const more=details(card,tr('更多操作','More actions'));more.parentElement.classList.add('fleet-node-more');
    button(tr('编辑名称与地址','Edit name and address'),more,()=>editServer(node));
    button(tr('入站与用户','Inbounds & users'),more,()=>manage(node)).disabled=!!node.revoked;
    button(tr('出口与路由','Outbounds & routing'),more,()=>manage(node,'mihomo')).disabled=!!node.revoked;
    button(tr('重新生成接入信息','Generate enrollment info'),more,async()=>{ui.enrollment.delete(node.id);await enroll(node);});
    button(tr('发布当前配置','Publish configuration'),more,async()=>{await request('/nodes/'+node.id+'/publish',{});await load();output(tr('配置已发布，等待子服务器应用。','Configuration published; waiting for the child server to apply it.'));}).disabled=!!node.revoked;
    button(tr('高级配置','Advanced configuration'),more,()=>{ui.modelID=node.id;go('advanced');}).disabled=!!node.revoked;
    const revoke=button(tr('吊销接入权限','Revoke access'),more,async()=>{if(await upConfirm(tr(`吊销“${node.name}”的接入权限？它将停止接收主机配置，已有代理进程可能仍继续运行。`,`Revoke access for “${node.name}”? It will stop receiving host policies; existing proxy processes may keep running.`),{danger:true})){await request('/nodes/'+node.id+'/revoke',{});ui.selected.delete(node.id);await load();output(tr('接入权限已吊销','Access revoked'));}});revoke.classList.add('fleet-danger');revoke.disabled=!!node.revoked;
  }
  function renderTasks(content) {
    const box=block(content,t('batch'),tr('先在“子服务器”选择目标，再选择操作。普通巡检不需要填写 JSON。','Select servers in the Servers tab, then choose an action. Routine checks need no JSON.'));
    const selected=(fleet.nodes||[]).filter(n=>ui.selected.has(n.id)&&!n.revoked);
    el('p',selected.length?tr(`目标：${selected.map(n=>n.name).join('、')}`,`Targets: ${selected.map(n=>n.name).join(', ')}`):tr('还没有选择服务器。','No servers selected.'),box,{className:'fleet-callout'});
    button(tr('返回选择服务器','Choose servers'),box,()=>go('nodes'));
    const form=el('form',undefined,box),kind=pick(form,tr('要执行的操作','Operation'),['collect-status','geo-update','test-egress','kernel-switch','cert-renew','agent-upgrade','defaults','quota','inbound'].map(k=>[k,t(k)]));
    const fields=el('div',undefined,form,{className:'fleet-form-grid'});let inputs={};
    const payloadHint=el('p','',form,{className:'fleet-hint'});
    const configure=()=>{fields.replaceChildren();inputs={};payloadHint.textContent='';
      if(kind.value==='test-egress'){inputs.proxy=field(fields,tr('子服务器本地 SOCKS5 代理','SOCKS5 proxy on child server'),'socks5://127.0.0.1:1080','url');inputs.url=field(fields,tr('测试地址（可选）','Test URL (optional)'),'','url');payloadHint.textContent=tr('填写子服务器上实际可用的 SOCKS5 入口地址。','Use an available SOCKS5 listener on the child server.');}
      if(kind.value==='kernel-switch')inputs.version=field(fields,tr('Mihomo 版本','Mihomo version'),'latest');
      if(kind.value==='agent-upgrade'){inputs.url=field(fields,tr('已验证的更新包 HTTPS 地址','Verified update HTTPS URL'),'','url');inputs.sha256=field(fields,'SHA256');}
      if(kind.value==='quota'){inputs.username=field(fields,t('username'));inputs.total=field(fields,tr('总流量（GB，0 为不限）','Total traffic (GB, 0 = unlimited)'),'0','number');inputs.total.min='0';inputs.total.step='any';inputs.expiryTime=field(fields,tr('到期时间（留空为不限）','Expiry (empty = unlimited)'),'','datetime-local');}
      if(kind.value==='cert-renew')payloadHint.textContent=tr('重新校验并加载主机已下发的证书。需要申请或续期证书时，请进入“主机维护”。','Revalidates and reloads certificates distributed by the host. Issue or renew certificates in Host maintenance.');
      if(kind.value==='defaults')payloadHint.textContent=tr('会将主机全局默认的出口、路由和核心基础设置覆盖到选中节点。','Replaces selected nodes’ outbounds, routing and core basics with host global defaults.');
      if(kind.value==='inbound')inputs.inbound=codeEditor(fields,tr('入站对象（高级操作）','Inbound object (advanced)'),{});
    };kind.onchange=configure;configure();
    const advanced=details(form,tr('高级：覆盖任务参数 JSON','Advanced: override task parameters with JSON')),override=field(advanced,tr('完整参数（留空使用上方表单）','Complete parameters (empty uses the form)'),'','textarea');override.spellcheck=false;
    const submit=el('button',t('execute'),form,{type:'submit',className:'primary-btn',disabled:!selected.length});
    form.onsubmit=async e=>{e.preventDefault();if(!form.reportValidity()||!selected.length)return;submit.disabled=true;
      try{let p={};for(const [key,input] of Object.entries(inputs)){if(key==='inbound')p[key]=JSON.parse(input.value);else if(key==='total')p[key]=Math.round(Number(input.value)*1024**3);else if(key==='expiryTime')p[key]=input.value?new Date(input.value).getTime():0;else if(input.value)p[key]=input.value.trim();}
        if(override.value.trim())p=JSON.parse(override.value);const configAction=['defaults','quota','inbound'].includes(kind.value);
        if(!p||typeof p!=='object'||Array.isArray(p))throw new Error(tr('任务参数必须是一个对象。','Task parameters must be an object.'));
        const required={ 'test-egress':['proxy'], 'kernel-switch':['version'], 'agent-upgrade':['url','sha256'], quota:['username'] }[kind.value]||[];
        if(required.some(key=>!String(p[key]??'').trim()))throw new Error(tr('请填写此操作需要的所有参数。','Complete the required parameters for this operation.'));
        if(kind.value==='agent-upgrade'&&(!/^https:\/\//i.test(p.url)||! /^[a-f\d]{64}$/i.test(p.sha256)))throw new Error(tr('更新包必须使用 HTTPS 地址，并提供 64 位 SHA256。','Use an HTTPS update URL and a 64-character SHA256.'));
        if(!configAction){const unsupported=selected.filter(n=>!n.status?.capabilities?.includes(kind.value));if(unsupported.length)throw new Error(tr('以下节点暂不支持此操作，请先接入或升级：','These nodes need to connect or upgrade first: ')+unsupported.map(n=>n.name).join(', '));}
        if(!['collect-status','test-egress'].includes(kind.value)&&!await upConfirm(tr(`将在 ${selected.length} 台服务器上执行“${t(kind.value)}”，是否继续？`,`Run “${t(kind.value)}” on ${selected.length} servers?`),{danger:true}))return;
        await request('/batch',configAction?{...p,nodes:selected.map(n=>n.id),action:kind.value}:{nodes:selected.map(n=>n.id),action:'task',kind:kind.value,payload:p});
        await load();output(tr('任务已提交。下方会自动更新执行状态。','Tasks submitted. Status updates automatically below.'));
      }catch(e){output(e.message,true);}finally{submit.disabled=false;}
    };
    const records=block(content,tr('最近任务','Recent tasks'),tr('每 15 秒更新；展开单条任务可查看具体结果。','Updates every 15 seconds. Expand a task to inspect its result.'));el('div',undefined,records,{id:'fleet-tasks'});updateTasks();
  }
  function updateTasks() {
    const list=root.querySelector('#fleet-tasks');if(!list)return;const opened=new Set(Array.from(list.querySelectorAll('details[open]')).map(d=>d.dataset.taskId));list.replaceChildren();
    const tasks=(fleet.tasks||[]).slice(-30).reverse();if(!tasks.length){empty(list,tr('还没有执行过任务','No tasks yet'),tr('提交巡检或批量操作后，会在这里显示进度。','Submitted checks and batch actions appear here.'));return;}
    const taskNames={'apply-state':tr('应用节点配置','Apply node configuration'),'purge':tr('停止服务并吊销节点','Stop services and revoke node'),'core-control':tr('控制代理内核','Control proxy core'),'traffic-reset':tr('重置流量','Reset traffic'),'vpngate-install':tr('安装 VPNGate 出口','Install VPNGate outbound'),'vpngate-reconnect':tr('重连 VPNGate','Reconnect VPNGate')};
    for(const task of tasks){const box=el('details',undefined,list,{className:'fleet-task'});box.dataset.taskId=task.id;box.open=opened.has(task.id);const head=el('summary',undefined,box);el('strong',taskNames[task.kind]||t(task.kind),head);el('span',(fleet.nodes||[]).find(n=>n.id===task.nodeId)?.name||task.nodeId,head);
      const status=task.result?.status||task.status;el('span',({pending:tr('排队中','Queued'),queued:tr('排队中','Queued'),leased:tr('执行中','Running'),running:tr('执行中','Running'),succeeded:tr('已完成','Completed'),success:tr('已完成','Completed'),done:tr('已完成','Completed'),failed:tr('失败','Failed'),expired:tr('已过期','Expired'),unsupported:tr('节点不支持','Unsupported'),superseded:tr('已被新配置替代','Superseded'),cancelled:tr('已取消','Cancelled')})[status]||status,head,{className:'fleet-task-status'});el('time',formatTime(task.createdAt),head);
      if(task.result?.error)el('p',task.result.error,box,{className:'fleet-error'});el('pre',JSON.stringify(task.result||{status:task.status,attempts:task.attempts},null,2),box,{className:'fleet-result'});
    }
  }
  function renderSubscriptions(content) {
    const box=block(content,t('subscriptions'),tr('汇总所有子服务器的用户订阅。订阅地址包含访问令牌，请只分享给对应用户。','Aggregated subscriptions across child servers. URLs contain access tokens; share only with the intended user.'));
    button(tr('读取用户订阅','Load subscriptions'),box,async()=>{const tokens=await request('/subscriptions');list.replaceChildren();if(!Object.keys(tokens).length){empty(list,tr('暂无用户订阅','No subscriptions yet'),tr('进入节点的“入站与用户”，添加用户后再读取。','Add users in a node’s Inbounds & users editor, then reload.'));return;}
      const publicPath=(await hostSettings()).subscriptionPath;
      for(const [name,token] of Object.entries(tokens)){const row=block(list,name);if(publicPath){const url=location.origin+publicPath.replace(/\/$/,'')+'/'+encodeURIComponent(token);const link=field(row,tr('订阅地址','Subscription URL'),url);link.readOnly=true;button(tr('复制订阅地址','Copy subscription URL'),row,async()=>{await copyText(url);output(tr('订阅地址已复制','Subscription URL copied'));});}else{const value=field(row,tr('订阅令牌','Subscription token'),token,'password');value.readOnly=true;button(tr('复制令牌','Copy token'),row,()=>copyText(token));}}
    },true);const list=el('div',undefined,box);
  }
  function renderBackup(content) {
    const box=block(content,tr('备份与恢复','Backup & restore'),tr('完整备份保存主机设置、节点身份、节点配置及恢复所需文件。迁移或更新前建议先下载。','Full backups contain host settings, node identities, policies and recovery files. Download one before migration or upgrades.'));
    el('a',t('backup'),box,{className:'primary-btn',href:(typeof panelURL==='function'?panelURL('api/fleet/backup'):'./api/fleet/backup'),download:'fleet-backup.json'});
    const grid=el('div',undefined,content,{className:'fleet-two-columns'});
    const legacy=block(grid,tr('导入历史数据','Import historical data'),tr('选择旧版 merit.json，完成后可下载转换日志。重复导入会合并既有节点。','Choose an old merit.json file. Download the conversion log afterward; repeated imports merge existing nodes.'));
    const file=field(legacy,t('import'),'','file');file.accept='.json';
    button(tr('导入并转换','Import and convert'),legacy,async()=>{if(!file.files[0])throw new Error(tr('请先选择 JSON 文件','Choose a JSON file first'));const result=await request('/import/merit',JSON.parse(await file.files[0].text()));await load();output(result);const blob=URL.createObjectURL(new Blob([JSON.stringify(result,null,2)],{type:'application/json'}));const a=el('a',t('downloadLog'),root.querySelector('#fleet-content'),{href:blob,download:'merit-conversion-log.json',className:'outline-btn'});a.onclick=()=>setTimeout(()=>URL.revokeObjectURL(blob),1000);});
    const restore=block(grid,t('restore'),tr('恢复会替换当前舰队数据，请确认备份来自可信来源。','Restoring replaces current fleet data. Use a trusted backup.')),restoreFile=field(restore,tr('完整舰队备份 JSON','Full fleet backup JSON'),'','file');restoreFile.accept='.json';
    button(t('restore'),restore,async()=>{if(!restoreFile.files[0])throw new Error(tr('请先选择备份文件','Choose a backup first'));if(await upConfirm(tr('恢复将替换当前主机与舰队数据，继续吗？','Restore and replace current host and fleet data?'),{danger:true})){await request('/backup',{action:'restore',backup:JSON.parse(await restoreFile.files[0].text())});await load();output(tr('备份恢复完成','Backup restored'));}});
    const dav=details(content,tr('WebDAV 远程备份','WebDAV remote backup')),davURL=field(dav,'WebDAV URL','','url');davURL.placeholder='https://backup.example/fleet-backup.json';const user=field(dav,t('username')),pass=field(dav,t('password'),'','password');pass.autocomplete='new-password';
    const credentials=()=>({url:davURL.value,username:user.value,password:pass.value});button(t('uploadBackup'),dav,async()=>output(await request('/backup',{action:'upload',webdav:credentials()})));button(tr('从 WebDAV 恢复','Restore from WebDAV'),dav,async()=>{if(await upConfirm(tr('从远程备份替换当前舰队数据？','Replace current fleet data with the remote backup?'),{danger:true})){output(await request('/backup',{action:'download',webdav:credentials()}));await load();}});
  }
  function renderMaintenance(content) {
    const identity=block(content,tr('主机身份','Host identity'),tr('子服务器使用公钥确认主机身份；请通过可信渠道核对。日常使用不需要修改它。','Child servers use this public key to verify the host. Confirm it over a trusted channel.'));
    el('code',fleet.publicKey,identity,{className:'fleet-command'});button(tr('复制公钥','Copy public key'),identity,async()=>{await copyText(fleet.publicKey);output(tr('主机公钥已复制','Host public key copied'));});
    const cert=block(content,t('certificates'),tr('集中管理证书并下发给绑定的子服务器。','Manage certificates centrally and distribute them to bound child servers.'));
    const certJSON=codeEditor(details(cert,tr('证书参数（高级）','Certificate parameters (advanced)')),tr('证书对象','Certificate object'),{name:'',domain:'',dns:'',automatic:false,certificate:'',privateKey:'',bindings:[]});
    button(tr('查看证书列表','View certificates'),cert,async()=>output(await request('/certificates')));
    for(const [label,path] of [['issue','issue'],['renew','renew'],['importCert','import'],['bind','bind']])button(t(label),cert,async()=>output(await request('/certificates/'+path,JSON.parse(certJSON.value))));
    const update=block(content,t('updateControl'),tr('先检查可用版本，再执行更新。子服务器升级在“任务记录”中操作。','Check the release before updating. Upgrade child servers from Task history.'));
    const updateJSON=codeEditor(details(update,tr('更新包信息','Update package details')),tr('更新参数','Update parameters'),{url:'',sha256:''});
    button(t('checkUpdate'),update,async()=>{const info=await request('/update/check');updateJSON.value=JSON.stringify({url:info.url,sha256:info.sha256},null,2);output(info);});button(t('updateControl'),update,async()=>{if(await upConfirm(tr('更新主机控制端？服务会短暂重启。','Update the host control panel? It will briefly restart.'),{danger:true}))output(await request('/update',JSON.parse(updateJSON.value)));});
  }
  function renderAdvanced(content) {
    const box=block(content,tr('节点高级配置','Advanced node configuration'),tr('日常添加入站、用户、出口和路由，请使用节点卡片上的“管理节点”。这里供需要直接编辑完整模型或辅助服务的用户使用。','Use Manage node for everyday inbounds, users, outbounds and routing. This area edits complete models and sidecars.'));
    const nodes=(fleet.nodes||[]).filter(n=>!n.revoked);if(!nodes.length){empty(box,tr('请先添加子服务器','Add a child server first'),tr('创建卡片后即可选择其高级配置。','Create a server card to edit its advanced configuration.'));return;}
    const select=pick(box,tr('要编辑的服务器','Server to edit'),[['',tr('选择服务器…','Choose a server…')],...nodes.map(n=>[n.id,n.name])],ui.modelID);
    const workspace=el('div',undefined,box);let generation=0;
    select.onchange=async()=>{const version=++generation;ui.modelID=select.value;workspace.replaceChildren();if(!select.value)return;el('p',tr('正在读取配置…','Loading configuration…'),workspace);try{const detail=await request('/nodes/'+select.value);if(version!==generation)return;workspace.replaceChildren();
      const target=select.value;const model=codeEditor(details(workspace,t('model')),t('model'),detail.draft||detail.desired?.model||{});
      button(t('save'),model.parentElement.parentElement,async()=>{if(await upConfirm(tr('发布完整模型会替换该节点配置，继续吗？','Publishing replaces this node’s complete configuration. Continue?'),{danger:true}))output(await request('/nodes/'+target+'/model',JSON.parse(model.value)));});
      const side=details(workspace,t('sidecars')),sidecars=codeEditor(side,t('sidecars'),detail.desired?.sidecars||[]);button(t('save'),side,async()=>output(await request('/nodes/'+target+'/sidecars',JSON.parse(sidecars.value))));
      const recipe=details(workspace,t('createSidecar'));el('p',tr('辅助代理服务单独运行，当前不计入用户流量。','Sidecar proxies run separately and are not included in per-user traffic.'),recipe,{className:'fleet-hint'});
      const config=codeEditor(recipe,tr('辅助服务参数','Sidecar parameters'),{kind:'shadowtls',port:443,username:'',password:'',domain:'',handshake:'cloudflare.com:443',certificateName:''});
      button(t('createSidecar'),recipe,async()=>{const result=await request('/nodes/'+target+'/sidecar',JSON.parse(config.value));sidecars.value=JSON.stringify(result.sidecars,null,2);output(result);});
    }catch(error){if(version===generation){workspace.replaceChildren();output(error.message,true);}}};if(select.value)select.onchange();
  }
  function update(data) {
    const previous=fleet;fleet=data;
    for(const key of ['all','online','pending','offline']){const count=root.querySelector('#fleet-count-'+key);if(count)count.textContent=String(key==='all'?data.nodes.length:data.nodes.filter(n=>state(n)===key).length);}
    // Polling must not replace forms, lose focus, collapse menus or erase drafts.
    const shape=nodes=>JSON.stringify(nodes.map(n=>[n.id,n.name,n.address,n.revoked,n.needsUpgrade]));
    if(previous&&shape(filtered())!==shape(filtered(previous)))renderCards();
    for(const node of data.nodes||[]) {
      const card=Array.from(root.querySelectorAll('[data-node-card]')).find(c=>c.dataset.nodeCard===node.id);if(!card)continue;
      const badge=card.querySelector('[data-node-state]');badge.textContent=stateText(node);badge.dataset.state=state(node);
      card.querySelector('[data-fleet-core]').textContent=state(node)==='pending'?tr('等待接入','Awaiting connection'):t(node.status?.running?'running':'stopped');card.querySelector('[data-fleet-sync]').textContent=syncText(node);card.querySelector('[data-fleet-seen]').textContent=formatTime(node.lastSeen);
      const usage=traffic(node);card.querySelector('[data-fleet-traffic]').textContent=bytes(usage.up+usage.down+usage.legacy);card.querySelector('[data-fleet-directions]').textContent=`${bytes(usage.up)} / ${bytes(usage.down)}`;card.querySelector('[data-fleet-version]').textContent=node.status?.version||'—';
      const primary=card.querySelector('.primary-btn');primary.textContent=state(node)==='pending'?tr('获取接入命令','Get enrollment command'):tr('管理节点','Manage node');if(!primary.hasAttribute('aria-busy'))primary.disabled=!!node.revoked;
    }
    updateTasks();
  }
  return {render,update,output};
};
