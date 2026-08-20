#!/usr/bin/env node
// gpt-image-generate: 调用 gpt-image-2 生成页面设计稿，支持参考图（图生图）。
// 用法:
//   node gpt-image-generate.mjs --prompt "描述" [选项]
//
// 选项:
//   --prompt / -p     图片描述（必填）
//   --image / -i      参考图路径，可重复传多张（--image a.png --image b.png），
//                     也可逗号分隔（--image a.png,b.png）；传入后走图生图端点
//   --out / -o        输出路径（默认 ./output.png）
//   --size            尺寸: 1024x1024 | 1536x1024 | 1024x1536 | auto（默认 1536x1024）
//   --quality         质量: low | medium | high | auto（默认 auto）
//   --n               生成张数（默认 1）
//   --style           风格: vivid | natural（默认 vivid，仅文生图生效）
//   --system          系统提示词，用于约束输出风格
//   --env             .env 文件路径（默认同目录 .env）
//
// 示例:
//   node gpt-image-generate.mjs -p "BOSS系统仪表盘页面，深色主题，左侧导航栏，右侧数据卡片"
//   node gpt-image-generate.mjs -p "登录页" -o login.png --size 1024x1536
//   node gpt-image-generate.mjs -p "保持整体布局，把主色改成蓝色" -i ref.png -o v2.png
//   node gpt-image-generate.mjs -p "融合两张图的风格" -i a.png -i b.png

import { readFileSync, writeFileSync, mkdirSync } from 'node:fs'
import { resolve, dirname, basename } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))

function loadEnv(envPath) {
  try {
    const content = readFileSync(envPath, 'utf-8')
    const env = {}
    for (const line of content.split('\n')) {
      const trimmed = line.trim()
      if (!trimmed || trimmed.startsWith('#')) continue
      const idx = trimmed.indexOf('=')
      if (idx === -1) continue
      env[trimmed.slice(0, idx).trim()] = trimmed.slice(idx + 1).trim()
    }
    return env
  } catch {
    return {}
  }
}

function parseArgs(argv) {
  const args = {
    prompt: '',
    images: [],
    out: './output.png',
    size: '1536x1024',
    quality: 'auto',
    n: 1,
    style: 'vivid',
    system: '',
    envPath: resolve(__dirname, '.env'),
  }
  for (let i = 2; i < argv.length; i += 2) {
    const key = argv[i]?.replace(/^--/, '').replace(/^-/, '')
    const val = argv[i + 1]
    if (!val || val.startsWith('-')) { i--; continue }
    if (key === 'p' || key === 'prompt') args.prompt = val
    else if (key === 'i' || key === 'image')
      args.images.push(...val.split(',').map(s => s.trim()).filter(Boolean))
    else if (key === 'o' || key === 'out') args.out = val
    else if (key === 'n') args.n = parseInt(val, 10)
    else if (key === 'env') args.envPath = resolve(val)
    else args[key] = val
  }
  return args
}

function buildPayload(args) {
  const body = {
    model: 'gpt-image-2',
    prompt: args.prompt,
    n: args.n,
    size: args.size,
    quality: args.quality,
    response_format: 'b64_json',
  }
  if (args.system) body.system_prompt = args.system
  if (args.style) body.style = args.style
  return body
}

// 图生图：/v1/images/edits 走 multipart。该端点不接受 style/response_format，
// system 提示词并入 prompt 前缀以保留风格约束能力。
function buildForm(args) {
  const form = new FormData()
  form.append('model', 'gpt-image-2')
  const prompt = args.system ? `${args.system}\n\n${args.prompt}` : args.prompt
  form.append('prompt', prompt)
  form.append('n', String(args.n))
  form.append('size', args.size)
  if (args.quality && args.quality !== 'auto') form.append('quality', args.quality)
  for (const path of args.images) {
    const buf = readFileSync(path)
    form.append('image', new File([buf], basename(path), { type: 'image/png' }), basename(path))
  }
  return form
}

async function requestImages(args, env) {
  const apiKey = env.OPENAI_API_KEY
  const baseUrl = (env.OPENAI_BASE_URL || 'https://api.openai.com').replace(/\/$/, '')
  const withRef = args.images.length > 0
  const url = withRef ? `${baseUrl}/v1/images/edits` : `${baseUrl}/v1/images/generations`

  console.log(`Generating image with gpt-image-2 (${withRef ? 'edit / reference' : 'text-to-image'})...`)
  console.log(`  prompt: ${args.prompt.slice(0, 80)}${args.prompt.length > 80 ? '...' : ''}`)
  if (withRef) console.log(`  reference: ${args.images.join(', ')}`)
  console.log(`  size: ${args.size}, quality: ${args.quality}${withRef ? '' : `, style: ${args.style}`}`)

  const resp = await fetch(url, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${apiKey}`,
      ...(withRef ? {} : { 'Content-Type': 'application/json' }),
    },
    body: withRef ? buildForm(args) : JSON.stringify(buildPayload(args)),
  })

  if (!resp.ok) {
    const errText = await resp.text()
    console.error(`API error ${resp.status}:`, errText.slice(0, 500))
    process.exit(1)
  }
  return resp.json()
}

async function generateImage(args) {
  const env = loadEnv(args.envPath)
  if (!env.OPENAI_API_KEY) {
    console.error('Error: OPENAI_API_KEY not found in', args.envPath)
    process.exit(1)
  }
  if (!args.prompt) {
    console.error('Error: --prompt is required')
    process.exit(2)
  }
  for (const p of args.images) {
    try {
      readFileSync(p)
    } catch {
      console.error(`Error: reference image not readable: ${p}`)
      process.exit(2)
    }
  }

  const data = await requestImages(args, env)
  if (!data.data || data.data.length === 0) {
    console.error('Error: no images returned')
    process.exit(1)
  }

  const outDir = dirname(resolve(args.out))
  mkdirSync(outDir, { recursive: true })

  for (let i = 0; i < data.data.length; i++) {
    const item = data.data[i]
    let outPath = args.out
    if (data.data.length > 1) {
      const dot = args.out.lastIndexOf('.')
      outPath = dot > 0
        ? `${args.out.slice(0, dot)}_${i + 1}${args.out.slice(dot)}`
        : `${args.out}_${i + 1}`
    }
    if (item.b64_json) {
      writeFileSync(outPath, Buffer.from(item.b64_json, 'base64'))
    } else if (item.url) {
      const imgResp = await fetch(item.url)
      const buf = Buffer.from(await imgResp.arrayBuffer())
      writeFileSync(outPath, buf)
    }
    console.log(`Saved: ${resolve(outPath)}`)
  }

  console.log(`Done. ${data.data.length} image(s) generated.`)
}

const args = parseArgs(process.argv)
await generateImage(args)
