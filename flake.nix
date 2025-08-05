{
	inputs = {
		nixpkgs.url = "github:nixos/nixpkgs/release-25.05";
		flake-parts.url = "github:hercules-ci/flake-parts";
	};
	outputs = inputs@{ nixpkgs, flake-parts, ... }:
		flake-parts.lib.mkFlake { inherit inputs; } {
			systems = nixpkgs.lib.platforms.unix;
			perSystem = { pkgs, ... }: {
				packages.default = pkgs.callPackage ./nix/package.nix {};
				devShells.default = pkgs.mkShell {
					name = "yacen-server-devshell";
					packages = with pkgs; [
						go
						gopls
						postgresql
					];
				};
			};
		};
}
