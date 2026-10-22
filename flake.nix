{
  description = "A lightweight CLI utility for automated Semantic Versioning (SemVer) management.";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      # Kept in sync by foonver's version_sync on release.
      version = "1.3.0";
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: {
        default = pkgs.buildGoModule {
          pname = "foonver";
          inherit version;

          src = self;
          vendorHash = "sha256-e6kBnZ2tRCJG3eKgDEjvHuSaSYZkmMIlFYdIpSVc6jQ=";

          ldflags = [
            "-s"
            "-w"
            "-X github.com/foonly/foonver/internal/config.AppVersion=${version}"
          ];

          nativeBuildInputs = [
            pkgs.installShellFiles
            pkgs.makeWrapper
          ];

          # Tests create temporary git repositories.
          nativeCheckInputs = [ pkgs.git ];
          preCheck = ''
            export HOME=$(mktemp -d)
            git config --global user.name "Nix Build"
            git config --global user.email "nix@localhost"
            git config --global init.defaultBranch main
          '';

          postInstall = ''
            export XDG_CONFIG_HOME=$(mktemp -d) XDG_DATA_HOME=$(mktemp -d) XDG_CACHE_HOME=$(mktemp -d)
            installShellCompletion --cmd foonver \
              --bash <($out/bin/foonver completion bash) \
              --zsh <($out/bin/foonver completion zsh) \
              --fish <($out/bin/foonver completion fish)
            wrapProgram $out/bin/foonver --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.git ]}
          '';

          meta = {
            description = "A lightweight CLI utility for automated Semantic Versioning (SemVer) management";
            homepage = "https://github.com/foonly/foonver";
            license = pkgs.lib.licenses.gpl3Only;
            mainProgram = "foonver";
          };
        };
      });

      overlays.default = final: prev: {
        foonver = self.packages.${final.system}.default;
      };
    };
}
