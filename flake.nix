{
  description = "MCP server for marc.info mailing list archive";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = {self, ...} @ inputs: let
    supportedSystems = [
      "x86_64-linux"
      "aarch64-linux"
      "x86_64-darwin"
      "aarch64-darwin"
    ];

    forEachSupportedSystem = f:
      inputs.nixpkgs.lib.genAttrs supportedSystems (
        system:
          f {
            inherit system;
            pkgs = import inputs.nixpkgs {inherit system;};
          }
      );
  in {
    packages = forEachSupportedSystem ({
      pkgs,
      system,
    }: rec {
      default = marc-mcp;

      marc-mcp = let
        lmd = self.lastModifiedDate or "19700101000000";
        buildDate = "${builtins.substring 0 4 lmd}-${builtins.substring 4 2 lmd}-${builtins.substring 6 2 lmd}T${builtins.substring 8 2 lmd}:${builtins.substring 10 2 lmd}:${builtins.substring 12 2 lmd}Z";
      in pkgs.buildGoModule {
        pname = "marc-mcp";
        version = self.shortRev or "dirty";
        src = self;

        vendorHash = "sha256-hX70z+6h9vf42y8fZu5cjf5UZzImf7hyYDBnHp2F+1Q=";

        subPackages = ["."];

        # Set CGO here (don’t use CGO_ENABLED = 0 as a derivation arg)
        env = {
          CGO_ENABLED = "0";
        };

        ldflags = [
          "-s"
          "-w"
          "-X=main.version=${self.ref or (self.shortRev or "dirty")}"
          "-X=main.commit=${self.rev or (self.shortRev or "dirty")}"
          "-X=main.date=${buildDate}"
        ];

        meta = with pkgs.lib; {
          description = "MCP server for marc.info mailing list archive";
          license = licenses.mit; # adjust if needed
          mainProgram = "marc-mcp";
        };
      };
    });

    apps = forEachSupportedSystem ({system, ...}: {
      default = {
        type = "app";
        program = "${self.packages.${system}.marc-mcp}/bin/marc-mcp";
      };
    });

    checks = forEachSupportedSystem ({system, ...}: {
      marc-mcp = self.packages.${system}.marc-mcp;
    });

    devShells = forEachSupportedSystem (
      {
        pkgs,
        system,
      }: {
        default = pkgs.mkShellNoCC {
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools
            delve
          ];

          env = {
            CGO_ENABLED = "0";
          };
        };
      }
    );

    formatter = forEachSupportedSystem ({pkgs, ...}: pkgs.nixfmt-rfc-style);
  };
}
