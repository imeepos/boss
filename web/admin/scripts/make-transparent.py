# 白底素材透明化:从边缘洪水填充近似白色 → alpha=0,边缘羽化;内容保留。
# 用法: python3 scripts/make-transparent.py src/assets/brand/<file.png> [...]
import sys
from collections import deque

from PIL import Image, ImageFilter

THRESH = 232  # 亮度阈值:高于此视为背景候选


def make_transparent(path: str) -> None:
    im = Image.open(path).convert('RGB')
    w, h = im.size
    px = im.load()
    lum = [[0] * h for _ in range(w)]
    for x in range(w):
        for y in range(h):
            r, g, b = px[x, y]
            lum[x][y] = (r * 299 + g * 587 + b * 114) // 1000

    mask = Image.new('L', (w, h), 255)
    seen = [[False] * h for _ in range(w)]
    q = deque()
    for x in range(w):
        for y in (0, h - 1):
            if lum[x][y] >= THRESH:
                q.append((x, y))
                seen[x][y] = True
    for y in range(h):
        for x in (0, w - 1):
            if lum[x][y] >= THRESH and not seen[x][y]:
                q.append((x, y))
                seen[x][y] = True

    mp = mask.load()
    while q:
        x, y = q.popleft()
        mp[x, y] = 0
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            nx, ny = x + dx, y + dy
            if 0 <= nx < w and 0 <= ny < h and not seen[nx][ny] and lum[nx][ny] >= THRESH:
                seen[nx][ny] = True
                q.append((nx, ny))

    mask = mask.filter(ImageFilter.GaussianBlur(1.2))
    out = im.convert('RGBA')
    out.putalpha(mask)
    out.save(path)
    print('OK', path)


if __name__ == '__main__':
    for p in sys.argv[1:]:
        make_transparent(p)
