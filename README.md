# Discussion Board

A simple, clean discussion board built with Go and PostgreSQL.

## Features

- 👥 **Guest Access**: Anyone can view posts
- 🔐 **User Authentication**: Register, login, and post
- 🎨 **Clean UI**: Modern, responsive design
- 🐳 **Docker Support**: Easy deployment with Docker Compose
- 📝 **Console Activation**: No email setup required

## Quick Start

1. **Clone and Setup**
   ```bash
   git clone <your-repo>
   cd discussionboard
   cp .env.example .env
   ```

2. **Run with Docker**
   ```bash
   docker-compose up
   ```

3. **Development Mode**
   ```bash
   make dev
   ```

## Usage

1. Visit `http://localhost:8080`
2. **As Guest**: View all posts immediately
3. **Register**: Create account → Check console for activation link → Login → Post

## Project Structure

```
├── cmd/api/              # Application handlers
├── internal/
│   ├── data/            # Database operations
│   └── config/          # Configuration management
├── templates/           # HTML templates
└── docker-compose.yml   # Docker setup
```

## Environment Variables

- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `PORT` - Application port (default: 8080)

## Development Commands

- `make dev` - Run in development mode
- `make build` - Build binary
- `make docker-run` - Run with Docker
- `make clean` - Clean up containers and volumes