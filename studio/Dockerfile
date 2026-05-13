# SPDX-License-Identifier: Apache-2.0

# Stage 1: Build the SPA
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

# Stage 2: Serve with Nginx
FROM nginx:1-alpine
RUN apk add --no-cache gettext
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY env.js.template /usr/share/nginx/html/env.js.template
COPY docker-entrypoint.sh /docker-entrypoint.d/40-envsubst-studio.sh
RUN chmod +x /docker-entrypoint.d/40-envsubst-studio.sh

EXPOSE 80
