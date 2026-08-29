#!/usr/bin/env bash
set -euo pipefail

# 统一从项目根目录管理本地 Compose 生命周期。
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENV_FILE="$PROJECT_ROOT/.env"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.yml"

compose_command() {
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

show_help() {
    cat <<'EOF'
WeKnora local runtime manager

Usage: scripts/start_all.sh [option]

Options:
  -h, --help                Show help
  -d, --docker              Start the core services
  -a, --all                 Start the core services
      --no-pull             Start without pulling images
  -s, --stop                Stop the core services
  -c, --check               Validate configuration and show status
  -l, --list                Show service status
  -p, --pull                Pull official service images
  -r, --restart SERVICE     Restart one service
  -v, --version             Show version
EOF
}

require_runtime() {
    if ! command -v docker >/dev/null 2>&1; then
        echo "Docker is not installed." >&2
        return 1
    fi
    if ! docker compose version >/dev/null 2>&1; then
        echo "Docker Compose is not installed." >&2
        return 1
    fi
    if ! docker info >/dev/null 2>&1; then
        echo "Docker is not running." >&2
        return 1
    fi
    if [[ ! -f "$ENV_FILE" ]]; then
        echo "Missing local .env file." >&2
        return 1
    fi
    compose_command config --quiet
}

ensure_frontend_dist() {
    if [[ ! -d "$PROJECT_ROOT/frontend/dist" ]]; then
        "$PROJECT_ROOT/scripts/build_frontend_dist.sh"
    fi
}

pull_images() {
    compose_command pull app docreader postgres redis
}

build_frontend_image() {
    ensure_frontend_dist
    compose_command build frontend
}

start_services() {
    local pull_mode="$1"
    require_runtime
    if [[ "$pull_mode" == "pull" ]]; then
        pull_images
    fi
    build_frontend_image
    compose_command up -d --no-build --pull never
    compose_command ps
}

stop_services() {
    require_runtime
    compose_command down --remove-orphans
}

check_environment() {
    require_runtime
    echo "Configuration is valid."
    compose_command ps
}

restart_service() {
    local service_name="$1"
    require_runtime
    compose_command up -d --no-deps --no-build --pull never "$service_name"
    compose_command ps "$service_name"
}

main() {
    if (($# == 0)); then
        start_services pull
        return
    fi

    case "$1" in
        -h|--help)
            show_help
            ;;
        -d|--docker|-a|--all)
            start_services pull
            ;;
        --no-pull)
            start_services skip
            ;;
        -s|--stop)
            stop_services
            ;;
        -c|--check)
            check_environment
            ;;
        -l|--list)
            require_runtime
            compose_command ps
            ;;
        -p|--pull)
            require_runtime
            pull_images
            ;;
        -r|--restart)
            if (($# < 2)); then
                echo "Missing service name." >&2
                return 2
            fi
            restart_service "$2"
            ;;
        -v|--version)
            echo "WeKnora local runtime manager 2.0.0"
            ;;
        *)
            echo "Unknown option: $1" >&2
            show_help >&2
            return 2
            ;;
    esac
}

main "$@"
