{ buildGoModule }:
buildGoModule {
	pname = "yacen-server";
	version = "2.2.0";
	
	src = ./.;
	
	vendorHash = null;
}
