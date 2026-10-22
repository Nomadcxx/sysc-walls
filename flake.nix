{
  description = "Terminal screensaver using sysc-Go animations";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
      version = "1.3.0";
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.buildGoModule {
            pname = "sysc-walls";
            inherit version;
            src = self;

            vendorHash = "sha256-etw1lTGVJiJKo67rIEpuuWgpUyXeAsnx46K2YnA2/BA=";
            subPackages = [ "cmd/daemon" "cmd/display" "cmd/client" ];
            nativeBuildInputs = [ pkgs.makeWrapper pkgs.pkg-config ];
            buildInputs = [ pkgs.wayland ];
            env.CGO_ENABLED = "1";
            ldflags = [
              "-s"
              "-w"
              "-X github.com/Nomadcxx/sysc-walls/internal/version.Version=${version}"
            ];

            postInstall = ''
              mv "$out/bin/daemon" "$out/bin/sysc-walls-daemon"
              mv "$out/bin/display" "$out/bin/sysc-walls-display"
              mv "$out/bin/client" "$out/bin/sysc-walls-client"

              mkdir -p "$out/share/sysc-walls/ascii"
              cp assets/ascii/*.txt "$out/share/sysc-walls/ascii/"
              wrapProgram "$out/bin/sysc-walls-display" \
                --set SYSC_WALLS_ASCII_DIR "$out/share/sysc-walls/ascii"

              install -Dm644 README.md "$out/share/doc/sysc-walls/README.md"
              install -Dm644 THIRD_PARTY_NOTICES.md "$out/share/doc/sysc-walls/THIRD_PARTY_NOTICES.md"
              install -Dm644 TROUBLESHOOTING.md "$out/share/doc/sysc-walls/TROUBLESHOOTING.md"
              install -Dm644 LICENSE "$out/share/licenses/sysc-walls/LICENSE"
            '';

            meta = with pkgs.lib; {
              description = "Terminal screensaver using sysc-Go animations";
              homepage = "https://github.com/Nomadcxx/sysc-walls";
              license = licenses.gpl3Only;
              platforms = platforms.linux;
              mainProgram = "sysc-walls-daemon";
            };
          };
        });

      nixosModules.default = { config, lib, pkgs, ... }:
        let
          cfg = config.services.sysc-walls;
          system = pkgs.stdenv.hostPlatform.system;
        in
        {
          options.services.sysc-walls = {
            enable = lib.mkEnableOption "sysc-walls idle screensaver";
            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${system}.default;
              defaultText = lib.literalExpression "inputs.sysc-walls.packages.${pkgs.system}.default";
              description = "sysc-walls package to install and run.";
            };
          };

          config = lib.mkIf cfg.enable {
            environment.systemPackages = [ cfg.package ];

            systemd.user.services.sysc-walls = {
              description = "sysc-walls idle daemon and screensaver";
              after = [ "graphical-session.target" ];
              wantedBy = [ "graphical-session.target" ];
              path = [ cfg.package pkgs.kitty pkgs.xprintidle ];
              serviceConfig = {
                Type = "simple";
                ExecStart = "${cfg.package}/bin/sysc-walls-daemon -start";
                Restart = "on-failure";
                RestartSec = 5;
                TimeoutStartSec = 10;
                TimeoutStopSec = 30;
              };
            };
          };
        };
    };
}
