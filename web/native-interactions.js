/* UnifiedProxy native gestures; m-ui interaction compatibility, GPL-3.0.
 * Retains switch/segment keyboard, drag, cancellation and disabled semantics. */
(function () {
  'use strict';
  const bars = '.segmented,.settings-tabbar,.mihomo-tabs,.mihomo-editor-tabs,#inbound-status-tabs,#security-extra-toggles';
  let gesture = null, suppress = null;
  const buttons = bar => Array.from(bar.querySelectorAll(':scope > button')).filter(b => !b.disabled);
  document.addEventListener('keydown', event => {
    const button = event.target.closest('button');
    const bar = button?.parentElement;
    if (!bar?.matches(bars) || !['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return;
    const items = buttons(bar); if (!items.length) return;
    event.preventDefault();
    const rtl = getComputedStyle(bar).direction === 'rtl';
    const direction = (event.key === 'ArrowRight' ? 1 : -1) * (rtl ? -1 : 1);
    const index = event.key === 'Home' ? 0 : event.key === 'End' ? items.length - 1 : (items.indexOf(button) + direction + items.length) % items.length;
    items[index].focus(); items[index].click();
  });
  document.addEventListener('pointerdown', event => {
    if (event.button !== 0 || event.isPrimary === false) return;
    const shell = event.target.closest('.switch,.theme-switch');
    const input = shell?.querySelector('input[type=checkbox]');
    const button = event.target.closest('button');
    const bar = button?.parentElement?.matches(bars) ? button.parentElement : null;
    if (input && !input.disabled) gesture = {shell,input,x:event.clientX,id:event.pointerId,moved:false};
    else if (bar && !button.disabled) gesture = {shell:bar,button,x:event.clientX,id:event.pointerId,moved:false};
    else return;
  });
  document.addEventListener('pointermove', event => {
    if (!gesture || gesture.id !== event.pointerId) return;
    gesture.moved ||= Math.abs(event.clientX-gesture.x)>6;
    gesture.targetX = event.clientX;
    if (gesture.moved && event.cancelable) event.preventDefault();
  },{passive:false});
  function finish(event, cancel) {
    const g = gesture; if (!g || g.id !== event.pointerId) return;
    gesture = null;
    if (!g.moved) return;
    suppress = {shell:g.shell,until:performance.now()+500};
    if (cancel || !g.shell.isConnected) return;
    if (g.input) {
      if (g.input.disabled) return;
      const rect = g.shell.getBoundingClientRect();
      const checked = (event.clientX > rect.left+rect.width/2) !== (getComputedStyle(g.shell).direction === 'rtl');
      if (checked !== g.input.checked) g.input.click();
    } else {
      const target = buttons(g.shell).find(b => {const r=b.getBoundingClientRect();return event.clientX>=r.left&&event.clientX<=r.right;});
      if (target && target !== g.button) target.click();
    }
  }
  window.addEventListener('pointerup',event=>finish(event,false));
  window.addEventListener('pointercancel',event=>finish(event,true));
  document.addEventListener('click', event => {
    if (event.isTrusted && event.detail !== 0 && suppress && performance.now()<suppress.until && suppress.shell.contains(event.target)) {
      event.preventDefault();event.stopImmediatePropagation();
    }
  },true);
}());
