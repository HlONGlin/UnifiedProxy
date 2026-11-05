/* A server workspace has a permanent URL; it shares only control authentication. */
window.UPServerWorkspace=function(){
  const root=document.documentElement,nodeID=root.dataset.nodeId,base=panelBase();
  const controlBase=nodeID?base.replace(/nodes\/[^/]+\/$/,''):base.replace(/host\/$/,'');
  const tr=(zh,en)=>(window.upGetLang?.()||'zh-CN').startsWith('zh')?zh:en;
  const fleetButton=document.querySelector('#fleet-nav');
  async function leave(event){
    if(event?.ctrlKey||event?.metaKey||event?.shiftKey||event?.altKey)return;
    event?.preventDefault();
    if(window.UPPanel?.isDirty?.()&&!await upConfirm(tr('当前编辑尚未保存，确认返回舰队？','Unsaved edits will be lost. Return to the fleet?'),{danger:true}))return;
    location.href=controlBase;
  }
  fleetButton.removeAttribute('data-view');
  fleetButton.onclick=leave;
  const bar=document.createElement('div');bar.className='server-workspace-bar';
  const back=document.createElement('a');back.className='outline-btn';back.href=controlBase;
  back.onclick=leave;
  const identity=document.createElement('div');identity.className='server-workspace-identity';
  const name=document.createElement('strong'),status=document.createElement('span');
  identity.append(name,status);bar.append(back,identity);
  document.querySelector('.content').prepend(bar);
  if(nodeID){
    document.querySelector('#nav [data-view=settings]').hidden=true;
  }
  function render(){
    back.textContent=tr('← 返回舰队','← Back to fleet');
    fleetButton.innerHTML=SF('multi-control','nav-icon')+'<span>'+tr('返回舰队','Back to fleet')+'</span>';
    const data=typeof app==='object'?app.state:null;
    if(nodeID){
      name.textContent=data?.nodeName||tr('服务器工作台','Server workspace');
      const seen=data?.lastSeen&&!data.lastSeen.startsWith('0001-')?new Date(data.lastSeen).toLocaleString(window.upGetLang?.()||'zh-CN'):tr('尚未连接','Never connected');
      status.textContent=(data?.online?tr('在线','Online'):tr('离线 / 等待心跳','Offline / awaiting heartbeat'))+' · '+(data?.nodeAddress||'')+' · '+tr('最近心跳：','Last heartbeat: ')+seen;
      bar.dataset.online=String(!!data?.online);
      document.title=(data?.nodeName||'UnifiedProxy')+' | UnifiedProxy';
    }else{
      name.textContent=tr('主机设置与全局默认','Host settings and global defaults');
      status.textContent=tr('管理账号、面板与共享默认配置','Manage the login, panel and shared defaults');
    }
    const eyebrow=document.querySelector('#view-dashboard .eyebrow');if(eyebrow)eyebrow.textContent=nodeID?name.textContent:'UnifiedProxy / Control';
    const caption=document.querySelector('#view-dashboard .caption');if(caption)caption.textContent=nodeID?tr('独立服务器工作台','Server workspace'):'UnifiedProxy';
  }
  function restoreView(){
    const view=location.hash.slice(1)||'dashboard';
    if(['dashboard','inbounds','mihomo','logs','connections','config','subtemplates',...(!nodeID?['settings']:[])].includes(view))setView(view);
    if(view==='mihomo')document.querySelector('[data-mihomo-tab=outbounds]')?.click();
  }
  window.addEventListener('hashchange',restoreView);
  window.addEventListener('up-langchange',render);
  document.addEventListener('up-state-rendered',render);
  window.addEventListener('DOMContentLoaded',restoreView,{once:true});
  render();
};
