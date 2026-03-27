const express = require('express');
const path = require('path');
const service = require('../services/shelterService');

const router = express.Router();

router.get('/health', (_req, res) => {
  res.json({ status: 'ok' });
});

router.post('/shelters', async (req, res) => {
  const result = await service.createShelter(req.body || {});
  if (result.error) {
    return res.status(result.error.code).json({ error: result.error.message });
  }
  res.status(201).json(result.shelter);
});

router.get('/shelters', async (_req, res) => {
  const shelters = await service.listShelters();
  res.json(shelters);
});

router.get('/shelters/:id', async (req, res) => {
  const id = Number(req.params.id);
  if (!id || id <= 0) {
    return res.status(400).json({ error: 'invalid shelter id' });
  }

  const shelter = await service.getShelter(id);
  if (!shelter) {
    return res.status(404).json({ error: 'shelter not found' });
  }

  res.json(shelter);
});

router.put('/shelters/:id', async (req, res) => {
  const id = Number(req.params.id);
  if (!id || id <= 0) {
    return res.status(400).json({ error: 'invalid shelter id' });
  }

  const result = await service.updateShelter(id, req.body || {});
  if (result.error) {
    return res.status(result.error.code).json({ error: result.error.message });
  }

  res.json(result.shelter);
});

router.delete('/shelters/:id', async (req, res) => {
  const id = Number(req.params.id);
  if (!id || id <= 0) {
    return res.status(400).json({ error: 'invalid shelter id' });
  }

  const result = await service.deleteShelter(id);
  if (result.error) {
    return res.status(result.error.code).json({ error: result.error.message });
  }

  res.status(204).send();
});

router.put('/shelters/:id/capacity', async (req, res) => {
  const id = Number(req.params.id);
  const { currentOccupancy } = req.body || {};

  if (!id || id <= 0) {
    return res.status(400).json({ error: 'invalid shelter id' });
  }

  if (currentOccupancy === undefined) {
    return res.status(400).json({ error: 'currentOccupancy is required' });
  }

  const result = await service.updateOccupancy(id, Number(currentOccupancy));
  if (result.error) {
    return res.status(result.error.code).json({ error: result.error.message });
  }

  res.json(result.occupancy);
});

router.get('/openapi.yaml', (req, res) => {
  res.sendFile(path.join(__dirname, '../../api/openapi.yaml'));
});

router.get('/swagger.json', (req, res) => {
  res.redirect(301, '/openapi.yaml');
});

router.get('/docs', (_req, res) => {
  res.type('html').send(`<!doctype html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Shelter Service API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`);
});

router.use((req, res) => {
  res.status(404).json({ error: 'not found' });
});

module.exports = router;
