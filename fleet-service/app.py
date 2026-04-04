import os
import json
import logging
from flask import Flask, request, jsonify, send_file, redirect
from flask_cors import CORS
import requests
from bson import ObjectId
from pymongo.errors import DuplicateKeyError
from db import init_db, get_db

app = Flask(__name__)
CORS(app)

# Setup logging
logging.basicConfig(level=logging.DEBUG)
app.logger.setLevel(logging.DEBUG)

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


def service_put(url, payload):
    """Make a PUT request to a service and return (data, status_code)"""
    try:
        resp = requests.put(url, json=payload, timeout=5)
    except requests.RequestException:
        return None, 503
    try:
        data = resp.json() if resp.text else None
    except json.JSONDecodeError:
        return None, 502
    return data, resp.status_code


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

    db = get_db()
    try:
        result = db.vehicles.insert_one({
            "type": vtype,
            "plate": plate,
            "capacity": capacity,
            "status": "AVAILABLE"
        })
        vehicle_id = str(result.inserted_id)
    except DuplicateKeyError:
        app.logger.warning(f"Vehicle with plate '{plate}' already exists")
        return jsonify({"error": f"vehicle with plate '{plate}' already exists"}), 409
    except Exception as e:
        app.logger.error(f"Error creating vehicle: {e}")
        return jsonify({"error": "failed to create vehicle"}), 500

    return jsonify({"id": vehicle_id, "type": vtype, "status": "AVAILABLE"}), 201


@app.route("/vehicles", methods=["GET"])
def list_vehicles():
    db = get_db()
    vehicles = []
    for doc in db.vehicles.find({}):
        doc['id'] = str(doc['_id'])
        del doc['_id']
        vehicles.append(doc)
    return jsonify(vehicles)


@app.route("/vehicles/<vehicle_id>", methods=["GET"])
def get_vehicle(vehicle_id: str):
    db = get_db()
    try:
        vehicle = db.vehicles.find_one({"_id": ObjectId(vehicle_id)})
    except:
        vehicle = None
    
    if not vehicle:
        return jsonify({"error": "vehicle not found"}), 404
    
    vehicle['id'] = str(vehicle['_id'])
    del vehicle['_id']
    return jsonify(vehicle)


@app.route("/vehicles/<vehicle_id>", methods=["PUT"])
def update_vehicle(vehicle_id: str):
    body = request.get_json(force=True, silent=True) or {}
    vtype = str(body.get("type", "")).strip()
    status = str(body.get("status", "")).strip()
    
    if not vtype and not status:
        return jsonify({"error": "no update fields provided"}), 400

    db = get_db()
    update_data = {}
    if vtype:
        update_data["type"] = vtype
    if status:
        update_data["status"] = status
    
    try:
        result = db.vehicles.update_one(
            {"_id": ObjectId(vehicle_id)},
            {"$set": update_data}
        )
        if result.matched_count == 0:
            return jsonify({"error": "vehicle not found"}), 404
    except Exception as e:
        return jsonify({"error": "failed to update vehicle"}), 500

    vehicle = db.vehicles.find_one({"_id": ObjectId(vehicle_id)})
    vehicle['id'] = str(vehicle['_id'])
    del vehicle['_id']
    return jsonify(vehicle)


@app.route("/vehicles/<vehicle_id>", methods=["DELETE"])
def delete_vehicle(vehicle_id: str):
    db = get_db()
    try:
        result = db.vehicles.delete_one({"_id": ObjectId(vehicle_id)})
        if result.deleted_count == 0:
            return jsonify({"error": "vehicle not found"}), 404
    except Exception as e:
        return jsonify({"error": "failed to delete vehicle"}), 500
    return jsonify({"message": "vehicle deleted"}), 204


@app.route("/trips", methods=["GET"])
def list_trips():
    db = get_db()
    trips = []
    for doc in db.trips.find({}):
        trip_id = str(doc['_id'])
        doc['id'] = trip_id
        doc['tripId'] = trip_id
        del doc['_id']
        trips.append(doc)
    return jsonify(trips)


@app.route("/trips", methods=["POST"])
def create_trip():
    body = request.get_json(force=True, silent=True) or {}
    app.logger.info(f"Create trip request body: {body}")
    
    vehicle_id = body.get("vehicleId")
    driver_id = body.get("driverId")
    cargo_id = body.get("cargoId")
    destination_id = body.get("destinationId")

    # Check each field individually for detailed error messages
    if vehicle_id is None:
        app.logger.warning(f"Missing vehicleId in request: {body}")
        return jsonify({"error": "vehicleId is required (got: null)"}), 400
    if driver_id is None:
        app.logger.warning(f"Missing driverId in request: {body}")
        return jsonify({"error": "driverId is required (got: null)"}), 400
    if cargo_id is None:
        app.logger.warning(f"Missing cargoId in request: {body}")
        return jsonify({"error": "cargoId is required (got: null)"}), 400
    if destination_id is None:
        app.logger.warning(f"Missing destinationId in request: {body}")
        return jsonify({"error": "destinationId is required (got: null)"}), 400
    
    app.logger.info(f"Trip request validated: vehicle_id={vehicle_id}, driver_id={driver_id}, cargo_id={cargo_id}, destination_id={destination_id}")

    db = get_db()
    
    # Verify vehicle availability
    try:
        vehicle = db.vehicles.find_one({"_id": ObjectId(vehicle_id)})
    except Exception as e:
        app.logger.error(f"Invalid vehicleId format: {vehicle_id}, error: {e}")
        return jsonify({"error": f"invalid vehicleId format: {vehicle_id}"}), 400
    
    if not vehicle:
        app.logger.warning(f"Vehicle not found: {vehicle_id}")
        return jsonify({"error": f"vehicle not found with ID: {vehicle_id}"}), 404
    
    vehicle_status = vehicle.get("status", "UNKNOWN").upper()
    if vehicle_status != "AVAILABLE":
        app.logger.warning(f"Vehicle {vehicle_id} not available - status: {vehicle_status}")
        return jsonify({"error": f"vehicle {vehicle_id} is not available (current status: {vehicle_status})"}), 409

    # Verify external services
    volunteer_data, code = service_get(f"{VOLUNTEER_SERVICE_URL}/volunteers/{driver_id}")
    if code != 200:
        if code == 404:
            return jsonify({"error": f"driver with ID {driver_id} not found"}), 404
        else:
            return jsonify({"error": "failed to verify driver with volunteer service"}), 503
    if volunteer_data.get("role") != "DRIVER":
        actual_role = volunteer_data.get("role", "unknown")
        return jsonify({"error": f"driver must have DRIVER role, but has {actual_role} role"}), 400
    
    resource_data, code = service_get(f"{RESOURCE_SERVICE_URL}/resources/{cargo_id}")
    if code != 200:
        if code == 404:
            return jsonify({"error": f"cargo/resource with ID {cargo_id} not found"}), 404
        else:
            return jsonify({"error": "failed to verify cargo with resource service"}), 503
    
    shelter_data, code = service_get(f"{SHELTER_SERVICE_URL}/shelters/{destination_id}")
    if code != 200:
        if code == 404:
            return jsonify({"error": f"shelter with ID {destination_id} not found"}), 404
        else:
            return jsonify({"error": "failed to verify shelter with shelter service"}), 503
    shelter_status = shelter_data.get("status", "unknown")
    if shelter_status.upper() != "OPEN":
        return jsonify({"error": f"shelter is not available for trips - current status: {shelter_status}"}), 400

    try:
        trip_result = db.trips.insert_one({
            "vehicle_id": vehicle_id,
            "driver_id": driver_id,
            "cargo_id": cargo_id,
            "destination_id": destination_id,
            "status": "SCHEDULED"
        })
        trip_id = str(trip_result.inserted_id)
        app.logger.info(f"Trip created: {trip_id}")
        
        # Update vehicle status to IN_USE
        try:
            # Try both string ID and ObjectId formats
            app.logger.info(f"Updating vehicle {vehicle_id} (type: {type(vehicle_id).__name__}) to IN_USE")
            result = db.vehicles.update_one(
                {"_id": vehicle_id},  # First try with string
                {"$set": {"status": "IN_USE"}}
            )
            app.logger.info(f"Vehicle update (string): matched={result.matched_count}, modified={result.modified_count}")
            if result.matched_count == 0:
                # If not found with string, try with ObjectId
                try:
                    result = db.vehicles.update_one(
                        {"_id": ObjectId(vehicle_id)},
                        {"$set": {"status": "IN_USE"}}
                    )
                    app.logger.info(f"Vehicle update (ObjectId): matched={result.matched_count}, modified={result.modified_count}")
                except Exception as oid_err:
                    app.logger.error(f"ObjectId conversion failed: {oid_err}")
            
            if result.matched_count == 0:
                app.logger.warning(f"Vehicle {vehicle_id} not found when updating status")
            elif result.modified_count == 0:
                app.logger.warning(f"Vehicle {vehicle_id} was not modified (already IN_USE?)")
        except Exception as ve:
            app.logger.error(f"Error updating vehicle status: {type(ve).__name__}: {ve}")
        
        # Update driver (volunteer) status to IN_USE
        try:
            app.logger.info(f"Updating driver {driver_id} status to IN_USE")
            _, code = service_put(
                f"{VOLUNTEER_SERVICE_URL}/volunteers/{driver_id}",
                {"status": "IN_USE"}
            )
            app.logger.info(f"Driver update response code: {code}")
            if code != 200:
                app.logger.warning(f"Failed to update driver {driver_id} status to IN_USE (code: {code})")
        except Exception as de:
            app.logger.error(f"Error updating driver status: {type(de).__name__}: {de}")
    except Exception as e:
        app.logger.error(f"Error in create_trip: {type(e).__name__}: {e}")
        return jsonify({"error": "failed to create trip"}), 500

    return jsonify({"id": trip_id, "tripId": trip_id, "status": "SCHEDULED"}), 201


@app.route("/trips/volunteer/<driver_id>", methods=["GET"])
def trips_for_volunteer(driver_id: str):
    db = get_db()
    trips = db.trips.find({
        "driver_id": int(driver_id) if driver_id.isdigit() else driver_id,
        "status": {"$in": ["SCHEDULED", "IN_PROGRESS", "BUSY"]}
    })
    has_active = any(trips)
    return jsonify({"volunteerId": driver_id, "hasActiveTrip": has_active})


@app.route("/trips/<trip_id>", methods=["GET"])
def get_trip(trip_id: str):
    db = get_db()
    try:
        trip = db.trips.find_one({"_id": ObjectId(trip_id)})
    except:
        trip = None
    
    if not trip:
        return jsonify({"error": "trip not found"}), 404
    
    trip_id_str = str(trip['_id'])
    trip['id'] = trip_id_str
    trip['tripId'] = trip_id_str
    del trip['_id']
    return jsonify(trip)


@app.route("/trips/<trip_id>", methods=["PUT"])
def update_trip_status(trip_id: str):
    body = request.get_json(force=True, silent=True) or {}
    status = str(body.get("status", "")).strip()
    if not status:
        return jsonify({"error": "status is required"}), 400

    db = get_db()
    try:
        trip = db.trips.find_one({"_id": ObjectId(trip_id)})
        if not trip:
            return jsonify({"error": "trip not found"}), 404
        
        # If completing trip, mark vehicle and driver as available
        if status == "COMPLETED":
            vehicle_id = trip.get("vehicle_id")
            driver_id = trip.get("driver_id")
            
            # Mark vehicle as AVAILABLE (try both string and ObjectId)
            try:
                result = db.vehicles.update_one(
                    {"_id": vehicle_id},  # Try string first
                    {"$set": {"status": "AVAILABLE"}}
                )
                if result.matched_count == 0:
                    try:
                        result = db.vehicles.update_one(
                            {"_id": ObjectId(vehicle_id)},
                            {"$set": {"status": "AVAILABLE"}}
                        )
                    except:
                        pass
            except Exception as e:
                print(f"error updating vehicle status: {e}")
            
            # Mark driver (volunteer) as AVAILABLE
            try:
                _, code = service_put(
                    f"{VOLUNTEER_SERVICE_URL}/volunteers/{driver_id}",
                    {"status": "AVAILABLE"}
                )
                if code != 200:
                    # Log warning but don't fail - trip update is still valid
                    print(f"warning: failed to update driver {driver_id} status to AVAILABLE (code: {code})")
            except Exception as e:
                print(f"error updating driver status: {e}")
        
        db.trips.update_one(
            {"_id": ObjectId(trip_id)},
            {"$set": {"status": status}}
        )
    except Exception as e:
        return jsonify({"error": "failed to update trip"}), 500
    
    return jsonify({"message": "trip updated"})


@app.route("/trips/<trip_id>", methods=["DELETE"])
def delete_trip(trip_id: str):
    db = get_db()
    try:
        trip = db.trips.find_one({"_id": ObjectId(trip_id)})
        if not trip:
            return jsonify({"error": "trip not found"}), 404
        
        # Mark vehicle and driver as AVAILABLE when trip is deleted
        vehicle_id = trip.get("vehicle_id")
        driver_id = trip.get("driver_id")
        trip_status = trip.get("status")
        
        # Only mark as available if trip was not already completed
        if trip_status != "COMPLETED":
            # Mark vehicle as AVAILABLE (try both string and ObjectId)
            try:
                result = db.vehicles.update_one(
                    {"_id": vehicle_id},  # Try string first
                    {"$set": {"status": "AVAILABLE"}}
                )
                if result.matched_count == 0:
                    try:
                        result = db.vehicles.update_one(
                            {"_id": ObjectId(vehicle_id)},
                            {"$set": {"status": "AVAILABLE"}}
                        )
                    except:
                        pass
            except Exception as e:
                print(f"error updating vehicle status when deleting: {e}")
            
            # Mark driver as AVAILABLE
            try:
                _, code = service_put(
                    f"{VOLUNTEER_SERVICE_URL}/volunteers/{driver_id}",
                    {"status": "AVAILABLE"}
                )
                if code != 200:
                    print(f"warning: failed to update driver {driver_id} status when deleting trip (code: {code})")
            except Exception as e:
                print(f"error updating driver status when deleting: {e}")
        
        result = db.trips.delete_one({"_id": ObjectId(trip_id)})
        if result.deleted_count == 0:
            return jsonify({"error": "trip not found"}), 404
    except Exception as e:
        return jsonify({"error": "failed to delete trip"}), 500
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
