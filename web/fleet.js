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
  function t(key) { const lang = window.upGetLang?.() || 'en'; const i = Math.max(0, locales.indexOf(lang)); return words[key]?.[i] || key; }
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
  let selected = sessionStorage.getItem('up-fleet-node') || '';
  window.fetch = function (input, options) {
    const request = new Request(input, options);
    const u = new URL(request.url);
    if (role === 'control' && selected && u.origin === location.origin && /\/api\//.test(u.pathname) && !/\/api\/(fleet|auth|config(?:$|\/)|sub-templates|client-mesh)/.test(u.pathname)) request.headers.set('X-Fleet-Node', selected);
    return originalFetch(request);
  };
  const apiURL = path => typeof panelURL === 'function' ? panelURL('api/fleet' + path) : './api/fleet' + path;
  async function request(path, body, method) {
    const response = await originalFetch(apiURL(path), {method: method || (body === undefined ? 'GET' : 'POST'), headers: {'Content-Type':'application/json'}, body: body === undefined ? undefined : JSON.stringify(body)});
    const data = await response.json(); if (!response.ok) throw new Error(data.message || String(response.status)); return data.data ?? data;
  }
  const button = document.querySelector('#fleet-nav') || document.createElement('button'); button.type='button'; if(!button.isConnected)document.querySelector('#nav')?.insertBefore(button,document.querySelector('#sidebar-logout'));
  const targetControl = document.createElement('label');
  targetControl.className = 'fleet-target-control';
  const targetSelect = element('select', undefined, targetControl, {id:'fleet-target-select', 'aria-label':'Fleet node'});
  const targetStatus = element('span', undefined, targetControl, {className:'fleet-target-status', 'aria-live':'polite'});
  if (role === 'control') document.querySelector('.top-actions')?.prepend(targetControl);
  const section = document.querySelector('#view-fleet') || document.createElement('section'); section.id='view-fleet'; section.classList.add('view'); if(!section.isConnected)document.querySelector('main.main')?.prepend(section);
  const style=document.createElement('style');style.textContent='.fleet-target-control{display:flex;align-items:center;gap:8px;min-width:0}.fleet-target-control select{max-width:220px;min-width:130px;height:var(--ctl);padding:0 30px 0 10px;border:1px solid var(--stroke);border-radius:var(--radius);background:var(--surface);color:var(--text-strong);font:inherit}.fleet-target-status{max-width:150px;overflow:hidden;color:var(--muted);font-size:var(--p-caption);text-overflow:ellipsis;white-space:nowrap}.fleet-toolbar{display:flex;gap:8px;flex-wrap:wrap;align-items:center;margin:12px 0}.fleet-block{overflow-wrap:anywhere;padding:18px;border:1px solid var(--stroke-2,#8885);border-radius:var(--radius,12px);margin:16px 0;background:var(--surface,transparent)}.fleet-block input,.fleet-block select,.fleet-block textarea{padding:8px;border:1px solid var(--stroke-2,#8885);border-radius:6px;background:var(--bg,transparent);color:inherit}.fleet-block textarea{width:100%;min-height:100px;box-sizing:border-box;font-family:monospace}.fleet-scroll{overflow:auto}.fleet-grid{min-width:880px;width:100%;border-collapse:collapse}.fleet-grid th,.fleet-grid td{padding:10px;text-align:start;border-bottom:1px solid var(--stroke-2,#8885)}.fleet-cards{display:grid;grid-template-columns:repeat(auto-fill,minmax(min(100%,360px),1fr));gap:12px;margin:16px 0}.fleet-node-card{min-width:0;padding:16px;border:1px solid var(--stroke-2,#8885);border-radius:8px;background:var(--surface,transparent)}.fleet-node-head{display:flex;align-items:flex-start;justify-content:space-between;gap:12px}.fleet-node-title{margin:0;font-size:var(--p-title-sm,18px);overflow-wrap:anywhere}.fleet-node-meta{display:grid;gap:6px;margin:12px 0;color:var(--muted);font-size:var(--p-body-sm)}.fleet-node-card input:not([type=checkbox]){width:100%;min-width:0;box-sizing:border-box;padding:8px;border:1px solid var(--stroke-2,#8885);border-radius:6px;background:var(--bg,transparent);color:inherit}.fleet-node-state{display:inline-flex;align-items:center;gap:6px;white-space:nowrap}.fleet-node-state::before{content:"";width:8px;height:8px;border-radius:50%;background:var(--red,#d44)}.fleet-node-state[data-online="true"]::before{background:var(--green,#29865b)}.fleet-enrollment{margin-top:12px;padding-top:12px;border-top:1px solid var(--stroke-2,#8885)}.fleet-enrollment code{display:block;max-height:100px;overflow:auto;padding:10px;background:var(--bg,transparent);overflow-wrap:anywhere;white-space:pre-wrap}.fleet-result{white-space:pre-wrap;overflow-wrap:anywhere;max-height:400px;overflow:auto}.fleet-error{color:var(--red,#d44)}@media(max-width:700px){.fleet-target-status{display:none}.fleet-target-control select{max-width:150px;min-width:100px}}';document.head.append(style);
  let fleet=null, modelID='';
  button.dataset.view='fleet';
  function element(tag,text,parent,props={}){const el=document.createElement(tag);if(text!==undefined)el.textContent=text;Object.assign(el,props);parent?.append(el);return el;}
  function action(label,parent,fn){const el=element('button',t(label),parent,{type:'button',className:'outline-btn'});el.onclick=async()=>{el.disabled=true;try{await fn();}catch(error){output(error.message,true);}finally{el.disabled=false;}};return el;}
  function output(data,error=false){const el=section.querySelector('#fleet-output');if(el){el.textContent=typeof data==='string'?data:JSON.stringify(data,null,2);el.classList.toggle('fleet-error',error);}}
  function input(label,parent,type='text'){const box=element('label',t(label)+' ',parent);return element('input',undefined,box,{type});}
  function updateTargetStatus(){
    if(role!=='control'||!fleet)return;
    const node=(fleet.nodes||[]).find(item=>item.id===selected);
    targetStatus.textContent=node?(node.online?`Online · ${node.status?.running?'Mihomo running':'Mihomo stopped'}`:'Offline'):'';
    targetStatus.title=targetStatus.textContent;
  }
  function syncTargetSelect(){
    if(role!=='control'||!fleet)return;
    const lang=window.upGetLang?.()||'en',zh=lang.startsWith('zh');
    targetSelect.replaceChildren();
    element('option',zh?'本机默认配置':'Local panel',targetSelect,{value:''});
    for(const node of fleet.nodes||[])element('option',`${node.name||node.id}${node.revoked?' · revoked':''}`,targetSelect,{value:node.id,disabled:node.revoked});
    if(selected&&!(fleet.nodes||[]).some(node=>node.id===selected&&!node.revoked)){
      selected='';
      sessionStorage.removeItem('up-fleet-node');
    }
    targetSelect.value=selected;
    updateTargetStatus();
  }
  async function selectTarget(value){
    if(value===selected)return true;
    const dirty=window.UPPanel?.isDirty?.();
    if(dirty&&!await upConfirm(t('confirm'),{danger:true})){targetSelect.value=selected;return false;}
    selected=value;sessionStorage.setItem('up-fleet-node',selected);targetSelect.value=selected;updateTargetStatus();
    await window.UPPanel?.refresh?.();
    return true;
  }
  targetSelect.onchange=()=>selectTarget(targetSelect.value).catch(error=>{targetSelect.value=selected;output(error.message,true);});
  async function load(){
    const status=section.querySelector('.fleet-load-status');
    if(status)status.textContent=role==='control'?'正在连接舰队服务…':'正在读取节点状态…';
    try{fleet=await request('');syncTargetSelect();render();}
    catch(error){
      section.replaceChildren();
      element('h2',t('title'),section);
      element('p',error.message,section,{className:'fleet-error',role:'alert'});
      action('refresh',section,load);
      throw error;
    }
  }
  function showEnrollment(result){
    const quote=value=>String.fromCharCode(39)+String(value).split(String.fromCharCode(39)).join(String.fromCharCode(39,34,39,34,39))+String.fromCharCode(39);
    output({...result,[t('command')]:'node agent join --data-dir /usr/local/unifiedproxy-node/data --control '+quote(location.origin)+' --control-key '+quote(result.controlKey)+' --token-stdin'});
  }
  function makeEnrollmentCommand(result){
    const quote=value=>String.fromCharCode(39)+String(value).split(String.fromCharCode(39)).join(String.fromCharCode(39,34,39,34,39))+String.fromCharCode(39);
    return 'node agent join --data-dir /usr/local/unifiedproxy-node/data --control '+quote(location.origin)+' --control-key '+quote(result.controlKey)+' --token-stdin';
  }
  function showNodeEnrollment(card,node,result){
    card.querySelector('.fleet-enrollment')?.remove();
    const zh=(window.upGetLang?.()||'').startsWith('zh');
    const box=element('div',undefined,card,{className:'fleet-enrollment'});
    element('strong',zh?'子主机接入信息（15 分钟内有效）':'Child host enrollment (valid for 15 minutes)',box);
    element('p',zh?'在子主机终端执行命令，然后粘贴下方一次性令牌。':'Run the command on the child host, then paste the one-time token below.',box);
    const command=element('code',makeEnrollmentCommand(result),box);
    action(zh?'复制命令':'Copy command',box,async()=>{await copyText(command.textContent);output(zh?'部署命令已复制':'Deployment command copied');});
    const token=element('code',result.token,box);
    action(zh?'复制令牌':'Copy token',box,async()=>{await copyText(token.textContent);output(zh?'接入令牌已复制':'Enrollment token copied');});
    box.dataset.enrollmentNode=node.id;
  }
  async function copyText(value){
    if(navigator.clipboard?.writeText){await navigator.clipboard.writeText(value);return;}
    const field=element('textarea',undefined,document.body,{value,readonly:true});field.select();const copied=document.execCommand('copy');field.remove();if(!copied)throw new Error('Clipboard access is unavailable');
  }
  button.onclick=()=>{if(typeof setView==='function'){labels.fleet='crumb.dashboard';setView('fleet');document.querySelector('#crumb').textContent=t('title');}load().catch(()=>{});};
  function render(){
  button.innerHTML=SF('multi-control','nav-icon')+'<span>'+t('title')+'</span>';section.replaceChildren();element('h2',t('title'),section);
    const toolbar=element('div',undefined,section,{className:'fleet-toolbar'});action('refresh',toolbar,load);
    const panel=element('div',undefined,section,{className:'fleet-block'});
    if(role==='control'){
      const zh=(window.upGetLang?.()||'').startsWith('zh');
      element('p',`${zh?'控制端公钥':'Control public key'}: ${fleet.publicKey}`,panel);
      const form=element('form',undefined,panel,{className:'fleet-toolbar'});form.onsubmit=event=>event.preventDefault();const name=input('name',form),address=input('address',form);name.required=true;address.type='url';address.required=true;address.placeholder='https://host.example:port';
      action('add',form,async()=>{
        if(!name.value.trim()||!address.value.trim())throw new Error(zh?'请填写节点名称和公开地址':'Enter a node name and public address');
        const node=await request('/nodes',{name:name.value.trim(),address:address.value.trim()});
        await load();
        const card=Array.from(section.querySelectorAll('[data-node-card]')).find(item=>item.dataset.nodeCard===node.id);
        card?.scrollIntoView({behavior:'smooth',block:'center'});
        const enrollment=await request('/tokens',{nodeId:node.id,name:node.name});
        if(card)showNodeEnrollment(card,node,enrollment);
        name.value='';address.value='';
      });
      const cards=element('div',undefined,section,{className:'fleet-cards'});
      for(const node of fleet.nodes||[]){
        const card=element('article',undefined,cards,{className:'fleet-node-card'});card.dataset.nodeCard=node.id;
        const head=element('div',undefined,card,{className:'fleet-node-head'});
        element('h3',(node.name||node.id)+(node.needsUpgrade?' · '+t('upgrade'):''),head,{className:'fleet-node-title'});
        const state=element('span',nodeConnectionState(node),head,{className:'fleet-node-state'});state.dataset.online=String(Boolean(node.online&&!node.revoked));state.dataset.nodeState=node.id;
        const meta=element('div',undefined,card,{className:'fleet-node-meta'});
        const addr=input('address',meta);addr.value=node.address||'';addr.dataset.nodeAddress=node.id;addr.onchange=()=>request('/nodes/'+node.id,{name:node.name,address:addr.value},'PUT').then(load).catch(e=>output(e.message,true));
        const status=element('div',nodeStatus(node),meta);status.dataset.fleetStatus=node.id;status.title=node.status?.error||'';
        const traffic=element('div',nodeTraffic(node),meta);traffic.dataset.fleetTraffic=node.id;
        const select=element('input',undefined,meta,{type:'checkbox',value:node.id,className:'fleet-selected','ariaLabel':t('select')+' '+node.name});
        const controls=element('div',undefined,card,{className:'fleet-toolbar'});
        action('edit',controls,async()=>{if(!await selectTarget(node.id))return;const detail=await request('/nodes/'+node.id);modelID=node.id;model.value=JSON.stringify(detail.draft||detail.desired?.model||{},null,2);sidecars.value=JSON.stringify(detail.desired?.sidecars||[],null,2);});
        action(zh?'重新生成接入信息':'Generate enrollment info',controls,async()=>showNodeEnrollment(card,node,await request('/tokens',{nodeId:node.id,name:node.name})));
        action('publish',controls,async()=>{output(await request('/nodes/'+node.id+'/publish',{}));});
        action('revoke',controls,async()=>{if(await upConfirm(t('confirm'),{danger:true})){await request('/nodes/'+node.id+'/revoke',{});await load();}});
        const manage=element('button',zh?'进入统一管理':'Manage from this panel',controls,{type:'button',className:'outline-btn'});
        manage.onclick=async()=>{if(!await selectTarget(node.id))return;if(typeof setView==='function'){setView('dashboard');document.querySelector('#crumb').textContent=zh?'仪表盘':'Dashboard';}};
      }
      const batch=element('div',undefined,section,{className:'fleet-block'});element('h3',t('batch'),batch);
      const kind=element('select',undefined,batch);for(const k of ['collect-status','test-egress','kernel-switch','geo-update','cert-renew','agent-upgrade','defaults','quota','inbound'])element('option',t(k),kind,{value:k});
      const payload=element('textarea',undefined,batch,{value:'{}','ariaLabel':'JSON'});
      action('execute',batch,async()=>{const nodes=Array.from(section.querySelectorAll('.fleet-selected:checked')).map(e=>e.value);const p=JSON.parse(payload.value);const configAction=['defaults','quota','inbound'].includes(kind.value);if(!configAction){const unsupported=(fleet.nodes||[]).filter(n=>nodes.includes(n.id)&&!n.status?.capabilities?.includes(kind.value));if(unsupported.length)throw new Error(t('upgrade')+': '+unsupported.map(n=>n.name).join(', '));}output(await request('/batch',configAction?{nodes,action:kind.value,...p}:{nodes,action:'task',kind:kind.value,payload:p}));});
      const edit=element('div',undefined,section,{className:'fleet-block'});element('h3',t('model'),edit);const model=element('textarea',undefined,edit,{value:'{}','ariaLabel':t('model')});action('save',edit,async()=>{if(!modelID)throw new Error(t('target'));output(await request('/nodes/'+modelID+'/model',JSON.parse(model.value)));});
      element('h3',t('sidecars'),edit);const sidecars=element('textarea',undefined,edit,{value:'[]','ariaLabel':t('sidecars')});action('save',edit,async()=>{if(!modelID)throw new Error(t('target'));output(await request('/nodes/'+modelID+'/sidecars',JSON.parse(sidecars.value)));});
      const sidecarRecipe=element('textarea',undefined,edit,{value:JSON.stringify({kind:'shadowtls',port:443,username:'',password:'',domain:'',handshake:'cloudflare.com:443',certificateName:''},null,2),ariaLabel:t('createSidecar')});
      action('createSidecar',edit,async()=>{if(!modelID)throw new Error(t('target'));const result=await request('/nodes/'+modelID+'/sidecar',JSON.parse(sidecarRecipe.value));sidecars.value=JSON.stringify(result.sidecars,null,2);output(result);});
      const certBox=element('details',undefined,section,{className:'fleet-block'});element('summary',t('certificates'),certBox);
      const certJSON=element('textarea',undefined,certBox,{value:JSON.stringify({name:'',domain:'',dns:'',automatic:false,certificate:'',privateKey:'',bindings:[]},null,2),ariaLabel:t('certificates')});
      action('refresh',certBox,async()=>output(await request('/certificates')));
      for(const [label,path] of [['issue','issue'],['renew','renew'],['importCert','import'],['bind','bind']])action(label,certBox,async()=>output(await request('/certificates/'+path,JSON.parse(certJSON.value))));
      const updates=element('details',undefined,section,{className:'fleet-block'});element('summary',t('updateControl'),updates);const updateJSON=element('textarea',undefined,updates,{value:JSON.stringify({url:'',sha256:''},null,2),ariaLabel:t('updateControl')});
      action('checkUpdate',updates,async()=>{const info=await request('/update/check');updateJSON.value=JSON.stringify({url:info.url,sha256:info.sha256},null,2);output(info);});
      action('updateControl',updates,async()=>{if(await upConfirm(t('confirm'),{danger:true}))output(await request('/update',JSON.parse(updateJSON.value)));});
      action('subscriptions',panel,async()=>output(await request('/subscriptions')));
      const backup=element('div',undefined,section,{className:'fleet-block'});element('a',t('backup'),backup,{href:apiURL('/backup'),download:'fleet-backup.json'});
      const importer=input('import',backup,'file');importer.accept='.json';importer.onchange=async()=>{try{const result=await request('/import/merit',JSON.parse(await importer.files[0].text()));await load();output(result);const link=element('a',t('downloadLog'),section,{href:URL.createObjectURL(new Blob([JSON.stringify(result,null,2)],{type:'application/json'})),download:'merit-conversion-log.json'});link.onclick=()=>setTimeout(()=>URL.revokeObjectURL(link.href),1000);}catch(e){output(e.message,true);}};
      const restore=input('restore',backup,'file');restore.accept='.json';restore.onchange=async()=>{try{if(await upConfirm(t('confirm'),{danger:true}))output(await request('/backup',{action:'restore',backup:JSON.parse(await restore.files[0].text())}));}catch(e){output(e.message,true);}};
      const dav=element('details',undefined,backup);element('summary','WebDAV',dav);const davURL=element('input',undefined,dav,{type:'url',placeholder:'https://…/fleet-backup.json','ariaLabel':'WebDAV URL'});const davUser=input('username',dav),davPass=input('password',dav,'password');davPass.autocomplete='new-password';action('uploadBackup',dav,async()=>output(await request('/backup',{action:'upload',webdav:{url:davURL.value,username:davUser.value,password:davPass.value}})));action('restore',dav,async()=>{if(await upConfirm(t('confirm'),{danger:true}))output(await request('/backup',{action:'download',webdav:{url:davURL.value,username:davUser.value,password:davPass.value}}));});
      element('h3',t('tasks'),section);const tasks=element('pre',undefined,section,{id:'fleet-tasks',className:'fleet-result'});tasks.textContent=taskSummary(fleet.tasks);
    } else {
      const url=input('controlURL',panel,'url'),key=input('key',panel),token=input('enrollmentToken',panel,'password');url.placeholder='https://control.example';
      action('join',panel,async()=>output(await request('/join',{url:url.value,key:key.value,token:token.value})));
      if(fleet.managed){element('pre',JSON.stringify(fleet.status,null,2),panel,{className:'fleet-result'});action('emergency',panel,async()=>{if(await upConfirm(t('confirm'),{danger:true}))output(await request('/break-glass',{}));});}
      else{
        const local=element('details',undefined,panel);element('summary',t('sidecars'),local);const localJSON=element('textarea',undefined,local,{value:'[]',ariaLabel:t('sidecars')});action('refresh',local,async()=>{localJSON.value=JSON.stringify(await request('/sidecars'),null,2);});action('save',local,async()=>output(await request('/sidecars',JSON.parse(localJSON.value))));
        const nodeID=input('legacyNode',local);const file=input('import',local,'file');file.accept='.json';file.onchange=async()=>{try{const result=await request('/import/merit?nodeId='+encodeURIComponent(nodeID.value),JSON.parse(await file.files[0].text()));output(result);}catch(error){output(error.message,true);}};
      }
    }
    element('h3',t('output'),section);element('pre','',section,{id:'fleet-output',className:'fleet-result',ariaLive:'polite'});
  }
  function nodeStatus(node){return t(node.online?'online':'offline')+' · Mihomo '+t(node.status?.running?'running':'stopped')+' · '+(node.status?.version||'—')+' · '+(node.status?.revision||0)+'/'+node.desiredRevision;}
  function nodeConnectionState(node){const zh=(window.upGetLang?.()||'').startsWith('zh');return node.revoked?(zh?'已吊销':'Revoked'):node.online?(zh?'在线':'Online'):(zh?'待接入 / 离线':'Pending / offline');}
  function nodeTraffic(node){let up=0,down=0,legacy=0;for(const [key,value] of Object.entries(node.status?.traffic||{})){if(!key.includes(':')){up+=value.up||0;down+=value.down||0;legacy+=value.legacyCombined||0;}}const format=value=>new Intl.NumberFormat().format(value);return 'Σ '+format(up+down+legacy)+' / ↑ '+format(up)+' / ↓ '+format(down)+(legacy?' / ↔ '+format(legacy):'');}
  function taskSummary(tasks){return JSON.stringify((tasks||[]).slice(-30).map(({id,nodeId,kind,status,attempts,result})=>({id,nodeId,kind,status,attempts,result})),null,2);}
  let polling=false;setInterval(async()=>{if(polling||role!=='control'||!section.classList.contains('active'))return;polling=true;try{fleet=await request('');for(const node of fleet.nodes||[]){for(const el of section.querySelectorAll('[data-fleet-status]'))if(el.dataset.fleetStatus===node.id){el.textContent=nodeStatus(node);el.title=node.status?.error||'';}for(const el of section.querySelectorAll('[data-fleet-traffic]'))if(el.dataset.fleetTraffic===node.id)el.textContent=nodeTraffic(node);for(const el of section.querySelectorAll('[data-node-state]'))if(el.dataset.nodeState===node.id){el.textContent=nodeConnectionState(node);el.dataset.online=String(Boolean(node.online&&!node.revoked));}}const tasks=section.querySelector('#fleet-tasks');if(tasks)tasks.textContent=taskSummary(fleet.tasks);}catch(error){output(error.message,true);}finally{polling=false;}},15000);
  button.textContent=t('title');window.addEventListener('up-langchange',()=>{button.textContent=t('title');if(fleet)render();if(section.classList.contains('active'))document.querySelector('#crumb').textContent=t('title');});
  window.UPFleet={locales,words,get selectedNode(){return selected;}};
  load().catch(()=>{});
}());
