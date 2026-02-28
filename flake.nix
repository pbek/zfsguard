{
  description = "ZFSGuard - TUI for ZFS snapshot management with health monitoring";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = builtins.replaceStrings [ "\n" " " ] [ "" "" ] (builtins.readFile ./VERSION);
      in
      {
        packages = rec {
          zfsguard = pkgs.buildGoModule {
            pname = "zfsguard";
            inherit version;
            src = ./.;

            # To update: run `nix build` and replace with the hash from the error message,
            # or set to `pkgs.lib.fakeHash` to get the correct hash.
            vendorHash = "sha256-ybSgbiKZqafvTmV04YDyBG5eI2/tJ0g8NLQckI0n31U=";

            subPackages = [
              "cmd/zfsguard"
              "cmd/zfsguard-monitor"
            ];

            ldflags = [
              "-s"
              "-w"
              "-X github.com/pbek/zfsguard/internal/version.Version=${version}"
            ];

            meta = with pkgs.lib; {
              description = "TUI for ZFS snapshot management with health monitoring and notifications";
              homepage = "https://github.com/pbek/zfsguard";
              license = licenses.gpl3Plus;
              maintainers = with lib.maintainers; [ pbek ];
              platforms = platforms.linux;
            };
          };
          default = zfsguard;
        };
      }
    )
    // {
      nixosModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          cfg = config.services.zfsguard;
          settingsFormat = pkgs.formats.yaml { };
          configFile = settingsFormat.generate "zfsguard-config.yaml" cfg.settings;
        in
        {
          options.services.zfsguard = {
            enable = lib.mkEnableOption "ZFSGuard monitoring service";

            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.system}.zfsguard;
              description = "The zfsguard package to use.";
            };

            settings = lib.mkOption {
              type = lib.types.submodule {
                freeformType = settingsFormat.type;
                options = {
                  monitor = {
                    interval_minutes = lib.mkOption {
                      type = lib.types.int;
                      default = 60;
                      description = "Interval in minutes between health checks.";
                    };
                    check_zfs = lib.mkOption {
                      type = lib.types.bool;
                      default = true;
                      description = "Whether to check ZFS pool health.";
                    };
                    check_smart = lib.mkOption {
                      type = lib.types.bool;
                      default = true;
                      description = "Whether to check SMART disk health.";
                    };
                    smart_devices = lib.mkOption {
                      type = lib.types.listOf lib.types.str;
                      default = [ ];
                      description = "List of devices to check. Empty means auto-detect.";
                    };
                  };
                  notify = {
                    shoutrrr_urls = lib.mkOption {
                      type = lib.types.listOf lib.types.str;
                      default = [ ];
                      description = ''
                        Shoutrrr notification URLs. Examples:
                        - "discord://token@id"
                        - "telegram://token@telegram?channels=channel"
                        - "gotify://host/token"
                        - "ntfy://ntfy.sh/topic"
                      '';
                    };
                    desktop = lib.mkOption {
                      type = lib.types.bool;
                      default = false;
                      description = "Enable local Linux desktop notifications via notify-send.";
                    };
                  };
                };
              };
              default = { };
              description = "ZFSGuard configuration settings.";
            };
          };

          config = lib.mkIf cfg.enable {
            systemd.services.zfsguard-monitor = {
              description = "ZFSGuard Health Monitor";
              wantedBy = [ "multi-user.target" ];
              after = [
                "zfs.target"
                "network-online.target"
              ];
              wants = [ "network-online.target" ];

              serviceConfig = {
                Type = "simple";
                ExecStart = "${cfg.package}/bin/zfsguard-monitor --config ${configFile}";
                Restart = "on-failure";
                RestartSec = "30s";

                # Security hardening
                NoNewPrivileges = false; # needs zfs commands
                ProtectHome = true;
                ProtectSystem = "strict";
                PrivateTmp = true;
                ReadOnlyPaths = [ "/" ];
                ReadWritePaths = [ "/dev" ];
              };

              path = with pkgs; [
                zfs
                smartmontools
                libnotify
              ];
            };

            # Provide the package in the system PATH for TUI usage
            environment.systemPackages = [ cfg.package ];
          };
        };
    };
}
