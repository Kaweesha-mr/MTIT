FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install --omit=dev --force || npm install --omit=dev
COPY . .
EXPOSE 8084
CMD ["npm", "start"]
