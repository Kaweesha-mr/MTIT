
const express = require('express');
const cors = require('cors');
const router = require('./routes/shelterRoutes');
const { init } = require('./db');
const config = require('./config');

async function start() {
  await init();

  const app = express();
  app.use(cors());
  app.use(express.json());
  app.use(router);

  app.listen(config.port, () => {
    console.log(`shelter-service running on :${config.port}`);
  });
}

start().catch((err) => {
  console.error('Failed to start shelter-service', err);
  process.exit(1);
});
