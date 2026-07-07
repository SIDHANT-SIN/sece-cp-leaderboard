# SECE CP Leaderboard

A modern, mobile-friendly Codeforces group leaderboard web app built with Go (Gin), Redis, Asynq, and Turso (SQLite).

## Features

- Fast, always-up-to-date leaderboard (no API calls on user visits)
- Asynq + Redis background job pipeline for handling Codeforces API calls
- Redis-based caching layer for leaderboard queries
- Composite database indexing for fast past/historical leaderboard queries
- Scheduled background rating updates, with manual refresh control available from the admin panel
- Past batches leaderboard, filterable by batch year
- SECE CP contest events listing
- ICPC PYQ (previous year questions) links section
- Admin dashboard for user/contest management
- Custom scoring formula and ranking logic
- Automated testing suite (unit, smoke, and integration tests) with GitHub Actions CI — includes linting and vulnerability scanning
- Modern dark UI, responsive design
- Minimal, consistent and secured admin panel

## Setup

1. **Clone the repo**
```bash
git clone https://github.com/SIDHANT-SIN/sece-cp-leaderboard.git
cd sece-cp-leaderboard
```

2. **Create a `.env` file**
```
ADMIN_USERNAME=your_admin_name
ADMIN_PASSWORD=sha256_hash_of_password
MAINTAINER_PASSWORD=your_password_hash_here
TURSO_DATABASE_URL=turso_url
TURSO_AUTH_TOKEN=auth_token
PORT=your_port
REDIS_URL=your_redis_url
```

- To generate a password hash:
  ```bash
  echo -n 'yourpassword' | sha256sum
  ```
  Use the hex string (without trailing dash/filename).

3. **Run the server**

Using `air` (live reload, config in `.air.toml`):
```bash
air
```

Or directly with `go run`:
```bash
go run src/main.go
```

4. **Visit**
- Leaderboard: [http://localhost:8080/](http://localhost:8080/)
- Admin: [http://localhost:8080/admin](http://localhost:8080/admin)

## License

MIT
