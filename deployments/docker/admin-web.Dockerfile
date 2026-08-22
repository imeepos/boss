# boss admin 前端:多阶段构建——Node 构建 + nginx 运行。
# 构建上下文:仓库根目录(docker build -f deployments/docker/admin-web.Dockerfile .)

FROM node:22-alpine AS builder
# 固定 pnpm 小版本:pnpm@latest 升级会把 ignored build scripts(esbuild)升级为硬失败,
# 2026-08-22 CI 曾因此中断(任务 2287);版本与本地开发环境一致。
RUN corepack enable && corepack prepare pnpm@11.10.0 --activate
WORKDIR /app
# pnpm-workspace.yaml 必须随锁文件同层 COPY:其中的 esbuild 构建白名单是 install 前置输入。
COPY web/admin/package.json web/admin/pnpm-lock.yaml web/admin/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile --prefer-offline
COPY web/admin/ ./
RUN pnpm build && find /app/dist -type f -exec chmod 644 {} \; && find /app/dist -type d -exec chmod 755 {} \;

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY deployments/docker/admin-web.nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
