{ pkgs }: {
  deps = [
    pkgs.nano
    pkgs.lsof
    pkgs.go
    pkgs.curl
    pkgs.git
    pkgs.bashInteractive
    pkgs.gcc
    pkgs.gnumake
    pkgs.cloudflared
  ];

  env = {
    GOPATH = "/home/runner/${builtins.getEnv "REPL_SLUG"}/go";
  };
}
