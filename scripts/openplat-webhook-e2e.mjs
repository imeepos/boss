#!/usr/bin/env node
// 开放平台 Webhook 端到端自验脚本(M2 投递器真实部署演练)。
// 覆盖事件产生(POST /openplat/apps/:id/test-event)→ 进入待投递(outbox)→
// 实际送达(POST 到订阅端点,带 HMAC-SHA256 签名)→ 结果落库(MarkResult
// status=1/http_status=200/attempts>=1);失败路径与恢复路径均可观测。
// 一键执行 + 演练后无残留(单事务 docker psql DELETE 唯一标记 openplat-e2e-%)。
//
// 设计要点(对齐 2026-08-30 持久化阶段2 + openplat-selftest.mjs 既有方向):
//   * 零 npm 依赖(node:crypto / node:http / node:child_process);
//   * 接收器跑在脚本进程内,监听 127.0.0.1:19880(避免占用 102 其他常用端口);
//   * 管理面调用需 admin 鉴权(--admin-key 或 ADMIN_API_KEY 环境变量);
//   * 清理走 docker exec -i boss-infra-postgres-1 psql 单事务(参照
//     acceptance-cleanup.sh 既有清理风格),按唯一标记 openplat-e2e-<ts>
//     回收 open_webhook_deliveries / open_webhook_subscriptions / open_apps。
//
// 用法:
//   node scripts/openplat-webhook-e2e.mjs --base http://127.0.0.1:28080 --admin-key <key> [--receiver-host 192.168.0.102]
//   ADMIN_API_KEY=<key> node scripts/openplat-webhook-e2e.mjs --base http://127.0.0.1:28080
//
// 注意:接收器监听 0.0.0.0:<port>(默认19880),boss-server 容器经主机路由到达;
// --receiver-host 必须是主机在容器视角可达的 IP,默认 192.168.0.102。
// 本脚本需在 102 主机(或与 boss-server 同网络可达的节点)上执行。
//
// 退出码:0=全链路 PASS;1=任一断言 FAIL(清理仍执行);2=环境不可用(无 admin key / 网络)。

import { createServer } from "node:http";
import { createHmac, createHash, randomUUID } from "node:crypto";
import { spawnSync } from "node:child_process";
import { setTimeout as sleep } from "node:timers/promises";

const PG_CONTAINER = process.env.BOSS_PG_CONTAINER || "boss-infra-postgres-1";
const RECEIVER_PORT = Number(process.env.BOSS_E2E_RECEIVER_PORT || 19880);
const POLL_TIMEOUT_MS = Number(process.env.BOSS_E2E_TIMEOUT_MS || 45000);
const POLL_INTERVAL_MS = Number(process.env.BOSS_E2E_INTERVAL_MS || 2000);

function parseArgs(argv) {
  const out = { base: "", adminKey: "", receiverHost: "" };
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === "--base") out.base = argv[++i];
    else if (argv[i] === "--admin-key") out.adminKey = argv[++i];
    else if (argv[i] === "--receiver-host") out.receiverHost = argv[++i];
    else if (argv[i] === "--help") {
      console.error("usage: openplat-webhook-e2e.mjs --base URL --admin-key KEY [--receiver-host HOST]");
      process.exit(2);
    }
  }
  if (!out.adminKey) out.adminKey = process.env.ADMIN_API_KEY || "";
  if (!out.receiverHost) out.receiverHost = process.env.BOSS_E2E_RECEIVER_HOST || "192.168.0.102";
  if (!out.base) { console.error("FAIL: --base 未提供"); process.exit(2); }
  if (!out.adminKey) { console.error("FAIL: --admin-key 未提供且 ADMIN_API_KEY 未设置"); process.exit(2); }
  return out;
}

const args = parseArgs(process.argv.slice(2));
const MARK = `openplat-e2e-${new Date().toISOString().replace(/[-:.TZ]/g, "").slice(0, 14)}`;
const base = args.base.replace(/\/$/, "");
const adminHeaders = { "X-API-Key": args.adminKey, "Content-Type": "application/json" };

const checks = [];
function record(name, ok, detail) {
  checks.push({ name, ok, detail });
  console.log(`${ok ? "PASS" : "FAIL"}  ${name}${detail ? " — " + detail : ""}`);
}

async function admin(method, path, body) {
  const init = { method, headers: adminHeaders };
  if (body !== undefined) init.body = JSON.stringify(body);
  const res = await fetch(base + path, init);
  let json = null;
  try { json = await res.json(); } catch {}
  if (!res.ok || (json && json.code !== undefined && json.code !== 0)) {
    const msg = json && json.message ? json.message : `status=${res.status}`;
    throw new Error(`admin ${method} ${path}: ${msg}`);
  }
  return json && json.data !== undefined ? json.data : json;
}

// 接收器:校验 X-BOSS-Signature(t=<ts>,v1=<hex>) 与 SignPayload(secret,ts,body)。
// 仅记录首条 POST(自验只需一发),返回 200。secret 通过可变闭包在创建应用
// 后注入,避免端口重启竞态(EADDRINUSE)。
function startReceiver(secretRef) {
  let gotRequest = null;
  const server = createServer((req, res) => {
    if (gotRequest) { res.writeHead(200); res.end(); return; }
    const chunks = [];
    req.on("data", (c) => chunks.push(c));
    req.on("end", () => {
      const body = Buffer.concat(chunks);
      const sig = (req.headers["x-boss-signature"] || "").toString();
      const m = /^t=(\d+),v1=([0-9a-f]+)$/.exec(sig);
      let sigOk = false;
      if (m && secretRef.value) {
        const bodyHash = createHash("sha256").update(body).digest("hex");
        const want = createHmac("sha256", secretRef.value).update(m[1] + "\n" + bodyHash).digest("hex");
        sigOk = want === m[2];
      }
      gotRequest = {
        event: req.headers["x-boss-event"],
        eventId: req.headers["x-boss-eventid"],
        timestamp: m ? m[1] : "",
        signatureHeader: sig,
        sigOk,
        bodyLen: body.length,
        bodySample: body.toString("utf8").slice(0, 200),
        receivedAt: new Date().toISOString(),
      };
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end('{"ok":true}');
    });
  });
  return new Promise((resolve, reject) => {
    server.once("error", reject);
    // 监听 0.0.0.0 以便 boss-server 容器经网关/NAT 到达;端口短暂占用,脚本结束立即释放。
    server.listen(RECEIVER_PORT, "0.0.0.0", () => resolve({ server, getReq: () => gotRequest }));
  });
}

async function run() {
  console.log(`== webhook E2E self-verification == marker=${MARK} base=${base}`);
  const secretRef = { value: "" };
  const { server, getReq } = await startReceiver(secretRef);
  let app, sub, eventId;
  try {
    app = await admin("POST", "/api/admin/v1/openplat/apps", {
      name: MARK, rateLimitRpm: 60, dailyQuota: 10000, sandbox: false,
    });
    record("创建应用", !!app && !!app.id && !!app.secret, `id=${app && app.id} appId=${app && app.appId}`);
    secretRef.value = (app && app.secret) || "";

    sub = await admin("POST", `/api/admin/v1/openplat/apps/${app.id}/subscriptions`, {
      eventType: "openplat.test",
      endpointUrl: `http://${args.receiverHost}:${RECEIVER_PORT}/webhook`,
    });
    record("创建订阅", !!sub && !!sub.id, `subId=${sub && sub.id}`);

    const emit = await admin("POST", `/api/admin/v1/openplat/apps/${app.id}/test-event`, {});
    eventId = emit.eventId;
    record("发出测试事件", !!eventId, `eventId=${eventId} queued=${emit.queued}`);
  } catch (e) {
    record("准备阶段", false, e.message);
    await cleanup({ app, sub });
    server.close();
    return summarize();
  }

  // 轮询投递结果:直到 status=1 或超时。
  let delivery = null;
  const deadline = Date.now() + POLL_TIMEOUT_MS;
  while (Date.now() < deadline) {
    await sleep(POLL_INTERVAL_MS);
    let list;
    try {
      const resp = await admin("GET", `/api/admin/v1/openplat/deliveries?subscriptionId=${sub.id}`);
      list = (resp && resp.items) || [];
    } catch (e) {
      record("查询投递列表", false, e.message);
      break;
    }
    const found = list.find((d) => d.eventId === eventId);
    if (found && found.status === 1) { delivery = found; break; }
    if (found && found.status === 2) {
      record("投递进入死信", false, `attempts=${found.attempts} http=${found.httpStatus} err=${found.lastError}`);
      break;
    }
  }
  record("投递完成(status=1)", !!delivery, delivery
    ? `deliveryId=${delivery.id} attempts=${delivery.attempts} httpStatus=${delivery.httpStatus} deliveredAt=${delivery.deliveredAt}`
    : `timeout ${POLL_TIMEOUT_MS}ms`);

  const got = getReq();
  record("接收器收到 POST", !!got, got
    ? `event=${got.event} eventId=${got.eventId} bodyLen=${got.bodyLen} receivedAt=${got.receivedAt}`
    : "未收到任何 POST");
  if (got) {
    record("签名校验通过", got.sigOk && got.eventId === eventId,
      `sigOk=${got.sigOk} eventId-header=${got.eventId} vs sent=${eventId}`);
  }

  await cleanup({ app, sub });
  server.close();
  return summarize();
}

async function cleanup({ app, sub }) {
  // 管理面删除订阅(若已建);docker psql 单事务清掉本次造数,真正零残留。
  if (sub && sub.id) {
    try { await admin("DELETE", `/api/admin/v1/openplat/subscriptions/${sub.id}`); }
    catch (e) { console.log(`WARN: delete subscription: ${e.message}`); }
  }
  if (app && app.id) {
    try { await admin("PUT", `/api/admin/v1/openplat/apps/${app.id}/status`, { status: 0 }); }
    catch (e) { console.log(`WARN: disable app: ${e.message}`); }
  }
  // 真零残留:删 deliveries + subscriptions + apps,按 MARK 前缀。
  const sql = `BEGIN;\n` +
    `DELETE FROM open_webhook_deliveries WHERE subscription_id IN\n` +
    `  (SELECT id FROM open_webhook_subscriptions WHERE app_id IN\n` +
    `    (SELECT id FROM open_apps WHERE name LIKE 'openplat-e2e-%'));\n` +
    `DELETE FROM open_webhook_subscriptions WHERE app_id IN\n` +
    `  (SELECT id FROM open_apps WHERE name LIKE 'openplat-e2e-%');\n` +
    `DELETE FROM open_apps WHERE name LIKE 'openplat-e2e-%';\n` +
    `COMMIT;\n`;
  const r = spawnSync("docker", ["exec", "-i", PG_CONTAINER, "psql", "-U", "boss", "-d", "boss", "-v", "ON_ERROR_STOP=1"],
    { input: sql, encoding: "utf8" });
  if (r.status !== 0) {
    console.log(`WARN: docker psql cleanup failed: ${r.stderr || r.stdout}`);
  } else {
    console.log(`清理: docker psql 删除 openplat-e2e-% 造数成功`);
  }
}

function summarize() {
  const failed = checks.filter((c) => !c.ok).length;
  console.log(`\n${checks.length - failed}/${checks.length} 项通过`);
  process.exit(failed ? 1 : 0);
}

run().catch((e) => { console.error("FATAL", e); process.exit(2); });