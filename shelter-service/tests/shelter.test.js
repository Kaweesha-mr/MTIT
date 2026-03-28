const request = require('supertest');
const express = require('express');
const router = require('../src/routes/shelterRoutes');

// Mock the service
jest.mock('../src/services/shelterService', () => ({
  createShelter: jest.fn().mockResolvedValue({ shelter: { id: 1, name: 'Test Shelter' } }),
  listShelters: jest.fn().mockResolvedValue([{ id: 1, name: 'Test Shelter' }]),
  getShelter: jest.fn().mockResolvedValue({ id: 1, name: 'Test Shelter' }),
  updateShelter: jest.fn().mockResolvedValue({ shelter: { id: 1, name: 'Updated Shelter' } }),
  deleteShelter: jest.fn().mockResolvedValue({ success: true }),
  updateOccupancy: jest.fn().mockResolvedValue({ occupancy: { id: 1, currentOccupancy: 50 } }),
}));

const app = express();
app.use(express.json());
app.use('/', router);

describe('Shelter API CRUD', () => {
  it('GET /health returns 200', async () => {
    const res = await request(app).get('/health');
    expect(res.statusCode).toEqual(200);
  });

  it('POST /shelters creates a shelter', async () => {
    const res = await request(app).post('/shelters').send({ name: 'Test' });
    expect(res.statusCode).toEqual(201);
  });

  it('GET /shelters lists shelters', async () => {
    const res = await request(app).get('/shelters');
    expect(res.statusCode).toEqual(200);
    expect(Array.isArray(res.body)).toBeTruthy();
  });

  it('GET /shelters/:id returns a shelter', async () => {
    const res = await request(app).get('/shelters/1');
    expect(res.statusCode).toEqual(200);
  });

  it('PUT /shelters/:id updates a shelter', async () => {
    const res = await request(app).put('/shelters/1').send({ name: 'Updated' });
    expect(res.statusCode).toEqual(200);
  });

  it('DELETE /shelters/:id deletes a shelter', async () => {
    const res = await request(app).delete('/shelters/1');
    expect(res.statusCode).toEqual(204);
  });
});
