/* Shared browser utilities for both HTTP and HTTPS panels. */
'use strict';
// Hold only translated pages until their static shell has been localized.
// Release on timeout as well, so a failed script never leaves an unusable page.
if (document.documentElement.hasAttribute('data-i18n-page')) {
  const root = document.documentElement;
  root.setAttribute('data-i18n-pending', '');
  let language = root.dataset.defaultLanguage || 'zh-CN';
  if (!root.dataset.workspace) {
    try { language = localStorage.getItem('up-language') || language; } catch {}
  }
  root.lang = language;
  root.dir = language === 'fa' ? 'rtl' : 'ltr';
  root.dataset.i18nLoading = language.startsWith('zh') ? '正在加载工作台…' : 'Loading workspace…';
  const style = document.createElement('style');
  style.textContent = 'html[data-i18n-pending] body{visibility:hidden}html[data-i18n-pending]::after{content:attr(data-i18n-loading);position:fixed;inset:0;display:grid;place-items:center;background:var(--bg,#f7f7f8);color:var(--text-strong,#242424);font:14px system-ui;z-index:2147483647}';
  document.head.append(style);
  const fallback = setTimeout(finish, 5000);
  function finish() {
    clearTimeout(fallback);
    root.removeAttribute('data-i18n-pending');
    delete root.dataset.i18nLoading;
    style.remove();
  }
  window.upFinishLanguageBoot = finish;
}
window.upRandomUUID = function () {
  if (typeof crypto.randomUUID === 'function') return crypto.randomUUID();
  // getRandomValues remains available on HTTP; keep UUIDs cryptographically random.
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const hex = Array.from(bytes, value => value.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
};
