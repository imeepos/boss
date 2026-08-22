# boss admin 前端:多阶段构建——Node 构建 + nginx 运行。
# 构建上下文:仓库根目录(docker build -f deployments/docker/admin-web.Dockerfile .)

FROM node:22-alpine AS builder
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app
COPY web/admin/package.json web/admin/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --prefer-offline
COPY web/admin/ ./
RUN pnpm build && find /app/dist -type f -exec chmod 644 {} \; && find /app/dist -type d -exec chmod 755 {} \;

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY deployments/docker/admin-web.nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
