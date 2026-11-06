/* Fleet controls reuse the panel's authentication, themes and native editors. */
(function () {
  'use strict';
  const role = document.documentElement.dataset.fleetRole;
  if (!role) return;
  const locales = ['en','zh-CN','zh-TW','ja','ru','fa','vi','es','tr','uk','pt-BR'];
  const words = {
    title: ['Fleet','舰队管理','艦隊管理','フリート管理','Управление узлами','مدیریت ناوگان','Quản lý cụm','Gestión de nodos','Filo yönetimi','Керування вузлами','Gerenciar frota'],
    refresh: ['Refresh','刷新','重新整理','更新','Обновить','بازخوانی','Làm mới','Actualizar','Yenile','Оновити','Atualizar'],
    target: ['Editing node','当前编辑节点','目前編輯節點','編集中のノード','Редактируемый узел','گره در حال ویرایش','Nút đang sửa','Nodo en edición','Düzenlenen düğüm','Вузол для редагування','Nó em edição'],
    global: ['Control / global defaults','控制端／全局默认','控制端／全域預設','制御／全体の既定値','Контроллер / общие настройки','کنترل / پیش‌فرض سراسری','Điều khiển / mặc định chung','Control / valores globales','Kontrol / genel varsayılanlar','Контролер / загальні налаштування','Controle / padrões globais'],
    token: ['Create enrollment token (15 min)','生成接入令牌（15 分钟）','產生接入權杖（15 分鐘）','登録トークンを作成（15分）','Создать токен (15 мин)','ایجاد توکن عضویت (۱۵ دقیقه)','Tạo mã tham gia (15 phút)','Crear token (15 min)','Katılım belirteci oluştur (15 dk)','Створити токен (15 хв)','Criar token (15 min)'],
    name: ['Name','名称','名稱','名前','Имя','نام','Tên','Nombre','Ad','Назва','Nome'],
    address: ['Public address','发布地址','發布位址','公開アドレス','Публичный адрес','نشانی عمومی','Địa chỉ công khai','Dirección pública','Genel adres','Публічна адреса','Endereço público'],
    add: ['Register node','登记节点','登記節點','ノードを登録','Зарегистрировать узел','ثبت گره','Đăng ký nút','Registrar nodo','Düğüm kaydet','Зареєструвати вузол','Registrar nó'],
    online: ['Online','在线','線上','オンライン','В сети','آنلاین','Trực tuyến','En línea','Çevrimiçi','У мережі','Online'],
    offline: ['Offline','离线','離線','オフライン','Не в сети','آفلاین','Ngoại tuyến','Desconectado','Çevrimdışı','Не в мережі','Offline'],
    publish: ['Publish','发布','發布','反映','Применить','انتشار','Áp dụng','Publicar','Yayınla','Застосувати','Publicar'],
    revoke: ['Revoke identity','吊销身份','撤銷身分','IDを失効','Отозвать ключ','لغو هویت','Thu hồi danh tính','Revocar identidad','Kimliği iptal et','Відкликати ключ','Revogar identidade'],
    edit: ['Edit','编辑','編輯','編集','Изменить','ویرایش','Sửa','Editar','Düzenle','Редагувати','Editar'],
    model: ['Native UnifiedProxy model (JSON)','UnifiedProxy 原生模型（JSON）','UnifiedProxy 原生模型（JSON）','UnifiedProxy モデル（JSON）','Модель UnifiedProxy (JSON)','مدل بومی UnifiedProxy (JSON)','Mô hình UnifiedProxy (JSON)','Modelo UnifiedProxy (JSON)','UnifiedProxy modeli (JSON)','Модель UnifiedProxy (JSON)','Modelo UnifiedProxy (JSON)'],
    sidecars: ['Sidecars (JSON; unmetered)','Sidecar（JSON；不计量）','Sidecar（JSON；不計量）','Sidecar（JSON・計量なし）','Sidecar (JSON; без учёта)','سرویس جانبی (JSON؛ بدون سنجش)','Sidecar (JSON; không đo lưu lượng)','Sidecars (JSON; sin medición)','Sidecar (JSON; ölçümsüz)','Sidecar (JSON; без обліку)','Sidecars (JSON; sem medição)'],
    tasks: ['Tasks / results','任务／结果','工作／結果','タスク／結果','Задания / результаты','وظایف / نتایج','Tác vụ / kết quả','Tareas / resultados','Görevler / sonuçlar','Завдання / результати','Tarefas / resultados'],
    batch: ['Batch action','批量操作','批次操作','一括操作','Групповая операция','عملیات گروهی','Thao tác hàng loạt','Acción por lotes','Toplu işlem','Групова операція','Ação em lote'],
    execute: ['Run on selected nodes','对选中节点执行','對選取節點執行','選択ノードで実行','Выполнить на выбранных','اجرا روی گره‌های انتخابی','Chạy trên nút đã chọn','Ejecutar en seleccionados','Seçili düğümlerde çalıştır','Виконати на вибраних','Executar nos selecionados'],
    import: ['Import merit.json','导入 merit.json','匯入 merit.json','merit.json をインポート','Импорт merit.json','ورود merit.json','Nhập merit.json','Importar merit.json','merit.json içe aktar','Імпорт merit.json','Importar merit.json'],
    backup: ['Download full fleet backup','下载完整舰队备份','下載完整艦隊備份','全体バックアップを取得','Скачать полную копию','دریافت پشتیبان کامل','Tải bản sao lưu toàn cụm','Descargar copia completa','Tam filo yedeğini indir','Завантажити повну копію','Baixar backup completo'],
    restore: ['Restore backup','恢复备份','還原備份','バックアップを復元','Восстановить копию','بازیابی پشتیبان','Khôi phục sao lưu','Restaurar copia','Yedeği geri yükle','Відновити копію','Restaurar backup'],
    join: ['Join control','接入控制端','接入控制端','コントローラーに参加','Подключить к контроллеру','پیوستن به کنترل','Tham gia bộ điều khiển','Conectar al control','Kontrole katıl','Підключити до контролера','Conectar ao controle'],
    key: ['Trusted control public key','可信控制端公钥','可信控制端公鑰','信頼するコントローラーの公開鍵','Доверенный ключ контроллера','کلید عمومی معتبر کنترل','Khóa công khai tin cậy','Clave pública de confianza','Güvenilen kontrol açık anahtarı','Довірений ключ контролера','Chave pública confiável'],
    emergency: ['Emergency local takeover','紧急本地接管','緊急本機接管','緊急ローカル引継ぎ','Аварийное локальное управление','کنترل اضطراری محلی','Tiếp quản cục bộ khẩn cấp','Control local de emergencia','Acil yerel devralma','Аварійне локальне керування','Controle local de emergência'],
    confirm: ['Confirm this operation?','确认执行此操作？','確認執行此操作？','この操作を実行しますか？','Подтвердить операцию?','این عملیات تأیید شود؟','Xác nhận thao tác?','¿Confirmar operación?','Bu işlem onaylansın mı?','Підтвердити операцію?','Confirmar operação?'],
    output: ['Result','结果','結果','結果','Результат','نتیجه','Kết quả','Resultado','Sonuç','Результат','Resultado'],
    save: ['Save and publish','保存并发布','儲存並發布','保存して反映','Сохранить и применить','ذخیره و انتشار','Lưu và áp dụng','Guardar y publicar','Kaydet ve yayınla','Зберегти й застосувати','Salvar e publicar'],
    upgrade: ['Upgrade needed','需要升级','需要升級','更新が必要','Нужно обновление','نیاز به ارتقا','Cần nâng cấp','Requiere actualización','Güncelleme gerekli','Потрібне оновлення','Atualização necessária'],
    select: ['Select','选择','選取','選択','Выбрать','انتخاب','Chọn','Seleccionar','Seç','Вибрати','Selecionar'],
    rebind: ['Rebind token','重新绑定令牌','重新綁定權杖','再登録トークン','Токен перепривязки','توکن اتصال مجدد','Mã liên kết lại','Token de revinculación','Yeniden bağlama belirteci','Токен переприв’язки','Token de revinculação']
  };
  Object.assign(words, {
    traffic: ['Traffic (bytes)','流量（字节）','流量（位元組）','通信量（バイト）','Трафик (байты)','ترافیک (بایت)','Lưu lượng (byte)','Tráfico (bytes)','Trafik (bayt)','Трафік (байти)','Tráfego (bytes)'],
    running: ['Running','运行中','執行中','稼働中','Работает','در حال اجرا','Đang chạy','En ejecución','Çalışıyor','Працює','Em execução'],
    stopped: ['Stopped','已停止','已停止','停止中','Остановлен','متوقف','Đã dừng','Detenido','Durduruldu','Зупинено','Parado'],
    certificates: ['Certificates','证书管理','憑證管理','証明書','Сертификаты','گواهی‌ها','Chứng chỉ','Certificados','Sertifikalar','Сертифікати','Certificados'],
    issue: ['Issue certificate','申请证书','申請憑證','証明書を発行','Выдать сертификат','صدور گواهی','Cấp chứng chỉ','Emitir certificado','Sertifika al','Видати сертифікат','Emitir certificado'],
    renew: ['Renew certificate','续期证书','續期憑證','証明書を更新','Продлить сертификат','تمدید گواهی','Gia hạn chứng chỉ','Renovar certificado','Sertifikayı yenile','Поновити сертифікат','Renovar certificado'],
    importCert: ['Import PEM pair','导入 PEM 证书对','匯入 PEM 憑證組','PEMペアをインポート','Импорт пары PEM','ورود جفت PEM','Nhập cặp PEM','Importar par PEM','PEM çifti aktar','Імпорт пари PEM','Importar par PEM'],
    bind: ['Bind certificate','绑定证书','綁定憑證','証明書を関連付け','Привязать сертификат','اتصال گواهی','Gắn chứng chỉ','Vincular certificado','Sertifika bağla','Прив’язати сертифікат','Vincular certificado'],
    createSidecar: ['Create sidecar','新建 Sidecar','新增 Sidecar','Sidecarを作成','Создать sidecar','ایجاد سرویس جانبی','Tạo sidecar','Crear sidecar','Sidecar oluştur','Створити sidecar','Criar sidecar'],
    checkUpdate: ['Check release','检查版本','檢查版本','リリースを確認','Проверить версию','بررسی نسخه','Kiểm tra phiên bản','Buscar versión','Sürümü denetle','Перевірити версію','Verificar versão'],
    updateControl: ['Update control','更新控制端','更新控制端','制御側を更新','Обновить контроллер','به‌روزرسانی کنترل','Cập nhật điều khiển','Actualizar control','Kontrolü güncelle','Оновити контролер','Atualizar controle'],
    command: ['Enrollment command; paste token into stdin','接入命令：通过标准输入粘贴令牌','接入命令：透過標準輸入貼上權杖','登録コマンド：標準入力にトークンを貼り付け','Команда регистрации: токен через stdin','دستور عضویت: توکن از ورودی استاندارد','Lệnh tham gia: dán mã vào stdin','Comando: pegar token en stdin','Katılım: belirteci stdin ile girin','Команда реєстрації: токен через stdin','Comando: cole token na entrada padrão'],
    downloadLog: ['Download conversion log','下载转换日志','下載轉換記錄','変換ログを保存','Скачать журнал','دریافت گزارش تبدیل','Tải nhật ký chuyển đổi','Descargar registro','Dönüşüm kaydını indir','Завантажити журнал','Baixar registro'],
    subscriptions: ['User subscriptions','用户订阅','使用者訂閱','ユーザー購読','Подписки пользователей','اشتراک کاربران','Đăng ký người dùng','Suscripciones','Kullanıcı abonelikleri','Підписки користувачів','Assinaturas']
  });
  function t(key) { const lang = window.upGetLang?.() || 'en'; const i = Math.max(0, locales.indexOf(lang)); return words[key]?.[i] || words[key]?.[0] || key; }
  Object.assign(words, {
    username: ['Username','用户名','使用者名稱','ユーザー名','Имя пользователя','نام کاربری','Tên người dùng','Usuario','Kullanıcı adı','Ім’я користувача','Usuário'],
    password: ['Password','密码','密碼','パスワード','Пароль','گذرواژه','Mật khẩu','Contraseña','Parola','Пароль','Senha'],
    controlURL: ['Control URL','控制端地址','控制端位址','制御URL','Адрес контроллера','نشانی کنترل','URL điều khiển','URL del control','Kontrol adresi','Адреса контролера','URL de controle'],
    enrollmentToken: ['Enrollment token','接入令牌','接入權杖','登録トークン','Токен регистрации','توکن عضویت','Mã tham gia','Token de registro','Katılım belirteci','Токен реєстрації','Token de registro'],
    legacyNode: ['Legacy node ID','旧节点 ID','舊節點 ID','旧ノードID','ID старого узла','شناسه گره قدیمی','ID nút cũ','ID del nodo anterior','Eski düğüm kimliği','ID старого вузла','ID do nó antigo'],
    uploadBackup: ['Upload backup','上传备份','上傳備份','バックアップを送信','Загрузить копию','بارگذاری پشتیبان','Tải bản sao lên','Subir copia','Yedeği yükle','Вивантажити копію','Enviar backup'],
    'collect-status': ['Collect status','采集状态','擷取狀態','状態を取得','Собрать состояние','دریافت وضعیت','Thu thập trạng thái','Recopilar estado','Durumu topla','Зібрати стан','Coletar estado'],
    'test-egress': ['Test outbound IP','检测出口 IP','檢測出口 IP','出口IPを確認','Проверить выходной IP','آزمون IP خروجی','Kiểm tra IP đầu ra','Probar IP de salida','Çıkış IP adresini denetle','Перевірити вихідний IP','Testar IP de saída'],
    'kernel-switch': ['Update Mihomo','更新 Mihomo','更新 Mihomo','Mihomoを更新','Обновить Mihomo','به‌روزرسانی Mihomo','Cập nhật Mihomo','Actualizar Mihomo','Mihomo güncelle','Оновити Mihomo','Atualizar Mihomo'],
    'geo-update': ['Update Geo data','更新 Geo 数据','更新 Geo 資料','Geoデータを更新','Обновить Geo-данные','به‌روزرسانی داده Geo','Cập nhật dữ liệu Geo','Actualizar datos Geo','Geo verilerini güncelle','Оновити Geo-дані','Atualizar dados Geo'],
    'cert-renew': ['Renew local certificate','续期本地证书','續期本機憑證','ローカル証明書を更新','Продлить локальный сертификат','تمدید گواهی محلی','Gia hạn chứng chỉ cục bộ','Renovar certificado local','Yerel sertifikayı yenile','Поновити локальний сертифікат','Renovar certificado local'],
    'agent-upgrade': ['Upgrade Agent','升级 Agent','升級 Agent','Agentを更新','Обновить агент','ارتقای عامل','Nâng cấp Agent','Actualizar agente','Agent güncelle','Оновити агент','Atualizar agente'],
    defaults: ['Apply global defaults','应用全局默认','套用全域預設','全体の既定値を適用','Применить общие настройки','اعمال پیش‌فرض سراسری','Áp dụng mặc định chung','Aplicar valores globales','Genel varsayılanları uygula','Застосувати загальні налаштування','Aplicar padrões globais'],
    quota: ['Change client quota','修改用户配额','修改使用者配額','ユーザーの上限を変更','Изменить квоту клиента','تغییر سهمیه کاربر','Đổi hạn mức người dùng','Cambiar cuota del cliente','Kullanıcı kotasını değiştir','Змінити квоту клієнта','Alterar cota do cliente'],
    inbound: ['Create inbound','创建入站','建立入站','インバウンドを作成','Создать входящее подключение','ایجاد ورودی','Tạo đầu vào','Crear entrada','Gelen bağlantı oluştur','Створити вхідне підключення','Criar entrada']
  });
  const originalFetch = window.fetch.bind(window);
  const selected = document.documentElement.dataset.nodeId || '';
  window.UPFleet={locales,words,get selectedNode(){return selected;}};
  if(role==='control' && document.documentElement.dataset.workspace!=='portal') {
    window.UPServerWorkspace?.();
    return;
  }
  const apiURL = path => typeof panelURL === 'function' ? panelURL('api/fleet' + path) : './api/fleet' + path;
  async function request(path, body, method) {
    const response = await originalFetch(apiURL(path), {method: method || (body === undefined ? 'GET' : 'POST'), headers: {'Content-Type':'application/json'}, body: body === undefined ? undefined : JSON.stringify(body)});
    if(response.status===401){location.href=panelURL('login');throw new Error('Session expired');}
    const data = await response.json(); if (!response.ok) throw new Error(data.message || String(response.status)); return data.data ?? data;
  }
  async function hostSettings(){
    const response=await originalFetch(panelURL('api/config'));
    const data=await response.json();if(!response.ok)throw new Error(data.message||String(response.status));return data.data;
  }
  const button = document.querySelector('#fleet-nav') || document.createElement('button'); button.type='button'; if(!button.isConnected)document.querySelector('#nav')?.insertBefore(button,document.querySelector('#sidebar-logout'));
  const section = document.querySelector('#view-fleet') || document.createElement('section'); section.id='view-fleet'; section.classList.add('view'); if(!section.isConnected)document.querySelector('main.main')?.prepend(section);
  const style=document.createElement('style');style.textContent='.fleet-target-control{display:flex;align-items:center;gap:8px;min-width:0}.fleet-target-control select{max-width:220px;min-width:130px;height:var(--ctl);padding:0 30px 0 10px;border:1px solid var(--stroke);border-radius:var(--radius);background:var(--surface);color:var(--text-strong);font:inherit}.fleet-target-status{max-width:150px;overflow:hidden;color:var(--muted);font-size:var(--p-caption);text-overflow:ellipsis;white-space:nowrap}.fleet-toolbar{display:flex;gap:8px;flex-wrap:wrap;align-items:center;margin:12px 0}.fleet-block{overflow-wrap:anywhere;padding:18px;border:1px solid var(--stroke-2,#8885);border-radius:var(--radius,12px);margin:16px 0;background:var(--surface,transparent)}.fleet-block input,.fleet-block select,.fleet-block textarea{padding:8px;border:1px solid var(--stroke-2,#8885);border-radius:6px;background:var(--bg,transparent);color:inherit}.fleet-block textarea{width:100%;min-height:100px;box-sizing:border-box;font-family:monospace}.fleet-scroll{overflow:auto}.fleet-grid{min-width:880px;width:100%;border-collapse:collapse}.fleet-grid th,.fleet-grid td{padding:10px;text-align:start;border-bottom:1px solid var(--stroke-2,#8885)}.fleet-cards{display:grid;grid-template-columns:repeat(auto-fill,minmax(min(100%,360px),1fr));gap:12px;margin:16px 0}.fleet-node-card{min-width:0;padding:16px;border:1px solid var(--stroke-2,#8885);border-radius:8px;background:var(--surface,transparent)}.fleet-node-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px}.fleet-node-title{margin:0;font-size:var(--p-title-sm,18px);overflow-wrap:anywhere}.fleet-node-meta{display:grid;gap:6px;margin:12px 0;color:var(--muted);font-size:var(--p-body-sm)}.fleet-node-card input:not([type=checkbox]){width:100%;min-width:0;box-sizing:border-box;padding:8px;border:1px solid var(--stroke-2,#8885);border-radius:6px;background:var(--bg,transparent);color:inherit}.fleet-node-state{display:inline-flex;align-items:center;gap:6px;white-space:nowrap}.fleet-node-state::before{content:"";width:8px;height:8px;border-radius:50%;background:var(--red,#d44)}.fleet-node-state[data-online="true"]::before{background:var(--green,#29865b)}.fleet-enrollment{margin-top:12px;padding-top:12px;border-top:1px solid var(--stroke-2,#8885)}.fleet-enrollment code{display:block;max-height:100px;overflow:auto;padding:10px;background:var(--bg,transparent);overflow-wrap:anywhere;white-space:pre-wrap}.fleet-result{white-space:pre-wrap;overflow-wrap:anywhere;max-height:400px;overflow:auto}.fleet-error{color:var(--red,#d44)}@media(max-width:700px){.fleet-target-status{display:none}.fleet-target-control select{max-width:150px;min-width:100px}}';if(role!=='control')document.head.append(style);
  let fleet=null, workspace;
  button.dataset.view='fleet';
  function element(tag,text,parent,props={}){const el=document.createElement(tag);if(text!==undefined)el.textContent=text;Object.assign(el,props);parent?.append(el);return el;}
  function action(label,parent,fn){const el=element('button',t(label),parent,{type:'button',className:'outline-btn'});el.onclick=async()=>{el.disabled=true;try{await fn();}catch(error){output(error.message,true);}finally{el.disabled=false;}};return el;}
  function output(data,error=false){if(role==='control'&&workspace){workspace.output(data,error);return;}const el=section.querySelector('#fleet-output');if(el){el.textContent=typeof data==='string'?data:JSON.stringify(data,null,2);el.classList.toggle('fleet-error',error);}}
  function input(label,parent,type='text'){const box=element('label',t(label)+' ',parent);return element('input',undefined,box,{type});}
  async function selectTarget(value,view='dashboard'){
    location.href=panelURL('nodes/'+encodeURIComponent(value)+'/')+(view&&view!=='dashboard'?'#'+view:'');
    return false;
  }
  async function load(){
    const status=section.querySelector('.fleet-load-status');
    if(status)status.textContent=role==='control'?'正在连接舰队服务…':'正在读取节点状态…';
    try{fleet=await request('');render();}
    catch(error){
      section.replaceChildren();
      element('h2',t('title'),section);
      element('p',error.message,section,{className:'fleet-error',role:'alert'});
      action('refresh',section,load);
      throw error;
    }
  }
  async function copyText(value){
    if(navigator.clipboard?.writeText){try{await navigator.clipboard.writeText(value);return;}catch{ /* Use selection copying when clipboard permissions are unavailable. */ }}
    const previous=document.activeElement;
    const field=element('textarea',undefined,document.querySelector('dialog[open]')||document.body,{value,readOnly:true});
    field.style.cssText='position:fixed;left:0;top:0;width:1px;height:1px;opacity:0';
    field.select();const copied=document.execCommand('copy');field.remove();previous?.focus();if(!copied)throw new Error('Clipboard access is unavailable');
  }
  button.addEventListener('click',()=>load().catch(()=>{}));
  function render(){
    if(section.classList.contains('active'))document.querySelector('#crumb').textContent=t('title');
    if(role==='control'){
      button.innerHTML=SF('multi-control','nav-icon')+'<span>'+t('title')+'</span>';
      workspace ||= window.UPFleetWorkspace({request,load,selectTarget,hostSettings,t,copyText});
      workspace.render(fleet);
      return;
    }
  button.innerHTML=SF('multi-control','nav-icon')+'<span>'+t('title')+'</span>';section.replaceChildren();element('h2',t('title'),section);
    const toolbar=element('div',undefined,section,{className:'fleet-toolbar'});action('refresh',toolbar,load);
    const panel=element('div',undefined,section,{className:'fleet-block'});
      const url=input('controlURL',panel,'url'),key=input('key',panel),token=input('enrollmentToken',panel,'password');url.placeholder='https://control.example';
      action('join',panel,async()=>output(await request('/join',{url:url.value,key:key.value,token:token.value})));
      if(fleet.managed){element('pre',JSON.stringify(fleet.status,null,2),panel,{className:'fleet-result'});action('emergency',panel,async()=>{if(await upConfirm(t('confirm'),{danger:true}))output(await request('/break-glass',{}));});}
      else{
        const local=element('details',undefined,panel);element('summary',t('sidecars'),local);const localJSON=element('textarea',undefined,local,{value:'[]',ariaLabel:t('sidecars')});action('refresh',local,async()=>{localJSON.value=JSON.stringify(await request('/sidecars'),null,2);});action('save',local,async()=>output(await request('/sidecars',JSON.parse(localJSON.value))));
        const nodeID=input('legacyNode',local);const file=input('import',local,'file');file.accept='.json';file.onchange=async()=>{try{const result=await request('/import/merit?nodeId='+encodeURIComponent(nodeID.value),JSON.parse(await file.files[0].text()));output(result);}catch(error){output(error.message,true);}};
      }
    element('h3',t('output'),section);element('pre','',section,{id:'fleet-output',className:'fleet-result',ariaLive:'polite'});
  }
  let polling=false;
  setInterval(async()=>{
    if(polling||role!=='control'||!section.classList.contains('active'))return;
    polling=true;
    try{fleet=await request('');workspace?.update(fleet);}
    catch(error){output(error.message,true);}
    finally{polling=false;}
  },15000);
  button.textContent=t('title');window.addEventListener('up-langchange',()=>{button.textContent=t('title');if(fleet)render();if(section.classList.contains('active'))document.querySelector('#crumb').textContent=t('title');});
  window.UPFleet={locales,words,get selectedNode(){return selected;}};
  if(role==='control'){
    button.click();
    document.querySelector('#crumb').textContent=t('title');
  } else {
    load().catch(()=>{});
  }
}());
