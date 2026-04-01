import os
import json
from flask import Flask, request, jsonify, send_file, redirect
from flask_cors import CORS
import requests
from db import init_db, get_conn

app = Flask(__name__)
CORS(app)

VOLUNTEER_SERVICE_URL = os.getenv("VOLUNTEER_SERVICE_URL", "http://localhost:8082").rstrip("/")
RESOURCE_SERVICE_URL = os.getenv("RESOURCE_SERVICE_URL", "http://localhost:8083").rstrip("/")
SHELTER_SERVICE_URL = os.getenv("SHELTER_SERVICE_URL", "http://localhost:8084").rstrip("/")
PORT = int(os.getenv("PORT", "8086"))
OPENAPI_PATH = os.path.join(os.path.dirname(__file__), "api", "openapi.yaml")


def service_get(url, expected_key=None):
    try:
        resp = requests.get(url, timeout=5)
    except requests.RequestException:
        return None, 503
    if resp.status_code != 200:
        return None, resp.status_code
    try:
        data = resp.json()
    except json.JSONDecodeError:
        return None, 502
    if expected_key and expected_key not in data:
        return None, 502
    return data, 200


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok"})


@app.route("/vehicles", methods=["POST"])
def create_vehicle():
    body = request.get_json(force=True, silent=True) or {}
    vtype = str(body.get("type", "")).strip()
    plate = str(body.get("plate", "")).strip()
    capacity = str(body.get("capacity", "")).strip()

    if not vtype or not plate or not capacity:
        return jsonify({"error": "type, plate and capacity are required"}), 400

    with get_conn() as conn:
        cur = conn.cursor()
        try:
            cur.execute(
                "INSERT INTO vehicles (type, plate, capacity, status) VALUES (%s, %s, %s, %s)",
                (vtype, plate, capacity, "AVAILABLE"),
            )
            vehicle_id = cur.lastrowid
        except Exception:
            cur.close()
            return jsonify({"error": "failed to create vehicle"}), 500
        cur.close()

    return jsonify({"id": vehicle_id, "type": vtype, "status": "AVAILABLE"}), 201


@app.route("/vehicles", methods=["GET"])
def list_vehicles():
    with get_conn() as conn:
        cur = conn.cursor(dictionary=True)
        cur.execute("SELECT id, type, plate, capacity, status FROM vehicles ORDER BY id")
        rows = cur.fetchall()
        cur.close()
    return jsonify(rows)


@app.route("/vehicles/<int:vehicle_id>", methods=["GET"])
def get_vehicle(vehicle_id: int):
    with get_conn() as conn:
        cur = conn.cursor(dictionary=True)
        cur.execute("SELECT id, type, plate, capacity, status FROM vehicles WHERE id = %s", (vehicle_id,))
        row = cur.fetchone()
        cur.close()
    if not row:
        return jsonify({"error": "vehicle not found"}), 404
    return jsonify(row)


@app.route("/vehicles/<int:vehicle_id>", methods=["PUT"])
def update_vehicle(vehicle_id: int):
    body = request.get_json(force=True, silent=True) or {}
    vtype = str(body.get("type", "")).strip()
    status = str(body.get("status", "")).strip()
    
    if not vtype and not status:
        return jsonify({"error": "no update fields provided"}), 400

    with get_conn() as conn:
        cur = conn.cursor()
        try:
            if vtype and status:
                cur.execute("UPDATE vehicles SET type = %s, status = %s WHERE id = %s", (vtype, status, vehicle_id))
            elif vtype:
                cur.execute("UPDATE vehicles SET type = %s WHERE id = %s", (vtype, vehicle_id))
            elif status:
                cur.execute("UPDATE vehicles SET status = %s WHERE id = %s", (status, vehicle_id))
            
            if cur.rowcount == 0:
                return jsonify({"error": "vehicle not found"}), 404
            conn.commit()
        except Exception:
            return jsonify({"error": "failed to update vehicle"}), 500
        finally:
            cur.close()
    return jsonify({"message": "vehicle updated"})


@app.route("/vehicles/<int:vehicle_id>", methods=["DELETE"])
def delete_vehicle(vehicle_id: int):
    with get_conn() as conn:
        cur = conn.cursor()
        try:
            cur.execute("DELETE FROM vehicles WHERE id = %s", (vehicle_id,))
            if cur.rowcount == 0:
                return jsonify({"error": "vehicle not found"}), 404
            conn.commit()
        except Exception:
            return jsonify({"error": "failed to delete vehicle"}), 500
        finally:
            cur.close()
    return jsonify({"message": "vehicle deleted"}), 204


@app.route("/trips", methods=["GET"])
def list_trips():
    with get_conn() as conn:
        cur = conn.cursor(dictionary=True)
        cur.execute("SELECT id, status FROM trips ORDER BY id")
        rows = cur.fetchall()
        cur.close()
    trips = [{"tripId": r["id"], "status": r["status"]} for r in rows]
    return jsonify(trips)


@app.route("/trips", methods=["POST"])
def create_trip():
    body = request.get_json(force=True, silent=True) or {}
    vehicle_id = body.get("vehicleId")
    driver_id = body.get("driverId")
    cargo_id = body.get("cargoId")
    destination_id = body.get("destinationId")

    if not all([vehicle_id, driver_id, cargo_id, destination_id]):
        return jsonify({"error": "vehicleId, driverId, cargoId, destinationId are required"}), 400

    # Verify vehicle availability
    with get_conn() as conn:
        cur = conn.cursor(dictionary=True)
        cur.execute("SELECT id, status FROM vehicles WHERE id = %s", (vehicle_id,))
        vehicle = cur.fetchone()
        if not vehicle:
            cur.close()
            return jsonify({"error": "vehicle not found"}), 404
        if vehicle["status"] != "AVAILABLE":
            cur.close()
            return jsonify({"error": "vehicle is not available"}), 409

        # Verify external services
        volunteer_data, code = service_get(f"{VOLUNTEER_SERVICE_URL}/volunteers/{driver_id}")
        if code != 200:
            return jsonify({"error": "driver verification failed"}), 400 if code == 404 else 503
        if volunteer_data.get("role") != "DRIVER":
            cur.close()
            return jsonify({"error": "volunteer must have DRIVER role"}), 400
        resource_data, code = service_get(f"{RESOURCE_SERVICE_URL}/resources/{cargo_id}")
        if code != 200:
            return jsonify({"error": "cargo verification failed"}), 400 if code == 404 else 503
        shelter_data, code = service_get(f"{SHELTER_SERVICE_URL}/shelters/{destination_id}")
        if code != 200:
            return jsonify({"error": "destination verification failed"}), 400 if code == 404 else 503
        if shelter_data.get("status") != "OPEN":
            cur.close()
            return jsonify({"error": "shelter not available or not prepared"}), 400

        try:
            cur.execute(
                "INSERT INTO trips (vehicle_id, driver_id, cargo_id, destination_id, status) VALUES (%s, %s, %s, %s, %s)",
                (vehicle_id, driver_id, cargo_id, destination_id, "SCHEDULED"),
            )
            trip_id = cur.lastrowid
            cur.execute("UPDATE vehicles SET status = %s WHERE id = %s", ("BUSY", vehicle_id))
            conn.commit()
        except Exception:
            cur.close()
            return jsonify({"error": "failed to create trip"}), 500
        cur.close()

    return jsonify({"tripId": trip_id, "status": "SCHEDULED"}), 201


@app.route("/trips/volunteer/<int:volunteer_id>", methods=["GET"])
def trips_for_volunteer(volunteer_id: int):
    with get_conn() as conn:
        cur = conn.cursor(dictionary=True)
        cur.execute(
            "SELECT id, status FROM trips WHERE driver_id = %s AND status IN ('SCHEDULED', 'IN_PROGRESS', 'BUSY')",
            (volunteer_id,),
        )
        rows = cur.fetchall()
        cur.close()
    has_active = len(rows) > 0
    return jsonify({"volunteerId": volunteer_id, "hasActiveTrip": has_active})


@app.route("/trips/<int:trip_id>", methods=["GET"])
def get_trip(trip_id: int):
    with get_conn() as conn:
        cur = conn.cursor(dictionary=True)
        cur.execute("SELECT id, vehicle_id, driver_id, cargo_id, destination_id, status FROM trips WHERE id = %s", (trip_id,))
        row = cur.fetchone()
        cur.close()
    if not row:
        return jsonify({"error": "trip not found"}), 404
    return jsonify(row)


@app.route("/trips/<int:trip_id>", methods=["PUT"])
def update_trip_status(trip_id: int):
    body = request.get_json(force=True, silent=True) or {}
    status = str(body.get("status", "")).strip()
    if not status:
        return jsonify({"error": "status is required"}), 400

    with get_conn() as conn:
        cur = conn.cursor()
        try:
            if status == "COMPLETED":
                cur.execute("SELECT vehicle_id FROM trips WHERE id = %s", (trip_id,))
                trip = cur.fetchone()
                if trip:
                    cur.execute("UPDATE vehicles SET status = 'AVAILABLE' WHERE id = %s", (trip[0],))
                    
            cur.execute("UPDATE trips SET status = %s WHERE id = %s", (status, trip_id))
            if cur.rowcount == 0:
                return jsonify({"error": "trip not found"}), 404
            conn.commit()
        except Exception:
            return jsonify({"error": "failed to update trip"}), 500
        finally:
            cur.close()
    return jsonify({"message": "trip updated"})


@app.route("/trips/<int:trip_id>", methods=["DELETE"])
def delete_trip(trip_id: int):
    with get_conn() as conn:
        cur = conn.cursor()
        try:
            cur.execute("DELETE FROM trips WHERE id = %s", (trip_id,))
            if cur.rowcount == 0:
                return jsonify({"error": "trip not found"}), 404
            conn.commit()
        except Exception:
            return jsonify({"error": "failed to delete trip"}), 500
        finally:
            cur.close()
    return jsonify({"message": "trip deleted"}), 204


@app.route("/swagger.json", methods=["GET"])
def swagger_json():
    return redirect("/openapi.yaml", code=301)


@app.route("/openapi.yaml", methods=["GET"])
def openapi_yaml():
    return send_file(OPENAPI_PATH, mimetype="application/yaml")


@app.route("/swagger", methods=["GET"])
@app.route("/swagger/", methods=["GET"])
@app.route("/docs", methods=["GET"])
@app.route("/docs/", methods=["GET"])
def docs():
    return """<!doctype html>
<html lang=\"en\">
<head>
    <meta charset=\"UTF-8\" />
    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\" />
    <title>Fleet Service API Docs</title>
    <link rel=\"stylesheet\" href=\"https://unpkg.com/swagger-ui-dist@5/swagger-ui.css\" />
</head>
<body>
    <div id=\"swagger-ui\"></div>
    <script src=\"https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js\"></script>
    <script>
        window.ui = SwaggerUIBundle({
            url: \"/openapi.yaml\",
            dom_id: \"#swagger-ui\",
            deepLinking: true,
            presets: [SwaggerUIBundle.presets.apis],
            layout: \"BaseLayout\"
        });
    </script>
</body>
</html>""", 200, {"Content-Type": "text/html; charset=utf-8"}


if __name__ == "__main__":
    init_db()
    app.run(host="0.0.0.0", port=PORT)
