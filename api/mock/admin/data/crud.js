// crud.js —— db 事实库通用增删改查 REST 路由(uuid 主键)
// /api/admin/v1/crud/{table} 与 /crud/{table}/{uuid},返回 null 交由上层 404。
'use strict';

const db = require('../../db.js');

const ALLOWED = new Set(db.TABLE_NAMES);

function tableOf(name) {
  return ALLOWED.has(name) ? db.tables[name] : null;
}

module.exports = {
  'GET /crud/{table}': ({ params }) => {
    const t = tableOf(params.table);
    return t ? { items: t.list(), total: t.rows.length } : null;
  },
  'GET /crud/{table}/{uuid}': ({ params }) => {
    const t = tableOf(params.table);
    const row = t && t.find(params.uuid);
    return row || null;
  },
  'POST /crud/{table}': ({ params, body }) => {
    const t = tableOf(params.table);
    if (!t || !body || typeof body !== 'object') return null;
    const row = t.insert(body);
    return { created: true, row };
  },
  'PUT /crud/{table}/{uuid}': ({ params, body }) => {
    const t = tableOf(params.table);
    const row = t && t.update(params.uuid, body);
    return row ? { updated: true, row } : null;
  },
  'DELETE /crud/{table}/{uuid}': ({ params }) => {
    const t = tableOf(params.table);
    return t && t.remove(params.uuid) ? { deleted: true, uuid: params.uuid } : null;
  },
};
