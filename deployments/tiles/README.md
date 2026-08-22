# PGIS 瓦片数据目录(commit 8 占位)

## 用途

`docker-compose.tiles.yml` 的 `tileserver-gl` 服务从此目录以 `:ro` 方式加载 PMTiles 单文件切片。

## 文件清单

当前**空**。部署时由 ops 拉取下列文件(总大小预估 < 1GB):

| 文件 | 来源 | 大小 | 用途 |
|---|---|---|---|
| `world-lowzoom.pmtiles` | [protomaps/PMTiles](https://protomaps.com/docs/frontends/maplibre) 公开示例或自建 | ~200MB | zoom 0-7 全球底图 |
| `philippines-highzoom.pmtiles` | OSM PBF 切片(geofabrik.de/asia/philippines.html) | ~700MB | zoom 8-14 PSGC 区域高分辨率 |
| `config.json` | tileserver-gl 配置 | <10KB | 数据源/样式声明 |

## 部署步骤

1. `docker compose -f docker-compose.tiles.yml up -d`
2. 服务监听 `127.0.0.1:18080`,通过 `deployments/docker-compose.nginx.yml` 反代到 `/tiles/` 路径
3. 前端 OL 瓦片源:`https://boss.ymm.cn/tiles/{z}/{x}/{y}.pbf`(自建)或临时 OSM 公开瓦片

## 不用 git 提交大文件

PMTiles 文件不进 git,部署时由 ops 拉取。`.gitkeep` 仅占位目录结构。