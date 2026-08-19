#!/usr/bin/env node
// shadcn-registry-cli: fetch shadcn/ui registry items, rewrite imports, install deps.
// The official shadcn CLI is non-functional in this environment (ERR_PACKAGE_PATH_NOT_EXPORTED zod/v3),
// so this script acts as a focused replacement for the common add/list/init workflow.
//
// Registry endpoint: https://ui.shadcn.com/r/styles/{style}/{name}.json
// Style comes from components.json -> "style" (default "default").
// All imports starting with "/registry/{style}/..." are rewritten into the project alias tree.
//
// Usage:
//   node .agents/skills/self-evolving/scripts/shadcn.mjs add <name> [name...]
//   node .agents/skills/self-evolving/scripts/shadcn.mjs list
//   node .agents/skills/self-evolving/scripts/shadcn.mjs init
//
// Flags:
//   --root=<dir>          project root (default: CWD)
//   --style=<name>        registry style override
//   --install=<tool>      installer: pnpm|npm|yarn|bun (default pnpm)
//   --force               overwrite existing component files
//   --skip-tokens         don't merge cssVars/tailwind extras
//   --dry-run             only print planned actions

import fs from 'node:fs';
import path from 'node:path';
import { execSync } from 'node:child_process';

const fsp = fs.promises;

// --------------------------------------------------------------------------

function die(msg) { console.error(`ERROR: ${msg}`); process.exit(1); }
function info(msg) { console.log(` ${msg}`); }
function warn(msg) { console.warn(`WARN: ${msg}`); }

function parseCli(argv) {
  const out = { command: null, components: [], flags: {} };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--force') out.flags.force = true;
    else if (a === '--dry-run') out.flags.dryRun = true;
    else if (a === '--skip-tokens') out.flags.skipTokens = true;
    else if (a.startsWith('--root=')) out.flags.root = a.slice(7);
    else if (a.startsWith('--style=')) out.flags.style = a.slice(7);
    else if (a.startsWith('--install=')) out.flags.installer = a.slice(10);
    else if (a.startsWith('--')) die(`unknown flag ${a}`);
    else if (!out.command) out.command = a;
    else out.components.push(a);
  }
  if (!out.command) die('missing command: add|list|init');
  return out;
}

const REGISTRY_PREFIX_PAT = /^(?:@\/registry\/[^/]+\/)/;

// --- config read ---------------------------------------------------------

async function readComponentsJson(root) {
  const p = path.join(root, 'components.json');
  if (!fs.existsSync(p)) die(`no components.json at ${p}`);
  const raw = JSON.parse(await fsp.readFile(p, 'utf8'));
  raw.root = root;
  return raw;
}

// --- network -------------------------------------------------------------

function fetchJson(url) {
  return fetch(url, { headers: { accept: 'application/json' } })
    .then(async r => {
      if (r.status === 404) {
        const body = await r.text().catch(() => r.statusText);
        throw new Error(`registry returned 404 at ${url}: ${String(body).slice(0, 120)}`);
      }
      if (!r.ok) throw new Error(`registry HTTP ${r.status} at ${url}`);
      return r.json();
    });
}

// --- path helpers ---------------------------------------------------------

function mergeDeep(base, incoming) {
  const out = { ...base };
  for (const [k, v] of Object.entries(incoming)) {
    if (v && typeof v === 'object' && !Array.isArray(v) && base[k] && typeof base[k] === 'object') {
      out[k] = mergeDeep(base[k], v);
    } else out[k] = v;
  }
  return out;
}

/**
 * Resolve registry item into real target dir and filename.
 * type → alias mapping (driven by components.json aliases; safe fallbacks).
 */
function resolveTarget(item, file) {
  const root = item.root;
  const aliases = item.aliases || {};
  if (file.target && typeof file.target === 'string' && file.target.trim()) {
    return path.dirname(path.join(root, ...file.target.split('/')));
  }
  const type = file.type || item.type || 'registry:ui';
  const mapped = type === 'registry:ui' ? aliases.ui
    : type === 'registry:hook' ? aliases.hooks
    : aliases[type.replace(/^registry:/, '')] ?? path.join('src', type.replace(/^registry:/, ''));
  if (!mapped) return path.join(root, file.path.split('/').slice(-2, -1)[0]);
  return path.join(root, mapped);
}

function resolveFilename(file) {
  return path.basename(file.path);
}

// --- import rewriting -----------------------------------------------------

// Force a TS-resolvable relative specifier: same-dir siblings must be "./x".
function rel(fileDir, target) {
  let r = path.relative(fileDir, target);
  if (!r.startsWith('.')) r = './' + r;
  return r;
}

function rewriteImports(content, fileDir, item) {
  const root = item.root;
  const aliases = item.aliases || {};
  const style = item.style;

  return content.split('\n').map(line => {
    const m = line.match(/^(import\s+[^'"]*)\s+from\s+("|')([^"']+)("|')\s*;?\s*$/);
    if (!m) return line;
    const prefix = m[1];
    const q = m[2];
    const p = m[3];
    // Only alias-style paths (@/...) are rewritten. Bare (react) and scoped
    // npm packages (@radix-ui/...) are left untouched.
    if (!p.startsWith('@/')) return line;
    const rewritten = REGISTRY_PREFIX_PAT.test(p)
      ? rewriteRegistry(p, fileDir, root, aliases, style)
      : rewriteAlias(p, fileDir, root, aliases);
    return rewritten ? `${prefix} from ${q}${rewritten}${q}` : line;
  }).join('\n');
}

function rewriteRegistry(p, fileDir, root, aliases, style) {
  const rest = p.slice(`@/registry/${style}/`.length);
  const [atom, ...sub] = rest.split('/');
  const restPath = sub.join('/');

  // Special case: canonical utility alias
  if (atom === 'lib' && (restPath === 'utils' || restPath === '')) {
    // `utils` maps to the project utility file (usually src/lib/cn)
    const alias = aliases.utils;
    return rel(fileDir, path.resolve(root, alias || 'src/lib/cn.ts'));
  }

  switch (atom) {
    case 'ui': {
      const dir = path.resolve(root, aliases.ui ?? 'src/components/ui');
      return rel(fileDir, path.join(dir, restPath));
    }
    case 'hooks': {
      const dir = path.resolve(root, aliases.hooks ?? 'src/hooks');
      return rel(fileDir, path.join(dir, restPath));
    }
    default: {
      const mapped = aliases[atom] ?? path.join('src', atom);
      return rel(fileDir, path.resolve(root, mapped, restPath));
    }
  }
}

function rewriteAlias(p, fileDir, root, aliases) {
  // Special-case: registry's canonical cn helper. The project's "utils" alias
  // points at the actual file (e.g. src/lib/cn), so @/lib/utils must resolve
  // to that file rather than a non-existent src/lib/utils.
  if (aliases.utils && (p === '@/lib/utils' || p.endsWith('/lib/utils'))) {
    return rel(fileDir, path.resolve(root, aliases.utils));
  }
  for (const [key, aliasTarget] of Object.entries(aliases)) {
    const bp = `@/${key}/`;
    if (p.startsWith(bp)) {
      const rest = p.slice(bp.length);
      return rel(fileDir, path.resolve(root, aliasTarget, rest));
    }
  }
  return rel(fileDir, path.resolve(root, p.slice(2)));
}

// --- registryIO -----------------------------------------------------------

const itemCache = new Map();

async function loadItem(name, style) {
  const url = `https://ui.shadcn.com/r/styles/${style}/${name}.json`;
  if (!itemCache.has(name)) {
    info(`[registry] fetch ${name}`);
    itemCache.set(name, await fetchJson(url));
  }
  return itemCache.get(name);
}

async function resolveDeps(name, style, seen) {
  seen = seen || new Set();
  if (seen.has(name)) return [];
  const item = await loadItem(name, style);
  let out = [];
  for (const dep of (item.registryDependencies || [])) {
    if (dep === name) continue;
    out = out.concat(await resolveDeps(dep, style, seen));
  }
  const key = `${item.type}::${item.name}`;
  if (!seen.has(key)) { out.push(item); seen.add(key); }
  return out;
}

// --- file writes ----------------------------------------------------------

async function ensureDir(p) { await fsp.mkdir(p, { recursive: true }); }

async function fileExists(p) { try { await fsp.access(p); return true; } catch { return false; } }

async function writeItems(items, flags) {
  for (const it of items) {
    const aliases = it.aliases;
    for (const f of it.files || []) {
      const dir = resolveTarget(it, f);
      const file = resolveFilename(f);
      const out = path.join(dir, file);
      const rel = path.relative(it.root, out);

      if (!flags.force && await fileExists(out)) {
        warn(`skip existing ${rel} (--force to overwrite)`);
        continue;
      }

      const content = rewriteImports(f.content || '', dir, it);

      if (!flags.dryRun) {
        await ensureDir(dir);
        await fsp.writeFile(out, content, 'utf8');
      }
      const tag = flags.dryRun ? '[dry-run]' : '[ok]';
      info(`${tag} ${rel}`);
    }

    // cssVars append is safe and idempotent-ish per key set per run.
    if (!flags.skipTokens && it.cssVars && it.cssVars.light && it.cssVars.dark) {
      if (!flags.dryRun) await mergeCssVarsBlock(it.cssVars, it.root);
      else info('[dry-run] append cssVars block to src/styles.css');
    }
  }
}

async function mergeCssVarsBlock(cssVars, root) {
  const inPath = path.join(root, 'src', 'styles.css');
  if (!fs.existsSync(inPath)) return warn('src/styles.css not found; skip cssVars merge');

  const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
  const open = `/* --- shadcn cssVars ${stamp} --- */\n`;
  const close = `/* --- /shadcn cssVars ${stamp} --- */\n`;
  const body = Object.entries(cssVars)
    .flatMap(([mode, vars]) => [
      `:root[data-theme="${mode}"] {`,
      ...Object.entries(vars).map(([k, v]) => `  --${k}: ${v};`),
      '}',
    ])
    .join('\n');

  // Writes are exclusive per override; we append once per item; repeated runs should be idempotent in spirit.
  // The project's tailwind.config.js will resolve hsl(var(--xxx-foreground)) style vars via @theme.
  const existing = fs.readFileSync(inPath, 'utf8');
  const marker = `/* --- shadcn cssVars ${stamp} --- */`;
  if (existing.includes(marker)) {
    info('same cssVars block already present; skip append');
    return;
  }
  fs.writeFileSync(inPath, existing + '\n' + open + body + '\n' + close, 'utf8');
  info('[ok] appended cssVars to src/styles.css');
}

// --- npm install ----------------------------------------------------------

function installDeps(deps, installer, cwd) {
  if (!deps.length) return null;
  const im = installer || 'pnpm';
  let bin;
  if (im === 'pnpm') bin = 'pnpm add -D';
  else if (im === 'npm') bin = 'npm install -D';
  else if (im === 'yarn') bin = 'yarn add -D';
  else if (im === 'bun') bin = 'bun add -d';
  else die(`unsupported installer ${im}`);
  info(`[dep] ${bin} ${deps.map(quote).join(' ')} ...`);
  try {
    execSync(`${bin} ${deps.map(quote).join(' ')}`, {
      encoding: 'utf8', cwd, stdio: 'inherit',
    });
    info('[dep] install OK');
  } catch (e) {
    warn(`dep install failed: ${String(e?.message || e).slice(0, 180)}; run manually`);
  }
}

function quote(s) { return /[\s'"]/.test(s) ? `"${s}"` : s; }

// --- list -----------------------------------------------------------------

async function cmdList() {
  const url = 'https://ui.shadcn.com/r';
  info(`[registry] ${url}`);
  const data = await fetchJson(url);
  console.log(
    data
      .map(x => `${x.name.padEnd(26)} ${x.type}`)
      .join('\n')
  );
  console.log(`--- ${data.length} items ---`);
}

// --- init -----------------------------------------------------------------

async function cmdInit(flags) {
  const root = path.resolve(flags.root ?? process.cwd());
  const lib = path.join(root, 'src', 'lib');
  await ensureDir(lib);
  const out = path.join(lib, 'cn.ts');
  if (!flags.force && await fileExists(out)) return warn(`${path.relative(root, out)} exists; --force to rewrite`);
  if (flags.dryRun) return info(`[dry-run] write ${path.relative(root, out)}`);
  await fsp.writeFile(out, [
    'import { type ClassValue, clsx } from "clsx";',
    'import { twMerge } from "tailwind-merge";',
    '',
    '/**',
    ' * Tailwind class-merge helper. Powered by clsx + tailwind-merge.',
    ' */',
    'export function cn(...inputs: ClassValue[]) {',
    '  return twMerge(clsx(inputs));',
    '}',
    '',
  ].join('\n') + '\n', 'utf8');
  info(`[init] ${path.relative(root, out)}`);
}

// --- add ------------------------------------------------------------------

async function cmdAdd(addList, flags) {
  const root = path.resolve(flags.root ?? process.cwd());
  if (!addList.length) die('add requires at least one component name, e.g. shadcn.mjs add button');
  const componentsJson = await readComponentsJson(root);
  const style = flags.style || componentsJson.style || 'default';

  const depItems = [];
  const seen = new Set();
  for (const name of addList) {
    const resolved = await resolveDeps(name, style, seen);
    depItems.push(...resolved);
  }

  const uniqueByName = [];
  const uniqueByNameSet = new Set();
  for (const it of depItems) {
    if (!uniqueByNameSet.has(it.name)) { uniqueByNameSet.add(it.name); uniqueByName.push(it); }
  }

  info(`resolved ${uniqueByName.length} item(s) for ${addList.join(', ')}`);

  // Attach root-derived aliases so rewriteImports can relativize correctly.
  for (const it of uniqueByName) {
    it.root = root;
    it.aliases = mergeDeep(componentsJson.aliases || {}, {});
    it.style = style;
  }

  await writeItems(uniqueByName, flags);

  const deps = computeDepSet(uniqueByName);
  if (!flags.dryRun && deps.length) installDeps(deps, flags.installer || 'pnpm', root);
  else if (!deps.length) info('no external npm deps listed in registry items');
}

function computeDepSet(items) {
  const set = new Map();
  for (const it of items) {
    for (const d of (it.dependencies || [])) {
      // Preserve pinned or scoped versions as registry lists them.
      set.set(normalizeDep(d), d);
    }
  }
  return [...set.values()];
}

function normalizeDep(d) {
  // Keep @scope/pkg format intact; use everything before optional ^~ as key.
  const m = d.match(/^(@[\w-]+\/[\w-]+)/);
  return m ? m[1] : d.split(/[\s@~^]/)[0];
}

// --------------------------------------------------------------------------

async function main() {
  const { command, components, flags } = parseCli(process.argv.slice(2));
  switch (command) {
    case 'add': await cmdAdd(components, flags); break;
    case 'list': await cmdList(); break;
    case 'init': await cmdInit(flags); break;
    default: die(`unknown command ${command}`);
  }
}

main().catch(e => { console.error(e); process.exit(1); });