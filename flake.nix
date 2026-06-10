{
  description = "Peril - boot.dev Learn Pub/Sub (RabbitMQ) project";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };

        # Project configuration
        projectName = "peril";
        rabbitImage = "rabbitmq:3.13-management";
        rabbitContainer = "peril_rabbitmq";
        amqpUrl = "amqp://guest:guest@localhost:5672/";

        # RabbitMQ helper scripts (Docker-backed, mirroring rabbit.sh)
        rabbitstart = pkgs.writeShellScriptBin "rabbitstart" ''
          if ! docker info >/dev/null 2>&1; then
            echo "✗ Docker daemon not reachable."
            echo "  Ensure 'virtualisation.docker.enable = true;' is set on the host"
            echo "  and that your user is in the 'docker' group."
            exit 1
          fi

          if docker inspect ${rabbitContainer} >/dev/null 2>&1; then
            echo "Starting existing ${rabbitContainer} container..."
            docker start ${rabbitContainer} >/dev/null
          else
            echo "Creating new ${rabbitContainer} container..."
            docker run -d --name ${rabbitContainer} \
              -p 5672:5672 -p 15672:15672 \
              ${rabbitImage} >/dev/null
          fi

          # Wait for the broker to accept connections
          printf "Waiting for RabbitMQ to be ready"
          for i in $(seq 1 30); do
            if docker exec ${rabbitContainer} rabbitmq-diagnostics -q ping >/dev/null 2>&1; then
              echo " ✓"
              echo "  AMQP:       ${amqpUrl}"
              echo "  Management: http://localhost:15672  (guest / guest)"
              exit 0
            fi
            printf "."
            sleep 1
          done
          echo ""
          echo "RabbitMQ did not become ready in time; check 'rabbitlogs'."
          exit 1
        '';

        rabbitstop = pkgs.writeShellScriptBin "rabbitstop" ''
          if docker inspect ${rabbitContainer} >/dev/null 2>&1; then
            docker stop ${rabbitContainer} >/dev/null && echo "Stopped ${rabbitContainer}"
          else
            echo "${rabbitContainer} container does not exist"
          fi
        '';

        rabbitstatus = pkgs.writeShellScriptBin "rabbitstatus" ''
          docker ps -a \
            --filter "name=${rabbitContainer}" \
            --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        '';

        rabbitlogs = pkgs.writeShellScriptBin "rabbitlogs" ''
          docker logs -f ${rabbitContainer}
        '';

        # Remove the container for a clean slate (handy when iterating)
        rabbitrm = pkgs.writeShellScriptBin "rabbitrm" ''
          if docker inspect ${rabbitContainer} >/dev/null 2>&1; then
            docker rm -f ${rabbitContainer} >/dev/null && echo "Removed ${rabbitContainer}"
          else
            echo "${rabbitContainer} container does not exist"
          fi
        '';

      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            # Go toolchain
            go
            golangci-lint
            gotools
            delve

            # RabbitMQ helpers
            rabbitstart
            rabbitstop
            rabbitstatus
            rabbitlogs
            rabbitrm
          ];

          shellHook = ''
            # Optional: the course hardcodes the connection string, but exposing
            # it lets your Go code read it via os.Getenv("RABBITMQ_URL") if you prefer.
            export RABBITMQ_URL="${amqpUrl}"

            echo "✓ ${projectName} dev environment ready"

            if ! command -v docker >/dev/null 2>&1; then
              echo "  ⚠ docker CLI not found on PATH."
              echo "    Set 'virtualisation.docker.enable = true;' on the host and"
              echo "    add your user to the 'docker' group."
            elif ! docker info >/dev/null 2>&1; then
              echo "  ⚠ docker daemon not reachable (running? are you in the 'docker' group?)."
            elif docker inspect ${rabbitContainer} >/dev/null 2>&1; then
              echo "  RabbitMQ container exists — 'rabbitstart' to (re)start, 'rabbitstatus' to check."
            else
              echo "  Run 'rabbitstart' to launch the RabbitMQ broker."
            fi

            echo "  Commands: rabbitstart, rabbitstop, rabbitstatus, rabbitlogs, rabbitrm"
            echo "  AMQP URL (\$RABBITMQ_URL): ${amqpUrl}"
            echo "  Management UI: http://localhost:15672 (guest / guest)"
          '';
        };
      }
    );
}
