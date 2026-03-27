const axios = require('axios');
const { pool } = require('../db');
const config = require('../config');

async function verifyIncident(incidentId) {
  try {
    const resp = await axios.get(`${config.incidentServiceUrl}/incidents/${incidentId}`);
    const status = (resp.data.status || '').toUpperCase();
    if (status !== 'ACTIVE') {
      return { ok: false, code: 400, message: 'incident is not active' };
    }
    return { ok: true };
  } catch (err) {
    if (err.response && err.response.status === 404) {
      return { ok: false, code: 400, message: 'incident not found' };
    }
    return { ok: false, code: 503, message: 'unable to verify incident' };
  }
}

async function createShelter(payload) {
  const incidentId = Number(payload.incidentId);
  const name = (payload.name || '').trim();
  const capacity = Number(payload.capacity);

  if (!incidentId || incidentId <= 0 || !name || !capacity || capacity <= 0) {
    return { error: { code: 400, message: 'incidentId, name and capacity are required' } };
  }

  const verification = await verifyIncident(incidentId);
  if (!verification.ok) {
    return { error: { code: verification.code, message: verification.message } };
  }

  const result = await pool.query(
    'INSERT INTO shelters (incident_id, name, capacity, status, location) VALUES ($1, $2, $3, $4, $5) RETURNING id, status, name, location',
    [incidentId, name, capacity, 'REQUEST', null],
  );

  const shelter = result.rows[0];
  return { shelter: { id: shelter.id, incidentId: incidentId, name: shelter.name, status: shelter.status, location: shelter.location } };
}

async function listShelters() {
  const result = await pool.query('SELECT id, name, status, location FROM shelters ORDER BY id');
  return result.rows;
}

async function getShelter(id) {
  const result = await pool.query(
    'SELECT id, name, capacity, status, current_occupancy, location FROM shelters WHERE id = $1',
    [id],
  );
  if (result.rowCount === 0) {
    return null;
  }
  const row = result.rows[0];
  return {
    id: row.id,
    name: row.name,
    capacity: row.capacity,
    status: row.status,
    currentOccupancy: row.current_occupancy,
    location: row.location,
  };
}

async function updateOccupancy(id, currentOccupancy) {
  const shelter = await getShelter(id);
  if (!shelter) return { error: { code: 404, message: 'shelter not found' } };

  if (currentOccupancy < 0 || currentOccupancy > shelter.capacity) {
    return { error: { code: 400, message: 'currentOccupancy must be between 0 and capacity' } };
  }

  await pool.query(
    'UPDATE shelters SET current_occupancy = $1 WHERE id = $2',
    [currentOccupancy, id],
  );

  return { occupancy: { id, occupancy: currentOccupancy } };
}

async function updateShelter(id, payload) {
  const shelter = await getShelter(id);
  if (!shelter) return { error: { code: 404, message: 'shelter not found' } };

  const name = (payload.name || '').trim();
  const capacity = Number(payload.capacity);
  const status = (payload.status || '').trim().toUpperCase();
  const location = (payload.location || '').trim() || null;

  if (!name || !capacity || capacity <= 0) {
    return { error: { code: 400, message: 'name and capacity are required' } };
  }

  if (capacity < shelter.currentOccupancy) {
    return { error: { code: 400, message: 'new capacity cannot be less than current occupancy' } };
  }

  if (status && status !== 'OPEN' && status !== 'REQUEST' && status !== 'CLOSED') {
    return { error: { code: 400, message: 'status must be OPEN, REQUEST, or CLOSED' } };
  }

  const result = await pool.query(
    'UPDATE shelters SET name = $1, capacity = $2, status = $3, location = $4 WHERE id = $5 RETURNING id, name, capacity, status, current_occupancy, location',
    [name, capacity, status || shelter.status, location, id],
  );

  const updated = result.rows[0];
  return {
    shelter: {
      id: updated.id,
      name: updated.name,
      capacity: updated.capacity,
      status: updated.status,
      currentOccupancy: updated.current_occupancy,
      location: updated.location,
    },
  };
}

async function deleteShelter(id) {
  const shelter = await getShelter(id);
  if (!shelter) return { error: { code: 404, message: 'shelter not found' } };

  await pool.query('DELETE FROM shelters WHERE id = $1', [id]);
  return { success: true };
}

module.exports = {
  createShelter,
  listShelters,
  getShelter,
  updateOccupancy,
  updateShelter,
  deleteShelter,
};
