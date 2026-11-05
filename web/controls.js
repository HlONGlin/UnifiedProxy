/* UnifiedProxy accessible controls; behavior adapted from UnifiedProxy (GPL-3.0). See NOTICE. */

(function(){
  'use strict';

  var STYLE_ID = 'up-swiftui-components-style';
  var ARROW = '<svg class="sf-symbol up-select-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>';

  var CSS = '';

  function injectStyle(){
    if(document.getElementById(STYLE_ID)) return;
    var style = document.createElement('style');
    style.id = STYLE_ID;
    style.textContent = CSS;
    (document.head || document.documentElement).appendChild(style);
  }

  
  var currentToast = null;
  var toastTimer = null;

  window.dynamicIslandToast = function(msg, isError, duration){
    if(toastTimer){
      clearTimeout(toastTimer);
      toastTimer = null;
    }
    if(!currentToast){
      currentToast = document.createElement('div');
      currentToast.className = 'dynamic-island-toast';
      currentToast.setAttribute('role', 'status');
      document.body.appendChild(currentToast);
    }

    currentToast.className = 'dynamic-island-toast' + (isError ? ' is-error' : '');
    var iconName = isError ? 'close-circle' : 'check-circle';
    var iconHtml = window.SF ? window.SF(iconName, 'toast-icon') : (window.AI ? window.AI(iconName, 'toast-icon') : '');
    currentToast.innerHTML = (iconHtml ? '<span class="toast-icon-wrap">' + iconHtml + '</span>' : '') + '<span class="toast-text">' + String(msg || '') + '</span>';
    currentToast.style.display = 'flex';

    var dur = typeof duration === 'number' ? duration : (isError ? 4000 : 2600);
    toastTimer = setTimeout(function(){
      if(currentToast){
        currentToast.style.transition = 'opacity var(--duration-fast) var(--easing-in)';
        currentToast.style.opacity = '0';
        currentToast.style.transform = 'translateX(-50%)';
        setTimeout(function(){
          if(currentToast && currentToast.parentNode){
            currentToast.parentNode.removeChild(currentToast);
          }
          currentToast = null;
        }, 120);
      }
    }, dur);
  };

  window.toast = window.dynamicIslandToast;

  
  var valueDesc = Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, 'value');
  var openWrap = null;

  function isSkipped(select){
    return select.multiple || select.size > 1 || !!select.querySelector('optgroup') || select.hasAttribute('data-no-swift-picker');
  }
  function isHiddenSelect(select){
    return select.hidden || select.style.display === 'none';
  }
  function signature(select){
    var out = '';
    for(var i = 0; i < select.options.length; i++){
      var option = select.options[i];
      out += option.value + '|' + option.textContent + '|' + (option.hidden ? 1 : 0) + '|' + (option.disabled ? 1 : 0) + ';';
    }
    return out + (select.disabled ? 1 : 0);
  }

  function closeAll(){
    if(!openWrap) return;
    openWrap.classList.remove('swift-picker-open');
    if(openWrap.up && openWrap.up.menu) openWrap.up.menu.hidden = true;
    openWrap = null;
  }

  function position(wrap){
    var menu = wrap.up.menu, rect = wrap.getBoundingClientRect();
    menu.style.width = Math.max(rect.width, 120) + 'px';
    menu.hidden = false;
    var height = menu.offsetHeight, top = rect.bottom + 4;
    if(top + height > window.innerHeight - 8 && rect.top - 4 - height > 8) top = rect.top - 4 - height;
    menu.style.left = Math.max(8, Math.min(rect.left, window.innerWidth - rect.width - 8)) + 'px';
    menu.style.top = top + 'px';
  }

  function open(wrap){
    if(wrap.up.select.disabled) return;
    if(openWrap === wrap){ closeAll(); return; }
    closeAll();
    rebuild(wrap);
    openWrap = wrap;
    wrap.classList.add('swift-picker-open');
    position(wrap);
    var active = wrap.up.menu.querySelector('.up-selected') || wrap.up.menu.querySelector('li');
    if(active) wrap.up.menu.scrollTop = Math.max(0, active.offsetTop - wrap.up.menu.clientHeight / 2 + active.offsetHeight / 2);
  }

  function rebuild(wrap){
    var select = wrap.up.select, menu = wrap.up.menu, html = '';
    for(var i = 0; i < select.options.length; i++){
      var option = select.options[i];
      if(option.hidden) continue;
      html += '<li data-index="' + i + '" class="' + (option.selected ? 'up-selected' : '') + (option.disabled ? ' up-disabled' : '') + '" role="option">' + option.textContent.replace(/[&<>]/g, function(ch){ return ch === '&' ? '&amp;' : ch === '<' ? '&lt;' : '&gt;'; }) + '</li>';
    }
    menu.innerHTML = html || '<li class="up-empty"></li>';
    wrap.up.sig = signature(select);
  }

  function refresh(wrap){
    var select = wrap.up.select;
    wrap.hidden = isHiddenSelect(select);
    wrap.classList.toggle('swift-picker-disabled', select.disabled);
    var option = select.options[select.selectedIndex];
    wrap.up.value.textContent = option ? option.textContent : '';
    if(signature(select) !== wrap.up.sig) rebuild(wrap);
    else {
      var items = wrap.up.menu.children;
      for(var i = 0; i < items.length; i++){
        if(items[i].dataset.index === undefined) continue;
        items[i].classList.toggle('up-selected', !!select.options[Number(items[i].dataset.index)]?.selected);
      }
    }
    if(openWrap === wrap) position(wrap);
  }

  function pick(wrap, index){
    var select = wrap.up.select;
    valueDesc.set.call(select, select.options[index].value);
    closeAll();
    refresh(wrap);
    select.dispatchEvent(new Event('change', { bubbles: true }));
  }

  function upgrade(select){
    if(select.dataset.swiftPicker || isSkipped(select) || isHiddenSelect(select)) return;
    select.dataset.swiftPicker = '1';
    var wrap = document.createElement('div');
    wrap.className = ('swift-picker ' + (select.className || '')).trim();
    wrap.tabIndex = 0;
    wrap.setAttribute('role', 'combobox');
    if(select.style.width) wrap.style.width = select.style.width;
    select.parentNode.insertBefore(wrap, select);
    wrap.appendChild(select);
    select.classList.add('up-native-select');
    var selection = document.createElement('span');
    selection.className = 'swift-picker-selection';
    var value = document.createElement('span');
    value.className = 'up-select-value';
    selection.appendChild(value);
    selection.insertAdjacentHTML('beforeend', ARROW);
    wrap.appendChild(selection);
    var menu = document.createElement('ul');
    menu.className = 'swift-picker-menu';
    menu.hidden = true;
    document.body.appendChild(menu);
    wrap.up = { select: select, menu: menu, value: value, sig: '' };

    Object.defineProperty(select, 'value', {
      configurable: true,
      get: function(){ return valueDesc.get.call(this); },
      set: function(next){ valueDesc.set.call(this, next); queueRefresh(select); }
    });

    wrap.addEventListener('click', function(event){ event.preventDefault(); open(wrap); });
    wrap.addEventListener('keydown', function(event){
      if(event.key === 'Enter' || event.key === ' '){
        event.preventDefault();
        var active = wrap.classList.contains('swift-picker-open') && wrap.up.menu.querySelector('.up-active[data-index]');
        if(active) pick(wrap, Number(active.dataset.index)); else open(wrap);
        return;
      }
      if(event.key === 'Escape'){ closeAll(); return; }
      if(!['ArrowDown','ArrowUp','Home','End'].includes(event.key)) return;
      event.preventDefault();
      if(!wrap.classList.contains('swift-picker-open')){ open(wrap); return; }
      var items = [].slice.call(wrap.up.menu.children).filter(function(item){ return !item.classList.contains('up-empty') && item.dataset.index !== undefined && !select.options[Number(item.dataset.index)].disabled; });
      if(!items.length) return;
      var current = items.findIndex(function(item){ return item.classList.contains('up-active'); });
      var next = event.key === 'Home' ? 0 : event.key === 'End' ? items.length - 1 : event.key === 'ArrowDown' ? Math.min(items.length - 1, current + 1) : Math.max(0, current < 0 ? items.length - 1 : current - 1);
      items.forEach(function(item, index){ item.classList.toggle('up-active', index === next); });
    });
    menu.addEventListener('mousedown', function(event){ event.preventDefault(); });
    menu.addEventListener('click', function(event){
      var item = event.target.closest('li');
      if(item && item.dataset.index !== undefined) pick(wrap, Number(item.dataset.index));
    });
    select.addEventListener('change', function(){ queueRefresh(select); });

    new MutationObserver(function(){ queueRefresh(select); }).observe(select, {
      childList: true,
      subtree: true,
      characterData: true,
      attributes: true,
      attributeFilter: ['hidden', 'disabled', 'class', 'style']
    });
    refresh(wrap);
  }

  var pending = new Set();
  function queueRefresh(select){
    if(pending.has(select)) return;
    pending.add(select);
    Promise.resolve().then(function(){
      pending.delete(select);
      var wrap = select.parentElement;
      if(wrap && wrap.up && wrap.up.select === select) refresh(wrap);
      else if(!select.dataset.swiftPicker) upgrade(select);
    });
  }

  function scan(root){
    (root || document).querySelectorAll('select').forEach(function(select){ upgrade(select); });
  }

  document.addEventListener('pointerdown', function(event){
    if(openWrap && !openWrap.contains(event.target) && !openWrap.up.menu.contains(event.target)) closeAll();
  }, true);
  document.addEventListener('keydown', function(event){
    if(event.key === 'Escape') closeAll();
  });
  document.addEventListener('scroll', function(){
    if(openWrap) position(openWrap);
  }, true);
  window.addEventListener('resize', function(){
    if(openWrap) position(openWrap);
  });
  document.addEventListener('reset', function(event){
    var form = event.target;
    if(!(form instanceof HTMLFormElement)) return;
    Promise.resolve().then(function(){ form.querySelectorAll('select').forEach(queueRefresh); });
  }, true);

  new MutationObserver(function(records){
    for(var i = 0; i < records.length; i++){
      var node = records[i].target;
      if(records[i].type === 'attributes'){
        if(node.tagName === 'SELECT') queueRefresh(node);
        continue;
      }
      records[i].addedNodes.forEach(function(added){
        if(added.nodeType !== 1) return;
        if(added.tagName === 'SELECT') upgrade(added);
        else if(added.querySelectorAll) added.querySelectorAll('select').forEach(function(select){ upgrade(select); });
      });
    }
  }).observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ['hidden', 'style', 'class'] });

  
  function upgradeCombobox(input, getOptions){
    if(!input || input.dataset.swiftCombobox) return input ? input.closest('.swift-combobox') : null;
    input.dataset.swiftCombobox = '1';
    input.classList.add('swift-combobox-input');
    input.autocomplete = 'off';

    var wrap = document.createElement('div');
    wrap.className = 'swift-combobox' + (input.disabled ? ' swift-picker-disabled' : '');
    wrap.tabIndex = -1;
    wrap.setAttribute('role', 'combobox');

    input.parentNode.insertBefore(wrap, input);
    var selection = document.createElement('div');
    selection.className = 'swift-picker-selection';
    selection.appendChild(input);
    selection.insertAdjacentHTML('beforeend', ARROW);
    wrap.appendChild(selection);

    var arrow = selection.querySelector('.up-select-arrow');
    var menu = document.createElement('ul');
    menu.className = 'swift-picker-menu';
    menu.hidden = true;
    document.body.appendChild(menu);

    wrap.up = {
      isCombobox: true,
      input: input,
      menu: menu,
      arrow: arrow,
      getOptions: getOptions || function(){ return []; },
      filterActive: false
    };

    function esc(s){
      return String(s || '').replace(/[&<>"]/g, function(ch){
        return ch === '&' ? '&amp;' : ch === '<' ? '&lt;' : ch === '>' ? '&gt;' : '&quot;';
      });
    }

    function rebuild(){
      var raw = typeof wrap.up.getOptions === 'function' ? wrap.up.getOptions() : (wrap.up.getOptions || []);
      var items = [];
      for(var i = 0; i < raw.length; i++){
        var opt = raw[i];
        if(typeof opt === 'string'){
          items.push({ value: opt, label: opt, sublabel: '' });
        } else if(opt && typeof opt === 'object'){
          var val = opt.value !== undefined ? String(opt.value) : (opt.code || opt.id || opt.label || '');
          var lbl = opt.label !== undefined ? String(opt.label) : (opt.name || val);
          var sub = opt.sublabel || opt.sub || (opt.code && opt.label ? opt.label : '');
          items.push({ value: val, label: lbl, sublabel: sub });
        }
      }

      var current = (input.value || '').trim();
      var filter = (wrap.up.filterActive ? current : '').toLowerCase();

      var list = items;
      if(filter){
        list = items.filter(function(it){
          return it.value.toLowerCase().indexOf(filter) !== -1 ||
                 it.label.toLowerCase().indexOf(filter) !== -1 ||
                 (it.sublabel && it.sublabel.toLowerCase().indexOf(filter) !== -1);
        });
      }

      if(!list.length){
        menu.innerHTML = '<li class="up-empty" style="color:var(--muted-2);cursor:default;font-style:italic">None</li>';
        return;
      }

      var html = '';
      for(var j = 0; j < list.length; j++){
        var item = list[j];
        var isSelected = current && (item.value.toLowerCase() === current.toLowerCase() || item.label.toLowerCase() === current.toLowerCase());
        html += '<li data-value="' + esc(item.value) + '" class="' + (isSelected ? 'up-selected' : '') + ' up-combobox-item" role="option">';
        if(item.sublabel && item.sublabel !== item.value){
          html += '<span class="up-combobox-val">' + esc(item.value) + '</span><span class="up-combobox-sub">' + esc(item.sublabel) + '</span>';
        } else {
          html += '<span class="up-combobox-val">' + esc(item.label) + '</span>';
        }
        html += '</li>';
      }
      menu.innerHTML = html;
    }

    function open(resetFilter){
      if(input.disabled) return;
      if(openWrap === wrap && !resetFilter){
        closeAll();
        return;
      }
      closeAll();
      wrap.up.filterActive = !resetFilter;
      rebuild();
      openWrap = wrap;
      wrap.classList.add('swift-picker-open');
      position(wrap);
      var active = menu.querySelector('.up-selected') || menu.querySelector('li:not(.up-empty)');
      if(active) menu.scrollTop = Math.max(0, active.offsetTop - menu.clientHeight / 2 + active.offsetHeight / 2);
    }

    function pick(value){
      input.value = value;
      closeAll();
      refresh();
      wrap.up.picking = true;
      try {
        input.dispatchEvent(new Event('input', { bubbles: true }));
        input.dispatchEvent(new Event('change', { bubbles: true }));
      } finally {
        wrap.up.picking = false;
      }
    }

    function refresh(){
      wrap.hidden = isHiddenSelect(input);
      wrap.classList.toggle('swift-picker-disabled', !!input.disabled);
      if(openWrap === wrap){
        if(input.disabled) closeAll();
        else position(wrap);
      }
    }

    input.addEventListener('focus', function(){
      if(!input.disabled && !wrap.classList.contains('swift-picker-open')){
        open(true);
      }
    });
    input.addEventListener('click', function(){
      if(!input.disabled && !wrap.classList.contains('swift-picker-open')){
        open(true);
      }
    });

    arrow.addEventListener('click', function(event){
      event.preventDefault();
      event.stopPropagation();
      if(input.disabled) return;
      if(wrap.classList.contains('swift-picker-open')){
        closeAll();
      } else {
        input.focus();
        open(true);
      }
    });

    input.addEventListener('input', function(){
      if(input.disabled || wrap.up.picking) return;
      wrap.up.filterActive = true;
      if(!wrap.classList.contains('swift-picker-open')){
        open(false);
      } else {
        rebuild();
        position(wrap);
      }
    });

    menu.addEventListener('mousedown', function(event){
      event.preventDefault();
    });
    menu.addEventListener('click', function(event){
      var item = event.target.closest('li');
      if(item && item.dataset.value !== undefined){
        pick(item.dataset.value);
      }
    });

    input.addEventListener('keydown', function(event){
      if(event.key === 'Escape'){
        if(wrap.classList.contains('swift-picker-open')){
          event.preventDefault();
          event.stopPropagation();
          closeAll();
        }
        return;
      }
      if(event.key === 'ArrowDown' || event.key === 'ArrowUp'){
        event.preventDefault();
        if(!wrap.classList.contains('swift-picker-open')){
          open(true);
          return;
        }
        var items = [].slice.call(menu.children).filter(function(it){ return !it.classList.contains('up-empty'); });
        if(!items.length) return;
        var current = items.findIndex(function(it){ return it.classList.contains('up-active'); });
        var next = event.key === 'ArrowDown' ? Math.min(items.length - 1, current + 1) : Math.max(0, current < 0 ? items.length - 1 : current - 1);
        items.forEach(function(it, idx){ it.classList.toggle('up-active', idx === next); });
        if(items[next]){
          menu.scrollTop = Math.max(0, items[next].offsetTop - menu.clientHeight / 2 + items[next].offsetHeight / 2);
        }
        return;
      }
      if(event.key === 'Enter'){
        if(wrap.classList.contains('swift-picker-open')){
          var active = menu.querySelector('li.up-active');
          if(active && active.dataset.value !== undefined){
            event.preventDefault();
            event.stopPropagation();
            pick(active.dataset.value);
          }
        }
      }
    });

    wrap.addEventListener('focusout', function(e){
      if(openWrap === wrap && (!e.relatedTarget || (!wrap.contains(e.relatedTarget) && !menu.contains(e.relatedTarget)))){
        closeAll();
      }
    });

    new MutationObserver(function(){
      refresh();
    }).observe(input, { attributes: true, attributeFilter: ['disabled', 'hidden', 'class', 'style'] });

    var inputDisabledDesc = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'disabled') || (typeof Element !== 'undefined' ? Object.getOwnPropertyDescriptor(Element.prototype, 'disabled') : null);
    if(inputDisabledDesc && inputDisabledDesc.set){
      Object.defineProperty(input, 'disabled', {
        configurable: true,
        get: function(){ return inputDisabledDesc.get.call(this); },
        set: function(val){
          inputDisabledDesc.set.call(this, val);
          refresh();
        }
      });
    }

    refresh();
    return wrap;
  }

  // The shared optical runtime owns navigation lenses and pointer gestures.
  function initDraggableLiquidSliders(){
    // Compatibility hook for retired decoration.
  }

  injectStyle();
  if(document.readyState === 'loading'){
    document.addEventListener('DOMContentLoaded', function(){
      scan();
      initDraggableLiquidSliders();
    });
  } else {
    scan();
    initDraggableLiquidSliders();
  }

  window.swiftPicker = {
    scan: scan,
    upgrade: upgrade,
    upgradeCombobox: upgradeCombobox,
    closeAll: closeAll,
    initSliders: initDraggableLiquidSliders
  };

  window.upUpgradeSelects = scan;
  window.upUpgradeCombobox = upgradeCombobox;
  window.upCloseSelects = closeAll;
  window.initDraggableLiquidSliders = initDraggableLiquidSliders;
})();
