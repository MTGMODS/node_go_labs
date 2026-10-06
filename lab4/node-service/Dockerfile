FROM node:22-alpine

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev
COPY src ./src
COPY test ./test
RUN npm test

ENV PORT=3004
EXPOSE 3004
CMD ["node", "src/server.js"]
