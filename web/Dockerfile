# ---------- 依赖安装阶段 ----------
FROM node:20-alpine AS deps

# 国内构建走 npmmirror 镜像，避免 npmjs.org 慢/超时
# 海外环境可删除此行（默认 registry.npmjs.org）
ENV npm_config_registry=https://registry.npmmirror.com

WORKDIR /app

COPY package.json package-lock.json .npmrc ./
RUN npm ci

# ---------- 构建阶段 ----------
FROM node:20-alpine AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# NEXT_PUBLIC_* 是构建时变量（Next.js 编译到产物里），通过 build arg 注入
# 默认留空：前端运行时按浏览器 hostname 自动适配，局域网/公网均可用
# 固定值场景（如自定义域名）在 .env 设置 NEXT_PUBLIC_API_BASE 覆盖
ARG NEXT_PUBLIC_API_BASE=
ENV NEXT_PUBLIC_API_BASE=$NEXT_PUBLIC_API_BASE

RUN npm run build

# ---------- 运行阶段 ----------
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
# Next.js standalone 模式下 server.js 默认监听 3000，这里对齐 3001（与 dev/start 一致）
ENV PORT=3001

# standalone 产物：仅包含运行所需文件，镜像体积小
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

EXPOSE 3001

CMD ["node", "server.js"]
