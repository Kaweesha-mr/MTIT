const express = require('express');
const cors = require('cors');
const path = require('path');
const { pool } = require('./db');
const { init } = require('./db');
const shelterRoutes = require('./routes/shelterRoutes');

const app = express();
const PORT = process.env.PORT || 8084;

// Middleware
app.use(cors());
app.use(express.json());

// Initialize database
init().catch(err => {
  console.error('Database initialization failed:', err);
  process.exit(1);
});

// Routes
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'shelter-service' });
});

app.use('/', shelterRoutes);

// API documentation endpoint
app.get('/openapi.yaml', (req, res) => {
  res.sendFile(path.join(__dirname, '../api/openapi.yaml'));
});

// Swagger/Docs endpoints
app.get('/swagger.json', (req, res) => {
  res.redirect('/openapi.yaml');
});

app.get('/docs', (req, res) => {
  res.sendFile(path.join(__dirname, '../index.html'));
});

app.get('/docs/', (req, res) => {
  res.sendFile(path.join(__dirname, '../index.html'));
});

app.get('/swagger', (req, res) => {
  res.sendFile(path.join(__dirname, '../index.html'));
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error(err);
  res.status(500).json({ error: 'Internal Server Error' });
});

// Start server
app.listen(PORT, () => {
  console.log(`Shelter service listening on port ${PORT}`);
});

// Graceful shutdown
process.on('SIGINT', async () => {
  console.log('Shutting down gracefully...');
  await pool.end();
  process.exit(0);
});
