/* UnifiedProxy palette controls. References: Vercel, Claude/Anthropic, Sentry,
 * Supabase, Stripe via awesome-design-md (MIT). m-ui application attribution in NOTICE. */
(function () {
  'use strict';
  const root = document.documentElement;
  const labels = ['T0 黑白 · 墨 · Vercel', 'T1 暖纸陶土 · Claude / Anthropic',
    'T2 夜紫荧光 · Sentry（首次默认暗色）', 'T3 翡翠绿 · Supabase', 'T4 靛紫 · Stripe'];
  const valid = value => /^t[0-4]$/.test(value || '');
  const get = key => { try { return localStorage.getItem(key); } catch { return null; } };
  const put = (key, value) => { try { localStorage.setItem(key, value); } catch {} };
  const readModes = () => { try { return JSON.parse(get('up-appearance-modes') || '{}') || {}; } catch { return {}; } };
  let modes = readModes();
  let transitionTimer;
  function colorTransition() {
    root.classList.add('up-theme-changing');
    clearTimeout(transitionTimer);
    transitionTimer=setTimeout(()=>root.classList.remove('up-theme-changing'),150);
  }
  let color = valid(get('up-color-theme')) ? get('up-color-theme') : (valid(root.dataset.colorTheme) ? root.dataset.colorTheme : 't0');
  let theme = modes[color] || get('up-theme') || (get('mui-is-ultra-dark') === 'true' ? 'dark' : get('mui-theme')) || (color === 't2' ? 'dark' : 'light');
  theme = theme === 'dark' ? 'dark' : 'light';
  function sync() {
    root.dataset.colorTheme = color;
    root.dataset.theme = theme;
    root.classList.toggle('dark', theme === 'dark');
    document.body?.classList.toggle('dark', theme === 'dark');
    document.querySelectorAll('[data-color-picker]').forEach(el => { el.value = color; });
    document.querySelectorAll('[data-palette]').forEach(el => el.setAttribute('aria-pressed', String(el.dataset.palette === color)));
    document.querySelectorAll('#sidebar-dark-toggle, #dark-mode, #theme-dark').forEach(el => { el.checked = theme === 'dark'; });
    document.querySelectorAll('[data-mode]').forEach(el => el.setAttribute('aria-pressed', String(el.dataset.mode === theme)));
    const setting = document.querySelector('[name="uiTheme"]');
    if (setting) setting.value = color;
  }
  function setMode(value, manual = true) {
    colorTransition();
    theme = value === 'dark' || value === true ? 'dark' : 'light';
    if (manual) { modes[color] = theme; put('up-appearance-modes', JSON.stringify(modes)); }
    put('up-theme', theme); put('mui-theme', theme); sync();
  }
  function setColor(value) {
    if (!valid(value)) return;
    colorTransition();
    const changed = value !== color;
    color = value;
    if (modes[color]) theme = modes[color];
    else if (changed && color === 't2') theme = 'dark';
    put('up-color-theme', color); put('up-theme', theme); sync();
    root.dispatchEvent(new CustomEvent('up-appearance-change', { detail: { color, theme } }));
  }
  function picker() {
    const el = document.createElement('select');
    el.setAttribute('data-color-picker', ''); el.setAttribute('data-no-swift-picker', '');
    el.setAttribute('aria-label', '配色主题 / Color theme');
    labels.forEach((label, i) => el.add(new Option(label, 't' + i)));
    el.value = color; el.onchange = () => setColor(el.value);
    return el;
  }
  window.UPTheme = { setMode, setColor, sync, labels, get color() { return color; }, get mode() { return theme; } };
  sync();
  document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('[data-appearance-slot]').forEach(slot => slot.append(picker()));
    const setting = document.querySelector('[name="uiTheme"]');
    if (setting) {
      setting.setAttribute('data-no-swift-picker', '');
      setting.replaceChildren(...labels.map((label, i) => new Option(label, 't' + i)));
      setting.onchange = () => setColor(setting.value);
    }
    document.querySelectorAll('[data-palette]').forEach(el => el.onclick = () => setColor(el.dataset.palette));
    document.querySelectorAll('[data-mode]').forEach(el => el.onclick = () => setMode(el.dataset.mode));
    sync();
    // renderSettings may populate the old server setting after the first paint.
    // An explicit browser choice wins until the user saves the form.
    document.addEventListener('up-state-rendered', sync);
  });
  window.addEventListener('storage', event => {
    if (event.key?.startsWith('up-')) {
      modes = readModes(); color = valid(get('up-color-theme')) ? get('up-color-theme') : color;
      theme = modes[color] || get('up-theme') || theme; sync();
    }
  });
}());
