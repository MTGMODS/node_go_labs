FROM node:22-alpine

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev
COPY src ./src
COPY public ./public
COPY scripts ./scripts
COPY test ./test
RUN npm test

ENV PORT=3006
ENV DATA_DIR=/data
EXPOSE 3006
CMD ["node", "src/server.js"]
