# Dione

A mobile friendly UI for ethminer. Can view multiple rigs and consolidated the metrics.

## Requirements

You should run the ethminer with api mode

Example:
```bash
./ethminer -P `your address` --api-port -9033
```

Now run dione

```bash
# Assume the above rig runs at 192.168.0.33:9033
# if running local use localhost:9033
DIONE_WORKER_ADDRESS=192.168.0.33:9033
# Multiple workers/rigs can be configure with ','
# DIONE_WORKER_ADDRESS=192.168.0.33:9033,192.168.0.34:9033,192.168.0.35:9033
# store the metrics in the local folder(same as where you execute the cmd)
DIONE_DB_PATH="./stats.db"
./dione
```

Open http://localhost:8080 to view the dashboard.