#!/usr/bin/env node
// gpt-image-analyze: 当主模型不支持图像输入时，用 gpt-5.6-sol 识图。
// 复用同目录 .env 的 OPENAI_API_KEY / OPENAI_BASE_URL（与生图脚本同一账号）。
// 用法:
//   node gpt-image-analyze.mjs --image <path> --prompt "问题" [选项]
//
// 选项:
//   --image / -i      图片路径，可重复传多张（-i a.png -i b.png），也可逗号分隔
//   --prompt / -p     识图指令（必填，如"描述这个页面的布局结构和配色"）
//   --system / -s     系统提示词（默认: 专业的 UI/UX 设计评审）
//   --model / -m      模型名（默认 gpt-5.6-sol）
//   --env             .env 文件路径（默认同目录 .env）
//
// 示例:
//   node gpt-image-analyze.mjs -i designs/login-v1.png -p "逐区块描述布局，列出所有颜色hex近似值"
//   node gpt-image-analyze.mjs -i v1.png -i v2.png -p "对比两稿差异"

import { readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))

function loadEnv(envPath) {
  try {
    const env = {}
    for (const line of readFileSync(envPath, 'utf-8').split('\n')) {
      const t = line.trim()
      if (!t || t.startsWith('#')) continue
      const idx = t.indexOf('=')
      if (idx === -1) continue
      env[t.slice(0, idx).trim()] = t.slice(idx + 1).trim()
    }
    return env
  } catch {
    return {}
  }
}

function parseArgs(argv) {
  const args = {
    images: [],
    prompt: '',
    system: 'You are a professional UI/UX design reviewer. Answer precisely and concretely in Chinese.',
    model: 'gpt-5.6-sol',
    diff: false,
    envPath: resolve(__dirname, '.env'),
  }
  for (let i = 2; i < argv.length; i += 2) {
    const key = argv[i]?.replace(/^--/, '').replace(/^-/, '')
    const val = argv[i + 1]
    if (key === 'diff') { args.diff = true; i--; continue }
    if (!val || val.startsWith('-')) { i--; continue }
    if (key === 'p' || key === 'prompt') args.prompt = val
    else if (key === 'i' || key === 'image')
      args.images.push(...val.split(',').map(s => s.trim()).filter(Boolean))
    else if (key === 's' || key === 'system') args.system = val
    else if (key === 'm' || key === 'model') args.model = val
    else if (key === 'env') args.envPath = resolve(val)
    else args[key] = val
  }
  return args
}

function imagePart(path) {
  const buf = readFileSync(path)
  const b64 = buf.toString('base64')
  const ext = path.toLowerCase().endsWith('.jpg') || path.toLowerCase().endsWith('.jpeg') ? 'jpeg' : 'png'
  return { type: 'image_url', image_url: { url: `data:image/${ext};base64,${b64}` } }
}

const DIFF_SYSTEM = `You are a meticulous design QA reviewer. You receive exactly two images: the FIRST is the design mockup (设计稿), the SECOND is the implementation screenshot (实现稿). Compare the implementation against the design and answer in Chinese with exactly this structure:

## 差异清单
A markdown table with columns: 序号 | 维度 (布局/间距/圆角/颜色/字体层级/组件形态/状态缺失/内容) | 设计稿 | 实现稿 | 严重度 (高/中/低).
Only list real, actionable deviations. Layout right or near-identical = not a deviation. Ignore trivial text placeholder differences unless they break layout. If there are no deviations, output "无显著差异" and skip the table.

## 修复提示词
A single fenced code block containing a prompt that can be sent directly to a frontend engineer or AI coding agent to fix ALL listed deviations: concrete CSS/component-level instructions (target element, current wrong state, expected state with values like hex/px), grouped by page region, ordered by severity. No greetings, no explanation outside the block.`

function diffPrompt(extra) {
  const base = '第一张图是设计稿，第二张图是实现稿。逐维度对比实现稿相对设计稿的偏差，按约定格式输出差异清单和修复提示词。'
  return extra ? `${base}\n\n补充上下文：${extra}` : base
}

async function analyze(args) {
  const env = loadEnv(args.envPath)
  const apiKey = env.OPENAI_API_KEY
  const baseUrl = (env.OPENAI_BASE_URL || 'https://api.openai.com').replace(/\/$/, '')
  if (!apiKey) {
    console.error('Error: OPENAI_API_KEY not found in', args.envPath)
    process.exit(1)
  }
  if (!args.prompt && !args.diff) {
    console.error('Error: --prompt is required (or use --diff with two images)')
    process.exit(2)
  }
  if (args.diff && args.images.length !== 2) {
    console.error('Error: --diff requires exactly 2 images: first the design mockup, then the implementation screenshot')
    process.exit(2)
  }
  for (const p of args.images) {
    try {
      readFileSync(p)
    } catch {
      console.error(`Error: image not readable: ${p}`)
      process.exit(2)
    }
  }

  const prompt = args.diff ? diffPrompt(args.prompt) : args.prompt
  const system = args.diff ? DIFF_SYSTEM : args.system
  const content = [{ type: 'text', text: prompt }, ...args.images.map(imagePart)]
  const resp = await fetch(`${baseUrl}/v1/chat/completions`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      model: args.model,
      messages: [
        { role: 'system', content: system },
        { role: 'user', content },
      ],
    }),
  })

  if (!resp.ok) {
    console.error(`API error ${resp.status}:`, (await resp.text()).slice(0, 500))
    process.exit(1)
  }

  const data = await resp.json()
  const text = data.choices?.[0]?.message?.content
  if (!text) {
    console.error('Error: no content returned:', JSON.stringify(data).slice(0, 300))
    process.exit(1)
  }
  console.log(text)
}

await analyze(parseArgs(process.argv))
