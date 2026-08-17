// 管理后台共享 UI 组件层:弹框/表单/分页/Toast/确认框/防抖。
// 页面内联脚本直接使用 window.UI.*,避免每个页面重复造轮子。
// 内部实现基于 jQuery($),页面也可直接使用 $ 简化 DOM 操作。
(function (global, $) {
  'use strict';

  function esc(s) {
    return String(s == null ? '' : s).replace(/[&<>"]/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c];
    });
  }

  // ---- Toast 轻提示 ----
  function toast(msg, type) {
    var cls = type === 'err' ? 'toast-err' : type === 'ok' ? 'toast-ok' : '';
    var el = $('<div class="toast ' + cls + '"></div>').text(msg);
    var box = $('.toast-box');
    if (!box.length) box = $('<div class="toast-box"></div>').appendTo('body');
    el.appendTo(box);
    setTimeout(function () { el.addClass('out'); }, 2400);
    setTimeout(function () { el.remove(); }, 2800);
  }

  // ---- Modal 弹框 ----
  // opts: { title, body(html), okText, cancelText, onOk, width }
  // onOk 返回 false 阻止关闭(校验失败);返回 Promise 失败也阻止关闭。
  function modal(opts) {
    var mask = $(
      '<div class="modal-mask"><div class="modal" style="max-width:' + (opts.width || 520) + 'px">' +
      '<div class="modal-head"><span>' + esc(opts.title || '') + '</span><span class="modal-x">×</span></div>' +
      '<div class="modal-body">' + (opts.body || '') + '</div>' +
      '<div class="modal-foot">' +
      '<button class="btn modal-cancel">' + esc(opts.cancelText || '取消') + '</button>' +
      '<button class="btn btn-primary modal-ok">' + esc(opts.okText || '确定') + '</button>' +
      '</div></div></div>'
    ).appendTo('body');
    function close() { mask.remove(); }
    mask.find('.modal-x, .modal-cancel').on('click', close);
    mask.on('click', function (ev) { if (ev.target === mask[0]) close(); });
    mask.find('.modal-ok').on('click', function () {
      // 对外仍传原生 DOM 节点,兼容页面内联脚本 mask.querySelector 用法
      var r = opts.onOk ? opts.onOk(mask[0], close) : true;
      if (r && typeof r.then === 'function') {
        r.then(function (ok) { if (ok !== false) close(); })
         .catch(function (e) { toast(e.message || '操作失败', 'err'); });
      } else if (r !== false) close();
    });
    return { el: mask[0], close: close };
  }

  // ---- 确认框 ----
  function confirmBox(msg, onOk) {
    modal({
      title: '操作确认', body: '<div style="padding:4px 0">' + esc(msg) + '</div>',
      okText: '确认', onOk: function (m, close) { close(); onOk && onOk(); }
    });
  }

  // ---- 表单字段构造(供 modal body 使用) ----
  // fields: [{ key, label, type(text|select|textarea|number), value, options[{v,t}], required, placeholder }]
  function formFields(fields) {
    var html = '';
    $.each(fields, function (_, f) {
      html += '<div class="form-item"><label>' + esc(f.label) +
        (f.required ? ' <i class="req">*</i>' : '') + '</label><div>';
      if (f.type === 'select') {
        var opts = f.options || [];
        html += '<select class="input" name="' + esc(f.key) + '">';
        $.each(opts, function (_, o) {
          html += '<option value="' + esc(o.v) + '"' +
            (String(o.v) === String(f.value == null ? '' : f.value) ? ' selected' : '') + '>' +
            esc(o.t) + '</option>';
        });
        html += '</select>';
      } else if (f.type === 'textarea') {
        html += '<textarea class="input" name="' + esc(f.key) + '" rows="3" placeholder="' + esc(f.placeholder || '') + '">' +
          esc(f.value == null ? '' : f.value) + '</textarea>';
      } else {
        html += '<input class="input" type="' + (f.type || 'text') + '" name="' + esc(f.key) + '" value="' +
          esc(f.value == null ? '' : f.value) + '" placeholder="' + esc(f.placeholder || '') + '">';
      }
      html += '</div></div>';
    });
    return '<form class="form">' + html + '</form>';
  }

  // 从 modal 容器内收集表单值(object)。required 缺失时 toast 报错并返回 null。
  function readForm(root, fields) {
    var out = {};
    root = $(root);
    for (var i = 0; i < fields.length; i++) {
      var f = fields[i];
      var el = root.find('[name="' + f.key + '"]');
      var v = $.trim(el.val() || '');
      if (f.required && !v) { toast((f.label) + ' 必填', 'err'); el.trigger('focus'); return null; }
      out[f.key] = v;
    }
    return out;
  }

  // ---- 分页 ----
  // 在 container 渲染分页条。opts: { total, page, pageSize, onChange(page) }
  // pageSize 切换时回到第 1 页。数据分片由页面自己 slice。
  function paginate(container, opts) {
    var total = opts.total || 0;
    var page = opts.page || 1;
    var pageSize = opts.pageSize || 10;
    var pages = Math.max(1, Math.ceil(total / pageSize));
    if (page > pages) page = pages;
    var html = '<span>共 ' + total + ' 条</span>' +
      '<select class="input page-size"><option' + (pageSize === 10 ? ' selected' : '') + '>10</option>' +
      '<option' + (pageSize === 20 ? ' selected' : '') + '>20</option>' +
      '<option' + (pageSize === 50 ? ' selected' : '') + '>50</option></select>' +
      '<span class="pg-btns">' +
      '<button class="btn pg-prev"' + (page <= 1 ? ' disabled' : '') + '>上一页</button>' +
      '<span class="pg-cur">' + page + ' / ' + pages + '</span>' +
      '<button class="btn pg-next"' + (page >= pages ? ' disabled' : '') + '>下一页</button>' +
      '</span>';
    container = $(container).html(html);
    container.find('.pg-prev').on('click', function () { if (page > 1) opts.onChange(page - 1, pageSize); });
    container.find('.pg-next').on('click', function () { if (page < pages) opts.onChange(page + 1, pageSize); });
    container.find('.page-size').on('change', function () {
      opts.onChange(1, parseInt(this.value, 10) || 10);
    });
  }

  // ---- 防抖 ----
  function debounce(fn, ms) {
    var t = null;
    return function () {
      var args = arguments, self = this;
      clearTimeout(t);
      t = setTimeout(function () { fn.apply(self, args); }, ms || 300);
    };
  }

  // 表格通用:加载中/空态/错误态
  function tbodyState(cols, text) {
    return '<tr><td colspan="' + cols + '" class="tbl-state">' + esc(text) + '</td></tr>';
  }

  global.UI = {
    esc: esc, toast: toast, modal: modal, confirm: confirmBox,
    formFields: formFields, readForm: readForm,
    paginate: paginate, debounce: debounce, tbodyState: tbodyState,
  };
})(window, jQuery);
