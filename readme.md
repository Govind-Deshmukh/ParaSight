# Parasight

Lightweight Go agent that runs alongside your application. Exposes logs and system metrics via HTTP for pull-based monitoring.

## Download

```bash
wget https://github.com/govind-deshmukh/parasight/raw/main/releases/parasight-linux-amd64-release.1.0.0
chmod +x parasight-linux-amd64-release.1.0.0
```

## Usage

```bash
./parasight-linux-amd64-release.1.0.0 -p 39998 -logs "app:/path/to/app.log,error:/path/to/error.log" -system_metrics "cpu,memory,disk"
```

## Build from source

```bash
go build -o parasight .
```

## Endpoints

| Endpoint        | Description              |
| --------------- | ------------------------ |
| `/app`          | Last 20 lines of app log |
| `/app?lines=50` | Last 50 lines (max 100)  |
| `/metrics`      | CPU, memory, disk stats  |
| `/health`       | Agent status             |

## Example Response

`GET /metrics`

```json
{
  "timestamp": 1706745600,
  "cpu": { "used_percent": 23.45, "free_percent": 76.55 },
  "memory": [
    { "type": "ram", "total_mb": 16384, "used_mb": 8192, "free_mb": 8192 },
    { "type": "swap", "total_mb": 4096, "used_mb": 512, "free_mb": 3584 }
  ],
  "disk": [{ "mount": "/", "total_gb": 500, "used_gb": 200, "free_gb": 300 }]
}
```

## Issues & Contact

For issues, raise them on the repository.

For help, contact: Govind Deshmukh (govind.ub47@gmail.com)

## License

MIT - Free to use and modify.
