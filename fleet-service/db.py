import os
from pymongo import MongoClient
from pymongo.errors import ConnectionFailure, ServerSelectionTimeoutError
import time

# MongoDB Configuration
MONGO_URI = os.getenv("MONGO_URI", "mongodb://mongo_root:mongo_pass@mongodb:27017/fleet_db?authSource=admin")
DB_NAME = os.getenv("DB_NAME", "fleet_db")

_client = None
_db = None


def get_client():
    """Get or create MongoDB client with retry logic"""
    global _client
    if _client is None:
        max_retries = 5
        retry_delay = 2
        for attempt in range(max_retries):
            try:
                _client = MongoClient(MONGO_URI, serverSelectionTimeoutMS=5000)
                # Test connection
                _client.admin.command('ping')
                print("Connected to MongoDB successfully")
                break
            except (ConnectionFailure, ServerSelectionTimeoutError) as e:
                if attempt < max_retries - 1:
                    print(f"MongoDB connection attempt {attempt + 1} failed, retrying in {retry_delay}s...")
                    time.sleep(retry_delay)
                else:
                    raise Exception(f"Failed to connect to MongoDB after {max_retries} attempts: {e}")
    return _client


def get_db():
    """Get MongoDB database connection"""
    global _db
    if _db is None:
        client = get_client()
        _db = client[DB_NAME]
    return _db


def init_db():
    """Initialize MongoDB collections and indexes"""
    db = get_db()
    
    # Create vehicles collection with index
    if 'vehicles' not in db.list_collection_names():
        db.create_collection('vehicles')
    
    db.vehicles.create_index('plate', unique=True)
    
    # Create trips collection with indexes
    if 'trips' not in db.list_collection_names():
        db.create_collection('trips')
    
    db.trips.create_index('vehicle_id')
    db.trips.create_index('driver_id')
    
    # Seed initial data
    if db.vehicles.count_documents({}) == 0:
        db.vehicles.insert_many([
            {"vehicle_id": 5, "type": "Truck", "plate": "TRK-001", "capacity": "5000kg", "status": "available"},
            {"vehicle_id": 6, "type": "Van", "plate": "VAN-001", "capacity": "2000kg", "status": "available"},
        ])
    
    print("MongoDB initialization complete")


def get_conn():
    """Return database connection object (for compatibility)"""
    return get_db()

