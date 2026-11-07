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
  function pick(parent,label,options,value) {const wrap=el('label',undefined,parent,{className:'fleet-field'});el('span',label,wrap);const select=el('select',undefined,wrap,{'aria-label':label});select.setAttribute('data-no-swift-picker','');for(const option of options){if(option?.group){const group=el('optgroup',option.group,select,{label:option.group});for(const [key,text] of option.options||[])el('option',text,group,{value:key});}else{const [key,text]=option;el('option',text,select,{value:key});}}if(value!==undefined)select.value=value;return select;}
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
    name.setCustomValidity(name.value.trim()?'':tr('请填写有效名称，不能只有空格。','Enter a name containing more than spaces.'));name.oninput=()=>name.setCustomValidity('');
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
    if(node.revoked)return tr('接入权限已撤销，节点会清空数据和已下载程序','Access revoked; the node clears its data and downloaded programs');if(node.status?.error)return tr('配置应用失败，请检查任务记录','Apply failed; check task history');
    if(!desired)return tr('尚未发布配置','No configuration published');
    return applied===desired?tr(`配置已同步 · 版本 ${applied}`,`In sync · revision ${applied}`):tr(`等待同步 · ${applied} → ${desired}`,`Awaiting sync · ${applied} → ${desired}`);
  }
  async function manage(node,view='dashboard') {closeDialog();await selectTarget(node.id,view);}
  async function nodeCoreAction(node,action) {
    const label={start:tr('启动 Mihomo','Start Mihomo'),stop:tr('停止 Mihomo','Stop Mihomo'),restart:tr('重启 Mihomo','Restart Mihomo')}[action]||action;
    if(action!=='start'&&!await upConfirm(tr(`将在“${node.name}”上${label}，是否继续？`,`Run ${label} on “${node.name}”?`),{danger:action==='stop'}))return;
    await request('/batch',{nodes:[node.id],action:'task',kind:'core-control',payload:{action}});
    await load();output(tr(`${label}任务已提交，请在任务记录查看结果。`,`${label} task submitted. Check task history for the result.`));
  }
  function steps(parent) {const list=el('ol',undefined,parent,{className:'fleet-steps'});for(const [title,hint] of [
    [tr('创建服务器卡片','Create a server card'),tr('填写名称，公网地址可留空自动识别','Enter a name; leave the public address blank for automatic detection')],
    [tr('在子服务器执行接入命令','Connect the child server'),tr('复制一条命令，自动安装、接入并启动','Copy one command to install, connect and start automatically')],
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
    el('p',tr('如果子服务器已绑定其他主机，终端会询问是否覆盖。确认覆盖将永久删除该服务器的节点数据和本地备份；原主机的卡片需在原主机上手动删除。','If the child server is bound to another host, the terminal will ask whether to replace it. Confirming permanently deletes its node data and local backups; remove the old card from the original host separately.'),body,{className:'fleet-callout'});
    const quote=value=>"'"+String(value).replaceAll("'","'\"'\"'")+"'";
    el('p',tr('在 Linux 子服务器以 root 执行。命令已包含一次性 token，会自动安装节点与缺失的 Mihomo、完成接入并启动服务。','Run as root on the Linux child server. The command includes its one-time token and installs the node and missing Mihomo core, enrolls and starts the service.'),body,{className:'fleet-callout'});
    const origin=field(body,tr('主机公网地址','Host public URL'),location.origin,'url',tr('支持 http://IP:端口 或 HTTPS 地址，无需先配置证书。不附加面板路径；反向代理需转发 /fleet/v1/。','Use http://IP:port or an HTTPS origin. No certificate is required for HTTP. Omit the panel path; reverse proxies must forward /fleet/v1/.'));origin.required=true;
    el('p',tr('HTTP 可直接接入，但不加密传输；需要传输加密时使用 HTTPS。','HTTP connects directly without transport encryption; use HTTPS when encryption is needed.'),body,{className:'fleet-hint'});
    el('h3',tr('复制整条命令，粘贴回车即可','Copy the complete command, paste and press Enter'),body);
    const command=el('code','',body,{className:'fleet-command'});
    const version=/^v\d+\.\d+\.\d+([+-][\dA-Za-z.-]+)?$/.test(info.installerVersion||'')?info.installerVersion:'latest';
    const manual=details(body,tr('已安装节点 / Docker：手动接入','Existing node / Docker: manual enrollment'));
    el('p',tr('自动脚本适用于 systemd。Docker 或自定义路径可使用下面的已含 token 命令，按实际路径调整；Docker 接入后重启节点容器。','The automatic installer requires systemd. For Docker or custom paths, adapt the token-inclusive command below and restart the node container after enrollment.'),manual,{className:'fleet-hint'});
    const manualCommand=el('code','',manual,{className:'fleet-command'});
    const setCommand=()=>{const url=origin.value.trim().replace(/\/+$/,'');const args=' --control '+quote(url)+' --control-key '+quote(info.controlKey)+' --token-stdin';const protocols=/^http:/i.test(url)?'=http,https':'=https';
      command.textContent='( set -euo pipefail; up_install_script="$(mktemp)"; trap \'rm -f "$up_install_script"\' EXIT; curl -fsSL --proto '+quote(protocols)+' --proto-redir '+quote(protocols)+' --connect-timeout 10 --max-time 60 '+quote(url+'/fleet/v1/install.sh')+' -o "$up_install_script"; printf \'%s\' '+quote(info.token)+' | bash "$up_install_script"'+args+' --node-id '+quote(node.id)+' --version '+quote(version)+' )';
      manualCommand.textContent='printf \'%s\' '+quote(info.token)+' | /usr/local/unifiedproxy-node/node agent join --data-dir /usr/local/unifiedproxy-node/data'+args;
    };origin.oninput=setCommand;setCommand();
    const validateOrigin=()=>{const url=new URL(origin.value);if(!['http:','https:'].includes(url.protocol)||url.username||url.password||url.search||url.hash||!['','/'].includes(url.pathname))throw new Error(tr('请填写 HTTP 或 HTTPS 源站地址，不附加面板路径、账号或参数。','Enter the HTTP or HTTPS origin without a panel path, credentials or query.'));if(Date.now()>info.until)throw new Error(tr('令牌已过期，请重新生成接入命令。','The token expired. Generate a new enrollment command.'));};
    const copy=button(tr('复制一键接入命令','Copy one-command setup'),body,async()=>{validateOrigin();await copyText(command.textContent);output(tr('一键接入命令已复制，到子服务器粘贴回车即可。','Setup command copied. Paste it in the child server terminal and press Enter.'));},true);
    body.insertBefore(copy,manual.parentElement);
    const manualCopy=button(tr('复制手动接入命令','Copy manual enrollment'),manual,async()=>{validateOrigin();await copyText(manualCommand.textContent);output(tr('手动接入命令已复制','Manual enrollment command copied'));});
    el('p',tr('命令含 15 分钟内有效的一次性 token，仅限当前卡片使用，请勿分享。无需单独粘贴令牌，也无需手动重启服务。','The command contains a one-time token valid for 15 minutes for this card. Keep it private. No separate token paste or manual service restart is needed.'),body,{className:'fleet-hint'});
    const expiry=el('p',tr(`有效期至 ${new Date(info.until).toLocaleTimeString()}，仅限此节点使用。`,`Expires at ${new Date(info.until).toLocaleTimeString()}; valid for this node only.`),body,{className:'fleet-hint'});
    const timer=setInterval(()=>{if(!dialog?.open||!dialog.contains(expiry)){clearInterval(timer);return;}if(Date.now()>info.until){expiry.textContent=tr('令牌已过期，请关闭窗口后重新生成。','Token expired. Close this window and generate another.');copy.disabled=true;manualCopy.disabled=true;clearInterval(timer);}},1000);
    el('p',tr('接入后通常在 15 秒内更新。子服务器只运行代理服务，日常配置都在此主机完成。','Status usually updates within 15 seconds. Manage day-to-day settings from this host.'),body,{className:'fleet-hint'});
    button(tr('完成，返回服务器列表','Done, return to servers'),footer(body),()=>{closeDialog();go('nodes');},true);
  }
  function addServer() {
    const body=modal(tr('添加子服务器','Add child server'),tr('先创建卡片，再把接入命令复制到子服务器。这里只登记服务器，不会直接连接 SSH。','Create a card, then run its enrollment command on the child server. This registers a server without connecting over SSH.'));
    const form=el('form',undefined,body,{id:'fleet-add-form'});
    const name=field(form,tr('服务器名称','Server name'),'','text',tr('例如：香港 01、东京中转','For example: Hong Kong 01 or Tokyo relay'));name.required=true;name.maxLength=120;name.autofocus=true;
    const address=field(form,tr('子服务器公网地址','Child server public address'),'','text',tr('选填。留空后由子服务器自动识别并通过心跳上报；填写 IP 或域名则优先使用，用于生成分享节点。','Optional. Leave blank to detect and report automatically; an IP or domain entered here overrides detection for shared nodes.'));address.placeholder=tr('留空自动识别','Leave blank to detect automatically');
    const actions=footer(form);button(tr('取消','Cancel'),actions,closeDialog);
    const submit=el('button',tr('创建并获取接入命令','Create and get enrollment command'),actions,{type:'submit',className:'primary-btn'});
    form.onsubmit=async e=>{e.preventDefault();if(!validateServer(form,name,address))return;submit.disabled=true;try{const node=await request('/nodes',{name:name.value.trim(),address:address.value.trim()});closeDialog();ui.tab='nodes';ui.query='';ui.filter='all';await load();await enroll(node);}catch(error){output(error.message,true);}finally{submit.disabled=false;}};
    name.focus();
  }
  function editServer(node) {
    const body=modal(tr('编辑服务器信息','Edit server details'),node.name),form=el('form',undefined,body);
    const name=field(form,t('name'),node.name);name.required=true;
    const address=field(form,tr('子服务器公网地址','Child server public address'),node.configuredAddress??node.address,'text',tr('手动地址优先；清空并保存即可恢复自动识别。','A manual address takes priority. Clear and save to restore automatic detection.'));address.placeholder=node.seenIP||tr('留空自动识别','Leave blank to detect automatically');
    const actions=footer(form);button(tr('取消','Cancel'),actions,closeDialog);
    const save=el('button',tr('保存信息','Save details'),actions,{type:'submit',className:'primary-btn'});
    form.onsubmit=async e=>{e.preventDefault();if(!validateServer(form,name,address))return;save.disabled=true;try{await request('/nodes/'+node.id,{name:name.value.trim(),address:address.value.trim()},'PUT');closeDialog();await load();output(tr('服务器信息已保存','Server details saved'));}catch(e){output(e.message,true);}finally{save.disabled=false;}};
  }
  function render(data) {
    fleet=data;root.classList.add('fleet-workspace');root.replaceChildren();
    for(const id of ui.selected)if(!fleet.nodes.some(n=>n.id===id))ui.selected.delete(id);
    const heading=el('div',undefined,root,{className:'fleet-heading'}),text=el('div',undefined,heading);
    el('span','UNIFIEDPROXY / FLEET',text,{className:'eyebrow'});el('h2',t('title'),text);
    el('p',tr('一个主机面板，管理所有子服务器的代理配置。','One host panel for every child server and its proxy settings.'),text,{className:'fleet-hint'});
    const actions=el('div',undefined,heading,{className:'fleet-toolbar'});button(t('refresh'),actions,load);button(tr('＋ 添加子服务器','＋ Add child server'),actions,addServer,true);
    const feedback=el('div',undefined,root,{id:'fleet-feedback',className:'fleet-feedback',hidden:true,role:'status'});feedback.setAttribute('aria-live','polite');if(notice)output(notice.data,notice.error);
    const tabs=el('nav',undefined,root,{className:'fleet-tabs','aria-label':tr('舰队工作区','Fleet workspace')});
    for(const [key,label] of [['certificates',tr('证书管理','Certificates')],['nodes',tr('子服务器','Servers')],['tasks',tr('任务记录','Task history')],['subscriptions',t('subscriptions')],['backup',tr('备份与迁移','Backup & migration')],['maintenance',tr('主机维护','Host maintenance')],['advanced',tr('高级配置','Advanced')]]) {
      const b=button(label,tabs,()=>go(key));b.dataset.fleetTab=key;if(key===ui.tab)b.setAttribute('aria-current','page');
    }
    const content=el('div',undefined,root,{id:'fleet-content'});
    ({nodes:renderNodes,certificates:renderCertificates,tasks:renderTasks,subscriptions:renderSubscriptions,backup:renderBackup,maintenance:renderMaintenance,advanced:renderAdvanced})[ui.tab](content);
  }
  function renderNodes(content) {
    const stats=el('div',undefined,content,{className:'fleet-stats'});
    for(const [key,label] of [['all',tr('子服务器总数','Total servers')],['online',tr('在线','Online')],['pending',tr('待接入','Awaiting connection')],['offline',tr('离线','Offline')]]) {
      const b=button('',stats,()=>{ui.filter=key;root.querySelector('#fleet-filter').value=key;renderCards();});b.className='fleet-stat';b.setAttribute('aria-pressed',String(ui.filter===key));el('span',label,b);el('strong',String(key==='all'?fleet.nodes.length:fleet.nodes.filter(n=>state(n)===key).length),b,{id:'fleet-count-'+key});
    }
    if(!fleet.nodes.length){const box=block(content,tr('接入你的第一台子服务器','Connect your first child server'),tr('按下面三步完成接入，然后就能在这个网页生成节点、设置出口和路由。','Follow these three steps to create proxy nodes and configure outbounds and routing here.'));steps(box);button(tr('添加第一台子服务器','Add your first child server'),box,addServer,true);}
    else{const help=details(content,tr('如何接入和管理子服务器？','How do I connect and manage servers?'));steps(help);}
    el('p',tr('节点每 15 秒独立发送心跳；90 秒未收到上报显示离线。更新等长任务不会暂停心跳。','Nodes send an independent heartbeat every 15 seconds; no report for 90 seconds means offline. Long maintenance tasks do not pause heartbeats.'),content,{className:'fleet-hint'});
    const tools=el('div',undefined,content,{className:'fleet-list-tools'});
    const search=field(tools,tr('搜索服务器','Search servers'),ui.query,'search');search.placeholder=tr('名称、地址或节点 ID','Name, address or node ID');search.id='fleet-search';search.oninput=()=>{ui.query=search.value;renderCards();};
    const filter=pick(tools,tr('连接状态','Connection status'),[['all',tr('全部状态','All states')],['online',t('online')],['pending',tr('待接入','Awaiting connection')],['offline',t('offline')],['revoked',tr('已吊销','Revoked')]],ui.filter);filter.id='fleet-filter';filter.onchange=()=>{ui.filter=filter.value;renderCards();};
    const choose=el('label',undefined,tools,{className:'fleet-check'}),all=el('input',undefined,choose,{type:'checkbox',id:'fleet-select-all'});el('span',tr('选择当前列表','Select visible servers'),choose);all.onchange=()=>{for(const n of filtered()){if(all.checked)ui.selected.add(n.id);else ui.selected.delete(n.id);}renderCards();};
    const batch=el('div',undefined,content,{className:'fleet-selection',id:'fleet-selection',hidden:true});el('span','',batch,{id:'fleet-selection-count'});const batchTasks=button(t('batch'),batch,()=>go('tasks'),true);batchTasks.dataset.fleetBatchTasks='';const deleteSelected=button(tr('删除已吊销项','Delete revoked'),batch,deleteSelectedRevoked);deleteSelected.classList.add('fleet-danger');deleteSelected.dataset.fleetDeleteSelected='';button(tr('清除选择','Clear selection'),batch,()=>{ui.selected.clear();renderCards();});
    el('div',undefined,content,{className:'fleet-cards',id:'fleet-cards'});renderCards();
  }
  function filtered(data=fleet){const q=ui.query.trim().toLowerCase();return (data.nodes||[]).filter(n=>(ui.filter==='all'||state(n)===ui.filter)&&(!q||[n.name,n.address,n.id].some(v=>String(v||'').toLowerCase().includes(q))));}
  function renderCards() {
    const cards=root.querySelector('#fleet-cards');if(!cards)return;cards.replaceChildren();
    const nodes=filtered();for(const node of nodes)renderCard(cards,node);
    if(!nodes.length&&fleet.nodes.length)empty(cards,tr('没有匹配的服务器','No matching servers'),tr('试试其他名称或清除状态筛选。','Try a different search or connection filter.'));
    const selected=(fleet.nodes||[]).filter(n=>ui.selected.has(n.id)),count=selected.length,taskCount=selected.filter(n=>!n.revoked).length,revokedCount=selected.filter(n=>n.revoked).length;root.querySelector('#fleet-selection').hidden=!count;root.querySelector('#fleet-selection-count').textContent=tr('已选择 '+count+' 台（'+taskCount+' 台可批量操作，'+revokedCount+' 台已吊销）',count+' selected ('+taskCount+' available for tasks, '+revokedCount+' revoked)');const taskButton=root.querySelector('[data-fleet-batch-tasks]'),deleteButton=root.querySelector('[data-fleet-delete-selected]');taskButton.textContent=tr('批量操作 ('+taskCount+')','Batch actions ('+taskCount+')');taskButton.disabled=taskCount===0;deleteButton.textContent=tr('删除已吊销项 ('+revokedCount+')','Delete revoked ('+revokedCount+')');deleteButton.disabled=revokedCount===0;const all=root.querySelector('#fleet-select-all');all.checked=nodes.length>0&&nodes.every(n=>ui.selected.has(n.id));all.indeterminate=!all.checked&&nodes.some(n=>ui.selected.has(n.id));
  }
  async function deleteFleetNodes(ids) {
    const targets=(fleet.nodes||[]).filter(n=>ids.includes(n.id)&&n.revoked);if(!targets.length)return;
    const names=targets.map(n=>n.name||n.id),shown=names.slice(0,5).join('、'),more=names.length>5?tr(' 等 '+(names.length-5)+' 台',' and '+(names.length-5)+' more'):'';
    const warning=tr('将永久删除 '+targets.length+' 张已吊销卡片（'+shown+more+'）及主机保存的节点草稿。删除后主机无法再向这些子服务器发送吊销清理指令；尚未收到指令的离线服务器可能仍保留并运行原配置。确定继续？','Permanently delete '+targets.length+' revoked server card(s) ('+shown+more+') and the host’s saved drafts? The host can no longer send these servers the cleanup instruction. An offline server that has not received it may keep running its existing configuration. Continue?');
    if(!await upConfirm(warning,{danger:true}))return;
    if(targets.length===1)await request('/nodes/'+encodeURIComponent(targets[0].id),{},'DELETE');
    else await request('/nodes',{nodes:targets.map(n=>n.id)},'DELETE');
    for(const node of targets)ui.selected.delete(node.id);
    await load();output(tr('已删除 '+targets.length+' 张服务器卡片。','Deleted '+targets.length+' server card(s).'));
  }
  function deleteSelectedRevoked(){return deleteFleetNodes(Array.from(ui.selected));}
  function renderCard(parent,node) {
    const card=el('article',undefined,parent,{className:'fleet-node-card'});card.dataset.nodeCard=node.id;
    const head=el('div',undefined,card,{className:'fleet-node-head'}),name=el('div',undefined,head);
    const title=el('h3',undefined,name,{className:'fleet-node-title'});
    title.textContent=node.name||node.id;
    const badge=el('span',stateText(node),name,{className:'fleet-node-state'});badge.dataset.state=state(node);badge.dataset.nodeState=node.id;
    const label=el('label',undefined,head,{className:'fleet-check'}),check=el('input',undefined,label,{type:'checkbox',className:'fleet-selected',checked:ui.selected.has(node.id),value:node.id,'aria-label':tr('选择 ','Select ')+node.name});check.onchange=()=>{check.checked?ui.selected.add(node.id):ui.selected.delete(node.id);renderCards();root.querySelector(`[data-node-card="${node.id}"] .fleet-selected`)?.focus();};
    el('p',node.address||tr('等待节点心跳自动识别公网地址','Waiting for a heartbeat to detect the public address'),card,{className:'fleet-node-address'});
    el('p',node.addressSource==='manual'?tr('手动指定地址','Manual address'):node.addressSource==='auto'?tr('自动识别地址','Automatically detected address'):tr('公网地址可留空，接入后自动识别','Public address is optional and detected after connection'),card,{className:'fleet-hint'});
    const facts=el('dl',undefined,card,{className:'fleet-node-facts'});
    const row=(label,value,key)=>{el('dt',label,facts);const out=el('dd',value,facts);if(key)out.dataset[key]=node.id;};
    row(tr('代理内核','Proxy core'),state(node)==='pending'?tr('等待接入','Awaiting connection'):t(node.status?.running?'running':'stopped'),'fleetCore');
    row(tr('配置状态','Configuration'),syncText(node),'fleetSync');
    const usage=traffic(node);row(tr('累计流量','Total traffic'),bytes(usage.up+usage.down+usage.legacy),'fleetTraffic');
    row(tr('上传 / 下载','Upload / download'),`${bytes(usage.up)} / ${bytes(usage.down)}`,'fleetDirections');
    row(tr('节点版本','Node version'),node.status?.version||'—','fleetVersion');
    row(tr('最近心跳','Last heartbeat'),formatTime(node.lastSeen),'fleetSeen');
    if(node.needsUpgrade)el('p',tr('节点版本较旧，部分操作需要先升级。','An older node version may need an upgrade for some actions.'),card,{className:'fleet-warning'});
    const actions=el('div',undefined,card,{className:'fleet-card-actions'});
    const main=button(node.revoked?tr('恢复接入权限','Restore access'):state(node)==='pending'?tr('获取接入命令','Get enrollment command'):tr('进入节点工作台','Open node workspace'),actions,async()=>{const current=fleet.nodes.find(n=>n.id===node.id);if(!current)return;if(current.revoked){if(!await upConfirm(tr(`恢复“${current.name}”的接入权限？尚未清空的子服务器可以重新连接。已经清空的子服务器需要重新接入。`,`Restore access for “${current.name}”? A child server that has not cleared itself can reconnect. A server that already wiped its data must be enrolled again.`),{danger:true}))return;await request('/nodes/'+current.id+'/restore',{});await load();output(tr('接入权限已恢复。尚未清空的子服务器可以重新连接。','Access restored. A child server that has not cleared itself can reconnect.'));return;}return state(current)==='pending'?enroll(current):manage(current);},true);
    const more=details(card,tr('更多操作','More actions'));more.parentElement.classList.add('fleet-node-more');
    button(tr('编辑名称与地址','Edit name and address'),more,()=>editServer(node));
    if(!node.revoked){
      button(node.status?.running?tr('停止 Mihomo','Stop Mihomo'):tr('启动 Mihomo','Start Mihomo'),more,()=>nodeCoreAction(node,node.status?.running?'stop':'start'));
      button(tr('重启 Mihomo','Restart Mihomo'),more,()=>nodeCoreAction(node,'restart'));
      button(tr('重新生成接入命令','Generate enrollment command'),more,async()=>{ui.enrollment.delete(node.id);await enroll(node);});
    }
    button(tr('发布当前配置','Publish configuration'),more,async()=>{await request('/nodes/'+node.id+'/publish',{});await load();output(tr('配置已发布，等待子服务器应用。','Configuration published; waiting for the child server to apply it.'));}).disabled=!!node.revoked;
    button(tr('高级配置','Advanced configuration'),more,()=>{ui.modelID=node.id;go('advanced');}).disabled=!!node.revoked;
    const revoke=button(tr('吊销接入权限','Revoke access'),more,async()=>{if(await upConfirm(tr(`吊销“${node.name}”的接入权限？它将停止接收主机配置。子服务器下次连上主机会停止并删除已下载的 Mihomo、Sidecar 和 VPNGate，并清空该节点的配置、证书、流量和接入数据。安装目录以外的程序不会删除。应急接管中的节点不会自动清空。`,`Revoke access for “${node.name}”? It will stop receiving host configuration. The next time the child server connects, it stops and deletes the downloaded Mihomo, Sidecar, and VPNGate programs, then clears that node's configuration, certificates, traffic, and enrollment data. Programs outside the install directory are kept. A node in emergency takeover is not cleared automatically.`),{danger:true})){await request('/nodes/'+node.id+'/revoke',{});ui.selected.delete(node.id);await load();output(tr('接入权限已吊销。子服务器会停止并删除已下载的代理程序，并清空本地数据。','Access revoked. The child server will stop and delete its downloaded proxy programs, then clear its local data.'));}});revoke.classList.add('fleet-danger');revoke.disabled=!!node.revoked;
    if(node.revoked){const remove=button(tr('删除服务器卡片','Delete server card'),more,()=>deleteFleetNodes([node.id]));remove.classList.add('fleet-danger');}
  }
  function renderTasks(content) {
    const box=block(content,t('batch'),tr('先在“子服务器”选择目标，再选择操作。普通巡检不需要填写 JSON。','Select servers in the Servers tab, then choose an action. Routine checks need no JSON.'));
    const selected=(fleet.nodes||[]).filter(n=>ui.selected.has(n.id)&&!n.revoked);
    el('p',selected.length?tr(`目标：${selected.map(n=>n.name).join('、')}`,`Targets: ${selected.map(n=>n.name).join(', ')}`):tr('还没有选择服务器。','No servers selected.'),box,{className:'fleet-callout'});
    button(tr('返回选择服务器','Choose servers'),box,()=>go('nodes'));
    const operationOptions=[{group:tr('日常检查','Routine checks'),options:[['collect-status',t('collect-status')],['test-egress',t('test-egress')]]},{group:tr('节点维护','Node maintenance'),options:[['core-start',tr('启动 Mihomo','Start Mihomo')],['core-stop',tr('停止 Mihomo','Stop Mihomo')],['core-restart',tr('重启 Mihomo','Restart Mihomo')],['kernel-switch',t('kernel-switch')],['geo-update',t('geo-update')],['agent-upgrade',t('agent-upgrade')],['cert-renew',t('cert-renew')]]},{group:tr('配置下发','Configuration'),options:[['defaults',t('defaults')],['inbound',t('inbound')]]},{group:tr('用户管理','User management'),options:[['quota',t('quota')]]}];
    const form=el('form',undefined,box),kind=pick(form,tr('要执行的操作','Operation'),operationOptions);
    const fields=el('div',undefined,form,{className:'fleet-form-grid'});let inputs={};
    const payloadHint=el('p','',form,{className:'fleet-hint'});
    const configure=()=>{fields.replaceChildren();inputs={};payloadHint.textContent='';
      if(kind.value==='test-egress'){inputs.proxy=field(fields,tr('子服务器本地 SOCKS5 代理','SOCKS5 proxy on child server'),'socks5://127.0.0.1:1080','url');inputs.url=field(fields,tr('测试地址（可选）','Test URL (optional)'),'','url');payloadHint.textContent=tr('填写子服务器上实际可用的 SOCKS5 入口地址。','Use an available SOCKS5 listener on the child server.');}
      if(kind.value==='kernel-switch')inputs.version=field(fields,tr('Mihomo 版本','Mihomo version'),'latest');
      if(kind.value==='agent-upgrade'){inputs.url=field(fields,tr('已验证的更新包 HTTPS 地址','Verified update HTTPS URL'),'','url');inputs.sha256=field(fields,'SHA256');}
      if(kind.value==='quota'){inputs.username=field(fields,t('username'));inputs.total=field(fields,tr('总流量（GB，0 为不限）','Total traffic (GB, 0 = unlimited)'),'0','number');inputs.total.min='0';inputs.total.step='any';inputs.expiryTime=field(fields,tr('到期时间（留空为不限）','Expiry (empty = unlimited)'),'','datetime-local');}
      if(kind.value.startsWith('core-'))payloadHint.textContent=tr('控制选中子服务器上的 Mihomo 进程。','Control the Mihomo process on the selected child servers.');
      if(kind.value==='cert-renew')payloadHint.textContent=tr('对该节点的本地托管证书执行续期；日常使用请在“证书管理”中选择证书。','Renew the selected child server local certificate. Use Certificates to choose a specific certificate.');
      if(kind.value==='defaults')payloadHint.textContent=tr('会将主机全局默认的出口、路由和核心基础设置覆盖到选中节点。','Replaces selected nodes’ outbounds, routing and core basics with host global defaults.');
      if(kind.value==='inbound')inputs.inbound=codeEditor(fields,tr('入站对象（高级操作）','Inbound object (advanced)'),{});
    };kind.onchange=configure;configure();
    const advanced=details(form,tr('高级：覆盖任务参数 JSON','Advanced: override task parameters with JSON')),override=field(advanced,tr('完整参数（留空使用上方表单）','Complete parameters (empty uses the form)'),'','textarea');override.spellcheck=false;
    const submit=el('button',t('execute'),form,{type:'submit',className:'primary-btn',disabled:!selected.length});
    form.onsubmit=async e=>{e.preventDefault();if(!form.reportValidity()||!selected.length)return;submit.disabled=true;
      try{let p={};for(const [key,input] of Object.entries(inputs)){if(key==='inbound')p[key]=JSON.parse(input.value);else if(key==='total')p[key]=Math.round(Number(input.value)*1024**3);else if(key==='expiryTime')p[key]=input.value?new Date(input.value).getTime():0;else if(input.value)p[key]=input.value.trim();}
        if(override.value.trim())p=JSON.parse(override.value);const coreActionName=kind.value.startsWith('core-')?kind.value.slice(5):'';const configAction=['defaults','quota','inbound'].includes(kind.value);const capability=coreActionName?'core-control':kind.value;
        if(!p||typeof p!=='object'||Array.isArray(p))throw new Error(tr('任务参数必须是一个对象。','Task parameters must be an object.'));
        const required={ 'test-egress':['proxy'], 'kernel-switch':['version'], 'agent-upgrade':['url','sha256'], quota:['username'] }[kind.value]||[];
        if(required.some(key=>!String(p[key]??'').trim()))throw new Error(tr('请填写此操作需要的所有参数。','Complete the required parameters for this operation.'));
        if(kind.value==='agent-upgrade'&&(!/^https:\/\//i.test(p.url)||! /^[a-f\d]{64}$/i.test(p.sha256)))throw new Error(tr('更新包必须使用 HTTPS 地址，并提供 64 位 SHA256。','Use an HTTPS update URL and a 64-character SHA256.'));
        if(!configAction){const unsupported=selected.filter(n=>!n.status?.capabilities?.includes(capability));if(unsupported.length)throw new Error(tr('以下节点暂不支持此操作，请先接入或升级：','These nodes need to connect or upgrade first: ')+unsupported.map(n=>n.name).join(', '));}
        if(coreActionName)p={action:coreActionName};
        if(!['collect-status','test-egress'].includes(kind.value)&&!await upConfirm(tr(`将在 ${selected.length} 台服务器上执行“${kind.options[kind.selectedIndex]?.textContent||kind.value}”，是否继续？`,`Run “${kind.options[kind.selectedIndex]?.textContent||kind.value}” on ${selected.length} servers?`),{danger:true}))return;
        await request('/batch',configAction?{...p,nodes:selected.map(n=>n.id),action:kind.value}:{nodes:selected.map(n=>n.id),action:'task',kind:capability,payload:p});
        await load();output(tr('任务已提交。下方会自动更新执行状态。','Tasks submitted. Status updates automatically below.'));
      }catch(e){output(e.message,true);}finally{submit.disabled=false;}
    };
    const records=block(content,tr('最近任务','Recent tasks'),tr('每 15 秒更新；展开单条任务可查看具体结果。','Updates every 15 seconds. Expand a task to inspect its result.'));el('div',undefined,records,{id:'fleet-tasks'});updateTasks();
  }
  function updateTasks() {
    const list=root.querySelector('#fleet-tasks');if(!list)return;const opened=new Set(Array.from(list.querySelectorAll('details[open]')).map(d=>d.dataset.taskId));list.replaceChildren();
    const tasks=(fleet.tasks||[]).slice(-30).reverse();if(!tasks.length){empty(list,tr('还没有执行过任务','No tasks yet'),tr('提交巡检或批量操作后，会在这里显示进度。','Submitted checks and batch actions appear here.'));return;}
    const taskNames={'apply-state':tr('应用节点配置','Apply node configuration'),'purge':tr('停止服务并吊销节点','Stop services and revoke node'),'core-control':tr('控制代理内核','Control proxy core'),'traffic-reset':tr('重置流量','Reset traffic'),'vpngate-install':tr('安装 VPNGate 出口','Install VPNGate outbound'),'vpngate-reconnect':tr('重连 VPNGate','Reconnect VPNGate')};
    taskNames['cert-issue']=tr('在子服务器申请证书','Issue certificate on child server');taskNames['cert-renew']=tr('续期子服务器证书','Renew child server certificate');taskNames['cert-bind']=tr('绑定子服务器证书','Bind child server certificate');taskNames['cert-credentials']=tr('加密保存 Cloudflare 凭据','Save encrypted Cloudflare credentials');
    taskNames['outbound-delay']=tr('测试出口延迟','Test outbound delay');
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
  function renderCertificates(content) {
    const nodes=(fleet.nodes||[]).filter(node=>!node.revoked),box=block(content,tr('子服务器证书','Child server certificates'),tr('证书、Cloudflare 凭据和自动续期配置只保存在所选子服务器。申请证书不需要先创建 TLS 入站；签发后可再绑定，也可直接在入站编辑页选择托管证书。Cloudflare 凭据在浏览器中先加密，再以密文发送。','Certificates, Cloudflare credentials, and renewal settings stay on the selected child server. You can issue a certificate before creating a TLS inbound, then bind it or select it directly in the inbound editor. Cloudflare credentials are encrypted in the browser before they are sent.'));
    if(!nodes.length){empty(box,tr('暂无可用子服务器','No child servers available'),tr('添加并接入子服务器后，可在这里单独申请和续期证书。','Add and connect a child server to manage its certificates here.'));return;}
    const workbench=el('div',undefined,box,{className:'fleet-certificate-workbench'});
    const toolbar=el('section',undefined,workbench,{className:'fleet-certificate-toolbar'});
    const nodeOptions=[['',tr('选择子服务器…','Select a child server…')],...nodes.map(node=>[node.id,node.name])];
    const server=pick(toolbar,tr('子服务器','Child server'),nodeOptions,nodes[0].id);
    server.closest('.fleet-field')?.classList.add('fleet-certificate-server');
    const serverState=el('div',undefined,toolbar,{className:'fleet-certificate-server-state'});
    el('span',tr('当前状态','Current status'),serverState);
    const serverStateValue=el('strong',tr('正在读取…','Loading…'),serverState);
    const refresh=button(tr('刷新证书','Refresh certificates'),toolbar,async()=>{await refreshNode();});refresh.classList.add('fleet-certificate-refresh');
    const cardGrid=el('div',undefined,workbench,{className:'fleet-certificate-card-grid'});
    const issueCard=el('section',undefined,cardGrid,{className:'fleet-certificate-panel fleet-certificate-panel-issue'});
    const dnsCard=el('section',undefined,cardGrid,{className:'fleet-certificate-panel fleet-certificate-panel-dns'});
    const bindingCard=el('section',undefined,cardGrid,{className:'fleet-certificate-panel fleet-certificate-panel-binding'});
    function panelHead(parent,title,hint){
      const head=el('header',undefined,parent,{className:'fleet-certificate-panel-head'});
      el('h4',title,head);if(hint)el('p',hint,head,{className:'fleet-hint'});return head;
    }
    panelHead(issueCard,tr('申请证书','Issue a certificate'),tr('证书任务在所选子服务器本地执行，不需要预先创建 TLS 入站。','The certificate task runs on the selected child server and does not require a TLS inbound first.'));
    const issueFields=el('div',undefined,issueCard,{className:'fleet-certificate-form-grid'});
    const name=field(issueFields,tr('证书名称','Certificate name'));
    const domain=field(issueFields,tr('证书域名','Certificate domain'));
    const dns=pick(issueFields,tr('验证方式','Challenge'),[['',tr('HTTP 验证（需开放 80 端口）','HTTP challenge (port 80 required)')],['dns_cf',tr('Cloudflare DNS','Cloudflare DNS')]],'');
    const target=pick(issueFields,tr('签发后绑定到 TLS 入站（可选）','Bind to a TLS inbound after issue (optional)'),[['',tr('稍后绑定','Bind later')]],'');
    const autoWrap=el('label',undefined,issueFields,{className:'fleet-check fleet-certificate-auto fleet-certificate-form-wide'}),automatic=el('input',undefined,autoWrap,{type:'checkbox',checked:true});el('span',tr('自动续期','Renew automatically'),autoWrap);
    const issueActions=el('div',undefined,issueCard,{className:'fleet-certificate-panel-actions'});

    panelHead(dnsCard,tr('Cloudflare 凭据','Cloudflare credentials'),tr('选择 Cloudflare DNS 验证后，可先加密保存 Token、账户 ID 和 Zone ID，再申请或续期证书。','When Cloudflare DNS is selected, securely save the token, account ID, and zone ID before issuing or renewing certificates.'));
    const credentialState=el('p',tr('尚未保存 Cloudflare 凭据','Cloudflare credentials are not saved'),dnsCard,{className:'fleet-certificate-credential-state'});
    const credentialGuide=el('p',tr('当前使用 HTTP 验证，不需要 Cloudflare 凭据。','HTTP validation is selected; Cloudflare credentials are not required.'),dnsCard,{className:'fleet-hint'});
    const dnsFields=el('div',undefined,dnsCard,{className:'fleet-certificate-form-grid'});
    const dnsToken=field(dnsFields,tr('Cloudflare API Token','Cloudflare API token'),'','password',tr('仅加密下发并保存在所选子服务器。','Encrypted for delivery and stored on the selected child server.'));dnsToken.autocomplete='new-password';
    const dnsAccount=field(dnsFields,tr('Cloudflare Account ID','Cloudflare account ID'),'','text',tr('可留空；填写后会随 Token 一起加密保存。','Optional; saved encrypted with the token.'));
    const dnsZone=field(dnsFields,tr('Cloudflare Zone ID','Cloudflare zone ID'),'','text',tr('可留空；填写后会随 Token 一起加密保存。','Optional; saved encrypted with the token.'));
    const credentialActions=el('div',undefined,dnsCard,{className:'fleet-certificate-panel-actions'});

    panelHead(bindingCard,tr('绑定已有证书','Bind an existing certificate'),tr('把已签发的证书绑定到该子服务器的 TLS 入站。新建入站时也可以直接在“Managed Certificate”中选择。','Bind an issued certificate to a TLS inbound on this child server. The inbound editor also exposes the same certificates under Managed Certificate.'));
    const bindFields=el('div',undefined,bindingCard,{className:'fleet-certificate-form-grid'});
    const certificateChoice=pick(bindFields,tr('已签发证书','Issued certificate'),[['',tr('选择证书…','Select a certificate…')]],'');
    const bindTarget=pick(bindFields,tr('TLS 入站','TLS inbound'),[['',tr('选择 TLS 入站…','Select a TLS inbound…')]],'');
    const bindingActions=el('div',undefined,bindingCard,{className:'fleet-certificate-panel-actions'});
    const bindButton=button(tr('绑定到所选 TLS 入站','Bind to selected TLS inbound'),bindingActions,async()=>{
      if(!server.value||!certificateChoice.value||!bindTarget.value)throw new Error(tr('请选择证书、子服务器和 TLS 入站。','Select a certificate, child server, and TLS inbound.'));
      const result=await request('/nodes/'+encodeURIComponent(server.value)+'/certificates/bind',{name:certificateChoice.value,inboundId:bindTarget.value});output(result);if(result.publishError)output(tr('绑定任务已排队，但配置发布失败：','The bind task was queued, but publishing the configuration failed: ')+result.publishError,true);
    },true);
    const targetHint=el('p','',bindingCard,{className:'fleet-hint'});

    const listSection=el('section',undefined,workbench,{className:'fleet-certificate-list-section'});
    const listHead=el('header',undefined,listSection,{className:'fleet-certificate-list-head'});
    const listTitle=el('div',undefined,listHead);
    el('h4',tr('托管证书','Managed certificates'),listTitle);
    const certificateCount=el('span',tr('正在读取…','Loading…'),listTitle,{className:'fleet-certificate-count'});
    el('p',tr('卡片中的“用于绑定”会把证书带入上方绑定区；如有可用 TLS 入站，可立即完成绑定。','Use for binding loads a certificate into the binding area above so an available TLS inbound can be selected immediately.'),listHead,{className:'fleet-hint'});
    const feedback=el('div',undefined,listSection,{className:'fleet-certificate-list'});
    let targetItems=[],generation=0,currentNode=null,saveCredentials;
    function base64Bytes(value){return Uint8Array.from(atob(value),character=>character.charCodeAt(0));}
    function base64String(value){let binary='';for(const byte of value)binary+=String.fromCharCode(byte);return btoa(binary);}
    function newTaskID(){return Array.from(crypto.getRandomValues(new Uint8Array(24))).map(value=>value.toString(16).padStart(2,'0')).join('');}
    function formatCertificateDate(value){if(!value)return{date:'—',remaining:'',tone:'muted'};const date=new Date(value);if(Number.isNaN(date.getTime()))return{date:String(value),remaining:'',tone:'muted'};const days=Math.ceil((date.getTime()-Date.now())/86400000),tone=days<0?'danger':days<=14?'warning':'valid',remaining=days<0?tr('已过期','Expired'):days===0?tr('今天到期','Expires today'):tr('剩余 '+days+' 天','in '+days+' days');return{date:date.toLocaleDateString(),remaining,tone};}
    async function encryptCredentials(nodeID,publicKey,taskID,credentials){
      if(!globalThis.crypto?.subtle)throw new Error(tr('当前浏览器不支持安全凭据加密。','This browser does not support secure credential encryption.'));
      try{
        const subtle=globalThis.crypto.subtle,encoder=new TextEncoder(),ephemeral=await subtle.generateKey({name:'X25519'},true,['deriveBits']),recipient=await subtle.importKey('raw',base64Bytes(publicKey),{name:'X25519'},false,[]),shared=await subtle.deriveBits({name:'X25519',public:recipient},ephemeral.privateKey,256),salt=await subtle.digest('SHA-256',encoder.encode('UnifiedProxy-fleet-secret-salt-v1\n'+nodeID)),hkdf=await subtle.importKey('raw',shared,'HKDF',false,['deriveKey']),key=await subtle.deriveKey({name:'HKDF',hash:'SHA-256',salt,info:encoder.encode('UnifiedProxy-fleet-secret-key-v1\n'+taskID+'\ncert-credentials')},hkdf,{name:'AES-GCM',length:256},false,['encrypt']),nonce=crypto.getRandomValues(new Uint8Array(12)),aad=encoder.encode('UnifiedProxy-fleet-secret-v1\n'+nodeID+'\n'+taskID+'\ncert-credentials'),ciphertext=await subtle.encrypt({name:'AES-GCM',iv:nonce,additionalData:aad,tagLength:128},key,encoder.encode(JSON.stringify(credentials))),ephemeralPublic=await subtle.exportKey('raw',ephemeral.publicKey);
        return {version:1,ephemeralPublicKey:base64String(new Uint8Array(ephemeralPublic)),nonce:base64String(nonce),ciphertext:base64String(new Uint8Array(ciphertext))};
      }catch(error){throw new Error(tr('浏览器无法使用 X25519 加密凭据，请更新浏览器后重试。','The browser could not encrypt credentials with X25519. Update the browser and try again.'));}
    }
    function fillTargetOptions(select,placeholder){
      select.replaceChildren();el('option',placeholder,select,{value:''});
      for(const item of targetItems)el('option',(item.name||item.type||item.id)+(item.port?' · '+item.port:'')+' · '+item.id,select,{value:item.id});
    }
    function syncBindButton(){const caps=currentNode?.status?.capabilities||[];bindButton.disabled=!(caps.includes('cert-bind')&&targetItems.length>0&&bindTarget.value&&certificateChoice.value);}
    function syncChallengeCard(){const cloudflare=dns.value==='dns_cf';dnsFields.hidden=!cloudflare;credentialGuide.hidden=cloudflare;dnsCard.classList.toggle('is-idle',!cloudflare);}
    function renderCertificateCard(cert){
      const card=el('article',undefined,feedback,{className:'fleet-certificate-card'});card.dataset.certificateName=cert.name;
      const head=el('header',undefined,card,{className:'fleet-certificate-card-head'}),identity=el('div',undefined,head,{className:'fleet-certificate-card-identity'});
      el('strong',cert.name,identity);el('code',cert.domain||tr('未指定域名','No domain'),identity);
      const badge=el('span',cert.automatic?tr('自动续期','Auto renew'):tr('手动续期','Manual'),head,{className:'fleet-certificate-badge'});badge.dataset.state=cert.lastError?'error':cert.automatic?'auto':'manual';
      const expiry=formatCertificateDate(cert.expiresAt),facts=el('dl',undefined,card,{className:'fleet-certificate-facts'});
      function fact(label,value,state=''){const item=el('div',undefined,facts,{className:'fleet-certificate-fact'});el('dt',label,item);const dd=el('dd',value,item);if(state)dd.dataset.state=state;}
      fact(tr('到期日期','Expires'),expiry.date,expiry.tone);
      fact(tr('有效期','Validity'),expiry.remaining||'—',expiry.tone);
      fact(tr('已绑定','Bound'),String(cert.bindings?.length||0));
      fact(tr('域名','Domain'),cert.domain||'—');
      if(cert.lastError)el('p',tr('续期失败：','Renewal failed: ')+cert.lastError,card,{className:'fleet-error fleet-certificate-card-error'});
      const actions=el('div',undefined,card,{className:'fleet-certificate-actions fleet-certificate-card-actions'});
      const renew=button(tr('立即续期','Renew now'),actions,async()=>{const task=await request('/tasks',{nodeId:server.value,kind:'cert-renew',payload:{name:cert.name},ttlSeconds:1200});output(task);});renew.disabled=!cert.domain;if(!cert.domain)renew.title=tr('该证书未配置域名，无法续期','Renewal is unavailable because this certificate has no domain');
      button(tr('用于绑定','Use for binding'),actions,()=>{certificateChoice.value=cert.name;syncBindButton();bindingCard.scrollIntoView({block:'start',behavior:'smooth'});setTimeout(()=>bindTarget.focus(),0);});
    }
    async function refreshNode(){
      const version=++generation;currentNode=null;targetItems=[];serverState.dataset.state='loading';serverStateValue.textContent=tr('正在读取…','Loading…');certificateCount.textContent=tr('正在读取…','Loading…');certificateChoice.replaceChildren();el('option',tr('选择证书…','Select a certificate…'),certificateChoice,{value:''});feedback.replaceChildren();targetHint.hidden=true;fillTargetOptions(target,tr('正在读取…','Loading…'));fillTargetOptions(bindTarget,tr('正在读取…','Loading…'));syncBindButton();
      if(!server.value){serverState.dataset.state='pending';serverStateValue.textContent=tr('未选择子服务器','No child server selected');fillTargetOptions(target,tr('先选择子服务器','Select a child server first'));fillTargetOptions(bindTarget,tr('先选择子服务器','Select a child server first'));credentialState.textContent=tr('先选择子服务器','Select a child server first');return;}
      let detail;
      try{detail=await request('/nodes/'+encodeURIComponent(server.value));}catch(error){if(version===generation){serverState.dataset.state='error';serverStateValue.textContent=tr('读取失败','Read failed');fillTargetOptions(target,error.message);fillTargetOptions(bindTarget,error.message);}throw error;}
      if(version!==generation||server.value!==detail.id)return;
      currentNode=detail;
      const model=detail.draft||detail.desired?.model||{};
      targetItems=(model.inbounds||[]).filter(item=>!item.reality?.enabled&&(item.tls||['hysteria2','hysteria2-realm','tuic','trusttunnel'].includes(String(item.type).toLowerCase())));
      fillTargetOptions(target,tr('稍后绑定','Bind later'));fillTargetOptions(bindTarget,tr('选择 TLS 入站…','Select a TLS inbound…'));
      const caps=detail.status?.capabilities||[],encryptionPublicKey=detail.encryptionPublicKey||detail.status?.encryptionPublicKey||'';
      serverState.dataset.state=caps.includes('cert-issue')?'online':'pending';serverStateValue.textContent=caps.includes('cert-issue')?tr('已连接，可管理证书','Connected, ready'):tr('等待重新连接或升级','Awaiting reconnect or upgrade');
      submit.disabled=!caps.includes('cert-issue');
      if(!caps.includes('cert-issue'))el('p',tr('请先升级此子服务器，再使用本地证书管理。','Upgrade this child server to enable local certificate management.'),feedback,{className:'fleet-hint'});
      const canSaveCredentials=caps.includes('cert-credentials')&&!!encryptionPublicKey;
      saveCredentials.disabled=!canSaveCredentials;
      credentialState.textContent=detail.status?.details?.cloudflareCredentialsConfigured?tr('Cloudflare 凭据已保存在此子服务器。','Cloudflare credentials are saved on this child server.'):canSaveCredentials?tr('尚未保存 Cloudflare 凭据','Cloudflare credentials are not saved'):tr('升级子服务器后即可安全保存 Cloudflare 凭据。','Upgrade the child server to securely save Cloudflare credentials.');
      const certs=detail.status?.details?.managedCertificates||[];certificateCount.textContent=certs.length+' '+tr('张证书','certificates');
      for(const cert of certs){el('option',cert.name+' · '+cert.domain,certificateChoice,{value:cert.name});renderCertificateCard(cert);}
      if(!certs.length)empty(feedback,tr('此子服务器尚无托管证书','No managed certificates on this server'),tr('可以先不选择 TLS 入站，独立申请证书。','Issue a certificate without selecting a TLS inbound first.'));
      targetHint.hidden=targetItems.length>0;targetHint.textContent=tr('此子服务器还没有 TLS 入站。创建安全为 TLS 的入站后，证书会出现在入站编辑页的“Managed Certificate”中。','This server has no TLS inbound yet. After creating one with TLS security, the certificate appears under Managed Certificate in the inbound editor.');
      syncBindButton();
    }
    certificateChoice.onchange=syncBindButton;
    bindTarget.onchange=syncBindButton;
    target.onchange=()=>{const inbound=targetItems.find(item=>item.id===target.value);if(inbound){if(inbound.tlsServerName)domain.value=inbound.tlsServerName;if(!name.value&&domain.value)name.value=domain.value.toLowerCase().replace(/^\*\./,'wildcard-').replace(/[^a-z0-9._-]/g,'-');}};
    const submit=button(tr('在此子服务器申请证书','Issue certificate on this server'),issueActions,async()=>{
      if(!server.value)throw new Error(tr('请选择子服务器。','Select a child server.'));
      if(!name.value.trim()||!domain.value.trim())throw new Error(tr('请填写证书名称和域名。','Enter a certificate name and domain.'));
      const payload={name:name.value.trim(),domain:domain.value.trim(),dns:dns.value,automatic:automatic.checked};if(target.value)payload.inboundId=target.value;
      const result=await request('/nodes/'+encodeURIComponent(server.value)+'/certificates/issue',payload);
      output(result);if(result.publishError)output(tr('证书任务已排队，但节点配置发布失败：','Certificate task was queued, but publishing the node configuration failed: ')+result.publishError,true);
    },true);
    dns.onchange=syncChallengeCard;syncChallengeCard();
    saveCredentials=button(tr('加密保存到此子服务器','Encrypt and save on this server'),credentialActions,async()=>{
      if(!server.value||!currentNode)throw new Error(tr('请选择在线子服务器。','Select an available child server.'));
      const publicKey=currentNode.encryptionPublicKey||currentNode.status?.encryptionPublicKey||'';
      if(!publicKey||!currentNode.status?.capabilities?.includes('cert-credentials'))throw new Error(tr('此子服务器尚未就绪，请先升级并等待重新连接。','This server is not ready. Upgrade it and wait for it to reconnect.'));
      if(!dnsToken.value.trim())throw new Error(tr('请输入 Cloudflare API Token。','Enter a Cloudflare API token.'));
      const taskId=newTaskID(),credentials={cloudflareToken:dnsToken.value.trim(),cloudflareAccountId:dnsAccount.value.trim(),cloudflareZoneId:dnsZone.value.trim()},encrypted=await encryptCredentials(server.value,publicKey,taskId,credentials),result=await request('/nodes/'+encodeURIComponent(server.value)+'/certificates/dns-credentials',{taskId:taskId,payload:encrypted});
      dnsToken.value='';dnsAccount.value='';dnsZone.value='';output(result);credentialState.textContent=tr('凭据加密任务已发送；完成后会保存在子服务器本地。','Encrypted credentials task sent; they will be saved on the child server when it completes.');
    });
    server.onchange=()=>{dnsToken.value='';dnsAccount.value='';dnsZone.value='';submit.disabled=!fleet.nodes.find(node=>node.id===server.value)?.status?.capabilities?.includes('cert-issue');refreshNode().catch(error=>output(error.message,true));};
    submit.disabled=!fleet.nodes.find(node=>node.id===server.value)?.status?.capabilities?.includes('cert-issue');
    refreshNode().catch(error=>output(error.message,true));
  }

  function renderMaintenance(content, certificatesOnly=false) {
    if(!certificatesOnly){
      const identity=block(content,tr('主机身份','Host identity'),tr('子服务器使用公钥确认主机身份；请通过可信渠道核对。日常使用不需要修改它。','Child servers use this public key to verify the host. Confirm it over a trusted channel.'));
      el('code',fleet.publicKey,identity,{className:'fleet-command'});button(tr('复制公钥','Copy public key'),identity,async()=>{await copyText(fleet.publicKey);output(tr('主机公钥已复制','Host public key copied'));});
    }
    const cert=block(content,tr('控制端托管证书','Control host certificates'),tr('此处保留控制端托管的兼容证书。子服务器请使用左侧“证书管理”，证书会在各节点本地申请和续期。','This section retains control-host certificates for compatibility. Use Certificates for child servers; those certificates are issued and renewed locally.'));
    const certificateList=el('div',undefined,cert,{className:'fleet-certificate-list'}),certificateNames=el('datalist',undefined,cert,{id:'fleet-certificate-names'});
    const listRefresh=button(tr('刷新证书列表','Refresh certificates'),cert,loadCertificates);
    const operation=pick(cert,tr('操作','Operation'),[['issue',tr('申请新证书','Issue a certificate')],['renew',tr('续期证书','Renew a certificate')],['import',tr('导入证书','Import a certificate')],['bind',tr('绑定到节点','Bind to a server')]],'issue');
    const name=field(cert,tr('证书名称','Certificate name'));name.required=true;name.setAttribute('list','fleet-certificate-names');
    const fields=el('div',undefined,cert,{className:'fleet-two-columns'});
    const domain=field(fields,tr('域名','Domain'));
    const dns=pick(fields,tr('DNS 验证','DNS challenge'),[['',tr('HTTP 验证（需要开放 80 端口）','HTTP challenge (port 80 must be open)')],['dns_cf','Cloudflare DNS']],'');
    const dnsToken=field(fields,tr('Cloudflare API Token'),'','password',tr('需要 DNS 编辑权限。仅用于签发；acme.sh 会保存在控制端用于续期。','Requires DNS edit permission. Used for issuance; acme.sh stores it on the control host for renewals.'));dnsToken.autocomplete='new-password';
    const automaticWrap=el('label',undefined,fields,{className:'fleet-check fleet-certificate-auto'}),automatic=el('input',undefined,automaticWrap,{type:'checkbox',checked:true});el('span',tr('自动续期','Renew automatically'),automaticWrap);
    const material=el('div',undefined,cert,{className:'fleet-two-columns'});
    const certificate=field(material,tr('证书 PEM','Certificate PEM'),'','textarea'),privateKey=field(material,tr('私钥 PEM','Private key PEM'),'','textarea');certificate.spellcheck=false;privateKey.spellcheck=false;
    const bindingEditor=el('div',undefined,cert,{className:'fleet-certificate-bindings'}),bindingRows=[];
    const nodeOptions=[['',tr('选择节点…','Select a server…')],...(fleet.nodes||[]).filter(node=>!node.revoked).map(node=>[node.id,node.name])];
    const nodeDetailCache=new Map();
    async function nodeDetail(id){if(!nodeDetailCache.has(id))nodeDetailCache.set(id,request('/nodes/'+encodeURIComponent(id)));try{return await nodeDetailCache.get(id);}catch(error){nodeDetailCache.delete(id);throw error;}}
    function addBindingRow(binding={}){
      const row=el('div',undefined,bindingEditor,{className:'fleet-certificate-binding'}),node=pick(row,tr('节点','Server'),nodeOptions,''),kind=pick(row,tr('目标类型','Target type'),[['inbound',tr('入站','Inbound')],['sidecar',tr('Naive sidecar','Naive sidecar')]],'inbound'),target=pick(row,tr('目标','Target'),[['',tr('先选择节点','Select a server first')]],'');
      node.value=binding.nodeId||'';kind.value=binding.sidecarId?'sidecar':'inbound';const targetID=binding.sidecarId||binding.inboundId||'';
      const entry={row,node,kind,target,generation:0};bindingRows.push(entry);
      const remove=button(tr('移除此绑定','Remove binding'),row,()=>{const index=bindingRows.indexOf(entry);if(index>=0)bindingRows.splice(index,1);row.remove();});remove.classList.add('fleet-certificate-remove');
      const refreshTargets=async()=>{const generation=++entry.generation;target.replaceChildren();el('option',node.value?tr('正在读取…','Loading…'):tr('先选择节点','Select a server first'),target,{value:''});if(!node.value)return;const selectedNode=node.value;try{const detail=await nodeDetail(selectedNode);if(generation!==entry.generation||selectedNode!==node.value)return;const model=detail.draft||detail.desired?.model||{},items=kind.value==='sidecar'?(detail.desired?.sidecars||[]).filter(item=>String(item.kind).toLowerCase()==='naive'):(model.inbounds||[]).filter(item=>!item.reality?.enabled&&(item.tls||['hysteria2','hysteria2-realm','tuic','trusttunnel'].includes(String(item.type).toLowerCase())));target.replaceChildren();el('option',tr('选择目标…','Select a target…'),target,{value:''});for(const item of items)el('option',`${item.name||item.kind||item.id}${item.port?` · ${item.port}`:''} · ${item.id}`,target,{value:item.id});if(!items.length)el('option',tr('此节点没有可绑定的 TLS 目标','No TLS targets on this server'),target,{value:''});}catch(error){if(generation===entry.generation){target.replaceChildren();el('option',error.message,target,{value:''});}}};
      node.onchange=refreshTargets;kind.onchange=refreshTargets;
      if(node.value)refreshTargets().then(()=>{if(targetID)target.value=targetID;});
    }
    button(tr('添加绑定','Add binding'),bindingEditor,addBindingRow);
    const submit=button(t('issue'),cert,async()=>{
      if(!name.value.trim())throw new Error(tr('请填写证书名称。','Enter a certificate name.'));
      const action=operation.value,payload={name:name.value.trim()};
      if(action==='issue'){if(!domain.value.trim())throw new Error(tr('申请证书时必须填写域名。','Enter a domain to issue a certificate.'));payload.domain=domain.value.trim();payload.dns=dns.value;payload.automatic=automatic.checked;if(dns.value==='dns_cf'){if(!dnsToken.value.trim())throw new Error(tr('请输入 Cloudflare API Token。','Enter a Cloudflare API token.'));payload.dnsCredentials={cloudflareToken:dnsToken.value.trim()};dnsToken.value='';}if(bindingRows.length)payload.bindings=bindingRows.map(({node,kind,target})=>{if(!node.value||!target.value)throw new Error(tr('每个绑定都需要选择节点和目标。','Choose a server and target for each binding.'));return kind.value==='sidecar'?{nodeId:node.value,sidecarId:target.value}:{nodeId:node.value,inboundId:target.value};});}
      if(action==='import'){payload.domain=domain.value.trim();payload.certificate=certificate.value;payload.privateKey=privateKey.value;payload.automatic=automatic.checked;if(!payload.certificate.trim()||!payload.privateKey.trim())throw new Error(tr('请填写证书和私钥 PEM。','Enter both certificate and private key PEM.'));if(payload.automatic&&!payload.domain)throw new Error(tr('自动续期需要填写域名。','A domain is required for automatic renewal.'));}
      if(action==='bind'){payload.bindings=bindingRows.map(({node,kind,target})=>{if(!node.value||!target.value)throw new Error(tr('每个绑定都需要选择节点和目标。','Choose a server and target for each binding.'));return kind.value==='sidecar'?{nodeId:node.value,sidecarId:target.value}:{nodeId:node.value,inboundId:target.value};});}
      const result=await request('/certificates/'+action,payload);output(result);await loadCertificates();
    });
    function syncCertificateForm(){
      const action=operation.value,isIssue=action==='issue',isImport=action==='import',isBind=action==='bind';
      domain.parentElement.hidden=action==='renew'||isBind;domain.required=isIssue;dns.parentElement.hidden=!isIssue;dnsToken.parentElement.hidden=!isIssue||dns.value!=='dns_cf';dnsToken.required=isIssue&&dns.value==='dns_cf';automaticWrap.hidden=action==='renew'||isBind;material.hidden=!isImport;bindingEditor.hidden=!(isIssue||isBind);
      certificate.required=isImport;privateKey.required=isImport;submit.textContent=t(({issue:'issue',renew:'renew',import:'importCert',bind:'bind'})[action]);
    }
    operation.onchange=()=>{if(operation.value==='import')automatic.checked=false;else if(operation.value==='issue')automatic.checked=true;syncCertificateForm();};dns.onchange=syncCertificateForm;syncCertificateForm();
    async function loadCertificates(){
      const items=await request('/certificates');certificateList.replaceChildren();certificateNames.replaceChildren();
      if(!items.length)empty(certificateList,tr('暂无托管证书','No managed certificates'),tr('申请或导入证书后，可在这里续期或管理绑定。','Issue or import a certificate to renew it or manage its bindings here.'));
      for(const item of items){
        el('option',undefined,certificateNames,{value:item.name});
        const row=el('div',undefined,certificateList,{className:'fleet-certificate-item'}),meta=el('div',undefined,row,{className:'fleet-certificate-meta'});
        el('strong',item.name,meta);el('span',`${item.domain||tr('未指定域名','No domain')} · ${tr('到期','Expires')} ${item.expiresAt?new Date(item.expiresAt).toLocaleDateString():'—'} · ${item.bindings?.length||0} ${tr('个绑定','bindings')}`,meta);if(item.automatic)el('span',tr('自动续期已开启','Automatic renewal enabled'),meta);if(item.lastError)el('span',tr('续期失败：','Renewal failed: ')+item.lastError,meta,{className:'fleet-error'});
        const actions=el('div',undefined,row,{className:'fleet-certificate-actions'});
        const focusForm=()=>{cert.scrollIntoView({block:'start',behavior:'smooth'});setTimeout(()=>{operation.focus();},0);};
        const renew=button(tr('续期','Renew'),actions,()=>{operation.value='renew';name.value=item.name;syncCertificateForm();focusForm();});renew.disabled=!item.domain;if(!item.domain)renew.title=tr('该证书未配置域名，无法续期','Renewal is unavailable because this certificate has no domain');
        button(tr('绑定','Bind'),actions,()=>{operation.value='bind';name.value=item.name;bindingRows.splice(0,bindingRows.length);bindingEditor.querySelectorAll('.fleet-certificate-binding').forEach(entry=>entry.remove());for(const binding of item.bindings||[])addBindingRow(binding);syncCertificateForm();focusForm();});
      }
      listRefresh.disabled=false;
    }
    loadCertificates().catch(error=>output(error.message,true));
    if(certificatesOnly)return;
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
    const shape=nodes=>JSON.stringify(nodes.map(n=>[n.id,n.name,n.address,n.configuredAddress,n.seenIP,n.addressSource,n.revoked,n.needsUpgrade]));
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
