#!/usr/bin/env node
// 开放平台集成方自助验收工具(Q4 M4/M5,docs/integration/open-platform.md)。
// 零依赖(Node>=22):对指定环境跑完整验收清单——签名请求、ping、沙箱样例订单、
// 生产隔离(沙箱查不到生产单)、Webhook 验答回放。
//
// 用法:
//   node scripts/openplat-selftest.mjs --base http://127.0.0.1:28080 --app op_xxx --secret ops_xxx
//   node scripts/openplat-selftest.mjs --base ... --app ... --secret ... --skip-prod-check
//
// 退出码:全部通过 0;任一失败 1(逐项打印 PASS/FAIL)。

import { createHmac, createHash } from "node:crypto";

function args() {
  const out = { skipProdCheck: false };
  const argv = process.argv.slice(2);
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === "--base") out.base = argv[++i];
    else if (argv[i] === "--app") out.app = argv[++i];
    else if (argv[i] === "--secret") out.secret = argv[++i];
    else if (argv[i] === "--skip-prod-check") out.skipProdCheck = true;
  }
  return out;
}

const cfg = args();
if (!cfg.base || !cfg.app || !cfg.secret) {
  console.error("usage: openplat-selftest.mjs --base URL --app APPID --secret SECRET [--skip-prod-check]");
  process.exit(2);
}

function sha256hex(s) {
  return createHash("sha256").update(s).digest("hex");
}

function signHeaders(method, path, body = "") {
  const ts = String(Math.floor(Date.now() / 1000));
  const nonce = crypto.randomUUID();
  const canonical = [cfg.app, method, path, ts, nonce, sha256hex(body)].join("\n");
  const sig = createHmac("sha256", cfg.secret).update(canonical).digest("hex");
  return {
    "X-BOSS-AppId": cfg.app,
    "X-BOSS-Timestamp": ts,
    "X-BOSS-Nonce": nonce,
    "X-BOSS-Signature": sig,
  };
}

async function signedGet(path) {
  const res = await fetch(cfg.base + path, { headers: signHeaders("GET", path) });
  let json = null;
  try { json = await res.json(); } catch {}
  return { status: res.status, json };
}

const results = [];
function record(name, ok, detail) {
  results.push({ name, ok, detail });
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${detail ? " — " + detail : ""}`);
}

// 1) ping:验签 + 连通
{
  const r = await signedGet("/api/open/v1/ping");
  const ok = r.status === 200 && r.json?.code === 0;
  record("ping(签名请求)", ok, ok ? `sandbox=${r.json.data.sandbox}` : `status=${r.status} body=${JSON.stringify(r.json)}`);
}

// 2) 沙箱样例清单 + 逐单查询
let sampleOrder;
{
  const r = await signedGet("/api/open/v1/sandbox/samples");
  const orders = r.json?.data?.orders ?? [];
  const ok = r.status === 200 && orders.length > 0;
  record("沙箱样例清单", ok, `${orders.length} 个样例`);
  if (ok) {
    sampleOrder = orders.find((o) => o.endsWith("0003"));
    const q = await signedGet(`/api/open/v1/orders/${sampleOrder}`);
    const dataOk = q.status === 200 && q.json?.data?.orderNo === sampleOrder;
    record("样例订单查询", dataOk, dataOk ? q.json.data.status : `status=${q.status}`);
  }
}

// 3) 生产隔离:沙箱应用查生产单号必须 404(除非显式跳过——例如用生产应用验收)
if (!cfg.skipProdCheck) {
  const r = await signedGet("/api/open/v1/orders/ORD-NOT-EXIST-0000");
  record("沙箱查生产单隔离(404)", r.status === 404, `status=${r.status}`);
}

// 4) Webhook 验答回放:本地复算 v1 签名与样例比对
{
  const r = await signedGet("/api/open/v1/sandbox/samples");
  const s = r.json?.data?.webhook;
  if (s) {
    const v1 = createHmac("sha256", "ops_sandbox_demo")
      .update(`${s.timestamp}\n${sha256hex(s.body)}`)
      .digest("hex");
    record("Webhook 验答回放", v1 === s.expectedV1, `eventId=${s.eventId}`);
  } else {
    record("Webhook 验答回放", false, "样例缺失");
  }
}

const failed = results.filter((r) => !r.ok);
console.log(`\n${results.length - failed.length}/${results.length} 项通过`);
process.exit(failed.length ? 1 : 0);
