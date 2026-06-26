FROM node:22-alpine AS builder
WORKDIR /src
ARG YARN_NPM_REGISTRY_SERVER=https://registry.npmjs.org
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ENV YARN_NPM_REGISTRY_SERVER=${YARN_NPM_REGISTRY_SERVER}
ENV HTTP_PROXY=${HTTP_PROXY}
ENV HTTPS_PROXY=${HTTPS_PROXY}
ENV NO_PROXY=${NO_PROXY}
COPY package.json yarn.lock .yarnrc.yml ./
COPY apps/web/package.json ./apps/web/package.json
COPY infra/fake-provider/package.json ./infra/fake-provider/package.json
RUN corepack enable \
    && corepack prepare yarn@4.17.0 --activate \
    && yarn install --immutable
COPY apps/web/ ./apps/web/
ARG NUXT_PUBLIC_API_BASE
ENV NUXT_PUBLIC_API_BASE=${NUXT_PUBLIC_API_BASE}
RUN yarn workspace @aeon-echoes/web generate

FROM nginx:1.27-alpine
COPY infra/web.nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /src/apps/web/.output/public /usr/share/nginx/html
