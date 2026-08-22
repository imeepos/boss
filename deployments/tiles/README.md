# PGIS 瓦片数据目录(commit 8 占位 + commit A7 实配)

## 用途

`docker-compose.tiles.yml` 的 `tileserver-gl` 服务从此目录以 `:ro` 方式加载 PMTiles 单文件切片 + `config.json` 数据源声明。

## 文件清单

当前含 `config.json`(commit A7 新增)。部署时由 ops 拉取下列文件(总大小预估 < 1GB):

| 文件 | 来源 | 大小 | 用途 |
|---|---|---|---|
| `config.json` | 仓库内置 | <2KB | 数据源/样式/healthcheck 端点声明 |
| `world-lowzoom.pmtiles` | [protomaps/PMTiles](https://protomaps.com/docs/frontends/maplibre) 公开或自建 | ~200MB | zoom 0-7 全球底图 |
| `philippines-highzoom.pmtiles` | OSM PBF 切片(geofabrik.de/asia/philippines.html) | ~700MB | zoom 8-14 PSGC 区域高分辨率(可选) |
| `fonts/` | protomaps fontset 或自建 | ~5MB | MVT 样式要求的字形目录(暂空) |

## 部署步骤

1. 拉 PMTiles 到 `data/`:
   ```bash
   cd deployments/tiles/data
   wget https://r2-public.protomaps.com/protomaps-sample-datasets/protomaps-basemap-opensource-20230408.pmtiles \
        -O world-lowzoom.pmtiles
   ```
2. 起服务:
   ```bash
   cd deployments
   docker compose -f docker-compose.tiles.yml up -d
   ```
3. 端到端验证:
   ```bash
   curl -fsS http://127.0.0.1:18080/health
   curl -fsS http://127.0.0.1:18080/data/world-lowzoom/metadata.json | jq .
   ```
4. nginx 反代(参见 `docker-compose.nginx.yml` 增项):
   - `/tiles/` → `tileserver-gl:8080/`
   - 前端 OL 切到 `PMTILES_TILE_URL = '/tiles/data/world-lowzoom/{z}/{x}/{y}.pbf'`
   - 修改 `web/admin/src/components/business/maps/tile-source.ts` 的 `LIGHT_TILE_URL` 即可,本期仍保留 OSM 兜底

## 端点约定

```
GET /health                                    → 200 OK
GET /data/world-lowzoom/metadata.json          → {name, center, zoom_min, zoom_max, ...}
GET /data/world-lowzoom/{z}/{x}/{y}.pbf        → 矢量瓦片二进制(MVT/PMTiles)
GET /styles/basic                              → GL JSON 样式
```

## 不用 git 提交大文件

PMTiles/字体不进 git,部署时由 ops 拉取。`.gitkeep` 仅占位目录结构(`data/` 空目录持久化)。

## Healthcheck 注意

docker-compose `healthcheck.test` 调用 `wget ... /health`。**PMTiles 文件不存在时** `/health` 会返回 503,容器会被反复重启。Ops 拉 PMTiles **前**先把容器 `disable:healthcheck` 或延迟 `start_period`(已设 30s);文件就位后 healthcheck 自动通过。