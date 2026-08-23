#!/usr/bin/env node
/**
 * Check that every data-i18n key in PORT pages exists in all locale dictionaries.
 * This is intentionally dependency-free so it can run in CI and on a fresh checkout.
 */
import { readFile, readdir } from 'node:fs/promises';
import { join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(fileURLToPath(new URL('..', import.meta.url)));
const pageDirs = ['docs/user', 'docs/worker'];
const localeFiles = {
  'docs/user': join(root, 'docs/user/locale.js'),
  'docs/worker': join(root, 'docs/worker/locale.js')
};
const locales = ['zh-CN', 'en-US', 'ms-MY'];
const sources = Object.fromEntries(await Promise.all(Object.entries(localeFiles).map(async ([dir, file]) => [dir, await readFile(file, 'utf8')])));

function localeKeys(source, locale) {
  const start = source.indexOf(`'${locale}':`);
  if (start < 0) return new Set();
  const nextLocale = locales.find((candidate) => candidate !== locale && source.indexOf(`'${candidate}':`, start + 1) >= 0);
  const next = nextLocale ? source.indexOf(`'${nextLocale}':`, start + 1) : -1;
  const end = source.indexOf('\n    },', start);
  const boundary = next >= 0 ? next : end;
  const block = source.slice(start, boundary < 0 ? source.length : boundary);
  return new Set([...block.matchAll(/'([^']+)'\s*:/g)].map((match) => match[1]));
}

const dictionaries = Object.fromEntries(Object.entries(sources).flatMap(([dir, source]) => locales.map((locale) => [`${dir}:${locale}`, localeKeys(source, locale)])));
const pages = [];
for (const dir of pageDirs) {
  const absoluteDir = join(root, dir);
  let names;
  try {
    names = await readdir(absoluteDir);
  } catch {
    continue;
  }
  for (const name of names.filter((item) => item.endsWith('.html'))) pages.push(join(absoluteDir, name));
}

const keysByPage = new Map();
for (const page of pages) {
  const html = await readFile(page, 'utf8');
  const keys = [...html.matchAll(/data-i18n\s*=\s*["']([^"']+)["']/g)].map((match) => match[1]);
  keysByPage.set(page, [...new Set(keys)]);
}

const errors = [];
for (const [page, keys] of keysByPage) {
  for (const key of keys) {
    for (const locale of locales) {
      const dir = page.includes('/docs/worker/') ? 'docs/worker' : 'docs/user';
      if (!dictionaries[`${dir}:${locale}`].has(key)) {
        errors.push(`${relative(root, page)}: ${locale} missing ${key}`);
      }
    }
  }
}

const totalKeys = new Set([...keysByPage.values()].flat()).size;
console.log(`Checked ${pages.length} pages, ${totalKeys} unique data-i18n keys, ${locales.length} locales.`);
if (errors.length) {
  console.error(errors.join('\n'));
  process.exitCode = 1;
} else {
  console.log('All data-i18n keys are defined in every locale.');
}
