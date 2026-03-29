const { Pool } = require('pg');
const config = require('./config');

const pool = new Pool({
  host: config.db.host,
  port: config.db.port,
  user: config.db.user,
  password: config.db.password,
  database: config.db.database,
  ssl: config.db.ssl,
  max: 10,
});

async function init() {
  const createTableSQL = `
    CREATE TABLE IF NOT EXISTS shelters (
      id SERIAL PRIMARY KEY,
      incident_id INTEGER NOT NULL,
      name TEXT NOT NULL,
      capacity INTEGER NOT NULL,
      current_occupancy INTEGER NOT NULL DEFAULT 0,
      status TEXT NOT NULL DEFAULT 'REQUEST',
      location TEXT DEFAULT NULL
    );
  `;
  await pool.query(createTableSQL);
}

module.exports = {
  pool,
  init,
};
