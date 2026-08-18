// 品牌素材生成:gpt-image-2,输出到 src/assets/brand/。
// 用法: node scripts/gen-brand-assets.mjs  (读取仓库根 .env 的 OPENAI_API_KEY/OPENAI_BASE_URL)
import { mkdir, writeFile } from 'node:fs/promises'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { readFileSync } from 'node:fs'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../../')
const env = Object.fromEntries(
  readFileSync(resolve(root, '.env'), 'utf8')
    .split('\n')
    .filter((l) => l.includes('='))
    .map((l) => [l.slice(0, l.indexOf('=')).trim(), l.slice(l.indexOf('=') + 1).trim()]),
)
const OUT = resolve(root, 'web/admin/src/assets/brand')
await mkdir(OUT, { recursive: true })

const BRAND =
  'Sphere Boss, a telecom business support system (BOSS) admin platform. Brand palette: deep navy #1B2A4A and warm gold #C9A45C. Style: premium, corporate, modern, clean.'

const tasks = [
  // logo 3 种风格
  { file: 'logo-mark-navy.png', size: '1024x1024', prompt: `Flat minimalist app icon logo mark for ${BRAND} A geometric sphere built from orbiting arcs with a subtle S monogram inside, solid deep navy silhouette on pure white background, crisp vector edges, centered, no text.` },
  { file: 'logo-mark-gradient.png', size: '1024x1024', prompt: `Modern gradient logo mark for ${BRAND} A luminous sphere of interwoven network lines and signal waves, navy-to-blue gradient with gold accent ring, glossy tech feel, pure white background, centered, no text.` },
  { file: 'logo-mark-lineart.png', size: '1024x1024', prompt: `Elegant thin line-art logo mark for ${BRAND} A single-weight golden line drawing of a globe with constellation nodes and one orbit ring, luxurious minimal linework, pure white background, centered, no text.` },
  // 登录页点缀素材
  { file: 'ornament-corner.png', size: '1024x1024', prompt: `Corner decorative flourish for a premium login page, ${BRAND} Delicate gold geometric line pattern of concentric quarter arcs and small dots, flowing from the top-left corner, isolated on transparent background, elegant, subtle.` },
  { file: 'ornament-ring.png', size: '1024x1024', prompt: `Abstract decorative ring ornament for corporate UI, ${BRAND} A dashed gold orbit ring with tiny glowing nodes like satellites, semi-transparent, isolated on transparent background, minimal.` },
  { file: 'ornament-ribbon.png', size: '1024x1024', prompt: `Smooth abstract silk ribbon wave in navy and gold gradients, flowing diagonal decorative element for a login page edge, isolated on transparent background, soft 3D sheen, no text.` },
  { file: 'ornament-dots.png', size: '1024x1024', prompt: `Scattered constellation dot grid ornament, small gold and navy circles of varying sizes with thin connecting lines, sparse and airy, isolated on transparent background, minimal corporate decoration.` },
  { file: 'ornament-shield.png', size: '1024x1024', prompt: `Minimal flat security shield badge icon with a keyhole, navy fill with gold outline and subtle glow, corporate trust symbol for a login form, isolated on transparent background, no text.` },
  // 登录页背景 1 张
  { file: 'login-bg-full.png', size: '1536x1024', prompt: `Wide login page background artwork for ${BRAND} An abstract city skyline at dusk made of fine navy line work with soft gold light trails, large calm negative space in the center-right for a login card, gentle gradient from deep navy to soft ivory, premium enterprise feel, no text, no logos.` },
  // 广告/宣传素材 3 张
  { file: 'ad-network.png', size: '1536x1024', prompt: `Marketing banner for ${BRAND} A stylized fiber-optic network spreading across a stylized Philippine archipelago map, glowing gold connection lines over deep navy, cinematic lighting, bold negative space at top for headline, no text.` },
  { file: 'ad-dashboard.png', size: '1536x1024', prompt: `Marketing banner for ${BRAND} Floating translucent analytics dashboard cards with charts and KPI rings above an abstract navy desk scene, gold highlight accents, clean enterprise SaaS aesthetic, negative space for headline, no readable text.` },
  { file: 'ad-service.png', size: '1536x1024', prompt: `Marketing banner for ${BRAND} A friendly technician character in navy uniform installing a glowing gold wifi sphere on a rooftop at golden hour, flat illustration style, warm premium palette, negative space for headline, no text.` },
]

async function gen({ file, size, prompt }) {
  for (let attempt = 1; attempt <= 3; attempt++) {
    try {
      const res = await fetch(`${env.OPENAI_BASE_URL}/v1/images/generations`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${env.OPENAI_API_KEY}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: 'gpt-image-2', prompt, size, n: 1 }),
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}: ${(await res.text()).slice(0, 200)}`)
      const json = await res.json()
      const b64 = json?.data?.[0]?.b64_json
      if (!b64) throw new Error('no b64_json in response')
      await writeFile(resolve(OUT, file), Buffer.from(b64, 'base64'))
      console.log('OK', file)
      return
    } catch (err) {
      console.error(`FAIL(${attempt}) ${file}:`, err.message)
      if (attempt === 3) process.exitCode = 1
      else await new Promise((r) => setTimeout(r, 3000))
    }
  }
}

for (const t of tasks) await gen(t)
console.log('done')
