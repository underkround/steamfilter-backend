{ pkgs ? import (fetchTarball {
    # nixos-18.09
    url = "https://github.com/NixOS/nixpkgs-channels/archive/50f41ea2fcf86def32799f75577a4fe5cfd1132e.tar.gz";
    sha256 = "1q0bxl5nxx1kabqvyzkdw91c5dnwpi2rwsgs5jdmnj7f0qqgdxh8";
  }) {}
}:

pkgs.stdenv.mkDerivation rec {
  name = "steamfilter-backend";
  buildInputs = with pkgs; [
    awscli
    go
    nodejs
  ];
  shellHook = ''
    export SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt
  '';
}
