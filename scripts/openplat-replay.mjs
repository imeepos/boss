#!/usr/bin/env node
// 开放平台回放工具(Q4 M5):按 fixture 回放请求并比对期望响应。
// 用法:
//   node scripts/openplat-replay.mjs --base http://127.0.0.1:28080 --app op_xxx --secret ops_xxx \
//        [--cases scripts/openplat/fixtures/sandbox-replay.json]
// fixture 格式见同目录 sandbox-replay.json;期望 data 为子集匹配(忽略 createdAt 等时变字段)。
// 退出码:全部通过 0,否则 1。

import { createHmac, createHash } from "node:crypto";
import { readFileSync } from "node:fs";

function args() {
  const out = { cases: new URL("./openplat/fixtures/sandbox-replay.json", import.meta.url).pathname };
  const argv = process.argv.slice(2);
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === "--base") out.base = argv[++i];
    else if (argv[i] === "--app") out.app = argv[++i];
    else if (argv[i] === "--secret") out.secret = argv[++i];
    else if (argv[i] === "--cases") out.cases = argv[++i];
  }
  return out;
}

const cfg = args();
if (!cfg.base || !cfg.app || !cfg.secret) {
  console.error("usage: openplat-replay.mjs --base URL --app APPID --secret SECRET [--cases FILE]");
  process.exit(2);
}

function signedHeaders(method, path, body = "") {
  const ts = String(Math.floor(Date.now() / 1000));
  const nonce = crypto.randomUUID();
  const canonical = [cfg.app, method, path, ts, nonce, createHash("sha256").update(body).digest("hex")].join("\n");
  return {
    "X-BOSS-AppId": cfg.app,
    "X-BOSS-Timestamp": ts,
    "X-BOSS-Nonce": nonce,
    "X-BOSS-Signature": createHmac("sha256", cfg.secret).update(canonical).digest("hex"),
  };
}

function subsetMatches(expect, actual) {
  for (const [k, v] of Object.entries(expect)) {
    if (actual?.[k] !== v) return false;
  }
  return true;
}

const fixtures = JSON.parse(readFileSync(cfg.cases, "utf8"));
let failed = 0;
for (const tc of fixtures.cases) {
  const { method, path, unsigned } = tc.request;
  const headers = unsigned ? {} : signedHeaders(method, path);
  let status = 0,
    json = null;
  try {
    const res = await fetch(cfg.base + path, { method, headers });
    status = res.status;
    try { json = await res.json(); } catch {}
  } catch (e) {
    console.log(`FAIL  ${tc.name} — request error: ${e.message}`);
    failed++;
    continue;
  }
  const exp = tc.expect;
  let ok = status === exp.status;
  let why = ok ? "" : `status ${status} != ${exp.status}`;
  if (ok && exp.code !== undefined) {
    ok = json?.code === exp.code;
    if (!ok) why = `code ${json?.code} != ${exp.code}`;
  }
  if (ok && exp.data) {
    ok = subsetMatches(exp.data, json?.data);
    if (!ok) why = `data ${JSON.stringify(json?.data)} !~ ${JSON.stringify(exp.data)}`;
  }
  if (ok && exp.dataFields) {
    ok = exp.dataFields.every((f) => json?.data && f in json.data);
    if (!ok) why = `dataFields missing in ${JSON.stringify(json?.data)}`;
  }
  console.log(`${ok ? "PASS" : "FAIL"}  ${tc.name}${why ? " — " + why : ""}`);
  if (!ok) failed++;
}
console.log(`\n${fixtures.cases.length - failed}/${fixtures.cases.length} 项通过`);
process.exit(failed ? 1 : 0);
