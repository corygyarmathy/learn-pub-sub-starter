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

        # Named volume for broker state, so queues/exchanges (and a known-good
        # cookie, once created) persist across stop/start. NOTE: this does NOT
        # prevent the first-boot cookie-permission failure; that is handled by the
        # restart-once logic in rabbitstart below.
        rabbitVolume = "peril_rabbitmq_data";

        # RabbitMQ helper scripts (Docker-backed, mirroring rabbit.sh)
        rabbitstart = pkgs.writeShellScriptBin "rabbitstart" ''
          # Poll until the broker answers, printing a dot per second. Returns 1
          # immediately if the container has exited (e.g. a crashed first boot),
          # so we don't wait out the whole timer before retrying.
          wait_for_broker() {
            local i
            for i in $(seq 1 30); do
              if docker exec ${rabbitContainer} rabbitmq-diagnostics -q ping >/dev/null 2>&1; then
                return 0
              fi
              if [ "$(docker inspect -f '{{.State.Running}}' ${rabbitContainer} 2>/dev/null)" != "true" ]; then
                return 1
              fi
              printf "."
              sleep 1
            done
            return 1
          }

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
              -v ${rabbitVolume}:/var/lib/rabbitmq \
              -p 5672:5672 -p 15672:15672 \
              ${rabbitImage} >/dev/null
          fi

          # Phase 1: broker process healthy. The RabbitMQ Docker image has a
          # long-standing quirk: on a fresh container's first boot the Erlang
          # cookie is written with the wrong permissions, so boot fails with
          # ".erlang.cookie: eacces". The maintainers' documented remedy is to
          # restart, which recreates the cookie correctly. We do that once here.
          printf "Waiting for the broker"
          if ! wait_for_broker; then
            printf " ✗\n"
            echo "First boot failed (known RabbitMQ cookie-permission quirk); restarting once..."
            docker restart ${rabbitContainer} >/dev/null
            printf "Waiting for the broker"
            if ! wait_for_broker; then
              printf " ✗\n"
              echo "Broker still not up after restart; check 'rabbitlogs'."
              exit 1
            fi
          fi
          printf " ✓\n"

          # Phase 2: published port reachable from the host (mirrors the Go client's dial)
          printf "Checking host can reach 127.0.0.1:5672"
          port_ok=""
          for i in $(seq 1 15); do
            if timeout 1 bash -c "echo > /dev/tcp/127.0.0.1/5672" >/dev/null 2>&1; then
              port_ok=1
              break
            fi
            printf "."
            sleep 1
          done
          if [ -z "$port_ok" ]; then
            printf " ✗\n"
            echo "Broker is up but 127.0.0.1:5672 is not reachable from the host."
            echo "Inspect the port mapping with: docker port ${rabbitContainer}"
            exit 1
          fi
          printf " ✓\n"

          echo "  AMQP:       ${amqpUrl}"
          echo "  Management: http://localhost:15672  (guest / guest)"
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

        # Remove the container AND its data volume for a true clean slate.
        rabbitrm = pkgs.writeShellScriptBin "rabbitrm" ''
          if docker inspect ${rabbitContainer} >/dev/null 2>&1; then
            docker rm -f ${rabbitContainer} >/dev/null && echo "Removed ${rabbitContainer}"
          else
            echo "${rabbitContainer} container does not exist"
          fi
          if docker volume inspect ${rabbitVolume} >/dev/null 2>&1; then
            docker volume rm ${rabbitVolume} >/dev/null && echo "Removed volume ${rabbitVolume}"
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
