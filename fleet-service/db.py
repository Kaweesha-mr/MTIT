import os
import mysql.connector
from mysql.connector import pooling


DB_CONFIG = {
    "host": os.getenv("DB_HOST", "localhost"),
    "port": int(os.getenv("DB_PORT", "3307")),
    "user": os.getenv("DB_USER", "fleet_user"),
    "password": os.getenv("DB_PASSWORD", "fleet_pass"),
    "database": os.getenv("DB_NAME", "fleet_db"),
    "autocommit": True,
}

_pool = pooling.MySQLConnectionPool(pool_name="fleet_pool", pool_size=5, **DB_CONFIG)


def get_conn():
    return _pool.get_connection()


def init_db():
    with get_conn() as conn:
        cur = conn.cursor()
        cur.execute(
            """
            CREATE TABLE IF NOT EXISTS vehicles (
                id INT AUTO_INCREMENT PRIMARY KEY,
                type VARCHAR(50) NOT NULL,
                plate VARCHAR(50) NOT NULL UNIQUE,
                capacity VARCHAR(50) NOT NULL,
                status VARCHAR(20) NOT NULL
            );
            """
        )
        cur.execute(
            """
            CREATE TABLE IF NOT EXISTS trips (
                id INT AUTO_INCREMENT PRIMARY KEY,
                vehicle_id INT NOT NULL,
                driver_id INT NOT NULL,
                cargo_id INT NOT NULL,
                destination_id INT NOT NULL,
                status VARCHAR(20) NOT NULL,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
            );
            """
        )
        cur.execute("ALTER TABLE vehicles AUTO_INCREMENT = 5;")
        cur.execute("ALTER TABLE trips AUTO_INCREMENT = 801;")
        cur.close()
