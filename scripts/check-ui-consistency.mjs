#!/usr/bin/env node
/**
 * UI consistency gate: hardcoded colors in both Android clients may only shrink.
 * Modes: gate (default) | --write-baseline | --stats | --matrix [--out <file>].
 * Dependency-free so it runs in CI and in any worktree.
 */
import { readdir, readFile, writeFile } from 'node:fs/promises';
import { relative, resolve } from 'node:path';
import process from 'node:process';

const repoRoot = resolve(import.meta.dirname, '..');
const PREFIX = '[ui-consistency]';
const BASELINE = 'scripts/ui-consistency-baseline.json';
// 6 位 RGB 或 8 位 ARGB 字面量;注释里的也算违规,门禁从严
const COLOR_RE = /Color\(0x[0-9A-Fa-f]{6,8}\)/g;

const USER_ROOT = 'mobile/user/android/app/src/main/java/com/ymm/boss/user';
const WORKER_ROOT = 'mobile/worker/android/app/src/main/java/com/ymm/boss/worker/ui';
const SCANS = [
  {
    end: 'user',
    dirs: [`${USER_ROOT}/page`, `${USER_ROOT}/ui`],
    exempt: [`${USER_ROOT}/ui/Theme.kt`, `${USER_ROOT}/ui/theme/Color.kt`, `${USER_ROOT}/ui/theme/RnPalette.kt`],
  },
  { end: 'worker', dirs: [WORKER_ROOT], exempt: [`${WORKER_ROOT}/theme/Color.kt`] },
];

async function walkKt(dir) {
  let entries;
  try {
    entries = await readdir(dir, { withFileTypes: true });
  } catch {
    return [];
  }
  const files = [];
  for (const entry of entries) {
    const full = resolve(dir, entry.name);
    if (entry.isDirectory()) files.push(...(await walkKt(full)));
    else if (entry.name.endsWith('.kt')) files.push(full);
  }
  return files;
}

async function scanFiles() {
  const files = [];
  for (const scan of SCANS) {
    for (const dir of scan.dirs) files.push(...(await walkKt(resolve(repoRoot, dir))));
  }
  return files.sort();
}

function isExempt(rel) {
  return SCANS.some((scan) => scan.exempt.includes(rel));
}

// 边界正则:名字前不能是标识符字符,既不漏行首也不被 isEmpty( 这类后缀污染
function refCount(text, name) {
  return [...text.matchAll(new RegExp(`(^|[^A-Za-z0-9_])${name}\\(`, 'gm'))].length;
}

// 捕获 name(...) 的实参列表;不含嵌套括号,覆盖现有调用形态
function argLists(text, name) {
  return [...text.matchAll(new RegExp(`(^|[^A-Za-z0-9_])${name}\\(([^()]*)\\)`, 'g'))].map((m) => m[2]);
}

function dpValues(text, name) {
  const values = [];
  for (const args of argLists(text, name)) {
    for (const dp of args.matchAll(/(\d+(?:\.\d+)?)\.dp/g)) values.push(dp[1]);
  }
  return values;
}

function weightSet(text) {
  return [...new Set([...text.matchAll(/FontWeight\.(\w+)/g)].map((m) => m[1]))].sort();
}

function skeletonType(text) {
  if (refCount(text, 'PinnedGradientPage')) return 'PinnedGradientPage';
  if (refCount(text, 'TopBar')) return 'TopBar';
  if (refCount(text, 'Scaffold')) return 'Scaffold';
  return '无';
}

async function collectViolations() {
  const violations = new Map();
  for (const file of await scanFiles()) {
    const rel = relative(repoRoot, file);
    if (isExempt(rel)) continue;
    const count = [...(await readFile(file, 'utf8')).matchAll(COLOR_RE)].length;
    if (count) violations.set(rel, count);
  }
  return violations;
}

function total(map) {
  return [...map.values()].reduce((sum, n) => sum + n, 0);
}

async function readBaseline() {
  try {
    return JSON.parse(await readFile(resolve(repoRoot, BASELINE), 'utf8'));
  } catch {
    return null;
  }
}

async function gate() {
  const violations = await collectViolations();
  const baseline = await readBaseline();
  if (!baseline) {
    for (const [rel, count] of violations) console.log(`${PREFIX} VIOLATION new file ${rel}: ${count}`);
    if (violations.size) {
      console.log(`${PREFIX} FAIL ${violations.size} file(s) / ${total(violations)} occurrence(s), no baseline yet; run: node scripts/check-ui-consistency.mjs --write-baseline`);
      process.exitCode = 1;
    } else {
      console.log(`${PREFIX} OK no violations, no baseline needed`);
    }
    return;
  }
  const base = baseline.violations ?? {};
  let failed = false;
  for (const [rel, count] of violations) {
    const prev = base[rel];
    if (prev === undefined) {
      console.log(`${PREFIX} VIOLATION new file ${rel}: ${count}`);
      failed = true;
    } else if (count > prev) {
      console.log(`${PREFIX} VIOLATION ${rel}: ${count} > baseline ${prev}`);
      failed = true;
    } else if (count < prev) {
      console.log(`${PREFIX} improved ${rel}: ${count} < baseline ${prev}, keep going`);
    }
  }
  for (const rel of Object.keys(base)) {
    if (!violations.has(rel)) console.log(`${PREFIX} improved ${rel}: cleared baseline ${base[rel]}, nice`);
  }
  if (failed) {
    console.log(`${PREFIX} FAIL ${violations.size} file(s) / ${total(violations)} occurrence(s) over baseline (baseline: ${Object.keys(base).length} file(s) / ${total(new Map(Object.entries(base)))})`);
    process.exitCode = 1;
  } else {
    console.log(`${PREFIX} OK ${violations.size} file(s) / ${total(violations)} occurrence(s), within baseline`);
  }
}

async function writeBaseline() {
  const violations = await collectViolations();
  const payload = {
    generatedAt: new Date().toISOString(),
    exempt: Object.fromEntries(SCANS.map((scan) => [scan.end, scan.exempt])),
    violations: Object.fromEntries([...violations].sort(([a], [b]) => (a < b ? -1 : 1))),
  };
  await writeFile(resolve(repoRoot, BASELINE), `${JSON.stringify(payload, null, 2)}\n`);
  console.log(`${PREFIX} baseline written to ${BASELINE}: ${violations.size} file(s) / ${total(violations)} occurrence(s)`);
}

function frequency(values) {
  const counts = new Map();
  for (const value of values) counts.set(value, (counts.get(value) ?? 0) + 1);
  return [...counts].sort(([, a], [, b]) => b - a);
}

function printFreq(title, pairs, unit) {
  console.log(`${PREFIX} ${title}: ${pairs.map(([v, n]) => `${v}${unit} x${n}`).join(', ') || '(none)'}`);
}

async function stats() {
  const files = await scanFiles();
  const corner = [];
  const fontSize = [];
  const padding = [];
  const weight = [];
  for (const file of files) {
    const text = await readFile(file, 'utf8');
    corner.push(...dpValues(text, 'RoundedCornerShape'));
    for (const m of text.matchAll(/fontSize\s*=\s*(\d+(?:\.\d+)?)\.sp/g)) fontSize.push(m[1]);
    padding.push(...dpValues(text, 'padding'));
    weight.push(...[...text.matchAll(/FontWeight\.(\w+)/g)].map((m) => m[1]));
  }
  console.log(`${PREFIX} stats over ${files.length} file(s), sorted by frequency`);
  printFreq('RoundedCornerShape', frequency(corner), 'dp');
  printFreq('fontSize', frequency(fontSize), 'sp');
  printFreq('padding', frequency(padding), 'dp');
  printFreq('FontWeight', frequency(weight), '');
}

async function matrix(outArg) {
  const rows = [];
  for (const file of await scanFiles()) {
    const text = await readFile(file, 'utf8');
    const rel = relative(repoRoot, file);
    const colors = [...text.matchAll(COLOR_RE)].length;
    const weights = weightSet(text).join(',') || '-';
    rows.push(`| ${rel} | ${skeletonType(text)} | ${colors} | ${refCount(text, 'Button')} | ${weights} |`);
  }
  const body = ['# 页面骨架四要素矩阵', '', '| 文件 | 骨架 | 硬编码颜色 | Button( | FontWeight |', '|---|---|---|---|---|', ...rows, ''].join('\n');
  if (outArg) {
    const out = resolve(repoRoot, outArg);
    await writeFile(out, body);
    console.log(`${PREFIX} matrix written to ${relative(repoRoot, out)}: ${rows.length} row(s)`);
  } else {
    process.stdout.write(body);
  }
}

const args = process.argv.slice(2);
const flagValue = (name) => {
  const index = args.indexOf(name);
  return index >= 0 ? args[index + 1] : undefined;
};

if (args.includes('--write-baseline')) await writeBaseline();
else if (args.includes('--stats')) await stats();
else if (args.includes('--matrix')) await matrix(flagValue('--out'));
else await gate();
