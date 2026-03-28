const DEFAULT_PORT = 8084;
const DEFAULT_INCIDENT_URL = 'http://localhost:8081';
const DEFAULT_DB_HOST = 'localhost';
const DEFAULT_DB_PORT = 5435;
const DEFAULT_DB_USER = 'shelter_user';
const DEFAULT_DB_PASSWORD = 'shelter_pass';
const DEFAULT_DB_NAME = 'shelters_db';

module.exports = {
  port: process.env.PORT || DEFAULT_PORT,
  incidentServiceUrl: (process.env.INCIDENT_SERVICE_URL || DEFAULT_INCIDENT_URL).replace(/\/$/, ''),
  db: {
    host: process.env.DB_HOST || DEFAULT_DB_HOST,
    port: parseInt(process.env.DB_PORT || DEFAULT_DB_PORT, 10),
    user: process.env.DB_USER || DEFAULT_DB_USER,
    password: process.env.DB_PASSWORD || DEFAULT_DB_PASSWORD,
    database: process.env.DB_NAME || DEFAULT_DB_NAME,
    ssl: (process.env.DB_SSLMODE || 'disable').toLowerCase() !== 'disable'
      ? { rejectUnauthorized: false }
      : false,
  },
};
