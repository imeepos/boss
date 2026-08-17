// store.js —— uuid 主键内存表引擎,提供真实增删改查
// 原地给种子行补 uuid(不改数组引用),派生视图零改动即可共享主键。
'use strict';

const crypto = require('crypto');

function createTable(name, rows) {
  for (const row of rows) {
    if (!row.uuid) row.uuid = crypto.randomUUID();
  }
  const find = (uuid) => rows.find((r) => r.uuid === uuid) || null;
  return {
    name,
    rows,
    list: () => rows.slice(),
    find,
    insert(data) {
      const row = { ...data, uuid: crypto.randomUUID() };
      rows.push(row);
      return row;
    },
    update(uuid, patch) {
      const row = find(uuid);
      if (!row) return null;
      const safe = { ...(patch || {}) };
      delete safe.uuid;
      Object.assign(row, safe);
      return row;
    },
    remove(uuid) {
      const i = rows.findIndex((r) => r.uuid === uuid);
      if (i < 0) return false;
      rows.splice(i, 1);
      return true;
    },
  };
}

module.exports = { createTable };
